# Operations

## Development

Create the shared networks once, copy `.env.example` to `.env`, and run
`docker compose up -d`. The backend is attached to both networks: `apexvoid-apps`
for service-to-service traffic and `apexvoid-data` for PostgreSQL. A Café
Compose project must join both networks, use the service alias `cafe`, and
must not publish its database or backend to the public host.

```bash
docker network create apexvoid-apps 2>/dev/null || true
docker network create apexvoid-data 2>/dev/null || true
cp .env.example .env
docker compose up -d
```

The backend receives the integration variables explicitly; Docker does not
implicitly pass values from `.env` into a container. In particular,
`DATABASE_PROVISIONING_URL` must use the Docker hostname `postgres`, not
`localhost`, and `INTEGRATIONS_ALLOWED_SERVICE_HOSTS` should include `cafe`.
The development PostgreSQL init hook creates `apexvoid_provisioner` only for a
fresh volume when `DATABASE_PROVISIONER_PASSWORD` is set. Existing volumes
require the DBA to create the separately managed role and grant it only the
provisioning privileges required by the documented workflow.

A separate Café Compose project joins the same externally managed networks:

```yaml
services:
  cafe:
    networks:
      apexvoid-apps:
        aliases: [cafe]
      apexvoid-data:

networks:
  apexvoid-apps:
    external: true
    name: apexvoid-apps
  apexvoid-data:
    external: true
    name: apexvoid-data
```

Only the Café backend needs both networks. Its browser-facing frontend may be
published separately, but the Enterprise backend must resolve the backend as
`cafe` on `apexvoid-apps`.

## Production startup

1. Create the shared networks, copy `.env.production.example` to
   `.env.production`, and replace every placeholder with a unique secret. Set
   `APEXVOID_PUBLIC_ORIGIN` to the HTTPS origin exposed by your TLS ingress.
   Configure `INTEGRATIONS_ALLOWED_SERVICE_HOSTS=cafe`, keep
   `DATABASE_PROVISIONING_URL` on the Docker `postgres` hostname, and use a
   separately managed `apexvoid_provisioner` role. Do not reuse the runtime
   `POSTGRES_USER` for application traffic.
2. Run `docker compose --env-file .env.production -f docker-compose.production.yml up -d --build`.
3. Confirm `GET /ready` through the frontend reverse proxy and sign in with the
   explicit bootstrap administrator. The server fails before startup if secure
   cookies, trusted origins, or non-default bootstrap credentials are absent.

The production Compose file builds the Go binary and static React bundle. Nginx
serves the bundle and proxies `/api`, `/health`, and `/ready` to the backend;
Vite and source bind mounts are not part of this path. TLS terminates at the
ingress/load balancer in front of Nginx, so `AUTH_COOKIE_SECURE=true` is always
required. The frontend port is loopback-bound by default (`FRONTEND_BIND_ADDRESS=127.0.0.1`);
the ingress is the only intended public listener. Nginx deliberately discards
client-supplied forwarding headers and sets the known external protocol to
HTTPS before proxying. Do not expose this listener directly without replacing
that trusted-ingress configuration.

The backend receives database host, port, name, user, and password as separate
environment values. It constructs an escaped PostgreSQL URI internally, so a
password containing URI-reserved characters remains valid. Never print the
resulting URI or place it in a shell command.

The production Compose file fails early when the assertion secret, trusted
service-host allowlist, provisioning URL, or provisioning encryption key is
missing. `DATABASE_PROVISIONING_KEY` must remain stable across restarts because
it decrypts provisioned application credentials. The PostgreSQL init hook only
creates the provisioner role on a new data volume; for an existing deployment,
create or rotate that role through the normal DBA process and update the URL in
the deployment secret store.

### Attachment storage

The backend image creates `/var/lib/apexvoid/attachments` as the unprivileged
`apexvoid` user with mode `0750`. Docker initializes a fresh named volume from
that image path, so the normal backend process can write attachments without
root privileges. Existing volumes retain their existing ownership. Repair only
an identified legacy volume while the application is stopped, using this
one-off command (it is not a long-running privileged service):

