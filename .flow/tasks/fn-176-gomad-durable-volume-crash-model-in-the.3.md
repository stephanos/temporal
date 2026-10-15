---
satisfies: [R1, R2]
---
# fn-176-gomad-durable-volume-crash-model-in-the.3 Rust crash enumeration, selection, lifecycle and self-contained recovery image

## Description
Port crash enumeration, crash selection, lifecycle advance and durable cloning, and define the helper-owned recovery image (R1, R2). This is the semantic core that the committed-workflow crash/reopen and SQLite durable gates depend on.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/volume/crash.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/volume/image.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/volume/mod.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/volume/**, tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs]

### Approach
- Port `EnumerateCrashStates` (volume_generated.go:291):
  - `CrashEnumerationLimits{States, Operations, Depth, Bytes, Work}`
  - the resumable `CrashExploration` cursor with `pendingSHA256` validation and stale-cursor rejection
  - `enumerationBudget.Take()` charging Work per visited cursor (`internal/model/volume/runtime.go`)
  - `Complete` and `Capacity` results
  - state identities (`crashStateIdentity`, `pendingIdentity`)
- Port `CrashVolumes` (:477), `AdvanceVolumeLifecycle` (:567; handles revoked, fs ESTALE, descriptors cleared before any image is exposed), `AdvanceVolumeLifecycleAfterDivergence` if reachable from wasi, and `SelectCrashOperations(seed)` (:628).
- Recovery image (decision; format change allowed): a versioned binary blob with magic, a version and length-prefixed sections, carrying everything recovery uses:
  - the durable tree
  - per-volume persisted state, history, `nextOperation` and `nextInode`
  - fault occurrence counters
  - the storage-model identity supplied by Go
- Image identity = SHA-256 over the blob. The old identity (SHA-256 of Go `json.Marshal({Snapshots, Occurrences})`) is retired, because it omitted recovery-relevant state.
- Recovery (`NewRecoveredEnvironment` and `loadStorageTree` parity):
  - the model identity in the image must equal the request's, else `invalid` before execution
  - lock-database paths inside an image are refused
  - capacity overflow on reload keeps Go's classification
  - occurrence counters continue from the image
- The Go reference image digest used by the harness becomes this blob's digest computed on the Go side. Add a Go encoder for the same blob, test-only, in the conformance test package, so the harness compares images structurally and by digest.

### Investigation targets
**Required:**
- `tools/gomad/internal/model/volume/volume_generated.go:291-760` (enumerate, crash, lifecycle, select)
- `tools/gomad/internal/model/volume/runtime.go` (`CloneDurable`, Work budget)
- `tools/gomad/internal/backends/wasm/wasi/volume.go:328-392` (CrashStorage, EnumerateStorageCrashStates, NewRecoveredEnvironment)
- `tools/gomad/internal/model/volume/work_test.go`, `tools/gomad/internal/model/volume/volume_test.go` (enumeration goldens)

## Acceptance
- [ ] The full volume conformance corpus passes, including crash, enumeration (with resume and capacity), selection, lifecycle and recovery scripts, with identical identities.
- [ ] Recovery refuses an image whose model identity differs from the request's, an image containing lock-database paths, and a truncated or digest-mismatched image (Rust unit tests and conformance scripts).
- [ ] Recovered runs continue occurrence counting from the image, as Go does (conformance script).


## Done summary
Enumeration, selection, lifecycle and image v1 (`GMVOLIMG`, SHA-256 identity) in Rust; Go owner encodes the same blob in production (`wasi/storage_image.go`). Recovery refuses model-identity mismatch, lock paths, truncation/digest mismatch; continues fault occurrences; admits Go-valid over-capacity durable states (corpus `recover-over-capacity-selection`). Image bound fs+1 MiB after measuring Temporal (≤7.5 MB, fs 512 MiB) and SQLite (≤85 KB, fs 8 MiB); oversize is `capacity: storage-image-bytes` on both owners (corpus `crash-image-capacity`).

## Evidence
- Commits: branch `volume-batch` (rebased on gomad 7406c9ed9e); see final report for tip
- Tests: see `.flow/artifacts/fn176-source-gates-2026-10-15.md`
- PRs:
