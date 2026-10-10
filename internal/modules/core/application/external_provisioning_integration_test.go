//go:build integration

package application

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIsolatedApplicationProvisioningAndMigrationRetries(t *testing.T) {
	source := os.Getenv("APEXVOID_TEST_DATABASE_URL")
	if source == "" {
		t.Skip("requires APEXVOID_TEST_DATABASE_URL and an isolated PostgreSQL test server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Database = "postgres"
	admin, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if err = admin.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	suffix := strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	appID := "photobooth" + suffix
	database, schema, role := canonicalApplicationDatabase(appID)
	platformDB := "platformtest" + suffix
	_, err = admin.Exec(ctx, "CREATE DATABASE "+quoteIdentifier(platformDB))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		_, _ = admin.Exec(cleanupCtx, "DROP DATABASE IF EXISTS "+quoteIdentifier(database)+" WITH (FORCE)")
		_, _ = admin.Exec(cleanupCtx, "DROP DATABASE IF EXISTS "+quoteIdentifier(platformDB)+" WITH (FORCE)")
		_, _ = admin.Exec(cleanupCtx, "DROP ROLE IF EXISTS "+quoteIdentifier(role))
	}()
	platformCfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	platformCfg.ConnConfig.Database = platformDB
	platform, err := pgxpool.NewWithConfig(ctx, platformCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer platform.Close()
	installation := uuid.New()
	setup := []string{
		`CREATE TABLE core_external_application_installations(id UUID PRIMARY KEY)`,
		`CREATE TABLE core_external_application_resources(application_id TEXT PRIMARY KEY,installation_id UUID NOT NULL UNIQUE REFERENCES core_external_application_installations(id),database_name TEXT UNIQUE NOT NULL,schema_name TEXT NOT NULL,role_name TEXT UNIQUE NOT NULL,encrypted_password BYTEA NOT NULL)`,
		`CREATE TABLE core_external_application_migrations(application_id TEXT NOT NULL REFERENCES core_external_application_resources(application_id),version INTEGER NOT NULL,checksum TEXT NOT NULL,applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),PRIMARY KEY(application_id,version))`,
	}
	for _, sql := range setup {
		if _, err = platform.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = platform.Exec(ctx, `INSERT INTO core_external_application_installations(id) VALUES($1)`, installation); err != nil {
		t.Fatal(err)
	}
	store := &ExternalStore{pool: platform, provisioningURL: source, provisioningKey: "integration-test-provisioning-key-at-least-32-chars", client: &http.Client{Timeout: 4 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	manifest := validManifestForTest()
	manifest.Application.ID = appID
	manifest.Service.FrontendRoute = "/apps/" + appID
	manifest.Permissions[0].Name = appID + ".report.read"
	manifest.Access.Permissions = []string{manifest.Permissions[0].Name}
	manifest.Database.Name, manifest.Database.Schema, manifest.Database.Role = database, schema, role

	created, err := store.provisionDatabase(ctx, installation, manifest)
	if err != nil {
		t.Fatal("first provisioning: ", err)
	}
	if created.Name != database || created.Schema != schema || created.Role != role {
		t.Fatal("unexpected resource scope")
	}
	second, err := store.provisionDatabase(ctx, installation, manifest)
	if err != nil {
		t.Fatal("retry provisioning: ", err)
	}
	if second.Password != created.Password {
		t.Fatal("repeated provisioning rotated database password")
	}
	if _, err = store.provisionDatabase(ctx, uuid.New(), manifest); err == nil {
		t.Fatal("another installation adopted an existing app database")
	}

	appPool, err := applicationPool(ctx, source, database, role, created.Password, schema)
	if err != nil {
		t.Fatal(err)
	}
	defer appPool.Close()
	var isSuper, canCreateDB, canCreateRole bool
	err = appPool.QueryRow(ctx, `SELECT rolsuper,rolcreatedb,rolcreaterole FROM pg_roles WHERE rolname=current_user`).Scan(&isSuper, &canCreateDB, &canCreateRole)
	if err != nil {
		t.Fatal(err)
	}
	if isSuper || canCreateDB || canCreateRole {
		t.Fatal("app role received privileged database capabilities")
	}
	if _, err = appPool.Exec(ctx, `CREATE SCHEMA forbidden_other_application`); err == nil {
		t.Fatal("application role created an unauthorized schema")
	}
	var actualOwner string
	if err = appPool.QueryRow(ctx, `SELECT pg_get_userbyid(nspowner) FROM pg_namespace WHERE nspname=$1`, schema).Scan(&actualOwner); err != nil {
		t.Fatal(err)
	}
	if actualOwner != role {
		t.Fatal("app schema not owned by its role")
	}
	var databaseOwner string
	if err = admin.QueryRow(ctx, `SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname=$1`, database).Scan(&databaseOwner); err != nil {
		t.Fatal(err)
	}
	if databaseOwner == role {
		t.Fatal("application login unexpectedly owns the entire database")
	}

	sql := fmt.Sprintf("CREATE TABLE %s.orders (id INTEGER PRIMARY KEY, description TEXT NOT NULL);", quoteIdentifier(schema))
	manifest.Migrations = []ExternalMigration{{Version: 1, Path: "/.well-known/apexvoid/migrations/001.sql", SHA256: migrationChecksum([]byte(sql))}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-ApexVoid-Enrollment-Code") != "" {
			t.Error("migration fetch leaked secret")
		}
		if r.URL.Path != "/.well-known/apexvoid/migrations/001.sql" {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(sql))
	}))
	defer server.Close()
	if err = store.applyMigrations(ctx, created, server.URL, manifest); err != nil {
		t.Fatal("first migration: ", err)
	}
	if err = store.applyMigrations(ctx, created, server.URL, manifest); err != nil {
		t.Fatal("idempotent migration retry: ", err)
	}
	var appVersions, platformVersions int
	if err = appPool.QueryRow(ctx, `SELECT count(*) FROM `+quoteIdentifier(schema)+`._apexvoid_migration_history`).Scan(&appVersions); err != nil {
		t.Fatal(err)
	}
	if err = platform.QueryRow(ctx, `SELECT count(*) FROM core_external_application_migrations WHERE application_id=$1`, appID).Scan(&platformVersions); err != nil {
		t.Fatal(err)
	}
	if appVersions != 1 || platformVersions != 1 {
		t.Fatalf("migration ledgers diverged: app=%d platform=%d", appVersions, platformVersions)
	}
	manifest.Migrations[0].SHA256 = strings.Repeat("a", 64)
	if err = store.applyMigrations(ctx, created, server.URL, manifest); err == nil {
		t.Fatal("modified migration artifact was accepted")
	}
	manifest.Database.Name = "apexvoid"
	if _, err = store.provisionDatabase(ctx, installation, manifest); err == nil {
		t.Fatal("app attempted to claim Enterprise database")
	}
}
