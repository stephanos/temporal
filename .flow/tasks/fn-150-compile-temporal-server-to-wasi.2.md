---
satisfies: [R1, R2, R3, R4]
---
# fn-150-compile-temporal-server-to-wasi.2 Replace SQLite driver and publish three WASI draft PRs

## Description
Split the compilation work into a gh stack based on gomad: Prometheus patch update, replace modernc SQLite with ncruces on all platforms, then WASI build target. Preserve native persistence behavior and publish three draft PRs.

## Acceptance
Three draft PRs linked by gh stack with the requested dependency order; native SQLite persistence tests, focused tests, WASI compilation and lint pass; driver runtime compatibility and remaining WASI limitations documented.

## Done summary
Published three draft PRs with gh stack in the requested order. PR 8 updates Prometheus, PR 9 replaces modernc SQLite on all platforms and preserves native persistence compatibility, and PR 10 adds the WASI build target and compilation report. All SQLite persistence test entries, focused native tests, native server build, affected WASI test compilation, WASI server build, module checks, and changed-package lint passed. Fresh independent review found no remaining actionable issues after fixing cancellation data loss. WASI execution and Gomad qualification remain unverified.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4ca90d673954391fcd95dafef9c8eac7d5194036, 8f75b231f8, 1ff842595e
- Tests: CGO_ENABLED=0 go test -mod=readonly -tags test_dep ./common/metrics/... -count=1, GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go build -mod=readonly ./common/metrics/..., CGO_ENABLED=0 go test -mod=readonly -tags test_dep ./cmd/server ./common/persistence/sql/sqlplugin/sqlite -count=1, CGO_ENABLED=0 go test -mod=readonly -tags test_dep ./common/persistence/tests -run ^TestSQLite -count=1 -timeout=15m, CGO_ENABLED=0 go test -mod=readonly -tags test_dep,sqlite3_dotlk ./cmd/server ./common/persistence/sql/sqlplugin/sqlite -count=1, CGO_ENABLED=0 go build -mod=readonly -tags disable_grpc_modules -o .tmp/temporal-server-native ./cmd/server, GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go test -mod=readonly -tags test_dep,sqlite3_dotlk -c -o .tmp/wasi-server.test.wasm ./cmd/server, GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go test -mod=readonly -tags test_dep,sqlite3_dotlk -c -o .tmp/wasi-sqlite.test.wasm ./common/persistence/sql/sqlplugin/sqlite, make temporal-server-wasi, make lint-code-fast GOLANGCI_LINT_BASE_REV=4ca90d673954391fcd95dafef9c8eac7d5194036 GOLANGCI_LINT_FIX=false, go mod tidy -diff, git diff --check
- PRs: https://github.com/stephanos/temporal/pull/8, https://github.com/stephanos/temporal/pull/9, https://github.com/stephanos/temporal/pull/10