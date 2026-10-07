package migrations

import "github.com/st-mich43l/apexvoid-CRM/internal/framework/module"

func All() []module.Migration {
	return []module.Migration{{
		Module:  "users",
		Version: 1,
		Name:    "create_users_and_sessions",
		UpSQL: `CREATE TABLE IF NOT EXISTS users_users (
  id UUID PRIMARY KEY,
  email TEXT NOT NULL,
  username TEXT,
  display_name TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('active', 'inactive', 'locked')),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  last_login_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS users_users_email_lower_key ON users_users (LOWER(email));
CREATE UNIQUE INDEX IF NOT EXISTS users_users_username_lower_key ON users_users (LOWER(username)) WHERE username IS NOT NULL;
CREATE TABLE IF NOT EXISTS users_sessions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users_users(id) ON DELETE CASCADE,
  access_token_hash BYTEA NOT NULL UNIQUE,
  refresh_token_hash BYTEA NOT NULL UNIQUE,
  access_expires_at TIMESTAMPTZ NOT NULL,
  refresh_expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  last_used_at TIMESTAMPTZ,
  user_agent TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS users_sessions_user_id_idx ON users_sessions (user_id);
CREATE INDEX IF NOT EXISTS users_sessions_refresh_expiry_idx ON users_sessions (refresh_expires_at);`,
		DownSQL: `DROP TABLE IF EXISTS users_sessions;
DROP TABLE IF EXISTS users_users;`,
	}, {
		Module:  "users",
		Version: 2,
		Name:    "force_bootstrap_password_change",
		UpSQL:   `ALTER TABLE users_users ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT FALSE;`,
		DownSQL: `ALTER TABLE users_users DROP COLUMN IF EXISTS must_change_password;`,
	}}
}
