# fn-176 source gates (2026-10-15)

Candidate: branch `volume-batch` rebased on gomad `2237e0276f`; gates on `67dfc9d9f4` plus the exhaustive-switch lint fix and identity re-pin (final tip in the worker report). Host: Linux/arm64 (shared, not a quiet measurement host). Toolchains: Go 1.27.1 (stock), Rust 1.94.1. Logs: `/home/agent/wt/volume-tmp/gates-67dfc9d/` (local, not retained in the repo).

| Gate | Command | Exit |
| --- | --- | --- |
| Rust fmt | `cargo +1.94.1 fmt --check` (wasmhost) | 0 |
| Rust clippy | `cargo +1.94.1 clippy --release --locked --all-targets [--features conformance\|diagnostic-profile] -- -D warnings` | 0 / 0 / 0 |
| Rust tests | `cargo +1.94.1 test --release --locked [--features diagnostic-profile]` | 0 / 0 |
| Go-vs-Rust volume differential | `make -C tools/gomad/internal/backends/wasm test-conformance` (46 volume scripts + namespace corpus) | 0 |
| validate | `make -C tools/gomad validate` | 0 |
| architecture | `GOMAD3_ARCH_ALL_PLATFORMS=1 go test -tags test_dep ./internal/archcheck` | 0 |
| lint | `make lint-code-gomad-wasm` / `make lint-code-fast` (base 2237e0276f), `go test ./cmd/tools/lintcode` | 0 / 0 / 0 |
| portable | model/evidence/host/target/runner suites; cli, runner (native exclusions), native toolchain/soak publishers | 0; 0 failures (fast-batch baseline 0) |
| WASM packages | `go test -p 4 -tags test_dep ./internal/backends/wasm/... ./internal/model/volume` | 0 |
| Temporal crash/reopen (non-frozen) | `GOMAD_WASM_TEMPORAL_RECOVERY_ROOT=… go test -run '^TestTemporalCommittedWorkflowFreshGuestCrashReopen$' ./wasi` | 0 (135 s; split-owned, Go served only fd_write; images 6.5–7.5 MB) |
| Revocation stress | split/transport revocation tests ×50 under 12 busy loops | 0 |

Reviews: per-task claude impl reviews (`claude:claude-opus-5-5:high`), round 1 NEEDS_WORK ×7 (all findings addressed), round 2 SHIP ×7.

Not run here (pending the frozen-candidate requalification): `test-wasm`, scheduling 32×25 + 32 forced replays, retained replays, operator, storage crash/reopen (frozen), network, clusters, `qualify-workloads`, SQLite selections and catalogue (mandatory for fn-176).
