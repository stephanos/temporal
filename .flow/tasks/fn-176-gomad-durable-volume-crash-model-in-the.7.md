---
satisfies: [R3, R4]
---
# fn-176-gomad-durable-volume-crash-model-in-the.7 Durable gates with the Rust volume, Go reference retention and docs

## Description
Pass the durable gates with the Rust volume, settle what Go keeps, and update the docs (R3, R4).

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasi/integration_test.go` (gate assertions), `tools/gomad3/backends/wasm/simulation/provider.go` (dead `crash-after` branch), `tools/gomad3/backends/wasm/wasi/volume.go`, `tools/gomad3/internal/archcheck/edges.go`, `tools/gomad3/backends/wasm/architecture_test.go`, `tools/gomad3/backends/wasm/README.md`, `tools/gomad3/backends/wasm/CLI.md`, `tools/gomad3/backends/wasm/QUALIFICATION.md`, `tools/gomad3/backends/wasm/wasmhost/protocol.md`, `tools/gomad3/README.md`, `.flow/artifacts/` (gate record)
**Touches:** [tools/gomad3/backends/wasm/wasi/integration_test.go, tools/gomad3/backends/wasm/simulation/provider.go, tools/gomad3/backends/wasm/wasi/volume.go, tools/gomad3/internal/archcheck/edges.go, tools/gomad3/backends/wasm/architecture_test.go, tools/gomad3/backends/wasm/README.md, tools/gomad3/backends/wasm/CLI.md, tools/gomad3/backends/wasm/QUALIFICATION.md, tools/gomad3/backends/wasm/wasmhost/protocol.md, tools/gomad3/README.md]

### Approach
- Gates (run alone; host memory budget):
  - `TestTemporalCommittedWorkflowFreshGuestCrashReopen` (integration_test.go:2067, needs `GOMAD_WASM_TEMPORAL_RECOVERY_ROOT`): enumeration complete, and per-state reopen with identical stdout, storage transcript and interrupted choice bytes.
  - `make -C tools/gomad3/backends/wasm qualify-workloads`, SQLite durable `success` and `canary` cells (`TestActualSQLiteDurableQualification`) with the env from the Makefile.
  - The Go fs-served counter must be 0 (non-Broker) or fd-1/2-only (Broker) in both.
- Native-catalogue-dependent SQLite selection cells (`TestActualSQLiteSelectedCrashQualification`) stay native-owned (fn-128/fn-149) and are out of R4.
- Retained-artifact replay: run the existing retained replay selectors (`backends/wasm/simulation` retained tests and `TestProviderActualCrashDurableRestart`) on newly produced evidence.
- What Go keeps (decision):
  - The generated `tools/gomad3/volume` projection stays. It is the R2 oracle and supplies decoder types, it is drift-tested against the native overlay, and it costs nothing to maintain.
  - The Go namespace and volume execution path in `wasi` stays only as the `remoteModel` reference and for stock-profile runs.
  - No production helper-owned or Broker run executes it. Assert this with the fs-served counter in the gates.
  - This deviates from "Go keeps only a decoder", because R2 needs a Go oracle.
- Delete the dead `crash-after` branch in `simulation/provider.go:741,774` (it is unreachable because `validateStorageModel` rejects the action), with a test.
- Update `internal/archcheck/edges.go` and `backends/wasm/architecture_test.go` only if imports changed.
- Docs:
  - README "Storage volumes" row and storage interrupt text (README.md:28,50,120-128)
  - protocol.md storage config, evidence streams, image frames and helper-originated `storage_crash`
  - CLI.md storage sections (324-339, 373-378)
  - QUALIFICATION.md SQLite rows (31-33, 129-151)
  - a Rust sentence in `tools/gomad3/README.md:25-31`
- Record gate results under `.flow/artifacts/`.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasi/integration_test.go:1635,1831,2067`
- `tools/gomad3/backends/wasm/Makefile:55-80` (qualify-workloads)
- `tools/gomad3/backends/wasm/qualification/sqlite_durable_test.go`
- `.flow/artifacts/wasm-section-acceptance-2026-10-12.md:50-80`

## Acceptance
- [ ] Temporal committed-workflow crash/reopen gate passes with the Rust volume (complete enumeration, every reopen identical).
- [ ] SQLite durable `success` and `canary` cells pass via `qualify-workloads`.
- [ ] Retained replay selectors pass on newly produced storage evidence; corrupt evidence is refused (not silently passed).
- [ ] Docs updated; gate record retained with commands, digests and host.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
