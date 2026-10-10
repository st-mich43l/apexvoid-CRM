package migrations

import "github.com/st-mich43l/apexvoid-CRM/internal/framework/module"

// All owns integration metadata only. External services never receive direct
// database access; their relationship with ApexVoid is always through HTTP.
func All() []module.Migration {
	return []module.Migration{{
		Module: "core", Version: 1, Name: "create_external_application_registry",
		UpSQL: `CREATE TABLE IF NOT EXISTS core_external_applications (
  id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  version TEXT NOT NULL,
  api_contract_version TEXT NOT NULL,
  service_identity TEXT NOT NULL UNIQUE,
  service_endpoint TEXT NOT NULL,
  health_endpoint TEXT NOT NULL,
  frontend_route TEXT NOT NULL DEFAULT '',
  settings_route TEXT NOT NULL DEFAULT '',
  access_match TEXT NOT NULL,
  access_permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  credential_hash TEXT NOT NULL,
  credential_revoked_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS core_external_application_permissions (
  application_id TEXT NOT NULL REFERENCES core_external_applications(id) ON DELETE CASCADE,
  name TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  scope TEXT NOT NULL,
  PRIMARY KEY (application_id, name)
);
CREATE TABLE IF NOT EXISTS core_external_application_workspaces (
  application_id TEXT NOT NULL REFERENCES core_external_applications(id) ON DELETE CASCADE,
  workspace_id UUID NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (application_id, workspace_id)
);`,
		DownSQL: `DROP TABLE IF EXISTS core_external_application_workspaces;
DROP TABLE IF EXISTS core_external_application_permissions;
DROP TABLE IF EXISTS core_external_applications;`,
	}, {
		Module: "core", Version: 2, Name: "harden_external_application_lifecycle",
		UpSQL: `ALTER TABLE core_external_applications ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE core_external_applications ADD COLUMN IF NOT EXISTS credential_rotated_at TIMESTAMPTZ NULL;
ALTER TABLE core_external_applications DROP CONSTRAINT IF EXISTS core_external_applications_status_check;
ALTER TABLE core_external_applications ADD CONSTRAINT core_external_applications_status_check CHECK (status IN ('registering', 'active', 'retired'));
CREATE TABLE IF NOT EXISTS core_external_permission_tombstones (
  name TEXT PRIMARY KEY,
  application_id TEXT NOT NULL,
  retired_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`,
		DownSQL: `DROP TABLE IF EXISTS core_external_permission_tombstones;
ALTER TABLE core_external_applications DROP CONSTRAINT IF EXISTS core_external_applications_status_check;
ALTER TABLE core_external_applications DROP COLUMN IF EXISTS credential_rotated_at;
ALTER TABLE core_external_applications DROP COLUMN IF EXISTS status;`,
	}, {
		Module: "core", Version: 3, Name: "audit_external_application_administration",
		UpSQL: `CREATE TABLE IF NOT EXISTS core_external_application_audit (
  id UUID PRIMARY KEY,
  application_id TEXT NOT NULL,
  actor_user_id UUID NOT NULL,
  workspace_id UUID NULL,
  action TEXT NOT NULL,
  request_id TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS core_external_application_audit_application_idx ON core_external_application_audit(application_id, created_at DESC);`,
		DownSQL: `DROP TABLE IF EXISTS core_external_application_audit;`,
	}, {
		Module: "core", Version: 4, Name: "add_external_application_installations",
		UpSQL: `ALTER TABLE core_external_applications ADD COLUMN IF NOT EXISTS workspace_default_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE core_external_applications ADD COLUMN IF NOT EXISTS installation_id UUID NULL;
ALTER TABLE core_external_applications ADD COLUMN IF NOT EXISTS database_name TEXT NULL;
ALTER TABLE core_external_applications ADD COLUMN IF NOT EXISTS database_schema TEXT NULL;
ALTER TABLE core_external_applications ADD COLUMN IF NOT EXISTS database_role TEXT NULL;
ALTER TABLE core_external_applications ADD COLUMN IF NOT EXISTS migration_bundle_version TEXT NULL;
ALTER TABLE core_external_applications ADD COLUMN IF NOT EXISTS installed_manifest JSONB NULL;
ALTER TABLE core_external_applications DROP CONSTRAINT IF EXISTS core_external_applications_status_check;
ALTER TABLE core_external_applications ADD CONSTRAINT core_external_applications_status_check CHECK (status IN ('registering', 'discovered', 'pending_approval', 'provisioning', 'migrating', 'verifying', 'failed', 'active', 'retired'));
CREATE UNIQUE INDEX IF NOT EXISTS core_external_applications_installation_idx ON core_external_applications(installation_id) WHERE installation_id IS NOT NULL;
CREATE TABLE IF NOT EXISTS core_external_application_installations (
  id UUID PRIMARY KEY,
  application_id TEXT NOT NULL,
  service_url TEXT NOT NULL,
  manifest JSONB NOT NULL,
  enrollment_code_hash TEXT NOT NULL,
  status TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  last_step TEXT NOT NULL DEFAULT 'discovered',
  error_message TEXT NOT NULL DEFAULT '',
  selected_workspace_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_by UUID NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT core_external_application_installations_status_check CHECK (status IN ('discovered', 'pending_approval', 'provisioning', 'migrating', 'verifying', 'failed', 'active', 'retired'))
);
CREATE INDEX IF NOT EXISTS core_external_application_installations_status_idx ON core_external_application_installations(status, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS core_external_application_installations_application_idx ON core_external_application_installations(application_id) WHERE status NOT IN ('failed', 'retired');`,
		DownSQL: `DROP TABLE IF EXISTS core_external_application_installations;
DROP INDEX IF EXISTS core_external_applications_installation_idx;
ALTER TABLE core_external_applications DROP COLUMN IF EXISTS workspace_default_enabled;
ALTER TABLE core_external_applications DROP COLUMN IF EXISTS installation_id;
ALTER TABLE core_external_applications DROP COLUMN IF EXISTS database_name;
ALTER TABLE core_external_applications DROP COLUMN IF EXISTS database_schema;
ALTER TABLE core_external_applications DROP COLUMN IF EXISTS database_role;
ALTER TABLE core_external_applications DROP COLUMN IF EXISTS migration_bundle_version;
ALTER TABLE core_external_applications DROP COLUMN IF EXISTS installed_manifest;
ALTER TABLE core_external_applications DROP CONSTRAINT IF EXISTS core_external_applications_status_check;
ALTER TABLE core_external_applications ADD CONSTRAINT core_external_applications_status_check CHECK (status IN ('registering', 'active', 'retired'));`,
	}, {
		Module: "core", Version: 5, Name: "external_application_resource_ownership_and_ledger",
		UpSQL: `ALTER TABLE core_external_application_installations ADD COLUMN service_credential_encrypted BYTEA NULL;
CREATE TABLE core_external_application_resources (
  application_id TEXT PRIMARY KEY,
  installation_id UUID NOT NULL UNIQUE REFERENCES core_external_application_installations(id),
  database_name TEXT NOT NULL UNIQUE,
  schema_name TEXT NOT NULL,
  role_name TEXT NOT NULL UNIQUE,
  encrypted_password BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE core_external_application_migrations (
  application_id TEXT NOT NULL REFERENCES core_external_application_resources(application_id),
  version INTEGER NOT NULL,
  checksum TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (application_id,version)
);`,
		DownSQL: `DROP TABLE IF EXISTS core_external_application_migrations;
DROP TABLE IF EXISTS core_external_application_resources;
ALTER TABLE core_external_application_installations DROP COLUMN IF EXISTS service_credential_encrypted;`,
	}, {
		Module: "core", Version: 6, Name: "reviewed_external_application_upgrades",
		UpSQL: `CREATE TABLE IF NOT EXISTS core_external_application_upgrades (
  id UUID PRIMARY KEY,
  application_id TEXT NOT NULL REFERENCES core_external_applications(id),
  manifest JSONB NOT NULL,
  manifest_sha256 TEXT NOT NULL,
  installed_manifest_sha256 TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('pending_approval','applying','applied','failed','superseded')),
  expires_at TIMESTAMPTZ NOT NULL,
  error_message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  applied_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS core_external_upgrade_history_idx ON core_external_application_upgrades(application_id,created_at DESC);`,
		DownSQL: `DROP TABLE IF EXISTS core_external_application_upgrades;`,
	}}
}
