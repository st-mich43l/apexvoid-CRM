package migrations

import "github.com/st-mich43l/apexvoid-CRM/internal/framework/module"

func All() []module.Migration {
	return []module.Migration{{
		Module:  "organization",
		Version: 1,
		Name:    "create_organizations_workspaces_memberships",
		UpSQL: `CREATE TABLE IF NOT EXISTS organization_organizations (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL CHECK (status IN ('active', 'inactive')),
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS organization_default_idx ON organization_organizations (is_default) WHERE is_default = TRUE;
CREATE TABLE IF NOT EXISTS workspace_workspaces (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organization_organizations(id) ON DELETE RESTRICT,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'UTC',
  status TEXT NOT NULL CHECK (status IN ('active', 'inactive')),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (organization_id, slug)
);
CREATE INDEX IF NOT EXISTS workspace_workspaces_organization_idx ON workspace_workspaces (organization_id);
CREATE TABLE IF NOT EXISTS workspace_memberships (
  id UUID PRIMARY KEY,
  workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users_users(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('active', 'invited', 'suspended')),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (workspace_id, user_id)
);
CREATE INDEX IF NOT EXISTS workspace_memberships_user_idx ON workspace_memberships (user_id);
CREATE INDEX IF NOT EXISTS workspace_memberships_workspace_idx ON workspace_memberships (workspace_id);`,
		DownSQL: `DROP TABLE IF EXISTS workspace_memberships;
DROP TABLE IF EXISTS workspace_workspaces;
DROP TABLE IF EXISTS organization_organizations;`,
	}}
}
