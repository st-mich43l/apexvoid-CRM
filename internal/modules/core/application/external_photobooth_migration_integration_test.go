//go:build integration

package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	coremigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/migrations"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

// The Photobooth repository is checked out by CI at the immutable revision recorded
// below. Keeping the source external prevents Enterprise from silently
// rewriting a published application migration while still exercising the
// exact artifacts used by the service.
const (
	photoboothMigrationRevision = "29e9e25f8fb687daa35d1d766cec4adcb718180d"
	photoboothMigration001Hash  = "49b5d9d0a1b1c5eeaf6f39f0d8284db8d8bc98d2688468fe933c2da6ebaea385"
	photoboothMigration002Hash  = "87ae9ab2a8525e58a988e619fa8348cf6cccfd0bfd29288aaeee4d391c93b4e0"
)

func TestExactPhotoboothMigrationArtifactsPassEnterprisePolicy(t *testing.T) {
	repository := os.Getenv("APEXVOID_PHOTOBOOTH_REPOSITORY")
	if repository == "" {
		t.Skip("set APEXVOID_PHOTOBOOTH_REPOSITORY to the pinned Photobooth repository checkout")
	}
	for _, item := range []struct {
		version int
		name    string
		hash    string
	}{
		{version: 1, name: "001_photobooth.sql", hash: photoboothMigration001Hash},
		{version: 2, name: "002_advanced_booking.sql", hash: photoboothMigration002Hash},
	} {
		body, err := os.ReadFile(filepath.Join(repository, "db", "migrations", item.name))
		if err != nil {
			t.Fatalf("read Photobooth migration %d from revision %s: %v", item.version, photoboothMigrationRevision, err)
		}
		actual := fmt.Sprintf("%x", sha256.Sum256(body))
		if actual != item.hash {
			t.Fatalf("Photobooth migration %d checksum changed: got %s, want %s", item.version, actual, item.hash)
		}
		if err := validateMigrationSQL(body); err != nil {
			t.Fatalf("Photobooth migration %d violates Enterprise policy: %v", item.version, err)
		}
	}
}

