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
	}}
}
