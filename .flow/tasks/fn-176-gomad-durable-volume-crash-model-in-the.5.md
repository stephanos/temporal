---
satisfies: [R1, R3]
---
# fn-176-gomad-durable-volume-crash-model-in-the.5 Helper-owned storage runs: protocol, helper-originated storage_crash and strict Go evidence decoder

## Description
Switch helper-owned storage runs (no Broker) to the Rust volume, and give Go a strict decoder for the helper-produced storage evidence (R1, R3). Broker runs follow in fn-176.6.

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/src/protocol.rs`, `tools/gomad3/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad3/backends/wasm/wasmhost/src/model.rs` (storage evidence streams, helper-originated `storage_crash`), `tools/gomad3/backends/wasm/wasi/process.go`, `tools/gomad3/backends/wasm/wasi/helper_model.go`, `tools/gomad3/backends/wasm/wasi/volume.go` (StorageImage becomes an opaque blob holder), `tools/gomad3/backends/wasm/wasi/storage_evidence.go` (new decoder), `tools/gomad3/backends/wasm/wasi/identity.go`, `tools/gomad3/backends/wasm/wasi/integration_test.go`, `tools/gomad3/backends/wasm/wasmhost/tests/engine.rs`
**Touches:** [tools/gomad3/backends/wasm/wasmhost/src/protocol.rs, tools/gomad3/backends/wasm/wasmhost/src/engine.rs, tools/gomad3/backends/wasm/wasmhost/src/model.rs, tools/gomad3/backends/wasm/wasi/process.go, tools/gomad3/backends/wasm/wasi/helper_model.go, tools/gomad3/backends/wasm/wasi/volume.go, tools/gomad3/backends/wasm/wasi/storage_evidence.go, tools/gomad3/backends/wasm/wasi/identity.go, tools/gomad3/backends/wasm/wasi/integration_test.go, tools/gomad3/backends/wasm/wasmhost/tests/engine.rs]

### Approach
- Gate: in helper-owned runs with `Config.Storage != nil`, the helper owns namespace and volume. This extends fn-175.4's gate, which excluded storage.
- Execute request additions (strict serde, `deny_unknown_fields`):
  - the `StorageModel`
  - the Go-computed storage-model identity
  - `StorageCrashLimits` and `StorageCrashSelection`
  - a `return_image` flag
  - an optional recovery-image descriptor (bytes and SHA-256); the image itself follows in chunked frames after the namespace image (fn-175.4 precedent)
- `bootIdentity` inputs (`simulation/provider.go:216-240`) stay unchanged.
- `return_image` (decision): the helper always computes the image identity, needed by `backend/model_evidence.go`. It ships the image blob only when the request asks, and the simulation provider asks for every storage run it may restart.
- Helper-originated crash: on `crash-before`, the helper stops before returning the syscall to the guest. It flushes partial evidence, computes enumeration (if limits were set), crash selection and the image, and reports the `storage_crash` terminal with an interrupted, unfinished choice prefix. Go no longer sends a `storage_crash` interrupt for gated runs. `helper_model.go:175-195` terminal validation still applies.
- Evidence out, at termination, in 64 KiB chunks:
  - stream `storage` (event lines)
  - stream `storage_state`: a versioned binary envelope with snapshots, operations, per-volume enumerations and the image identity, with a total-length and SHA-256 trailer so truncation is detectable
  - stream `storage_image` (blob)
- Go decodes these strictly into the existing `volume.VolumeSnapshot`, `volume.Operation`, `volume.CrashEnumeration` and `StorageImage`. Undecodable, truncated or trailing-data evidence is refused as `invalid` (never a silent pass), matching `collectHelper`.
- Bounds: image ≤ sum of volume capacities + metadata (≤ `FilesystemBytes` + 1 MiB), else `capacity`.
- Ordering parity with `process.go:383-394`: enumerate each configured volume in order, then crash selection. An enumeration error is `invalid` before image production.
- Parity: extend the Go-owner (`remoteModel=true`) vs helper-owner pattern to storage fixtures: SQLite guest recovery, sync crash/reopen, a fault run and a `crash-before` run. Require equal stdout, transcript digest, storage transcript, snapshots, operations, enumerations and recovered history after reopen from the returned image.
- Retained replay (`simulation/retained.go`) re-executes and stores no image; confirm the new evidence decodes through `BindWASMVolumeEvidence` (`simulation/provider.go:846-880`) unchanged.
- Identity: `wasi/identity.go` binds `volume.ImplementationSHA256()`. Keep it while the Go reference path exists. The helper digest already binds the Rust model (protocol.md cache key section).

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasi/process.go:238-470,600-760` (storage request fields, evidence collection, interrupt and terminal validation)
- `tools/gomad3/backends/wasm/wasi/helper_model.go:100-229`
- `tools/gomad3/backends/wasm/simulation/provider.go:140-240,520-560,730-880`
- `tools/gomad3/backends/wasm/backend/model_evidence.go`
- `tools/gomad3/backends/wasm/wasmhost/src/model.rs:241-320` (evidence queue and flush)

### Key context
- fn-153.20 moves Go helper-frame readers to the shared strict JSON decoder. Use it for any JSON parts; the binary envelope decoder must refuse trailing data, truncation and unknown versions without a hand-written token scanner.

## Acceptance
- [ ] Non-broker storage runs make zero storage callbacks to Go (the Go fs-served counter from fn-175.5 is 0), including fault and `crash-before` runs.
- [ ] Go-owner vs helper-owner parity passes on the storage fixtures above, including reopen from the helper-returned image.
- [ ] Invalid storage config, crash selection or recovery image → `invalid` before `_start`; truncated or corrupt storage evidence → `invalid` refusal (Go unit tests in `storage_evidence_test.go` and `wasmhost/tests/engine.rs`).
- [ ] `go test -tags test_dep -count=1 -run 'TestStorage|TestSQLiteStorage|TestHelper' ./backends/wasm/wasi ./backends/wasm/simulation` (from `tools/gomad3`) passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
