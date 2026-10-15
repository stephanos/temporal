---
satisfies: [R1, R2]
---
# fn-176-gomad-durable-volume-crash-model-in-the.4 Rust storage fault engine, storage-model validation and namespace-to-volume routing

## Description
Port the storage fault engine, storage-model validation and the namespace-to-volume routing into Rust (R1, R2), so that a storage-configured Rust namespace answers every storage operation, including faults and `crash-before`.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/volume/faults.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/volume/config.rs` (new: validation), `tools/gomad/internal/backends/wasm/wasmhost/src/namespace.rs` (storage routing branches), `tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/volume/**, tools/gomad/internal/backends/wasm/wasmhost/src/namespace.rs, tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs, tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/process_test.go, tools/gomad/internal/backends/wasm/wasi/helper_model_test.go]

Conductor-authorized integrated-batch correction (2026-10-15): address the task .4 integration review's split-output accounting defect now, ahead of its task .6 owner. A remote stdout write followed by local stdout close and an invalid write must share Go's offered-output capacity and terminal count; later remote writes must adopt prior local charges. Preserve descriptor-based routing. Add focused Go-owner versus split regressions, retain the same review receipt, and cite this correction in task .6 evidence. Original acceptance remains unchanged.

### Approach
- Port `storageDecision` (wasi/volume.go:233):
  - `resource\0op` occurrence counting
  - first-match fault lookup
  - action mapping: `io-error`/`sync-error` → 29, `no-space` → 51, `short-write`/`torn-write` shorten the count and torn also gives 29, `crash-before` → modeled storage crash
  - the runtime `invalid("storage write fault must shorten the matched write")`
- `StorageEvent` lines: render the exact Go JSON line, `json.Marshal(StorageEvent)` field order with omitempty, so the storage transcript bytes and the `capacity("storage-transcript")` boundary against `Limits.TranscriptBytes` match Go byte for byte.
- Port `validateStorageModel` (:71) completely: volume/fault/lock-database bounds, writable-preopen and capacity checks, the fault action/operation matrix, ambiguity checks, and lock-path rules. `crash-after` stays rejected. Rust re-validates in addition to Go, and either side's refusal is `invalid` before execution.
- SQLite lock exception: `sqliteLockPath`/`storagePath` (:392-414). `.lock`, `-shm` and `-shm/…` nodes stay namespace-only and never reach the volume.
- Namespace routing: add the storage branches the Go namespace has at namespace.go 335-388 (close, sync), 457-465 (read), 538 (`storageWrite`), 586-598 (truncate), 731-757 (open), 806-816 (mkdir), 836-842 (remove) and 943-953 (rename), with `refreshStorageNode` and `storageResizeAllowed` semantics. Keep namespace filesystem-bytes and volume capacity as distinct classifications.
- `crash-before` produces a typed Rust error that fn-176.5 maps to a helper-originated `storage_crash` termination.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/volume.go` (whole file)
- `tools/gomad/internal/backends/wasm/wasi/namespace.go:335-388,425-610,674-960` (storage branches)
- `tools/gomad/internal/backends/wasm/wasi/volume_test.go:23-280`

## Acceptance
- [ ] The full volume conformance corpus passes through the Rust namespace with storage routing (not only through the volume module directly), including every fault action, storage-transcript capacity and the lock exception.
- [ ] Invalid storage models (each `validateStorageModel` rule, including `crash-after`) are refused by the Rust validator with `invalid` (Rust unit tests mirroring `wasi/volume_test.go` cases).


## Done summary
Fault engine, Go-identical event lines, storage-model validation (every `validateStorageModel` rule, `crash-after` refused, unit-tested in `volume/config.rs`) and namespace storage routing (`namespace.rs`: close, sync, read, write, truncate, open, mkdir, remove, rename; lock exception; filesystem-bytes vs volume capacity). Corpus passes through the Rust namespace.

## Evidence
- Commits: branch `volume-batch` (rebased on gomad 7406c9ed9e); see final report for tip
- Tests: see `.flow/artifacts/fn176-source-gates-2026-10-15.md`
- PRs:
