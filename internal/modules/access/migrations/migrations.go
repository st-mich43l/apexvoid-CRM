package migrations

import "github.com/st-mich43l/apexvoid-CRM/internal/framework/module"

func All() []module.Migration {
	return []module.Migration{{
		Module:  "access",
		Version: 1,
		Name:    "create_roles_and_assignments",
		UpSQL: `CREATE TABLE IF NOT EXISTS access_roles (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  system BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS access_user_roles (
  user_id UUID NOT NULL REFERENCES users_users(id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, role_id)
);
CREATE TABLE IF NOT EXISTS access_role_permissions (
  role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
  permission_name TEXT NOT NULL,
  PRIMARY KEY (role_id, permission_name)
);
CREATE INDEX IF NOT EXISTS access_user_roles_role_idx ON access_user_roles (role_id);
CREATE INDEX IF NOT EXISTS access_role_permissions_permission_idx ON access_role_permissions (permission_name);`,
		DownSQL: `DROP TABLE IF EXISTS access_role_permissions;
DROP TABLE IF EXISTS access_user_roles;
DROP TABLE IF EXISTS access_roles;`,
	}, {
		Module:  "access",
		Version: 2,
		Name:    "scope_roles_to_workspaces",
		UpSQL: `ALTER TABLE access_roles ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspace_workspaces(id) ON DELETE CASCADE;
ALTER TABLE access_roles DROP CONSTRAINT IF EXISTS access_roles_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS access_roles_global_name_idx ON access_roles (name) WHERE workspace_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS access_roles_workspace_name_idx ON access_roles (workspace_id, name) WHERE workspace_id IS NOT NULL;
CREATE TABLE IF NOT EXISTS access_workspace_membership_roles (
  membership_id UUID NOT NULL REFERENCES workspace_memberships(id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
  PRIMARY KEY (membership_id, role_id)
);
CREATE INDEX IF NOT EXISTS access_workspace_membership_roles_role_idx ON access_workspace_membership_roles (role_id);`,
		DownSQL: `DROP TABLE IF EXISTS access_workspace_membership_roles;
DROP INDEX IF EXISTS access_roles_global_name_idx;
DROP INDEX IF EXISTS access_roles_workspace_name_idx;
ALTER TABLE access_roles DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE access_roles ADD CONSTRAINT access_roles_name_key UNIQUE (name);`,
	}}
}
