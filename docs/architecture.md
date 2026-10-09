# ApexVoid Architecture

## Backend layers

Each module owns the behavior for its business concepts. A module may add only the layers it needs:

```text
transport/http  ->  application  ->  domain
                         |
                         v
                 repository contract
                         ^
                         |
                 infrastructure/postgres
```

Transport owns request/response DTOs and HTTP mapping. Application services coordinate use cases, repositories, transactions, permissions, and events. Domain code contains business rules and depends on none of HTTP, PostgreSQL, configuration, or logging implementations. PostgreSQL adapters implement contracts owned by the application/domain side.

Module-owned tables use `snake_case` and a module prefix such as `contacts_contacts` or `crm_leads`. Domain timestamps use UTC `time.Time` values and PostgreSQL `TIMESTAMPTZ`; `created_at` and `updated_at` are added when a persisted domain requires them.

The Phase 2 `core` module is a framework/system module, so it has application and transport layers but no domain or persistence layer. Its discovery flow is:

```text
HTTP handler -> core application service -> framework metadata registry -> DTO response
```

That is a real non-business flow without inventing persistent example records.

## Module composition

The application composition root constructs platform services and compiled-in modules. Modules are registered explicitly, dependency-resolved, initialized, and then asked to register routes through the small framework route contract. Modules do not use package `init()` registration or global service locators.

Module-to-module communication uses either a narrow public contract under `modules/<module>/api` for synchronous reads or the in-process typed event bus for reactions. A module must declare dependencies in its descriptor before using another module's public contract.

## Application composition

An application is a compiled product-facing composition of modules, distinct
from a platform module. Platform modules provide authentication, users,
organizations/workspaces, access control, customization, and framework
services. Applications consume those services through explicit public
contracts and preserve workspace context at every boundary.

Application modules register `framework/application.Descriptor` during module
initialization. A descriptor includes a stable ID, display metadata, module
dependencies, required permissions/capabilities, a frontend entry route,
navigation identity, and an optional application-owned settings route. Runtime
validation confirms every referenced module, permission, and capability was
registered. The core discovery endpoint exposes this metadata at
`/api/v1/framework/applications`.

React modules remain compiled into the web release. The frontend module
registry verifies the matching entry route and navigation ID and the
Applications page reports a backend/frontend mismatch. It never downloads or
executes JavaScript from application metadata. Explicit registration in the Go
and React composition roots is intentional; runtime Go plugin loading is not
supported.

## Organization, workspace, and tenant context

The `organization` module owns the Phase 4 ownership boundary:

```text
User -> workspace_memberships -> workspace_workspaces -> organization_organizations
```

Users may have active memberships in multiple workspaces. The selected workspace is carried by the `X-ApexVoid-Workspace` request header; when a user has exactly one active workspace, the backend may resolve it without the header. The `organization/api` middleware authenticates the user, resolves the workspace, verifies the active membership and active organization/workspace status, and stores a typed `WorkspaceContext` in the request context. A workspace ID supplied by a client is never trusted without this membership check.

Workspace-owned repositories receive an explicit workspace ID or `WorkspaceContext` and must include it in every lookup. The framework entity metadata supports `global`, `organization`, and `workspace` scopes so future modules can declare ownership without adding ad-hoc tenant behavior.

Platform roles remain installation-wide. Workspace role definitions carry a `workspace_id`, and role assignments are made through `access_workspace_membership_roles`. Permission definitions explicitly declare `platform` or `workspace` scope. Platform authorization only returns platform-scoped permissions; workspace authorization returns platform permissions from platform roles plus workspace permissions from the active membership. The built-in workspace administrator is provisioned transactionally during initial setup and dynamically grants all registered workspace-scoped permissions, so new workspace capabilities do not require role synchronization. It is separate from the global platform administrator and cannot grant platform administration.

Initial setup is authoritative on the backend: `GET /api/v1/setup/status` reports whether an organization exists and `POST /api/v1/setup/organization` creates the organization, default workspace, administrator membership, and workspace administrator role in one transaction. The operation is idempotently rejected after setup is complete. Membership removal, suspension, and administrator-role changes are rejected when they would leave zero active workspace administrators. Organization/workspace/member events are published only after their state-change transaction completes, using typed payloads. The frontend keeps the selected workspace in local storage only as a convenience; the backend remains the authorization boundary. Query keys for workspace data include the workspace ID, switching invalidates the whole workspace query namespace, and the central API client adds the workspace header automatically.

## Persistence and transactions

PostgreSQL is the only external data dependency. Business repositories belong to their owning module under `infrastructure/postgres`; the platform database package owns the pool and transaction boundary helper. Application use cases call `TxManager.WithTransaction`; repositories retrieve the active transaction explicitly from the context and never start nested transactions.

Modules also own migration metadata through `module.MigrationProvider`. The registry assembles migrations in resolved module dependency order and then by version. Phase 2 has no persistent framework state, so no migration files are registered.

Successful application events are published after the surrounding database transaction commits. The in-process bus is synchronous and not durable; an outbox is intentionally deferred until an integration requirement exists.

Post-commit publishers log subscriber failures with event context. A subscriber
failure cannot roll back an already committed transaction and does not imply
durable retry; applications that need delivery guarantees must introduce a
concrete integration requirement and corresponding durable design.

## Frontend modules

The React application composes compiled-in `AppModule` definitions in `app/bootstrap/modules.ts`. The frontend module registry resolves declared dependencies, combines route definitions, and feeds a deterministic navigation registry into the shell. Backend metadata is authoritative for backend modules/entities/permissions; frontend modules remain compiled UI code and never execute backend-provided JavaScript.
