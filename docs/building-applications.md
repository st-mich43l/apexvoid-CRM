# Building a compiled-in ApexVoid application

An ApexVoid application is compiled together with the platform. It is not a
runtime plugin, a marketplace package, or code stored in the database. This
keeps deployment, authorization, upgrades, and frontend execution explicit.

## Backend

1. Create `internal/modules/<application>/` with the layers actually needed:
   `domain`, `application`, `infrastructure/postgres`, `transport/http`, and
   `api` for narrow public contracts.
2. Implement `module.Module`. Its descriptor declares a stable lowercase name,
   semantic version, and every module dependency.
3. In `Register`, register entity definitions, fields, permissions,
   capabilities, events, extension implementations, and the application
   descriptor. Registration must return errors; do not use `init()` or a global
   service locator.
4. Keep synchronous cross-module dependencies behind a small interface in the
   owning module's `api` package. Declare that module in the descriptor before
   using its interface.
5. Put HTTP DTO validation and error mapping in `transport/http`. Register
   routes with `module.RouteRegistry`; require authentication, workspace
   context, and the precise permission for every workspace-owned operation.
6. Add migrations through `Migrations()`. Each migration is module-owned and
   ordered after declared dependencies. Application services own transaction
   boundaries through `TxManager` and publish events with `AfterCommit`.
7. Add the module explicitly in `internal/app/bootstrap.go`. Construct its
   dependencies there; modules must not reach into unrelated implementations.

The following is a minimal application registration, without adding a business
feature:

```go
func (Module) Register(ctx *module.Context) error {
    if err := ctx.Permissions.Register(permission.Definition{
        Name: "sample.record.read", Module: "sample", DisplayName: "Read samples",
        Scope: permission.ScopeWorkspace,
    }); err != nil { return err }
    return ctx.Applications.Register(application.Descriptor{
        ID: "sample", DisplayName: "Sample", Description: "A compiled sample app.", Version: "1.0.0",
        ModuleDependencies: []string{"sample"},
        // Startup dependencies are not user access rules.
        RequiredPermissions: []string{"sample.record.read"},
        Frontend: application.Frontend{EntryRoute: "/sample", NavigationID: "sample"},
        APIContractVersion: "v1",
        Access: application.Access{Entry: application.PermissionPolicy{
            Match: application.PermissionMatchAll,
            Permissions: []string{"sample.record.read"},
        }},
    })
}
```

The runtime validates the application descriptor after all modules initialize:
module dependencies, static permissions, capabilities, and access-policy
permissions must have registered successfully. `RequiredPermissions` expresses
registration dependencies only. `Access.Entry` and optional `Access.Settings`
are the user authorization policies: each explicitly requires either all or
any listed permissions, whose registered scope (platform or workspace) is
enforced by the discovery service. The non-business fixture in
`internal/framework/runtime/runtime_test.go` exercises this contract.

## Frontend

1. Create `web/src/modules/<application>/` and export one compiled `AppModule`.
2. Add typed React routes and permission-aware navigation. Reuse shared UI
   primitives from `web/src/components/ui.tsx` and the existing AppShell.
3. Add `application: { id, entryRoute, navigationID, apiContractVersion }` to
   the module. All four values must exactly match the backend descriptor.
4. Register the frontend module in `web/src/app/bootstrap/modules.ts`.
5. Test module resolution, routes, navigation permissions, and error/empty
states. The Applications page compares backend discovery metadata against the
compiled frontend contract and reports mismatches clearly. Compatibility does
not authorize access: backend-provided entry/settings authorization controls
whether action links are rendered, and every target route remains server-side
protected.

The browser never evaluates JavaScript received from the backend. Application
settings belong to the application route declared in its manifest; platform,
organization, workspace, and user settings remain owned by their respective
platform services.

## External application enrollment

External services can be registered without copying their metadata into an Enterprise deployment. The service must expose a versioned manifest at:

```text
GET /.well-known/apexvoid/manifest.json
X-ApexVoid-Enrollment-Code: <one-time-code>
```

The service URL must be an approved host and redirects are rejected. Enterprise parses the manifest with unknown fields disabled and validates the application ID, API contract, application-owned permissions, gateway paths, database identifiers, migration ordering, and SHA-256 pins before showing the approval screen. The one-time enrollment code is the operator-controlled trust bootstrap; deployments that require signed manifests should enforce that at the service boundary before exposing the manifest.

Example manifest:

```json
{
  "manifest_version": "v1",
  "application": { "id": "reports", "display_name": "Reports", "description": "Trusted reports", "version": "1.0.0", "api_contract_version": "v1" },
  "service": { "identity": "reports-service", "health_path": "/health", "enrollment_path": "/.well-known/apexvoid/enroll", "frontend_route": "/apps/reports", "api_route": "/api" },
  "database": { "name": "apexvoid_reports", "schema": "reports", "role": "apexvoid_reports", "migration_bundle_version": "1.0.0" },
  "permissions": [{ "name": "reports.report.read", "display_name": "Read reports", "description": "View reports", "scope": "workspace" }],
  "access": { "match": "all", "permissions": ["reports.report.read"] },
  "migrations": [{ "version": 1, "path": "/.well-known/apexvoid/migrations/001-init.sql", "sha256": "<64 lowercase hexadecimal characters>" }]
}
```

The administrator must explicitly approve registration, permissions, database, schema, and migrations, and select at least one workspace. Approval creates an isolated database, role, schema, and migration ledger using `DATABASE_PROVISIONING_URL`; the runtime `DATABASE_URL` is not used as an implicit PostgreSQL administrator. The migration worker fetches only the manifest-declared same-origin paths, enforces a size limit, verifies each checksum, rejects privileged/destructive SQL, and records applied checksums in `<schema>._apexvoid_migration_history`.

After migrations, Enterprise posts a short-lived enrollment exchange to `service.enrollment_path`. The payload contains the application identity, generated service credential, dedicated database name/schema/role/password, migration bundle, and API contract version. The raw enrollment code and credentials are never written to the Enterprise database or logs; the service is responsible for storing its credentials securely. A failed attempt leaves the database and ledger intact and records a bounded error so a retry cannot silently destroy state.

Successful external applications are surfaced through the normal framework discovery endpoint and are added to the sidebar on the next bounded refresh or workspace change. Their availability is workspace-specific: a newly enrolled application is not globally visible to every workspace. The existing manual registration and credential rotation endpoints remain available for compatibility, but guided enrollment is the preferred path.

For local development, set `INTEGRATIONS_ALLOWED_SERVICE_HOSTS` to the service DNS names and provide `DATABASE_PROVISIONING_URL` only when testing approval. Do not mount the Docker socket, add a second PostgreSQL container, or grant an external service access to the Enterprise database.
