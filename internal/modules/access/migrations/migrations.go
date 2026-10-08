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
	}, {
		Module:  "access",
		Version: 3,
		Name:    "protect_role_identity_and_assignment_scope",
		UpSQL: `
ALTER TABLE access_workspace_membership_roles ADD COLUMN IF NOT EXISTS workspace_id UUID;
UPDATE access_workspace_membership_roles wmr SET workspace_id = m.workspace_id FROM workspace_memberships m WHERE m.id = wmr.membership_id AND wmr.workspace_id IS NULL;
ALTER TABLE access_workspace_membership_roles ALTER COLUMN workspace_id SET NOT NULL;
DO $$ BEGIN
  ALTER TABLE workspace_memberships ADD CONSTRAINT workspace_memberships_id_workspace_key UNIQUE (id, workspace_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
  ALTER TABLE access_roles ADD CONSTRAINT access_roles_id_workspace_key UNIQUE (id, workspace_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
  ALTER TABLE access_workspace_membership_roles ADD CONSTRAINT access_workspace_membership_roles_membership_scope_fk FOREIGN KEY (membership_id, workspace_id) REFERENCES workspace_memberships (id, workspace_id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
  ALTER TABLE access_workspace_membership_roles ADD CONSTRAINT access_workspace_membership_roles_role_scope_fk FOREIGN KEY (role_id, workspace_id) REFERENCES access_roles (id, workspace_id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
CREATE OR REPLACE FUNCTION access_validate_protected_role_identity() RETURNS trigger LANGUAGE plpgsql AS $fn$
BEGIN
  IF NEW.name = 'administrator' AND NOT (NEW.system = TRUE AND NEW.workspace_id IS NULL) THEN
    RAISE EXCEPTION 'administrator is a protected global system role' USING ERRCODE = '23514';
  END IF;
  IF NEW.name = 'workspace_administrator' AND NOT (NEW.system = TRUE AND NEW.workspace_id IS NOT NULL) THEN
    RAISE EXCEPTION 'workspace_administrator is a protected workspace system role' USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$fn$;
DROP TRIGGER IF EXISTS access_protected_role_identity ON access_roles;
CREATE TRIGGER access_protected_role_identity BEFORE INSERT OR UPDATE OF name, system, workspace_id ON access_roles FOR EACH ROW EXECUTE FUNCTION access_validate_protected_role_identity();`,
		DownSQL: `DROP TRIGGER IF EXISTS access_protected_role_identity ON access_roles;
DROP FUNCTION IF EXISTS access_validate_protected_role_identity();
ALTER TABLE access_workspace_membership_roles DROP CONSTRAINT IF EXISTS access_workspace_membership_roles_role_scope_fk;
ALTER TABLE access_workspace_membership_roles DROP CONSTRAINT IF EXISTS access_workspace_membership_roles_membership_scope_fk;
ALTER TABLE access_roles DROP CONSTRAINT IF EXISTS access_roles_id_workspace_key;
ALTER TABLE workspace_memberships DROP CONSTRAINT IF EXISTS workspace_memberships_id_workspace_key;
ALTER TABLE access_workspace_membership_roles DROP COLUMN IF EXISTS workspace_id;`,
	}}
}
