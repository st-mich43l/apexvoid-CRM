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
	}}
}
