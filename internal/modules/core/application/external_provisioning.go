package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var migrationForbiddenPattern = regexp.MustCompile(`(?i)(drop\s|truncate\s|alter\s+(system|database|role)|create\s+(database|role|schema)|grant\s|revoke\s|create\s+extension|copy\s+[^;]*\s+program|pg_(read_file|write_file|execute_server_program)|set\s|security\s+definer|\b(public|pg_catalog|core_|workspace_|apexvoid_)\w*\.)`)

type provisionedDatabase struct {
	Name     string
	Schema   string
	Role     string
	Password string
}

func (s *ExternalStore) provisionDatabase(ctx context.Context, manifest ExternalManifest) (provisionedDatabase, error) {
	if strings.TrimSpace(s.provisioningURL) == "" {
		return provisionedDatabase{}, errors.New("database provisioning is not configured; set DATABASE_PROVISIONING_URL")
	}
	admin, err := pgxpool.New(ctx, s.provisioningURL)
	if err != nil {
		return provisionedDatabase{}, fmt.Errorf("connect provisioning database: %w", err)
	}
	defer admin.Close()
	if err := admin.Ping(ctx); err != nil {
		return provisionedDatabase{}, fmt.Errorf("ping provisioning database: %w", err)
	}
	name, schema, role := manifest.Database.Name, manifest.Database.Schema, manifest.Database.Role
	var currentOwner string
	err = admin.QueryRow(ctx, `SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname=$1`, name).Scan(&currentOwner)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return provisionedDatabase{}, err
	}
	if err == nil && currentOwner != role {
		return provisionedDatabase{}, fmt.Errorf("database %q already exists and is not owned by the declared role", name)
	}
	var roleSuper, roleCreateDB, roleCreateRole bool
	roleErr := admin.QueryRow(ctx, `SELECT rolsuper,rolcreatedb,rolcreaterole FROM pg_roles WHERE rolname=$1`, role).Scan(&roleSuper, &roleCreateDB, &roleCreateRole)
	if roleErr != nil && !errors.Is(roleErr, pgx.ErrNoRows) {
		return provisionedDatabase{}, roleErr
	}
	if roleErr == nil && (roleSuper || roleCreateDB || roleCreateRole) {
		return provisionedDatabase{}, errors.New("declared application role has unsafe PostgreSQL privileges")
	}
	password, err := newCredential()
	if err != nil {
		return provisionedDatabase{}, err
	}
	if errors.Is(roleErr, pgx.ErrNoRows) {
		if _, err = admin.Exec(ctx, `CREATE ROLE `+quoteIdentifier(role)+` LOGIN PASSWORD `+quoteLiteral(password)+` NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT`); err != nil {
			return provisionedDatabase{}, fmt.Errorf("create application role: %w", err)
		}
	} else if _, err = admin.Exec(ctx, `ALTER ROLE `+quoteIdentifier(role)+` LOGIN PASSWORD `+quoteLiteral(password)+` NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT`); err != nil {
		return provisionedDatabase{}, fmt.Errorf("rotate application role password: %w", err)
	}
	if currentOwner == "" {
		if _, err = admin.Exec(ctx, `CREATE DATABASE `+quoteIdentifier(name)+` OWNER `+quoteIdentifier(role)); err != nil {
			return provisionedDatabase{}, fmt.Errorf("create application database: %w", err)
		}
	}
	appPool, err := applicationPool(ctx, s.provisioningURL, name, role, password, schema)
	if err != nil {
		return provisionedDatabase{}, err
	}
	defer appPool.Close()
	if _, err = appPool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+quoteIdentifier(schema)+` AUTHORIZATION `+quoteIdentifier(role)); err != nil {
		return provisionedDatabase{}, fmt.Errorf("create application schema: %w", err)
	}
	if _, err = appPool.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+quoteIdentifier(schema)+`._apexvoid_migration_history (version INTEGER PRIMARY KEY, path TEXT NOT NULL, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		return provisionedDatabase{}, fmt.Errorf("create migration ledger: %w", err)
	}
	return provisionedDatabase{Name: name, Schema: schema, Role: role, Password: password}, nil
}

func (s *ExternalStore) applyMigrations(ctx context.Context, database provisionedDatabase, serviceURL string, manifest ExternalManifest, enrollmentCode string) error {
	appPool, err := applicationPool(ctx, s.provisioningURL, database.Name, database.Role, database.Password, database.Schema)
	if err != nil {
		return err
	}
	defer appPool.Close()
	for _, migration := range manifest.Migrations {
		endpoint, err := manifestEndpoint(serviceURL, migration.Path)
		if err != nil {
			return err
		}
		body, err := httpRequestLimit(ctx, s.client, http.MethodGet, endpoint, enrollmentCode, nil, maxMigrationBytes)
		if err != nil {
			return fmt.Errorf("fetch migration %d: %w", migration.Version, err)
		}
		if len(body) > maxMigrationBytes {
			return fmt.Errorf("migration %d exceeds size limit", migration.Version)
		}
		if migrationChecksum(body) != strings.ToLower(migration.SHA256) {
			return fmt.Errorf("migration %d checksum does not match the approved manifest", migration.Version)
		}
		if migrationForbiddenPattern.Match(body) {
			return fmt.Errorf("migration %d: %w", migration.Version, ErrMigrationPolicy)
		}
		var previousChecksum string
		ledgerPath := fmt.Sprintf("%s._apexvoid_migration_history", quoteIdentifier(database.Schema))
		queryErr := appPool.QueryRow(ctx, `SELECT checksum FROM `+ledgerPath+` WHERE version=$1`, migration.Version).Scan(&previousChecksum)
		if queryErr == nil {
			if previousChecksum != strings.ToLower(migration.SHA256) {
				return fmt.Errorf("migration %d checksum differs from the applied ledger", migration.Version)
			}
			continue
		}
		if !errors.Is(queryErr, pgx.ErrNoRows) {
			return queryErr
		}
		tx, err := appPool.Begin(ctx)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, string(body))
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO `+ledgerPath+`(version,path,checksum) VALUES($1,$2,$3)`, migration.Version, migration.Path, strings.ToLower(migration.SHA256))
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %d: %w", migration.Version, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func applicationPool(ctx context.Context, rawURL, database, role, password, schema string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(rawURL)
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	parsed.Path = "/" + database
	config, err = pgxpool.ParseConfig(parsed.String())
	if err != nil {
		return nil, err
	}
	config.ConnConfig.User = role
	config.ConnConfig.Password = password
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.MaxConns = 2
	config.MinConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(checkCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect application database: %w", err)
	}
	return pool, nil
}

func quoteIdentifier(value string) string { return `"` + strings.ReplaceAll(value, `"`, `""`) + `"` }
func quoteLiteral(value string) string    { return `'` + strings.ReplaceAll(value, `'`, `''`) + `'` }
