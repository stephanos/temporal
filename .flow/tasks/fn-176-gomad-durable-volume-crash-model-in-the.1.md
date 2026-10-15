---
satisfies: [R2]
---
# fn-176-gomad-durable-volume-crash-model-in-the.1 Volume conformance harness: storage ops, faults, enumeration and recovery against the Go projection

## Description
Extend the fn-175 conformance harness with a `internal/model/volume` model before any Rust volume code (R2). The Go reference is the generated projection `tools/gomad/internal/model/volume` plus the wasi fault engine (`wasi/volume.go`), exercised through a storage-configured `Environment`. The harness must cover operations, faults, crash enumeration, crash selection, lifecycle and recovery, so the later port tasks land against it.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/volume_conformance_test.go` (new), `tools/gomad/internal/backends/wasm/wasi/testdata/volume-conformance/` (golden corpus), `tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs` (`internal/model/volume` model selector and record kinds), `tools/gomad/internal/backends/wasm/wasmhost/protocol.md` (Conformance driver section)
**Touches:** [tools/gomad/internal/backends/wasm/wasi/volume_conformance_test.go, tools/gomad/internal/backends/wasm/wasi/testdata/volume-conformance/**, tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md]

### Approach
- Session setup record: `StorageModel` (volumes, limits, faults, `SQLiteLockDatabases`), namespace config and optionally a recovery image from an earlier step in the same script.
- Operation records:
  - the fs ops routed to the volume (namespace.go: close, sync, read, write, truncate, open, mkdir, remove, rename), each with an explicit `now`
  - `enumerate{volume, limits, cursor?}`
  - `crash{selections}`
  - `select{seed}`
  - `recover{image}`
- Per step, compare raw reply bytes and errno, the storage event line(s) appended, and on request:
  - snapshots
  - operations (per volume, in sorted key order)
  - the enumeration, including every `CrashState.Identity`, `Volume`, `PendingSHA256`, `Complete` and `Capacity`, and the cursor
  - the image digest (the fn-176.3 definition; until then, Go's identity)
  - the fn-175 namespace digest
- Seed the scripted corpus from the existing Go tests:
  - `internal/model/volume/volume_test.go` (sync visibility, resumable enumeration, ordering after crash, durable clone, fault observer, Work budget resume and stale cursor, lifecycle restart)
  - `internal/model/volume/work_test.go`
  - `wasi/volume_test.go` (errno mapping, file and dir sync crash/reopen, fault occurrence, crash fault and limits, enumeration rights, SQLite lock opt-in)
- Add generated sequences: seeded fault placement over write, sync, rename and unlink, followed by enumerate → crash(selection) → recover → continue.
- Add a recorded sample from the `storage_sqlite` guest's storage operations.
- Pin the clock contract: every operation carries the virtual `now` used for volume timestamps, which the helper takes from its model clock, including in broker runs (fn-176.6).
- Native parity is transitive: the projection is drift-tested against the native overlay (`internal/model/volume/projection_test.go`), and this harness tests Rust against the projection. Do not add a native comparison here.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/volume.go:71-464` (model validation, init, decision engine, crash, recovery)
- `tools/gomad/internal/model/volume/volume_test.go`, `tools/gomad/internal/model/volume/work_test.go` (enumeration goldens)
- `tools/gomad/internal/backends/wasm/wasi/volume_test.go` (wasi-level storage cases)
- fn-175.1's `namespace_conformance_test.go` and `conformance.rs` (harness to extend)

### Key context
- `crash-after` is rejected by `validateStorageModel` but referenced by `simulation/provider.go:741,774`. Include a script asserting that the Go reference refuses it, so the Rust port mirrors the refusal; fn-176.7 resolves the dead provider branch.

## Acceptance
- [ ] `go test -tags test_dep -run TestVolumeConformance ./internal/backends/wasm/wasi` (from `tools/gomad`) runs scripted, generated and recorded volume corpora against the Go reference and the conformance helper, reporting the first divergence with seed and step.
- [ ] The checked-in corpus covers every fault action (`io-error`, `sync-error`, `no-space`, `short-write`, `torn-write`, `crash-before`), the invalid short/torn-write runtime case, Work-budget resume, stale cursor rejection, enumeration capacity, lock-database exclusion and recovery from an image; a Go drift test guards it.
- [ ] Until fn-176.2 lands, the Rust side answers `unsupported` and the Rust corpus test is ignored with a reason.


## Done summary
Verified the existing fn-176.1 volume conformance harness on the integrated `gomad` candidate: the Rust volume tests, 46-script Go/Rust conformance gate, and focused Go storage suite are green. The initial Go baseline exposed a stale release helper; rebuilding the repository's normal helper target resolved the tooling mismatch without source changes.

baseline: red initially (`cd tools/gomad && go test -tags test_dep -count=1 -run 'TestVolumeConformance|TestStorage|TestSQLiteStorage' ./internal/backends/wasm/wasi` used a stale helper), resolved by `make -C tools/gomad/internal/backends/wasm helper`; green on rerun

GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)
GATE_SKIPPED:smoke:docs-only - cumulative diff classified tier-B (no executable paths touched)
GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-15T05:24:21Z..2026-10-15T05:29:23Z] (Codex SHIP after correcting the review base from the post-merge empty range to `2237e0276f50f3b4b4b244f32a564ec17caca110..cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d`)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ebcc33c75c957843a87021a7cecdcd53a75e1658
- Tests: cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked volume, make -C tools/gomad/internal/backends/wasm test-conformance, cd tools/gomad && go test -tags test_dep -count=1 -run 'TestVolumeConformance|TestStorage|TestSQLiteStorage' ./internal/backends/wasm/wasi, GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched), GATE_SKIPPED:smoke:docs-only - cumulative diff classified tier-B (no executable paths touched), GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)
- PRs: