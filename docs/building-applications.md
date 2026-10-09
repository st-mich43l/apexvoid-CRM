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
        RequiredPermissions: []string{"sample.record.read"},
        Frontend: application.Frontend{EntryRoute: "/sample", NavigationID: "sample"},
    })
}
```

The runtime validates the application descriptor after all modules initialize:
module dependencies, required permissions, and required capabilities must have
registered successfully. The non-business fixture in
`internal/framework/runtime/runtime_test.go` exercises this contract.

## Frontend

1. Create `web/src/modules/<application>/` and export one compiled `AppModule`.
2. Add typed React routes and permission-aware navigation. Reuse shared UI
   primitives from `web/src/components/ui.tsx` and the existing AppShell.
3. Add `application: { id, entryRoute, navigationID }` to the module. Those
   values must exactly match the backend application descriptor.
4. Register the frontend module in `web/src/app/bootstrap/modules.ts`.
5. Test module resolution, routes, navigation permissions, and error/empty
   states. The Applications page compares backend discovery metadata against
   the compiled frontend contract and reports mismatches clearly.

The browser never evaluates JavaScript received from the backend. Application
settings belong to the application route declared in its manifest; platform,
organization, workspace, and user settings remain owned by their respective
platform services.
