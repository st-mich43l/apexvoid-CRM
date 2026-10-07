package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
)

var ErrNoMigrationChange = errors.New("no migration change")

type MigrationRunner struct {
	pool       *pgxpool.Pool
	migrations []module.Migration
}

func NewMigrationRunner(pool *pgxpool.Pool, migrations []module.Migration) *MigrationRunner {
	return &MigrationRunner{pool: pool, migrations: append([]module.Migration{}, migrations...)}
}

func (r *MigrationRunner) Up(ctx context.Context) error {
	if len(r.migrations) == 0 {
		return ErrNoMigrationChange
	}
	if err := r.ensureHistory(ctx); err != nil {
		return err
	}
	for _, migration := range r.migrations {
		var applied bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM apexvoid_schema_migrations WHERE module = $1 AND version = $2)`, migration.Module, migration.Version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s:%d: %w", migration.Module, migration.Version, err)
		}
		if applied {
			continue
		}
		if strings.TrimSpace(migration.UpSQL) == "" {
			return fmt.Errorf("migration %s:%d has empty up SQL", migration.Module, migration.Version)
		}
		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s:%d: %w", migration.Module, migration.Version, err)
		}
		if _, err = tx.Exec(ctx, migration.UpSQL); err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO apexvoid_schema_migrations (module, version, name) VALUES ($1, $2, $3)`, migration.Module, migration.Version, migration.Name)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s:%d: %w", migration.Module, migration.Version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s:%d: %w", migration.Module, migration.Version, err)
		}
	}
	return nil
}

func (r *MigrationRunner) Down(ctx context.Context) error {
	if len(r.migrations) == 0 {
		return ErrNoMigrationChange
	}
	if err := r.ensureHistory(ctx); err != nil {
		return err
	}
	var moduleName, name string
	var version uint64
	err := r.pool.QueryRow(ctx, `SELECT module, version, name FROM apexvoid_schema_migrations ORDER BY applied_at DESC, module DESC, version DESC LIMIT 1`).Scan(&moduleName, &version, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoMigrationChange
	}
	if err != nil {
		return fmt.Errorf("find latest migration: %w", err)
	}
	var target *module.Migration
	for i := range r.migrations {
		if r.migrations[i].Module == moduleName && r.migrations[i].Version == version {
			target = &r.migrations[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("migration %s:%d (%s) is not registered", moduleName, version, name)
	}
	if strings.TrimSpace(target.DownSQL) == "" {
		return fmt.Errorf("migration %s:%d has empty down SQL", moduleName, version)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration rollback: %w", err)
	}
	if _, err = tx.Exec(ctx, target.DownSQL); err == nil {
		_, err = tx.Exec(ctx, `DELETE FROM apexvoid_schema_migrations WHERE module = $1 AND version = $2`, moduleName, version)
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("rollback migration %s:%d: %w", moduleName, version, err)
	}
	return tx.Commit(ctx)
}

func (r *MigrationRunner) ensureHistory(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS apexvoid_schema_migrations (module TEXT NOT NULL, version BIGINT NOT NULL, name TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (module, version))`)
	return err
}
