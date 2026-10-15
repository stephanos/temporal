---
satisfies: [R1, R3]
---
# fn-176-gomad-durable-volume-crash-model-in-the.5 Helper-owned storage runs: protocol, helper-originated storage_crash and strict Go evidence decoder

## Description
Switch helper-owned storage runs (no Broker) to the Rust volume, and give Go a strict decoder for the helper-produced storage evidence (R1, R3). Broker runs follow in fn-176.6.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/volume/crash.rs`, `tools/gomad/internal/backends/wasm/wasi/identity_test.go`, `tools/gomad/internal/backends/wasm/wasi/storage_evidence_test.go`, `tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs` (storage evidence streams, helper-originated `storage_crash`), `tools/gomad/internal/backends/wasm/wasi/process.go`, `tools/gomad/internal/backends/wasm/wasi/helper_model.go`, `tools/gomad/internal/backends/wasm/wasi/volume.go` (StorageImage becomes an opaque blob holder), `tools/gomad/internal/backends/wasm/wasi/storage_evidence.go` (new decoder), `tools/gomad/internal/backends/wasm/wasi/identity.go`, `tools/gomad/internal/backends/wasm/wasi/integration_test.go`, `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/volume/crash.rs, tools/gomad/internal/backends/wasm/wasi/identity_test.go, tools/gomad/internal/backends/wasm/wasi/storage_evidence_test.go, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/volume.go, tools/gomad/internal/backends/wasm/wasi/storage_evidence.go, tools/gomad/internal/backends/wasm/wasi/identity.go, tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs]

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
- `tools/gomad/internal/backends/wasm/wasi/process.go:238-470,600-760` (storage request fields, evidence collection, interrupt and terminal validation)
- `tools/gomad/internal/backends/wasm/wasi/helper_model.go:100-229`
- `tools/gomad/internal/backends/wasm/cluster/provider.go:140-240,520-560,730-880`
- `tools/gomad/internal/backends/wasm/backend/model_evidence.go`
- `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs:241-320` (evidence queue and flush)

### Key context
- fn-153.20 moves Go helper-frame readers to the shared strict JSON decoder. Use it for any JSON parts; the binary envelope decoder must refuse trailing data, truncation and unknown versions without a hand-written token scanner.

Conductor scope correction (2026-10-15): the acceptance-named storage_evidence_test.go is admitted for end-to-end classification controls; existing decoder unit controls in volume_test.go are reused.


Conductor scope correction (2026-10-15): identity_test.go is admitted for decoder/image source-binding regression and its required digest pin update.


Conductor scope correction (2026-10-15): volume/crash.rs is admitted to enforce strict nested crash_limits decoding after the implementation review finding. Storage stream-bound regressions use the already admitted test files.
### Reopened completion-review correction (2026-10-15)

Completion finding `finding-b09c3c9a375db075908fa7df12d439b8` leaves R1 unproved: duplicate and unknown operation selections currently reach guest execution and finalize as infrastructure. Task .5 is reopened; prior source acceptance and its original evidence remain historical at `fba4865efd:.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.5.md`.

Preserve the pre-execution refusal requirement and valid selected-state/catalogue replay. Selections can name operations created during the guest run, so request-only ID validation cannot distinguish unknown IDs from valid future operations. Carry bounded prior-observation context for an explicit operation selection: each volume's actual pending operations and existing pending-state digest. Validate volume references, duplicate/zero/unknown IDs and dependency closure against this context in Go and Rust before guest startup. Bind it through existing request, boot and retained-input identities. At the crash boundary, a different actual pending state is a state/replay divergence, rather than an admitted malformed selection. Export context from enumeration observations and thread it through existing catalogue/selected-state consumers; do not infer an inventory from selected IDs alone or remove those consumers.

The conductor admits the required storage transport/evidence, strict validation, result/request, catalogue, WASM boot/retained projection and regression surfaces for this correction, including `storage_transport.rs`, `wasi/volume_test.go`, `qualification/native_crash.go` and associated selected-state tests. Actual native volume semantics remain unchanged. No new dependency or generic provenance framework is needed. Root owns integration, same-receipt completion review and task completion; the implementation worker owns isolated source checkpoints.

## Acceptance
- [x] Malformed operation selections are `invalid` before guest execution in Go and Rust, including duplicate/zero/unknown IDs and invalid dependency closure; an unreachable `_start` proves execution did not begin.
- [x] Valid prior-observation selections retain their catalogue/selected-state replay and recovery behavior; context/actual-state mismatches are explicitly refused without being misclassified as infrastructure.
- [x] Non-broker storage runs make zero storage callbacks to Go (the Go fs-served counter from fn-175.5 is 0), including fault and `crash-before` runs.
- [x] Go-owner vs helper-owner parity passes on the storage fixtures above, including reopen from the helper-returned image.
- [x] Invalid storage config, crash selection or recovery image → `invalid` before `_start`; truncated or corrupt storage evidence → `invalid` refusal (Go unit tests in `storage_evidence_test.go` and `wasmhost/tests/engine.rs`).
- [x] `go test -tags test_dep -count=1 -run 'TestStorage|TestSQLiteStorage|TestHelper' ./internal/backends/wasm/wasi ./internal/backends/wasm/cluster` (from `tools/gomad`) passes.


