# Phase 8 verification

The following evidence was collected for the Phase 8 foundation changes.

| Area | Status | Evidence |
|---|---|---|
| Core framework | PASS | `go vet ./...`, `go test ./...`, `go build ./...` |
| Module registration | PASS | Existing deterministic/cycle/missing-dependency tests plus framework package tests |
| Application architecture | PASS | `internal/framework/runtime/runtime_test.go` registers a non-business fixture and validates discovery and unknown contracts |
| Authentication | PASS | Existing users tests plus PostgreSQL integration suite |
| RBAC | PASS | PostgreSQL integration suite covers role boundaries and administrator protection |
| Multi-workspace | PASS | PostgreSQL integration suite covers workspace isolation and membership restrictions |
| Customization | PASS | PostgreSQL customization integration coverage passed |
| Shared UI | PASS | `npm --prefix web run lint`, `typecheck`, and `test` (8 files, 17 tests) |
| Light/dark mode | PASS | Existing ThemeProvider tests passed in the frontend suite |
| Administration | PASS | Existing frontend and PostgreSQL workflow coverage passed |
| Secure configuration | PASS | `internal/platform/config` tests verify production cookie, origin, and bootstrap validation |
| Production deployment | PENDING | Production Compose renders with isolated `apexvoid-phase8_*` volumes; build/start/readiness/restart persistence smoke test requires explicit local Docker approval |
| Developer extensibility | PASS | Application guide and non-business registration fixture test |
| Existing module compatibility | PASS | `go test -tags integration ./tests/integration` against local PostgreSQL passed in 12.055s |

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
docker compose -p apexvoid-phase8 --env-file .env.production.example \
  -f docker-compose.production.yml config
```

All completed commands passed. Vite emitted its standard warning that the
single minified client chunk is over 500 kB; this is a performance optimization
opportunity, not a failed build or framework correctness defect.
