# Phase 8 verification

This matrix records verification performed on the Phase 8 foundation branch.
`PASS` is supported by an executed command or test; `PENDING` is deliberately
not treated as a production-freeze pass.

| Requirement | Status | Evidence |
|---|---|---|
| Module registry | PASS | `go test ./...`; deterministic dependency, duplicate, capability, and permission validation tests |
| Application registration | PASS | Registry/runtime unit tests validate immutable descriptors, access-policy references, and `vN` API contract versions |
| Application discovery | PASS | Authenticated workspace discovery integration coverage; unauthenticated discovery route returns `401` |
| Application authorization | PASS | Core service tests cover all-of, any-of, platform/workspace scopes, and separate settings access; frontend mounted tests hide unauthorized actions |
| Multi-workspace isolation | PASS | PostgreSQL integration suite and isolated stack attachment read from a second workspace returned `404` |
| Authentication | PASS | PostgreSQL integration suite plus isolated login and initial-password-change smoke test |
| Customization contracts | PASS | PostgreSQL integration suite passed |
| Non-root attachment storage | PASS | Fresh Compose volume: backend `uid=100(apexvoid)`, directory `0750`, uploaded file `apexvoid:apexvoid 0600`; upload/download and post-restart read-back passed. A simulated legacy root-owned directory was repaired by the documented one-off command to `apexvoid:apexvoid 0750` |
| PostgreSQL migrations | PASS | Fresh production Compose stack reached readiness; PostgreSQL integration suite passed |
| Production images | PASS | Isolated Compose built backend and Nginx frontend images successfully |
| Reverse proxy | PASS | Isolated loopback Nginx proxy returned `/ready`; a direct SPA route returned `200` HTML |
| Secure cookies | PENDING | Login response sets `HttpOnly; Secure; SameSite=Lax`; browser acceptance through an operator-managed HTTPS ingress is not available in the isolated HTTP-only Compose test |
| Persistent volumes | PASS | Full isolated container restart preserved the uploaded attachment and it downloaded with matching bytes |
| Database backup | PASS | Quiesced custom-format `pg_dump -Fc` archive was created from the isolated stack |
| Database restore | PASS | Archive restored with `pg_restore` into fresh `apexvoid-phase8-restore` volumes; readiness, login, and record access passed |
| Attachment backup/restore | PASS | Attachment tar archive restored into the fresh stack and returned byte-for-byte matching content |
| Application version semantics | PASS | Go validation and frontend registry tests cover matching and incompatible declared API contract versions |
| Platform UI consistency | PASS | Frontend lint/typecheck and 9 mounted/unit test files passed; Application directory covers available, unauthorized, and contract-issue states |
| Existing Contacts compatibility | PASS | PostgreSQL integration suite passed, including attachment and tenant-isolation paths |
| Existing CRM compatibility | PASS | PostgreSQL integration suite passed, including lead conversion and opportunity paths |
| Developer application integration | PASS | Compiled-in registration fixture and application guide exercise the extension contract |

## Commands run

```text
go vet ./...
go test ./...
go build ./...
APEXVOID_TEST_DATABASE_URL=… go test -tags integration ./tests/integration
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
docker compose -p apexvoid-phase8-config --env-file .env.production.example \
  -f docker-compose.production.yml config
docker compose -p apexvoid-phase8-test --env-file .env.production.example \
  -f docker-compose.production.yml up -d --build
```

The production smoke test used only the project-scoped
`apexvoid-phase8-test` and `apexvoid-phase8-restore` containers, networks, and
volumes with loopback ports `18080` and `18081`. It created synthetic data
only. The verification included backend/frontend image builds, PostgreSQL and
backend health, SPA fallback, secure-cookie attributes, initial setup,
non-root contact attachment upload/download, cross-workspace denial, full
restart persistence, a quiesced backup, and an isolated restore/read-back.

Vite emitted its standard single-chunk size warning during production builds;
this is a performance follow-up, not a build or framework correctness failure.