## Done summary
Reopened correction for completion finding finding-b09c3c9a375db075908fa7df12d439b8 is complete. Explicit crash selections now carry bounded prior-observation context (each volume's pending operations and pending digest). Go (`ValidateStorageCrashSelection`, in `Run` and `BootConfiguration`) and Rust (`validate`/`validate_context`) refuse volume, duplicate, zero, unknown-ID, missing-observation and non-dependency-closed selections as `invalid` before guest start. An unreachable `_start` with zero execute nanos/peak memory proves this. At the crash boundary, an actual pending state that differs from the admitted context is refused as `invalid` with the exact message "storage crash context diverged from prior observation", never as infrastructure. Complete evidence carries the v3 context section exactly when crash limits are set. Selected-state and catalogue consumers thread the context through.

Commits: e4fb5adca6 (RED), 01c419c40a (fix), e41f6841da (review P2: context presence tied to limits), fa92d0fb7c and 9c854e5ee2 (review P3s), 75c13cf844 (frozen-run finding: removed `reflect` from WASM pure sources; exact equality). RED was observed for the Rust and Go selection admission at e4fb5adca6, for context presence, and for TestWASMArchitecture; all are GREEN at 75c13cf844.

The full frozen 84-stage requalification of source 75c13cf844 passed (84/84 exit 0, PASS 2026-10-15T19:07:17Z; source sha256:a8a9bdb0...b0a6, helper 70e8dbe6...ad48). It includes the Temporal committed-workflow crash/reopen gate, the qualify-workloads success/canary cells, all four WASM SQLite selected states (whose context came from the primary catalogue's 4 pending operations) and the authenticated catalogue. Native execution remains deferred (fn-128/fn-149).

Independent impl review (flowctl claude, claude-opus-5-5:high, base 7e2a9aa1c3, same receipt): NEEDS_WORK → SHIP → SHIP.

Full evidence, commands and exit codes: .flow/artifacts/fn176-5-selection-admission-close-2026-10-15.md.

The prior source acceptance and its original evidence remain historical at fba4865efd:.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.5.md (implementation provenance cd2cf0c9f6/0127791360/49021ebfed/629e735ca6; checkpoints bd539d703e, 695bacc98a, f7217179d0, 55c6505464).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e4fb5adca6, 01c419c40a, e41f6841da, fa92d0fb7c, 9c854e5ee2, 75c13cf844
- Tests: cargo +1.94.1 test --manifest-path wasmhost/Cargo.toml --release --locked (exit 0), cargo +1.94.1 clippy --release --locked --all-targets -- -D warnings (exit 0), make -C tools/gomad/internal/backends/wasm test-conformance (exit 0), make -C tools/gomad validate (exit 0), go test -tags test_dep -count=1 -run 'TestStorage|TestSQLiteStorage|TestHelper|TestImplementationIdentity|TestVolume|TestNamespace' ./internal/backends/wasm/wasi (exit 0), go test -tags test_dep -count=1 ./internal/backends/wasm/cluster ./internal/backends/wasm/qualification ./internal/backends/wasm/backend (exit 0), go test -tags test_dep -count=1 -run '^TestWASMArchitecture' ./internal/backends/wasm (exit 0 at 75c13cf844), make lint-code-gomad-wasm GOLANGCI_LINT_BASE_REV=7e2a9aa1c3 (0 issues), frozen driver /home/agent/wt/fn176-qual-20261015T1753/run-qualification.sh 75c13cf844e4fb348978f9ac22dc3f1820e1d940 a1f0be7e96e2746cb9c51598766a5ae96d8552c2 (84/84 stages exit 0, PASS)
- PRs:
### Evidence detail (fn176-close, 2026-10-15)
- Full commands, exit codes, RED/GREEN logs and frozen-run bindings: `.flow/artifacts/fn176-5-selection-admission-close-2026-10-15.md`.
- Frozen run root `/home/agent/wt/fn176-qual-20261015T1753` (84/84 exit 0); superseded attempt `/home/agent/wt/fn176-qual-20261015T1716` (ordinary exit 2, TestWASMArchitecture `reflect`, fixed in 75c13cf844).
- Historical pre-reopen evidence: `fba4865efd:.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.5.md`.