func TestExactPhotoboothUpgradeApprovalWorkflow(t *testing.T) {
	repository := os.Getenv("APEXVOID_PHOTOBOOTH_REPOSITORY")
	source := os.Getenv("APEXVOID_TEST_DATABASE_URL")
	if repository == "" || source == "" {
		t.Skip("requires APEXVOID_PHOTOBOOTH_REPOSITORY and APEXVOID_TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	readMigration := func(name, expected string) []byte {
		body, err := os.ReadFile(filepath.Join(repository, "db", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		if actual := fmt.Sprintf("%x", sha256.Sum256(body)); actual != expected {
			t.Fatalf("Photobooth %s checksum changed: got %s, want %s", name, actual, expected)
		}
		if err := validateMigrationSQL(body); err != nil {
			t.Fatalf("Photobooth %s violates Enterprise policy: %v", name, err)
		}
		return body
	}
	migration001 := readMigration("001_photobooth.sql", photoboothMigration001Hash)
	migration002 := readMigration("002_advanced_booking.sql", photoboothMigration002Hash)

	adminConfig, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	adminConfig.ConnConfig.Database = "postgres"
	admin, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	suffix := strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	platformDB := "apexvoid_photobooth_test_" + suffix
	const applicationID = "photobooth"
	const databaseName = "apexvoid_photobooth"
	const schemaName = "photobooth"
	const roleName = "apexvoid_photobooth"
	for _, name := range []string{databaseName, roleName} {
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`
		if name == roleName {
			query = `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$1)`
		}
		if err := admin.QueryRow(ctx, query, name).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Skipf("refusing to use existing Photobooth database or role %q", name)
		}
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+quoteIdentifier(platformDB)); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		_, _ = admin.Exec(cleanupCtx, `DROP DATABASE IF EXISTS `+quoteIdentifier(databaseName)+` WITH (FORCE)`)
		_, _ = admin.Exec(cleanupCtx, `DROP DATABASE IF EXISTS `+quoteIdentifier(platformDB)+` WITH (FORCE)`)
		_, _ = admin.Exec(cleanupCtx, `DROP ROLE IF EXISTS `+quoteIdentifier(roleName))
	}()
	platformURL, err := databaseURLForTest(source, platformDB)
	if err != nil {
		t.Fatal(err)
	}
	platformConfig, err := pgxpool.ParseConfig(platformURL)
	if err != nil {
		t.Fatal(err)
	}
	platform, err := pgxpool.NewWithConfig(ctx, platformConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer platform.Close()
	if err := database.NewMigrationRunner(platform, coremigrations.All()).Up(ctx); err != nil {
		t.Fatal(err)
	}

	credential := strings.Repeat("c", 64)
	var next ExternalManifest
	var updateRequests atomic.Int32
	releaseFirstDiscovery := make(chan struct{})
	firstDiscoveryStarted := make(chan struct{})
	discoveryDone := make(chan struct{})
	releasedDiscovery := false
	defer func() {
		if !releasedDiscovery {
			close(releaseFirstDiscovery)
		}
	}()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ExternalManifestPath:
			requestNumber := updateRequests.Add(1)
			if requestNumber == 2 {
				close(firstDiscoveryStarted)
				<-releaseFirstDiscovery
			}
			raw, _ := json.Marshal(next)
			mac := hmac.New(sha256.New, mustDecodeHex(hashCredential(credential)))
			_, _ = mac.Write([]byte(upgradeManifestContext + r.Header.Get("X-ApexVoid-Update-Challenge") + "\n"))
			_, _ = mac.Write(raw)
			w.Header().Set("X-ApexVoid-Update-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(raw)
		case "/.well-known/apexvoid/migrations/001_photobooth.sql":
			_, _ = w.Write(migration001)
		case "/.well-known/apexvoid/migrations/002_advanced_booking.sql":
			_, _ = w.Write(migration002)
		default:
			http.NotFound(w, r)
		}
	}))
	server.Start()
	defer server.Close()
	_, port, _ := net.SplitHostPort(server.Listener.Addr().String())
	serviceURL := "http://photobooth:" + port
	initial := ExternalManifest{
		ManifestVersion: SupportedExternalContractVersion,
		Application:     ManifestApplication{ID: applicationID, DisplayName: "ApexVoid Photobooth", Description: "Coffee counter and photo booth booking", Version: "0.1.0", APIContractVersion: "v1"},
		Service:         ManifestService{Identity: "photobooth-service", HealthPath: "/health", EnrollmentPath: "/.well-known/apexvoid/enroll", FrontendRoute: "/apps/photobooth", APIRoute: "/api"},
		Database:        ManifestDatabase{Name: databaseName, Schema: schemaName, Role: roleName, MigrationBundleVersion: "0.1.0"},
		Permissions:     []ExternalPermission{{Name: "photobooth.booking.read", DisplayName: "Read bookings", Scope: permission.ScopeWorkspace}},
		Access:          frameworkapplication.PermissionPolicy{Match: frameworkapplication.PermissionMatchAll, Permissions: []string{"photobooth.booking.read"}},
		Migrations:      []ExternalMigration{{Version: 1, Path: "/.well-known/apexvoid/migrations/001_photobooth.sql", SHA256: photoboothMigration001Hash}},
	}
	next = initial
	next.Application.Version = "0.2.0"
	next.Database.MigrationBundleVersion = "0.2.0"
	next.Migrations = append(next.Migrations, ExternalMigration{Version: 2, Path: "/.well-known/apexvoid/migrations/002_advanced_booking.sql", SHA256: photoboothMigration002Hash})

	clientTransport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
	}}
	store := NewExternalStore(platform, permission.NewRegistry(), nil, source, strings.Repeat("p", 48))
	store.client = &http.Client{Transport: clientTransport, Timeout: 5 * time.Second}
	_, _, err = store.Register(ctx, RegisterExternalInput{Application: ExternalApplication{ID: applicationID, DisplayName: initial.Application.DisplayName, Description: initial.Application.Description, Version: initial.Application.Version, APIContractVersion: initial.Application.APIContractVersion, ServiceIdentity: initial.Service.Identity, ServiceEndpoint: serviceURL, HealthEndpoint: serviceURL + "/health", FrontendRoute: initial.Service.FrontendRoute, Access: initial.Access, Permissions: initial.Permissions}, Credential: credential})
	if err != nil {
		t.Fatal(err)
	}
	installationID := uuid.New()
	if _, err := platform.Exec(ctx, `INSERT INTO core_external_application_installations(id,application_id,service_url,manifest,enrollment_code_hash,status,expires_at,last_step,error_message,selected_workspace_ids) VALUES($1,$2,$3,$4,$5,'active',NOW()+INTERVAL '1 hour','active','', '[]'::jsonb)`, installationID, applicationID, serviceURL, mustJSON(initial), hashCredential("installation-code")); err != nil {
		t.Fatal(err)
	}
	resource, err := store.provisionDatabase(ctx, installationID, initial)
	if err != nil {
		t.Fatal("provision Photobooth database: ", err)
	}
	if err := store.applyMigrations(ctx, resource, serviceURL, initial); err != nil {
		t.Fatal("apply Photobooth migration 001: ", err)
	}
	if _, err := platform.Exec(ctx, `UPDATE core_external_applications SET installation_id=$2,database_name=$3,database_schema=$4,database_role=$5,migration_bundle_version=$6,installed_manifest=$7 WHERE id=$1`, applicationID, installationID, resource.Name, resource.Schema, resource.Role, initial.Database.MigrationBundleVersion, mustJSON(initial)); err != nil {
		t.Fatal(err)
	}
	appPool, err := applicationPool(ctx, source, resource.Name, resource.Role, resource.Password, resource.Schema)
	if err != nil {
		t.Fatal(err)
	}
	defer appPool.Close()
	workspaceID, boothID, itemID, bookingID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	if _, err := appPool.Exec(ctx, `INSERT INTO photobooth.photobooth_items(id,workspace_id,sku,name,kind,price_vnd,created_by) VALUES($1,$2,'legacy-package','Legacy package','photo',100000,$3)`, itemID, workspaceID, actorID); err != nil {
		t.Fatal("seed legacy item: ", err)
	}
	if _, err := appPool.Exec(ctx, `INSERT INTO photobooth.photobooth_booths(id,workspace_id,name,created_by) VALUES($1,$2,'Legacy booth',$3)`, boothID, workspaceID, actorID); err != nil {
		t.Fatal("seed legacy booth: ", err)
	}
	if _, err := appPool.Exec(ctx, `INSERT INTO photobooth.photobooth_bookings(id,workspace_id,booth_id,package_id,guest_name,start_at,end_at,status,package_name,price_vnd,created_by) VALUES($1,$2,$3,$4,'Legacy guest',NOW()+INTERVAL '1 day',NOW()+INTERVAL '1 day 1 hour','reserved','Legacy package',100000,$5)`, bookingID, workspaceID, boothID, itemID, actorID); err != nil {
		t.Fatal("seed legacy booking: ", err)
	}
	preview, err := store.PreviewUpdate(ctx, applicationID)
	if err != nil {
		t.Fatal("create Photobooth upgrade review: ", err)
	}
	if len(preview.PendingMigrations) != 1 || preview.PendingMigrations[0].Version != 2 {
		t.Fatalf("unexpected Photobooth upgrade preview: %#v", preview.PendingMigrations)
	}
	go func() {
		defer close(discoveryDone)
		store.discoverUpdates(ctx)
	}()
	select {
	case <-firstDiscoveryStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("discovery did not start before approval")
	}
	approved, err := store.ApproveUpdate(ctx, applicationID, preview.ID, true, true, preview.ManifestSHA256)
	if err != nil {
		t.Fatal("approve Photobooth upgrade: ", err)
	}
	close(releaseFirstDiscovery)
	releasedDiscovery = true
	<-discoveryDone
	if approved.Status != "applied" {
		t.Fatalf("approval status = %q, want applied", approved.Status)
	}
	var version, bundle, availableVersion string
	var available bool
	if err := platform.QueryRow(ctx, `SELECT version,migration_bundle_version,update_available,available_version FROM core_external_applications WHERE id=$1`, applicationID).Scan(&version, &bundle, &available, &availableVersion); err != nil {
		t.Fatal(err)
	}
	if version != "0.2.0" || bundle != "0.2.0" || available || availableVersion != "0.2.0" {
		t.Fatalf("installed Photobooth state = version %s bundle %s available=%v available_version=%s", version, bundle, available, availableVersion)
	}
	var historyCount, platformLedgerCount, eventCount int
	if err := appPool.QueryRow(ctx, `SELECT COUNT(*) FROM photobooth._apexvoid_migration_history`).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if err := platform.QueryRow(ctx, `SELECT COUNT(*) FROM core_external_application_migrations WHERE application_id=$1`, applicationID).Scan(&platformLedgerCount); err != nil {
		t.Fatal(err)
	}
	if err := appPool.QueryRow(ctx, `SELECT COUNT(*) FROM photobooth.photobooth_booking_events`).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 2 || platformLedgerCount != 2 || eventCount != 1 {
		t.Fatalf("Photobooth upgrade ledgers/events = app %d platform %d events %d", historyCount, platformLedgerCount, eventCount)
	}
	var bookingStatus, bookingRef string
	if err := appPool.QueryRow(ctx, `SELECT status,booking_ref FROM photobooth.photobooth_bookings WHERE id=$1`, bookingID).Scan(&bookingStatus, &bookingRef); err != nil {
		t.Fatal(err)
	}
	if bookingStatus != "confirmed" || bookingRef == "" {
		t.Fatalf("legacy booking was not upgraded: status=%s ref=%q", bookingStatus, bookingRef)
	}
}

func databaseURLForTest(raw, databaseName string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	parsed.Path = "/" + databaseName
	return parsed.String(), nil
}

func mustDecodeHex(value string) []byte {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return decoded
}
