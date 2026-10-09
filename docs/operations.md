# Operations

## Development

Copy `.env.example` to `.env` and run `docker compose up -d`. Development uses
the Vite service and deliberately retains local bootstrap defaults. Do not use
that Compose file for a public deployment.

## Production startup

1. Copy `.env.production.example` to `.env.production` and replace every
   placeholder with a unique secret. Set `APEXVOID_PUBLIC_ORIGIN` to the HTTPS
   origin exposed by your TLS ingress.
2. Run `docker compose --env-file .env.production -f docker-compose.production.yml up -d --build`.
3. Confirm `GET /ready` through the frontend reverse proxy and sign in with the
   explicit bootstrap administrator. The server fails before startup if secure
   cookies, trusted origins, or non-default bootstrap credentials are absent.

The production Compose file builds the Go binary and static React bundle. Nginx
serves the bundle and proxies `/api`, `/health`, and `/ready` to the backend;
Vite and source bind mounts are not part of this path. TLS terminates at the
ingress/load balancer in front of Nginx, so `AUTH_COOKIE_SECURE=true` is always
required.

## Upgrade and restart

Build and start the same Compose command. Application startup applies ordered,
module-owned migrations before serving traffic. Restart only after a successful
`/ready` check; named PostgreSQL and attachment volumes survive container
recreation.

## Backup and restore

Back up the database and attachments together:

```bash
docker compose --env-file .env.production -f docker-compose.production.yml exec -T postgres \
  pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" > apexvoid.sql
docker compose --env-file .env.production -f docker-compose.production.yml exec -T backend \
  tar czf - -C /var/lib/apexvoid/attachments . > apexvoid-attachments.tgz
```

Restore into an isolated environment first, stop the backend, restore the SQL
with `psql`, then restore the attachment archive to the matching named volume.
Verify `/ready`, authenticate, and confirm an attachment can be read before
switching traffic. Keep database dumps and attachment archives encrypted and
access-controlled.

## Diagnostics

Use `docker compose ... ps` for health, `docker compose ... logs backend` for
request IDs and startup diagnostics, `/health` for process liveness, and
`/ready` for database readiness. API errors include a request ID but never
return database traces or credentials.
