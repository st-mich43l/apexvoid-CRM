# Phase 6.4 review matrix

| Confirmed issue | Fix | Code locations | Regression coverage |
| --- | --- | --- | --- |
| Opportunity edit sent transition-only pipeline and stage fields | Added explicit create/patch contracts and a date-aware patch mapper | `web/src/core/api/client.ts`, `web/src/modules/crm/index.tsx` | `TestCRMLeadConversionAndSparsePatch` |
| Browser date-only values did not match Go `time.Time` JSON parsing | Strict `YYYY-MM-DD` transport type with explicit null/clear handling and date-only responses | `internal/modules/crm/transport/http/workspace.go` | CRM HTTP integration test |
| Numeric custom fields were posted as strings | Typed renderer serializes finite decimals and safe integers as JSON numbers | `web/src/modules/crm/index.tsx`, `internal/modules/customization/domain/models.go` | CRM PostgreSQL integration test and domain unit tests |
| Last active Open stage could be archived | Validate the complete resulting stage aggregate inside the workspace transaction | `internal/modules/crm/application/service.go` | CRM PostgreSQL integration test |
| Stage restore/reorder could collide with unique positions | Transactional temporary positions plus normalized final ordering | `internal/modules/crm/application/service.go`, `internal/modules/crm/infrastructure/postgres/repository.go` | CRM pipeline integration coverage |
| Field inventory permission did not match the designer | Read inventory/schema with `customization.schema.read`; reserve manage permission for writes | `internal/modules/customization/transport/http/handler.go` | Customization integration test |
| Sections and saved views lacked management lifecycle endpoints | Added workspace-scoped section list/update and owner/manager saved-view update/delete | `internal/modules/customization/{application,transport,http,infrastructure}` | `TestCustomizationManagementAndTenantIsolation` |
| Saved-view filters were stored without typed validation | Validate fields, bounded operators, and values against effective schema metadata | `internal/modules/customization/domain/models.go` | Customization domain unit tests |

The saved-view UI deliberately maps supported CRM filters to the existing parameterized `search`, `pipeline_id`, and stage/query contracts. It does not introduce a generic query language or dynamic SQL.
