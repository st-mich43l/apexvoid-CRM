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

## Persistence and transactions

PostgreSQL is the only external data dependency. Business repositories belong to their owning module under `infrastructure/postgres`; the platform database package owns the pool and transaction boundary helper. Application use cases call `TxManager.WithTransaction`; repositories retrieve the active transaction explicitly from the context and never start nested transactions.

Modules also own migration metadata through `module.MigrationProvider`. The registry assembles migrations in resolved module dependency order and then by version. Phase 2 has no persistent framework state, so no migration files are registered.

Successful application events are published after the surrounding database transaction commits. The in-process bus is synchronous and not durable; an outbox is intentionally deferred until an integration requirement exists.

## Frontend modules

The React application composes compiled-in `AppModule` definitions in `app/bootstrap/modules.ts`. The frontend module registry resolves declared dependencies, combines route definitions, and feeds a deterministic navigation registry into the shell. Backend metadata is authoritative for backend modules/entities/permissions; frontend modules remain compiled UI code and never execute backend-provided JavaScript.
