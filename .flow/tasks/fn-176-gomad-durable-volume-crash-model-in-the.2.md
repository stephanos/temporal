---
satisfies: [R1, R2]
---
# fn-176-gomad-durable-volume-crash-model-in-the.2 Rust volume persistence core: durable/volatile state, sync semantics, snapshots and identities

## Description
Port the volume persistence model into Rust (R1, R2): durable versus volatile state, pending persistence operations and history, sync semantics (file and directory sync, rename/unlink durability, partial writes), snapshots, operations and SHA-256 identities. Enumeration, lifecycle and recovery follow in fn-176.3.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/volume/mod.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/volume/state.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/volume/hash.rs` (new: framed hasher), `tools/gomad/internal/backends/wasm/wasmhost/src/main.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/volume/**, tools/gomad/internal/backends/wasm/wasmhost/src/main.rs, tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs]

### Approach
- Source of truth: `tools/gomad/internal/model/volume/volume_generated.go` (projection of the native overlay `gomadfs/volume.go`) and `fs_generated.go`/`handles_generated.go` for the fs/handle layer the volume hooks into.
- Port `VolumeConfig`, `VolumeLimits`, `diskState`, `persistenceEffect`, `persistenceOperation`, `volumeState{persisted, pending, history, nextOperation, last}` (volume_generated.go:131-195), `ConfigureVolumes` (:197), `syncNodeLocked` (:816), `Operation`, `SnapshotEntry` and `VolumeSnapshot`.
- Identities must be byte-identical to Go, because they are compared against native catalogues (`qualification/native_crash.go` `MatchedNativeCrashEnumeration`):
  - Port the framed hasher (`internal/model/volume/internal/hashwire/hash_generated.go`; domain tags such as `gomad3-volume-snapshot/v1` and `gomad3-volume-effect/v1`) and the `writeHash*` helpers exactly.
  - Add unit vectors generated from Go for every identity function.
- Capacity: `VolumeCapacityError` resources keep their Go classification. Only `volume_bytes` exhaustion maps to errno 51 (`no-space`); others become `capacity("storage-"+resource)` (`wasi/volume.go:278-310`).
- Use ordered containers for every map whose order reaches identities or evidence.
- Hook points: expose the storage-backed node operations the Rust namespace (fn-175.2/.3) calls for nodes under a volume mount. Keep the namespace module storage-agnostic through a small trait or an `Option<VolumeHandle>` on nodes, mirroring Go's `descriptor.storage` and `node.storagePath`.

### Investigation targets
**Required:**
- `tools/gomad/internal/model/volume/volume_generated.go:1-480,780-1378` (state model, configure, sync, identities)
- `tools/gomad/internal/model/volume/internal/hashwire/hash_generated.go` (framing)
- `tools/gomad/internal/model/volume/runtime.go`, `tools/gomad/internal/model/volume/clock.go`, `tools/gomad/internal/model/volume/recorder.go` (hand-written adapters)
- `tools/gomad/internal/backends/wasm/wasi/volume.go:152-232,392-464` (init, tree load, write and resize paths)

### Key context
- The Go projection stays in place, generated and drift-tested; it is the R2 oracle. Do not edit it.

## Acceptance
- [ ] The volume conformance corpus passes for all non-crash scripts (operations, sync visibility, faults that do not crash, capacity) with identical replies, events, snapshots, operations and identities.
- [ ] Identity unit vectors (Go-generated) pass in `cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked volume`.


## Done summary
Verified the already-integrated Rust volume persistence core from merge `cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d` against the generated Go projection: the exact Rust volume tests, 46-script conformance corpus, and focused Go storage suite pass. No source correction was required; the retained broader source-gate record is `.flow/artifacts/fn176-source-gates-2026-10-15.md`.

Implementation provenance is merge `cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d` and original Rust core commit `b2dcd21f5ac797a41ae166dd739c956575ad0283`; the lifecycle baseline is `500e4d921d6e68fc623fdde3da45e9a347803855`, while independent review intentionally used historical base `2237e0276f50f3b4b4b244f32a564ec17caca110`.

baseline: green

GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)
GATE_SKIPPED:smoke:docs-only - cumulative diff classified tier-B (no executable paths touched)
GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)

Tier: session (jev-unavailable(no_key))

stage: impl-review - ran [2026-10-15T05:40:25Z..2026-10-15T05:42:53Z] (Codex `gpt-6.1-sol` at high; three-axis SHIP on `2237e0276f50f3b4b4b244f32a564ec17caca110..500e4d921d6e68fc623fdde3da45e9a347803855`; reviewer and writer are from the same Codex family)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d, b2dcd21f5ac797a41ae166dd739c956575ad0283
- Tests: cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked volume, make -C tools/gomad/internal/backends/wasm test-conformance, cd tools/gomad && go test -tags test_dep -count=1 -run 'TestVolumeConformance|TestStorage|TestSQLiteStorage' ./internal/backends/wasm/wasi, GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched), GATE_SKIPPED:smoke:docs-only - cumulative diff classified tier-B (no executable paths touched), GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)
- PRs: