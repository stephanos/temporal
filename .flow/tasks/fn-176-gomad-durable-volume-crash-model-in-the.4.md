---
satisfies: [R1, R2]
---
# fn-176-gomad-durable-volume-crash-model-in-the.4 Rust storage fault engine, storage-model validation and namespace-to-volume routing

## Description
Port the storage fault engine, storage-model validation and the namespace-to-volume routing into Rust (R1, R2), so that a storage-configured Rust namespace answers every storage operation, including faults and `crash-before`.

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/src/volume/faults.rs` (new), `tools/gomad3/backends/wasm/wasmhost/src/volume/config.rs` (new: validation), `tools/gomad3/backends/wasm/wasmhost/src/namespace.rs` (storage routing branches), `tools/gomad3/backends/wasm/wasmhost/src/conformance.rs`
**Touches:** [tools/gomad3/backends/wasm/wasmhost/src/volume/**, tools/gomad3/backends/wasm/wasmhost/src/namespace.rs, tools/gomad3/backends/wasm/wasmhost/src/conformance.rs]

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
- `tools/gomad3/backends/wasm/wasi/volume.go` (whole file)
- `tools/gomad3/backends/wasm/wasi/namespace.go:335-388,425-610,674-960` (storage branches)
- `tools/gomad3/backends/wasm/wasi/volume_test.go:23-280`

## Acceptance
- [ ] The full volume conformance corpus passes through the Rust namespace with storage routing (not only through the volume module directly), including every fault action, storage-transcript capacity and the lock exception.
- [ ] Invalid storage models (each `validateStorageModel` rule, including `crash-after`) are refused by the Rust validator with `invalid` (Rust unit tests mirroring `wasi/volume_test.go` cases).


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
