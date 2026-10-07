# ApexVoid CRM

ApexVoid is a modular business application framework built as a Go/React modular monolith. CRM domain capabilities will be added as explicit compiled-in modules in later phases; Phase 1 contains framework infrastructure only.

## Architecture

- Backend: Go, chi, `log/slog`, pgx, PostgreSQL
- Frontend: React, TypeScript, Vite, Tailwind CSS, React Router, TanStack Query
- Runtime: explicit module registry, dependency resolution, metadata registries, synchronous typed event bus, and extension points
- Infrastructure: PostgreSQL only; no cache or message broker is required
- API: versioned REST under `/api/v1`

Application startup creates one owned framework runtime, registers built-in modules, resolves dependencies, validates metadata, and only then starts HTTP serving.

## Framework concepts

- Modules declare identity, version, dependencies, and registration behavior.
- Entities and fields describe strongly typed business models without replacing relational tables.
- Permissions and capabilities are explicit registries; authorization and roles are intentionally deferred.
- Events use namespaced definitions and generic typed subscriber/publisher functions.
- Extensions attach named implementations to explicit extension points with deterministic ordering.
- Metadata aggregates framework definitions for read-only discovery by API clients.

The built-in `core` module registers only `core.example`, `core.example.read`, `core.example.created`, `core.auditable`, and `core.navigation` to exercise the framework. No CRM entities are included.

## Development

Requirements: Go 1.25+, Node.js 22+, npm, Docker, and Docker Compose.

```bash
cp .env.example .env
docker compose up -d
```

Endpoints:

- Frontend: http://localhost:5173
- Backend: http://localhost:8080
- Health: `GET /health`
- Readiness: `GET /ready`
- Modules: `GET /api/v1/framework/modules`
- Entities: `GET /api/v1/framework/entities`
- Entity detail: `GET /api/v1/framework/entities/{entity}`
- Permissions: `GET /api/v1/framework/permissions`

Configuration defaults are in `config/application.yaml`; environment variables override them. Use `.env.example` as the local template and do not commit secrets.

## Frontend discovery

The Framework page in the web shell consumes the typed discovery APIs and displays installed modules, registered entities, and permissions. A navigation contribution registry exists in Go for future module-driven navigation; dynamic navigation rendering is intentionally deferred.

## Testing

```bash
make test
make lint
make fmt
make build
```

The test suite covers dependency ordering, missing dependencies, cycles, entity/field validation, permission validation, typed event delivery, extension ordering, and metadata discovery routes. CI additionally runs Go formatting validation and the frontend production build.

## Repository structure

```text
cmd/server/          HTTP application entrypoint
internal/app/        application bootstrap and discovery routes
internal/framework/  module, entity, field, permission, event, extension, metadata, runtime
internal/modules/    compiled-in modules, currently only core
internal/platform/   config, PostgreSQL, health, logging, HTTP
web/                 React framework shell and typed API client
```

Framework definitions are currently held in Go code. There are no framework metadata tables or CRM migrations because no persistent framework state is required yet.
