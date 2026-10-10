# Temporal server WASI compilation

Assessment date: 2026-10-09. Stack base: `15f56644664f3d3749bab2387aa97936a1cac6dd` on `gomad`.

The existing `cmd/server` entrypoint links as a `wasip1/wasm` executable using stock Go 1.27.0. The experiment lives in the separate `temporal_wasm` clone. SQLite, Prometheus, and the server's normal service graph remain included.

The draft PR stack contains the Prometheus patch update, the all-platform SQLite driver replacement, and the WASI build target, in that order.

## Build

From this clone's root:

```sh
make temporal-server-wasi
```

This writes `temporal-server.wasm`, ignored by Git. The target disables cgo and uses the repository's usual build tags plus `sqlite3_dotlk`:

```sh
GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go build -mod=readonly \
  -tags disable_grpc_modules,sqlite3_dotlk \
  -o temporal-server.wasm ./cmd/server
```

## Dependency changes

Prometheus v1.21.0's `process_collector_wasip1_js.go` has an implicit JavaScript filename constraint despite its explicit `wasip1 || js` build tag. Updating to v1.21.1 selects its renamed [unsupported-platform implementation](https://github.com/prometheus/client_golang/blob/v1.21.1/prometheus/process_collector_not_supported.go). Metrics remain enabled.

[ncruces/go-sqlite3 v0.35.6](https://github.com/ncruces/go-sqlite3/tree/v0.35.6) replaces modernc SQLite on every platform. This release translates SQLite WebAssembly into Go with wasm2go. It needs neither cgo nor an embedded WebAssembly interpreter. The root module removes modernc SQLite and its libc dependencies; ncruces raises several selected `golang.org/x` versions.

The adapter retains Temporal's `sqlite` persistence plugin and `database/sql` driver names. It handles duplicate-key and table-exists errors, registers FTS5 on each connection, and enables double-quoted strings for the existing visibility schema. Named in-memory databases use the driver's shared `memdb` VFS. The connection wrapper implements reset and validation so cancellation of a transaction preserves the in-memory database.

Timestamp parameters retain the previous layout and fractional seconds, including pointer and `driver.Valuer` inputs. The driver's automatic decoder reads both existing timestamps and zone-less schema defaults. Keeping the storage layout preserves comparisons against existing database rows. The [driver documentation](https://github.com/ncruces/go-sqlite3/blob/v0.35.6/driver/driver.go) describes its DSN options.

Native builds use the driver's default file locking. WASI uses `sqlite3_dotlk` for [portable file locking and in-process WAL shared memory](https://github.com/ncruces/go-sqlite3/blob/v0.35.6/vfs/README.md). Databases accessed concurrently must use compatible locking implementations.

## Verification

The native server and WASI server link successfully. Native tests cover server SQL registrations, SQLite error classification, timestamp precision and compatibility, schema loading, shared memory, and transaction cancellation. All SQLite persistence test entries pass on the available Darwin/arm64 host:

```sh
CGO_ENABLED=0 go test -mod=readonly -tags test_dep \
  ./cmd/server ./common/persistence/sql/sqlplugin/sqlite -count=1
CGO_ENABLED=0 go test -mod=readonly -tags test_dep \
  ./common/persistence/tests -run '^TestSQLite' -count=1 -timeout=15m
CGO_ENABLED=0 go test -mod=readonly -tags test_dep,sqlite3_dotlk \
  ./cmd/server ./common/persistence/sql/sqlplugin/sqlite -count=1
```

The server and SQLite package test binaries also compile for WASI:

```sh
GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go test -mod=readonly \
  -tags test_dep,sqlite3_dotlk -c -o .tmp/wasi-server.test.wasm ./cmd/server
GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go test -mod=readonly \
  -tags test_dep,sqlite3_dotlk -c -o .tmp/wasi-sqlite.test.wasm \
  ./common/persistence/sql/sqlplugin/sqlite
```

The changed packages pass `make lint-code-fast` with `GOLANGCI_LINT_FIX=false`; `go mod tidy -diff` and `git diff --check` are clean. Independent review identified the transaction cancellation regression, which the existing regression test reproduced. After the fix, a fresh review found no actionable issues and independently repeated the cancellation test ten times. Writer and reviewers were Codex-family models.

## Runtime work remains

No WASI guest execution, server startup, functional-test execution, or determinism qualification has been performed. The native tests cover the three compatibility gaps identified in the initial compilation experiment: shared in-memory databases, zone-less timestamps, and double-quoted SQL strings. WASI execution still needs to verify these behaviors in the guest. The driver replacement also requires new Gomad qualification; historical modernc libc evidence describes the previous implementation.

The initial SQLite PR CI run fails Gomad compatibility-pack qualification because a server-target request still requires modernc libc in the dependency closure. The replacement removes that dependency. Those packs need reconciliation and the replacement driver needs qualification before claiming Gomad support. CI also identified older `x/sys` and `x/text` selections in the nested mixedbrain module; the SQLite PR updates them, and its race/coverage test binary compiles with `test_dep` and read-only module resolution.

All seven native Linux/arm64 SQLite functional CI jobs passed on PR head `601f430d86`. Subsequent source corrections scope root checks to maintained packages, preserve archived evidence and runtime overlay bytes, register SQL providers in converter tests, and apply the mandatory Go 1.27 formatter updates. Focused tests, formatter idempotence, shellcheck, changed-package lint and generated validation pass locally. Other CI jobs encounter DockerHub unauthenticated pull quotas, and native Gomad qualification remains unresolved. These corrections do not establish a fully passing CI run.

Compiling the full functional-test package and running a one-box server are subsequent milestones. The remembered precedent is Polar Signals' [“(Mostly) Deterministic Simulation Testing in Go”](https://www.polarsignals.com/blog/posts/2024/05/28/mostly-dst-in-go), which combines WASI with runtime changes for scheduling, time, and randomness.
