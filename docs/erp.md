# ERP inside ApexVoid Enterprise

ERP is a first-class **compiled-in application suite** in the same Go backend,
React frontend, and PostgreSQL runtime as CRM. ERP is not another core service,
repository, Docker service, or authority for authentication and RBAC.

## Current catalog

The `erp` module currently owns products and services:

- `erp.product` entity with `erp_products` persistence.
- `/api/v1/erp/products` and `/api/v1/erp/products/{id}` workspace-scoped APIs.
- Workspace permissions: `erp.product.read`, `create`, `update`, `archive`.
- React entry `/erp` and catalog at `/erp/products` in the existing app shell.
- Product kinds: physical goods and services; separate unit, SKU, description,
  active/archive status, currency, and exact decimal unit prices.
- SKU uniqueness is per workspace, including archived items, and updates use
  optimistic `version` checks.
- Each query and mutation uses the authenticated workspace context; a caller
  cannot choose a workspace in a business request body.
- Unit price is persisted as PostgreSQL `NUMERIC(20,4)` and transmitted as a
  decimal string; avoid `float64`/`number` in financial calculations.

The catalog is a common ERP source of product definitions, **not** an
inventory ledger, invoice book, or account balance. Creating a product does not
create stock, an invoice, or an accounting entry.

## Boundaries for future ERP modules

- **CRM** continues to own leads, pipelines, and opportunities.
- **Contacts** continues to own customer and company records. Do not duplicate
  customer identities in ERP.
- **ERP Sales** should own quotations and sales orders, and refer to
  opportunities and contacts using narrow domain contracts.
- **ERP Inventory** should own stock transactions and balances; do not store
  quantity on the product catalog as an authoritative stock balance.
- **ERP Purchasing** should own supplier orders and receipts.
- **ERP Invoicing/Accounting** should own invoice and ledger state separately.
  Monetary arithmetic must use exact decimals with explicit currency rules.

Future modules should follow existing `internal/modules` layering and register
their own permissions, schema migrations, metadata, and routes. Avoid reaching
into another module's PostgreSQL tables from application logic. Add intermodule
contracts in the owning module's public `api` package.

## Verification

`go test ./...` covers product validation. The PostgreSQL-backed integration
suite checks registration alongside CRM, authenticated CRUD, SKU collision,
decimal precision, workspace access, optimistic locking, and archive/restore.

This catalog is the first usable ERP capability. Sales, purchasing, stock
movements, invoicing, accounting and country-specific tax compliance are not
implemented yet.