```bash
docker compose --env-file .env.production -f docker-compose.production.yml stop backend
docker compose --env-file .env.production -f docker-compose.production.yml run --rm --no-deps \
  --user root --entrypoint sh backend -ec \
  'chown -R apexvoid:apexvoid /var/lib/apexvoid/attachments && chmod 0750 /var/lib/apexvoid/attachments && find /var/lib/apexvoid/attachments -type d -exec chmod 0750 {} + && find /var/lib/apexvoid/attachments -type f -exec chmod 0600 {} +'
docker compose --env-file .env.production -f docker-compose.production.yml start backend
```

Run this only for the selected deployment after a backup; it changes ownership
and modes in that deployment's attachment volume.

## Upgrade and restart

Build and start the same Compose command. Application startup applies ordered,
module-owned migrations before serving traffic. Restart only after a successful
`/ready` check; named PostgreSQL and attachment volumes survive container
recreation.

## Backup and restore

Back up the database and attachments together. First quiesce writes by stopping
the backend; leave PostgreSQL running. The dump is custom format and therefore
must be restored with `pg_restore`, not `psql`.

```bash
docker compose --env-file .env.production -f docker-compose.production.yml stop backend
docker compose --env-file .env.production -f docker-compose.production.yml exec -T postgres \
  sh -c 'exec pg_dump -Fc -U "$POSTGRES_USER" "$POSTGRES_DB"' > apexvoid.dump
docker compose --env-file .env.production -f docker-compose.production.yml run --rm --no-deps \
  --entrypoint tar backend czf - -C /var/lib/apexvoid/attachments . > apexvoid-attachments.tgz
docker compose --env-file .env.production -f docker-compose.production.yml start backend
```

The database environment variables in the commands resolve inside the
PostgreSQL container. Store both artifacts outside Git, encrypted at rest, and
with access limited to approved operators.

### Isolated restore drill

Restores overwrite the target database and attachment volume. Never use these
commands against development or production resources. Create a new restore
directory containing the two artifacts and a copied production environment
file, choose a unique project name, and use a different loopback port:

```bash
mkdir apexvoid-restore && cd apexvoid-restore
cp /path/to/.env.production .env.restore
sed -i 's/^FRONTEND_PORT=.*/FRONTEND_PORT=18080/' .env.restore
docker compose -p apexvoid-restore --env-file .env.restore \
  -f /path/to/docker-compose.production.yml up -d postgres
docker compose -p apexvoid-restore --env-file .env.restore \
  -f /path/to/docker-compose.production.yml exec -T postgres \
  sh -c 'exec pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists' < apexvoid.dump
docker compose -p apexvoid-restore --env-file .env.restore \
  -f /path/to/docker-compose.production.yml up -d backend
docker compose -p apexvoid-restore --env-file .env.restore \
  -f /path/to/docker-compose.production.yml cp apexvoid-attachments.tgz backend:/tmp/apexvoid-attachments.tgz
docker compose -p apexvoid-restore --env-file .env.restore \
  -f /path/to/docker-compose.production.yml exec -T backend \
  sh -c 'rm -rf /var/lib/apexvoid/attachments/* && tar xzf /tmp/apexvoid-attachments.tgz -C /var/lib/apexvoid/attachments && find /var/lib/apexvoid/attachments -type d -exec chmod 0750 {} + && find /var/lib/apexvoid/attachments -type f -exec chmod 0600 {} +'
docker compose -p apexvoid-restore --env-file .env.restore \
  -f /path/to/docker-compose.production.yml up -d frontend
curl --fail http://127.0.0.1:18080/ready
```

The restored backend applies only pending migrations at startup. Authenticate
through the configured HTTPS ingress, retrieve a restored business record, and
download a known attachment before treating the restore drill as successful.
Destroy only the explicitly created restore project and its volumes after the
drill is accepted.

## Diagnostics

Use `docker compose ... ps` for health, `docker compose ... logs backend` for
request IDs and startup diagnostics, `/health` for process liveness, and
`/ready` for database readiness. API errors include a request ID but never
return database traces or credentials.
