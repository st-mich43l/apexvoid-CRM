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

## External application enrollment (v1 — secure bootstrap)

An independent Docker application is NOT compiled into Enterprise. Its Go backend
and React/Vite frontend remain in the standalone repository and its business
database lives on the **existing** PostgreSQL server, separately owned.

### Docker startup and the administrator form

After starting, the standalone service should print its application ID, Docker
DNS service URL, `/.well-known/apexvoid/manifest.json`, and a randomly generated
**single-use, high-entropy enrollment code of at least 32 characters**. The
operator opens Enterprise → Applications → Connect service, enters the service
URL and code, reviews the discovered contract, explicitly approves all five
installation operations, and selects the workspaces to enable.

Never print permanent application service credentials or database passwords to
Docker logs. The enrollment code is sensitive even if it expires quickly. The
service must persist a consumed-code marker and reject re-use. A previously
completed identical enrollment may return a safe idempotent success without
re-writing secrets.

### Authenticated discovery over internal HTTP

Enterprise calls `GET /.well-known/apexvoid/manifest.json` **without sending
the enrollment code as a header**. The service MUST HMAC-SHA256 the exact
manifest response bytes using the secret enrollment code:

- Domain prefix: `apexvoid-manifest-v1\n` (literal newline at end).
- HMAC key: UTF-8 bytes of the enrollment code.
- Response header: `X-ApexVoid-Manifest-Signature: sha256=<64 hex characters>`.
- The response must be the *exact* bytes signed; no middleware reformatting.
- Use the public Go helper `integration.SignManifest(code, manifestJSON)`.

Enterprise checks the signature with constant-time comparison, enforces the
configured Docker DNS hostname allowlist, disables redirects, enforces small
response limits and validates the strict `v1` manifest contract. A SHA-256
migration checksum is anchored by this authenticated manifest. Migration GETs
are same-origin and do not expose the pairing code.

The manifest JSON shape remains the versioned one below:

```json
{
  "manifest_version": "v1",
  "application": {
    "id": "cafe", "display_name": "ApexVoid Café",
    "description": "Coffee counter and photo booth booking",
    "version": "0.1.0", "api_contract_version": "v1"
  },
  "service": {
    "identity": "cafe-service", "health_path": "/health",
    "enrollment_path": "/.well-known/apexvoid/enroll",
    "frontend_route": "/apps/cafe", "api_route": "/api"
  },
  "database": {
    "name": "apexvoid_cafe", "schema": "cafe",
    "role": "apexvoid_cafe", "migration_bundle_version": "0.1.0"
  },
  "permissions": [{
    "name": "cafe.order.read", "display_name": "View orders",
    "description": "View café orders", "scope": "workspace"
  }],
  "access": {"match": "all", "permissions": ["cafe.order.read"]},
  "migrations": [{
    "version": 1,
    "path": "/.well-known/apexvoid/migrations/001-init.sql",
    "sha256": "<exact SHA256 of published SQL bytes>"
  }]
}
```

### Protected credential enrollment

Enterprise creates the dedicated database and role with an explicitly
configured `DATABASE_PROVISIONING_URL` administrator identity. It encrypts
the role password at rest with a stable, independent
`DATABASE_PROVISIONING_KEY` (at least 32 random characters). Back up this
key: changing it invalidates resumption of unfinished installations. Neither
the application role nor the Enterprise runtime role should be the database
provisioner.

App IDs determine immutable database ownership names: `cafe` maps to database
`apexvoid_cafe`, schema `cafe`, and role `apexvoid_cafe`. For IDs with `-`
or `.`, those separators are canonicalized to `_`; database/role collisions
are rejected. A pre-existing unclaimed database/role is **never** adopted.
The restricted login owns only its schema, not the entire database, and has
no superuser, role creation or database creation privileges. Migration SQL is
executed with that restricted login; statement timeouts and transaction locks
apply. No Python, shell or Docker socket execution is required.

After approved migrations, Enterprise `POST`s an AES-256-GCM JSON envelope to
the service's `enrollment_path`. The one-time code is never sent over HTTP:

```json
{
  "version": "v1",
  "nonce": "<base64url-no-padding 12-byte random nonce>",
  "ciphertext": "<base64url-no-padding AEAD encrypted credential JSON>"
}
```

Encryption key: SHA-256 of UTF-8 `apexvoid-enrollment-v1:` concatenated
with the enrollment code. AEAD associated data:
`apexvoid-enrollment-v1` (UTF-8). Plaintext JSON contains
`application_id`, `service_credential`, `api_contract_version`, and
`database: {name, schema, role, password, migration_bundle_version}`.
Use public `integration.OpenEnrollment(code, payload)` to authenticate/decrypt,
validate expected identity, persist secrets securely, and mark the code
consumed before acknowledging completion.

Enterprise also sends a fresh random
`X-ApexVoid-Enrollment-Challenge` header. After **persisting** the received
service secret, the application MUST return
`X-ApexVoid-Enrollment-Ack: sha256=<HMAC hex>`. Compute that HMAC using the
permanent service credential as key and UTF-8
`apexvoid-enrollment-ack-v1\n` concatenated with the challenge as its
message. Use the SDK helper
`integration.SignEnrollmentAcknowledgement(serviceCredential, challenge)`.
Enterprise checks this proof before activating the application. Do not return
a successful acknowledgement when credentials have not been durably saved.

This protects permanent bootstrap secrets on an administrator-controlled
internal HTTP network, but TLS is additionally required for deployments
across untrusted networks. Protect the enrollment code at its source;
low-entropy operator-supplied secrets are unacceptable.

### Idempotence, review and lifecycle

Each installation has an explicit persisted ownership identity. Another
application cannot adopt its database, role or migration history. Failed
attempts reuse encrypted role credentials without silently rotating the
password. PostgreSQL migrations are run in transactions under application
advisory locks; the application and Enterprise each track migration checksums.
SQL regex screening is **only defense in depth**: the actual boundary is the
database/role isolation, authenticated reviewed manifest, restricted database
grants and administrator approval. Do not treat checksums as a publisher trust
mechanism.

The app remains disabled in the gateway until enrollment, service health
verification, workspace grants and final activation succeed. Final activation
uses a transaction. Failed attempts remain tracked and can be retried.

The main Applications page refreshes discovery on a bounded 30-second polling
interval; no Enterprise image rebuild is needed to show newly registered
external apps. The existing manual registration and external service APIs
continue for compatibility. Updates are currently **detected**, not silently
installed or migrated; a separate reviewed update workflow is required.

For application deployment, provision a trusted shared Docker network and add
the service DNS hostname to `INTEGRATIONS_ALLOWED_SERVICE_HOSTS`. Configure
`DATABASE_PROVISIONING_URL` and `DATABASE_PROVISIONING_KEY` in Enterprise
secrets. The standalone service must store its enrollment state in a persistent
mount/secret store; process-local memory is insufficient across restarts.
