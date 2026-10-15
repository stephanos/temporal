---
satisfies: [R3, R4]
---
# fn-176-gomad-durable-volume-crash-model-in-the.7 Durable gates with the Rust volume, Go reference retention and docs

## Description
Pass the durable gates with the Rust volume, settle what Go keeps, and update the docs (R3, R4).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/integration_test.go` (gate assertions), `tools/gomad/internal/backends/wasm/cluster/provider.go` (dead `crash-after` branch), `tools/gomad/internal/backends/wasm/wasi/volume.go`, `tools/gomad/internal/archcheck/edges.go`, `tools/gomad/internal/backends/wasm/architecture_test.go`, `tools/gomad/internal/backends/wasm/README.md`, `tools/gomad/internal/backends/wasm/CLI.md`, `tools/gomad/internal/backends/wasm/QUALIFICATION.md`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`, `tools/gomad/README.md`, `.flow/artifacts/` (gate record)
**Touches:** [tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/cluster/provider.go, tools/gomad/internal/backends/wasm/wasi/volume.go, tools/gomad/internal/archcheck/edges.go, tools/gomad/internal/backends/wasm/architecture_test.go, tools/gomad/internal/backends/wasm/README.md, tools/gomad/internal/backends/wasm/CLI.md, tools/gomad/internal/backends/wasm/QUALIFICATION.md, tools/gomad/internal/backends/wasm/wasmhost/protocol.md, tools/gomad/README.md]

### Approach
- Gates (run alone; host memory budget):
  - `TestTemporalCommittedWorkflowFreshGuestCrashReopen` (integration_test.go:2067, needs `GOMAD_WASM_TEMPORAL_RECOVERY_ROOT`): enumeration complete, and per-state reopen with identical stdout, storage transcript and interrupted choice bytes.
  - `make -C tools/gomad/internal/backends/wasm qualify-workloads`, SQLite durable `success` and `canary` cells (`TestActualSQLiteDurableQualification`) with the env from the Makefile.
  - The Go fs-served counter must be 0 (non-Broker) or fd-1/2-only (Broker) in both.
- Native-catalogue-dependent SQLite selection cells (`TestActualSQLiteSelectedCrashQualification`) stay native-owned (fn-128/fn-149) and are out of R4.
- Retained-artifact replay: run the existing retained replay selectors (`internal/backends/wasm/cluster` retained tests and `TestProviderActualCrashDurableRestart`) on newly produced evidence.
- What Go keeps (decision):
  - The generated `tools/gomad/internal/model/volume` projection stays. It is the R2 oracle and supplies decoder types, it is drift-tested against the native overlay, and it costs nothing to maintain.
  - The Go namespace and volume execution path in `wasi` stays only as the `remoteModel` reference and for stock-profile runs.
  - No production helper-owned or Broker run executes it. Assert this with the fs-served counter in the gates.
  - This deviates from "Go keeps only a decoder", because R2 needs a Go oracle.
- Delete the dead `crash-after` branch in `simulation/provider.go:741,774` (it is unreachable because `validateStorageModel` rejects the action), with a test.
- Update `internal/archcheck/edges.go` and `internal/backends/wasm/architecture_test.go` only if imports changed.
- Docs:
  - README "Storage volumes" row and storage interrupt text (README.md:28,50,120-128)
  - protocol.md storage config, evidence streams, image frames and helper-originated `storage_crash`
  - CLI.md storage sections (324-339, 373-378)
  - QUALIFICATION.md SQLite rows (31-33, 129-151)
  - a Rust sentence in `tools/gomad/README.md:25-31`
- Record gate results under `.flow/artifacts/`.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:1635,1831,2067`
- `tools/gomad/internal/backends/wasm/Makefile:55-80` (qualify-workloads)
- `tools/gomad/internal/backends/wasm/qualification/sqlite_durable_test.go`
- `.flow/artifacts/wasm-section-acceptance-2026-10-12.md:50-80`

## Acceptance
- [ ] Temporal committed-workflow crash/reopen gate passes with the Rust volume (complete enumeration, every reopen identical).
- [ ] SQLite durable `success` and `canary` cells pass via `qualify-workloads`.
- [ ] Retained replay selectors pass on newly produced storage evidence; corrupt evidence is refused (not silently passed).
- [ ] Docs updated; gate record retained with commands, digests and host.


## Done summary
Docs (wasm README/CLI/QUALIFICATION, protocol.md storage/split/driver, tools/gomad README); dead `crash-after` provider branch removed with test; gates assert Go fs-served only fd 1/2 writes (Temporal gate helper, SQLite durable cells). Temporal committed crash/reopen gate passes on the Rust volume (non-frozen run). Frozen-candidate requalification (incl. qualify-workloads SQLite success/canary, selected-state and catalogue controls, retained replays) pending conductor go-ahead.

## Evidence
- Commits: branch `volume-batch` (rebased on gomad 7406c9ed9e); see final report for tip
- Tests: see `.flow/artifacts/fn176-source-gates-2026-10-15.md`
- PRs:
