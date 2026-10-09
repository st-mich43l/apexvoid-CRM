#!/bin/sh
set -eu

# This runs only when PostgreSQL initializes a fresh data volume. Existing
# deployments must create the dedicated role through their normal DBA process.
if [ -z "${DATABASE_PROVISIONER_PASSWORD:-}" ]; then
  exit 0
fi

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<'SQL'
\getenv provisioner_password DATABASE_PROVISIONER_PASSWORD
SELECT format(
  'CREATE ROLE apexvoid_provisioner LOGIN PASSWORD %L NOSUPERUSER CREATEDB CREATEROLE NOINHERIT',
  :'provisioner_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'apexvoid_provisioner')\gexec

SELECT format(
  'ALTER ROLE apexvoid_provisioner PASSWORD %L NOSUPERUSER CREATEDB CREATEROLE NOINHERIT',
  :'provisioner_password'
)
WHERE EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'apexvoid_provisioner')\gexec
SQL
