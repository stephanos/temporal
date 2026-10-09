---
satisfies: [R1, R2, R3, R4]
---
# fn-150-compile-temporal-server-to-wasi.1 Resolve dependencies and link the WASI server

## Description
Compile cmd/server for wasip1/wasm. Repair the Prometheus platform constraint using its upstream patch release. Evaluate a WASI-compatible SQLite driver before deciding whether to omit the provider. Preserve native driver behavior and record compiler evidence and remaining runtime limitations.

## Acceptance
R1: Linked WASI executable and reproducible build command. R2: Native SQL registrations retained; WASI SQLite availability explicitly documented. R3: Prometheus compiles. R4: Focused checks and documentation retained.

## Done summary
Linked the existing Temporal server entrypoint to WASI in the separate temporal_wasm clone. Added a reproducible Make target, selected ncruces SQLite only for WASI (or explicit native test opt-in), preserved native modernc defaults and all SQL plugin registrations, and fixed Prometheus platform selection with an upstream patch update. Added driver error and timestamp-precision tests. Documented runtime limitations separately from compilation success.

Fresh verification passed: server WASI build, affected WASI test compilation, native tests with both drivers, lint with both driver selections, module tidy check, and diff whitespace check. Independent fresh-context review found no actionable issues. No WASI execution, server startup, or determinism claim is made.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: make temporal-server-wasi, CGO_ENABLED=0 go test -mod=readonly -tags test_dep ./cmd/server ./common/persistence/sql/sqlplugin/sqlite -count=1, CGO_ENABLED=0 go test -mod=readonly -tags test_dep,sqlite_ncruces,sqlite3_dotlk ./cmd/server ./common/persistence/sql/sqlplugin/sqlite -count=1, GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go test -mod=readonly -tags test_dep,sqlite3_dotlk -c -o .tmp/wasi-server.test.wasm ./cmd/server, GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go test -mod=readonly -tags test_dep,sqlite3_dotlk -c -o .tmp/wasi-sqlite.test.wasm ./common/persistence/sql/sqlplugin/sqlite, make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT_FIX=false, make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT_FIX=false BUILD_TAG=sqlite_ncruces,sqlite3_dotlk, go mod tidy -diff, git diff --check
- PRs: