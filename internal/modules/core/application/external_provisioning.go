package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type provisionedDatabase struct {
	Name     string
	Schema   string
	Role     string
	Password string
}

func canonicalApplicationDatabase(id string) (name, schema, role string) {
	schema = strings.NewReplacer("-", "_", ".", "_").Replace(id)
	name = "apexvoid_" + schema
	role = name
	return
}

// Each external application has exactly one ownership record and one
// installation identity. A pre-existing unclaimed database is NEVER adopted.
// The encrypted role password is stable across failed attempts and retries.
func (s *ExternalStore) provisionDatabase(ctx context.Context, installationID uuid.UUID, manifest ExternalManifest) (provisionedDatabase, error) {
	if s.provisioningURL == "" || len(s.provisioningKey) < 32 {
		return provisionedDatabase{}, errors.New("database provisioning is not configured; set DATABASE_PROVISIONING_URL and DATABASE_PROVISIONING_KEY")
	}
	name, schema, role := canonicalApplicationDatabase(manifest.Application.ID)
	if manifest.Database.Name != name || manifest.Database.Schema != schema || manifest.Database.Role != role || len(name) > 63 {
		return provisionedDatabase{}, errors.New("app database, schema and role must match its canonical application identity")
	}
	admin, err := pgxpool.New(ctx, s.provisioningURL)
	if err != nil {
		return provisionedDatabase{}, fmt.Errorf("connect database provisioner: %w", err)
	}
	defer admin.Close()
	if err = admin.Ping(ctx); err != nil {
		return provisionedDatabase{}, fmt.Errorf("ping database provisioner: %w", err)
	}

	var storedInstallation uuid.UUID
	var storedDB, storedSchema, storedRole string
	var encrypted []byte
	err = s.pool.QueryRow(ctx, `SELECT installation_id,database_name,schema_name,role_name,encrypted_password
		FROM core_external_application_resources WHERE application_id=$1`, manifest.Application.ID).
		Scan(&storedInstallation, &storedDB, &storedSchema, &storedRole, &encrypted)
	newClaim := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !newClaim {
		return provisionedDatabase{}, err
	}
	password := ""
	if newClaim {
		// Never take over databases/roles created by other apps or operators.
		var dbExists, roleExists bool
		if err := admin.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`, name).Scan(&dbExists); err != nil {
			return provisionedDatabase{}, err
		}
		if err := admin.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$1)`, role).Scan(&roleExists); err != nil {
			return provisionedDatabase{}, err
		}
		if dbExists || roleExists {
			return provisionedDatabase{}, errors.New("database or role already exists without a verified application ownership claim")
		}
		password, err = newCredential()
		if err != nil {
			return provisionedDatabase{}, err
		}
		encrypted, err = protectProvisioningPassword(s.provisioningKey, password)
		if err != nil {
			return provisionedDatabase{}, err
		}
		_, err = s.pool.Exec(ctx, `INSERT INTO core_external_application_resources(application_id,installation_id,database_name,schema_name,role_name,encrypted_password)
			VALUES($1,$2,$3,$4,$5,$6)`, manifest.Application.ID, installationID, name, schema, role, encrypted)
		if err != nil {
			return provisionedDatabase{}, fmt.Errorf("reserve exclusive database ownership: %w", err)
		}
	} else {
		if storedDB != name || storedSchema != schema || storedRole != role {
			return provisionedDatabase{}, errors.New("application resource ownership does not match the application manifest")
		}
		if storedInstallation != installationID {
			var previousStatus string
			if err = s.pool.QueryRow(ctx, `SELECT status FROM core_external_application_installations WHERE id=$1`, storedInstallation).Scan(&previousStatus); err != nil {
				return provisionedDatabase{}, err
			}
			if previousStatus != "failed" {
				return provisionedDatabase{}, errors.New("application resource ownership belongs to another active installation")
			}
			if _, err = s.pool.Exec(ctx, `UPDATE core_external_application_resources SET installation_id=$2 WHERE application_id=$1 AND installation_id=$3`, manifest.Application.ID, installationID, storedInstallation); err != nil {
				return provisionedDatabase{}, fmt.Errorf("transfer failed application resource ownership: %w", err)
			}
		}
		password, err = recoverProvisioningPassword(s.provisioningKey, encrypted)
		if err != nil {
			return provisionedDatabase{}, err
		}
	}
	var roleSuper, roleCreateDB, roleCreateRole, roleLogin bool
	err = admin.QueryRow(ctx, `SELECT rolsuper,rolcreatedb,rolcreaterole,rolcanlogin FROM pg_roles WHERE rolname=$1`, role).Scan(&roleSuper, &roleCreateDB, &roleCreateRole, &roleLogin)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = admin.Exec(ctx, `CREATE ROLE `+quoteIdentifier(role)+` LOGIN PASSWORD `+quoteLiteral(password)+` NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT`)
		if err != nil {
			return provisionedDatabase{}, fmt.Errorf("create isolated app role: %w", err)
		}
	} else if err != nil {
		return provisionedDatabase{}, err
	} else if roleSuper || roleCreateDB || roleCreateRole || !roleLogin {
		return provisionedDatabase{}, errors.New("existing app role has elevated or invalid PostgreSQL privileges")
	}
	var provisioner string
	if err = admin.QueryRow(ctx, `SELECT current_user`).Scan(&provisioner); err != nil {
		return provisionedDatabase{}, err
	}
	var roleMemberships bool
	err = admin.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=(SELECT oid FROM pg_roles WHERE rolname=$1))`, role).Scan(&roleMemberships)
	if err != nil {
		return provisionedDatabase{}, err
	}
	if roleMemberships {
		return provisionedDatabase{}, errors.New("application database role must not be a member of other roles")
	}
	var provisionerHasRole bool
	err = admin.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1
		FROM pg_auth_members membership
		JOIN pg_roles member ON member.oid = membership.member
		JOIN pg_roles granted ON granted.oid = membership.roleid
		WHERE member.rolname=$1 AND granted.rolname=$2 AND membership.set_option
	)`, provisioner, role).Scan(&provisionerHasRole)
	if err != nil {
		return provisionedDatabase{}, err
	}
	if !provisionerHasRole {
		if _, err = admin.Exec(ctx, `GRANT `+quoteIdentifier(role)+` TO `+quoteIdentifier(provisioner)+` WITH SET TRUE`); err != nil {
			return provisionedDatabase{}, fmt.Errorf("grant provisioner access to app role: %w", err)
		}
		defer func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = admin.Exec(cleanupCtx, `REVOKE `+quoteIdentifier(role)+` FROM `+quoteIdentifier(provisioner))
		}()
	}

	var dbOwner string
	err = admin.QueryRow(ctx, `SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname=$1`, name).Scan(&dbOwner)
	if errors.Is(err, pgx.ErrNoRows) {
		// The provisioning administrator owns the DB. The runtime role does not
		// receive database CREATE, ensuring it cannot own arbitrary schemas.
		_, err = admin.Exec(ctx, `CREATE DATABASE `+quoteIdentifier(name))
		if err != nil {
			return provisionedDatabase{}, fmt.Errorf("create app database: %w", err)
		}
	} else if err != nil {
		return provisionedDatabase{}, err
	} else {
		var provisioner string
		if err = admin.QueryRow(ctx, `SELECT current_user`).Scan(&provisioner); err != nil {
			return provisionedDatabase{}, err
		}
		if dbOwner != provisioner {
			return provisionedDatabase{}, errors.New("application database owner is not the approved provisioning role")
		}
	}
	_, err = admin.Exec(ctx, `REVOKE ALL ON DATABASE `+quoteIdentifier(name)+` FROM PUBLIC`)
	if err != nil {
		return provisionedDatabase{}, err
	}
	_, err = admin.Exec(ctx, `GRANT CONNECT ON DATABASE `+quoteIdentifier(name)+` TO `+quoteIdentifier(role))
	if err != nil {
		return provisionedDatabase{}, err
	}

	adminConfig, err := pgxpool.ParseConfig(s.provisioningURL)
	if err != nil {
		return provisionedDatabase{}, err
	}
	adminConfig.ConnConfig.Database = name
	adminConfig.MaxConns = 2
	adminConfig.MinConns = 0
	dbAdmin, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		return provisionedDatabase{}, err
	}
	defer dbAdmin.Close()
	if err = dbAdmin.Ping(ctx); err != nil {
		return provisionedDatabase{}, err
	}
	if _, err = dbAdmin.Exec(ctx, `REVOKE CREATE ON SCHEMA public FROM PUBLIC`); err != nil {
		return provisionedDatabase{}, err
	}
	if _, err = dbAdmin.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+quoteIdentifier(schema)); err != nil {
		return provisionedDatabase{}, err
	}
	if _, err = dbAdmin.Exec(ctx, `ALTER SCHEMA `+quoteIdentifier(schema)+` OWNER TO `+quoteIdentifier(role)); err != nil {
		return provisionedDatabase{}, err
	}
	var schemaOwner string
	err = dbAdmin.QueryRow(ctx, `SELECT pg_get_userbyid(nspowner) FROM pg_namespace WHERE nspname=$1`, schema).Scan(&schemaOwner)
	if err != nil {
		return provisionedDatabase{}, err
	}
	if schemaOwner != role {
		return provisionedDatabase{}, errors.New("app schema is not owned by its isolated role")
	}
	appPool, err := applicationPool(ctx, s.provisioningURL, name, role, password, schema)
	if err != nil {
		return provisionedDatabase{}, err
	}
	defer appPool.Close()
	if _, err = appPool.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+quoteIdentifier(schema)+`._apexvoid_migration_history (version INTEGER PRIMARY KEY, path TEXT NOT NULL, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		return provisionedDatabase{}, err
	}
	return provisionedDatabase{Name: name, Schema: schema, Role: role, Password: password}, nil
}

func (s *ExternalStore) applyMigrations(ctx context.Context, database provisionedDatabase, serviceURL string, manifest ExternalManifest) error {
	appPool, err := applicationPool(ctx, s.provisioningURL, database.Name, database.Role, database.Password, database.Schema)
	if err != nil {
		return err
	}
	defer appPool.Close()
	ledgerPath := quoteIdentifier(database.Schema) + "._apexvoid_migration_history"
	for _, migration := range manifest.Migrations {
		endpoint, err := manifestEndpoint(serviceURL, migration.Path)
		if err != nil {
			return newMigrationFailure(MigrationFetchFailed, migration.Version, migration.Path, "The migration path is invalid and could not be retrieved.", err)
		}
		body, err := httpRequestLimit(ctx, s.client, http.MethodGet, endpoint, "", nil, maxMigrationBytes)
		if err != nil {
			return newMigrationFailure(MigrationFetchFailed, migration.Version, migration.Path, "The application service did not return the migration SQL.", fmt.Errorf("%w: %v", ErrMigrationFetch, err))
		}
		if len(body) > maxMigrationBytes {
			return newMigrationFailure(MigrationFetchFailed, migration.Version, migration.Path, "The migration SQL exceeds the maximum allowed size.", ErrMigrationFetch)
		}
		checksum := strings.ToLower(migration.SHA256)
		if migrationChecksum(body) != checksum {
			return newMigrationFailure(MigrationChecksumMismatch, migration.Version, migration.Path, "Downloaded bytes differ from the checksum pinned in the signed manifest.", ErrMigrationChecksum)
		}
		if policyErr := validateMigrationSQL(body); policyErr != nil {
			return newMigrationFailure(MigrationPolicyRejected, migration.Version, migration.Path, "The SQL contains a privileged, session-changing, server-side, or cross-application operation.", fmt.Errorf("%w: %v", ErrMigrationPolicy, policyErr))
		}
		tx, err := appPool.Begin(ctx)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('apexvoid_migration_install'))`)
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		var previous string
		err = tx.QueryRow(ctx, `SELECT checksum FROM `+ledgerPath+` WHERE version=$1`, migration.Version).Scan(&previous)
		if err == nil {
			_ = tx.Rollback(ctx)
			if previous != checksum {
				return fmt.Errorf("migration %d checksum changed after execution", migration.Version)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			return err
		} else {
			_, err = tx.Exec(ctx, string(body))
			if err == nil {
				_, err = tx.Exec(ctx, `INSERT INTO `+ledgerPath+`(version,path,checksum) VALUES($1,$2,$3)`, migration.Version, migration.Path, checksum)
			}
			if err != nil {
				_ = tx.Rollback(ctx)
				return migrationExecutionFailure(migration.Version, migration.Path, err)
			}
			if err = tx.Commit(ctx); err != nil {
				return err
			}
		}
		// The platform ledger is a second, independent integrity record.
		var platformChecksum string
		err = s.pool.QueryRow(ctx, `SELECT checksum FROM core_external_application_migrations WHERE application_id=$1 AND version=$2`, manifest.Application.ID, migration.Version).Scan(&platformChecksum)
		if err == nil && platformChecksum != checksum {
			return fmt.Errorf("platform checksum changed for migration %d", migration.Version)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			_, err = s.pool.Exec(ctx, `INSERT INTO core_external_application_migrations(application_id,version,checksum) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, manifest.Application.ID, migration.Version, checksum)
		}
		if err != nil {
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
	config.ConnConfig.Database = database
	config.ConnConfig.User = role
	config.ConnConfig.Password = password
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.ConnConfig.RuntimeParams["statement_timeout"] = "30000"
	config.ConnConfig.RuntimeParams["lock_timeout"] = "5000"
	config.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "30000"
	config.MaxConns = 2
	config.MinConns = 0
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = pool.Ping(checkCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect application database: %w", err)
	}
	return pool, nil
}

func quoteIdentifier(value string) string { return `"` + strings.ReplaceAll(value, `"`, `""`) + `"` }
func quoteLiteral(value string) string    { return `'` + strings.ReplaceAll(value, `'`, `''`) + `'` }
