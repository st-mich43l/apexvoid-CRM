# Phase 8 baseline audit

Audit performed against `master` commit `777814fa62fb07f0e0a34b05a9ec0f0ad0392cc4` before Phase 8 changes.

| Area | Baseline | Action in Phase 8 |
|---|---|---|
| Module lifecycle | Implemented and correct: deterministic ordering, duplicate/missing/cycle checks, migrations, typed metadata registries | Preserve; extend validation only for application contracts |
| Entity, permission, capability, event, extension metadata | Implemented with duplicate and basic metadata validation | Preserve; application manifests now validate references to registered contracts |
| Platform identity, organization, workspace, RBAC | Implemented with dedicated modules and integration coverage for tenant isolation and administrator protection | Preserve; no authorization weakening |
| Customization | Implemented for supported workspace entities, effective schemas, form sections, values, and saved views | Preserve existing Contacts/CRM compatibility; no dynamic SQL engine |
| Application architecture | Missing: Contacts and CRM were modules with compiled frontend code but no typed backend application identity or discoverable frontend contract | Add typed application registry, discovery endpoint, compiled frontend contract verification, and fixture test |
| Shared UI foundation | Implemented: AppShell, theme tokens, core primitives, navigation registry, loading/empty/error states | Add an Applications platform workflow and align navigation sections to Platform/Workspace/Administration/Settings |
| API/error infrastructure | Implemented: versioned routes, request IDs, safe common error body, auth/workspace middleware | Retain current contracts; do not introduce a parallel API framework |
| Transactions/events | Implemented: transaction helper, post-commit hooks, in-process typed event bus | Correct CRM's silent post-commit event failure; document non-durable semantics |
| Production configuration and deployment | Incomplete: secure production bootstrap validation, reverse-proxy routing, production Compose path, and operations runbook were missing | Add production validation, Nginx API proxy, static frontend production Compose, environment template, and runbook |
| Developer experience/docs | Incomplete: README described an earlier pre-CRM state and no authoritative application authoring guide existed | Refresh README/architecture and add the application guide |

Not required before future application development: a marketplace, runtime code
upload, database-stored JavaScript, runtime Go plugins, arbitrary settings
key/value storage, a dynamic SQL engine, or an event broker. Those are outside
the stable compiled-in framework contract.
