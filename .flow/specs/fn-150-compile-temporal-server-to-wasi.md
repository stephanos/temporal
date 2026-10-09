# Compile Temporal server to WASI

Build and link the existing cmd/server entrypoint as wasip1/wasm using stock Go, isolated in temporal_wasm. This milestone proves compilation. Native SQLite compatibility belongs to the all-platform driver replacement. WASI server startup, external networking and deterministic execution remain future work.

## Acceptance Criteria

- **R1:** A reproducible command emits a linked WASI WebAssembly executable for cmd/server.
- **R2:** Preserve all SQL provider registrations. Replace modernc SQLite with ncruces on every platform while preserving native persistence behavior and the sqlite plugin name.
- **R3:** Correct the Prometheus WASI compile failure using its upstream patch release.
- **R4:** Verify the WASI build, compile the affected tests for WASI, run focused native tests and all SQLite persistence test entries with the replacement driver, and document limitations and artifact details.

## Approach

Publish three stacked draft PRs based on gomad using gh stack. First update prometheus/client_golang from v1.21.0 to v1.21.1. Then replace modernc with ncruces/go-sqlite3 v0.35.6 on all platforms, preserving timestamp storage and decoding, shared in-memory databases, schema quoting, FTS5 and cancellation behavior. Finally add make temporal-server-wasi with sqlite3_dotlk for portable locking, and retain verification evidence and runtime limits in docs/research/gomad/2026-10-09-wasi-compilation.md.

The user's all-platform replacement instruction supersedes the initial platform-specific adapter design recorded in task 1.
