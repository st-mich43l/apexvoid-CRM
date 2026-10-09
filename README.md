# ApexVoid Enterprise

ApexVoid is a modular business application framework built as a Go/React modular monolith. The runtime includes the shared platform foundation and compiled-in Contacts, CRM, and ERP applications. ERP starts with a workspace-scoped Products & Services catalog, extending the same modular Go/React application without a new service.

## Architecture

- Backend: Go, chi, `log/slog`, pgx, PostgreSQL
- Frontend: React, TypeScript, Vite, Tailwind CSS, React Router, TanStack Query
- Runtime: explicit module and application registries, deterministic dependency resolution, metadata registries, synchronous typed event bus, extension points, module-owned HTTP transport, and user/RBAC modules
- Infrastructure: PostgreSQL only; no cache or message broker is required
- API: versioned REST under `/api/v1`

Application startup creates one owned framework runtime, registers built-in modules, resolves dependencies, validates metadata, and only then starts HTTP serving.

See [docs/architecture.md](docs/architecture.md) for layer responsibilities, transaction boundaries, migration ownership, and frontend module composition. See [ERP architecture](docs/erp.md) for the initial product catalog and future ERP domain boundaries.

## Framework concepts

- Modules declare identity, version, dependencies, and registration behavior.
- Applications declare their compiled-in identity, backend dependencies, static contracts, explicit all-of/any-of entry and settings authorization, frontend entry route, navigation identity, and API contract version. Backend discovery is authenticated and workspace-aware; its metadata never executes code in the browser.
- Entities and fields describe strongly typed business models without replacing relational tables.
- Permissions and capabilities are explicit registries; users and RBAC assignments are owned by the platform modules.
- Events use namespaced definitions and generic typed subscriber/publisher functions.
- Extensions attach named implementations to explicit extension points with deterministic ordering.
- Metadata aggregates framework definitions for read-only discovery by API clients.

The built-in `core` module registers only `core.example`, `core.example.read`, `core.example.created`, `core.auditable`, and `core.navigation` to exercise the framework. Its discovery handlers live under `internal/modules/core/transport/http`; no CRM entities are included.

Users authenticate with short-lived opaque access tokens and rotating refresh tokens in HttpOnly cookies. Organizations, workspaces, memberships, workspace context, and workspace-scoped RBAC are platform-owned. Permission definitions explicitly declare platform or workspace scope: the protected platform `administrator` role dynamically grants platform permissions, while the protected workspace administrator dynamically grants workspace permissions for its active membership. No tokens or password hashes are returned by the API.

## Development

Requirements: Go 1.25+, Node.js 22+, npm, Docker, and Docker Compose.

```bash
cp .env.example .env
docker compose up -d
```

Endpoints:

- Frontend: http://localhost:8386
- Backend: http://localhost:6868
- Health: `GET /health`
- Readiness: `GET /ready`
- Modules: `GET /api/v1/framework/modules`
- Applications: `GET /api/v1/framework/applications`
- Entities: `GET /api/v1/framework/entities`
- Entity detail: `GET /api/v1/framework/entities/{entity}`
- Permissions: `GET /api/v1/framework/permissions`
- Login: `POST /api/v1/auth/login`
- Current user: `GET /api/v1/auth/me`
- Users: `/api/v1/users`
- Roles: `/api/v1/access/roles`
- Setup status: `GET /api/v1/setup/status`
- Initial setup: `POST /api/v1/setup/organization`
- Workspaces: `GET /api/v1/workspaces`, current context at `GET /api/v1/workspace`
- Workspace members and roles: `/api/v1/workspace/members`, `/api/v1/workspace/roles`
- Workspace context is selected with `X-ApexVoid-Workspace`; the API verifies active membership before resolving any workspace-scoped operation.

Configuration defaults are in `config/application.yaml`; environment variables override them. Use `.env.example` as the local template and do not commit secrets.

Migration metadata is owned by modules and ordered by module dependencies. Users and access migrations create the identity and RBAC tables:

```bash
make migrate-up
make migrate-down
make test-integration
```

For a fresh development deployment, the default administrator is `admin` / `admin` when the database has no users. The credential is immediately forced through the password-change flow before administration APIs are available. Production rejects those defaults and requires explicit non-default bootstrap credentials, secure cookies, and trusted origins. The password is stored as an Argon2id hash, never as reversible plaintext or encryption.

## Frontend structure

The Framework and Applications pages consume typed discovery APIs. Frontend modules are compiled in through `web/src/app/bootstrap/modules.ts`; each module may contribute routes and navigation. The core module owns login, protected routing, setup, workspace selection, and organization settings, while the administration module provides platform user and role management. A navigation registry provides deterministic ordering without a runtime plugin system. See [building applications](docs/building-applications.md) for the supported extension workflow.

## Production operations

Copy `.env.production.example` to `.env.production`, set every secret and public HTTPS origin, then run:

```bash
docker compose --env-file .env.production -f docker-compose.production.yml up -d --build
```

The production Compose path builds static frontend assets served by Nginx and proxies API requests to the Go backend; it does not use Vite. Its frontend listener is loopback-bound by default for a trusted HTTPS ingress, database URI components are safely escaped by the backend, and attachment storage runs under the non-root runtime user. See [operations](docs/operations.md) for upgrade, backup, restore, attachment-volume repair, and diagnostics guidance.

## Testing

```bash
make test
make lint
make fmt
make build
```

The test suite covers dependency ordering, missing dependencies, cycles, entity/field validation, permission validation, typed event delivery, extension ordering, metadata discovery routes, and frontend theme/navigation/workspace-RBAC behavior. PostgreSQL integration tests exercise setup idempotency, tenant isolation, workspace role boundaries, administrator semantics, and last-administrator protection. CI runs Go formatting, vet/tests/build, frontend lint/typecheck/tests/build, and the PostgreSQL integration suite.

## Repository structure

```text
cmd/server/          HTTP application entrypoint
internal/app/        application bootstrap and discovery routes
internal/framework/  module, entity, field, permission, event, extension, metadata, runtime
internal/modules/    compiled-in platform and business modules
internal/platform/   config, PostgreSQL, health, logging, HTTP
web/src/core/        platform API client and application shell
web/src/framework/   frontend module, navigation, and metadata contracts
web/src/modules/     compiled-in frontend modules
```

Framework definitions are held in Go code. Organization, workspace, membership, workspace-role, Contacts, CRM, ERP, and customization state is persisted in PostgreSQL. Application and frontend contracts are compiled into the deployed release.
