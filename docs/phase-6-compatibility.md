# Phase 6 compatibility notes

## Existing contracts retained

- Compiled entity definitions remain registered through the framework entity registry. Runtime configuration may decorate an entity, but it cannot create arbitrary entities or executable behavior.
- `contacts_custom_field_definitions` and `contacts_contacts.custom_values` remain the only definition and value stores for `contacts.contact` custom fields. The customization service reads them through the Contacts public service instead of copying them.
- Every business table is workspace-scoped. New relationships use composite workspace-aware foreign keys where the referenced table supports them.
- Application services own transaction boundaries through `database.TxManager`; repository calls use the transaction carried in context.
- HTTP handlers derive workspace and principal from middleware. Clients never provide a workspace identifier in request bodies.
- React modules are compiled-in `AppModule`s and use workspace/user query keys. The existing AppShell, theme provider, and semantic tokens stay in place.

## Phase 6 design choices

- Runtime fields, form sections, and saved views are stored by the new `customization` module for entities other than Contacts. An effective-schema read combines compiled metadata with workspace configuration.
- Contacts remains backward compatible through an adapter: built-in Contacts metadata comes from the framework registry, and its existing field definitions are projected into the effective schema.
- Field values are plain JSON values validated against a strict type allow-list. Filters and sorting are structured configuration only; JavaScript, SQL, and expressions are not accepted.
- CRM will consume the configuration API for `crm.lead` and `crm.opportunity` custom values. Its conversion use case will run inside one transaction and reuse the Contacts service with the same transaction context.

## Delivery sequencing

1. Shared configuration contracts, schema API, and reusable field controls.
2. CRM pipeline/stage persistence and configuration APIs.
3. Leads, opportunities, conversion, lifecycle, and integrity tests.
4. CRM screens, settings, saved views, and responsive validation.
