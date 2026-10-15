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
- [ ] Malformed operation selections are `invalid` before guest execution in Go and Rust, including duplicate/zero/unknown IDs and invalid dependency closure; an unreachable `_start` proves execution did not begin.
- [ ] Valid prior-observation selections retain their catalogue/selected-state replay and recovery behavior; context/actual-state mismatches are explicitly refused without being misclassified as infrastructure.
- [ ] Non-broker storage runs make zero storage callbacks to Go (the Go fs-served counter from fn-175.5 is 0), including fault and `crash-before` runs.
- [ ] Go-owner vs helper-owner parity passes on the storage fixtures above, including reopen from the helper-returned image.
- [ ] Invalid storage config, crash selection or recovery image → `invalid` before `_start`; truncated or corrupt storage evidence → `invalid` refusal (Go unit tests in `storage_evidence_test.go` and `wasmhost/tests/engine.rs`).
- [ ] `go test -tags test_dep -count=1 -run 'TestStorage|TestSQLiteStorage|TestHelper' ./internal/backends/wasm/wasi ./internal/backends/wasm/cluster` (from `tools/gomad`) passes.


## Done summary
Verified helper-owned storage execution, fault and crash-before parity, recovery from returned images, and strict Go evidence refusal. Bound the production storage decoder and image encoder into the Go model identity, corrected storage-state admission for JSON expansion and per-volume enumerations, and made nested crash-limit decoding strict.

Implementation provenance is merge cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d, helper-storage implementation 0127791360, image-bound follow-up 49021ebfed, and inherited review corrections 629e735ca6. This task's checkpoints are bd539d703e, 695bacc98a, f7217179d0 and 55c6505464. Lifecycle baseline is 714b7de43b7de8f19a06876182cd213cf00d424f. Independent implementation review examined historical base 2237e0276f50f3b4b4b244f32a564ec17caca110 through 55c6505464, covering the inherited implementation rather than an empty receipt-only range.

baseline: green. The exact task Quick command passed pre-edit with explicit stock Go and native /tmp. WASI took 62.918 seconds; the cluster selection passed in 0.002 seconds. Similar-code investigation reused existing envelope corruption/version/trailing/JSON/log/image controls in volume_test.go, extended TestStorageOwnerParity for fault-image reopen, and added actual Run refusal controls in storage_evidence_test.go. The conductor admitted storage_evidence_test.go, identity_test.go and volume/crash.rs through task set-description before edits.

Acceptance evidence:
- Non-broker Go-owner/helper-owner comparisons require equal output, transcript digest, storage log, snapshots, operations, enumeration, image identity and interrupted choice prefixes. Fault cases cover io-error, short-write and sync-error, including equal recovered histories and zero Go filesystem callbacks after reopen. Existing SQLite recovery checks exercise file/directory sync, durable crash selection, all six fault actions and interrupted-prefix replay.
- TestStorageEvidenceRunRefusesDamagedHelperStreams captures a real SQLite helper envelope, verifies unchanged transport replay, and requires invalid refusal with no image/snapshots after truncation, corruption or trailing bytes. Existing TestStorageEvidence* controls in volume_test.go retain strict decoding coverage. The first new replay fixture attempt omitted ReturnStorageImage; correcting that test request reached the intended decoder path without a production workaround.
- TestStorageEvidenceLargeEnumerationOwnerParity produces 45,065,293 bytes of enumeration JSON, exceeding both the former 2,621,440-byte cap and the corrected base allowance without enumeration. Its image is 135,937 bytes; both owners reopen and read x. TestStorageEvidenceEscapedMetadataOwnerParity produces 3,634,997 bytes of snapshot/operation JSON with no enumeration, versus the former 2,129,920-byte cap and an 823,189-byte image. Fifty empty files use 4000-byte control-character paths.
- TestStorageEvidenceStateBudgetSaturatesAndCountsVolumes checks aggregation, saturation and real owner parity for maximal uint64 budgets with a tiny actual enumeration. TestStorageEvidenceStreamAdmissionKeepsIndependentBounds accepts each exact storage-state/image limit and refuses the next byte. Recovery-image bounds remain filesystem bytes plus 1 MiB.
- Rust invalid_storage_configuration_and_images_refuse_before_guest_start covers invalid storage models, unknown crash-selection volumes, unknown nested crash-limit fields, truncated images and digest mismatch. An unreachable _start, zero peak memory and absent storage-state evidence prove pre-execution refusal.
- TestImplementationIdentityBindsStorageEvidenceSources failed for both omitted files before the identity fix. The first focused identity run also exposed the inherited stale digest pin bf4d4e... against actual eda61f...; the pin was deliberately refreshed with each declared source identity change.

Final source gates passed with local release helper SHA256 7ccd46787a123e3e7d7b108d31d778a4223b258de4f606fe9c66fcdb935bdb64 and Go model identity sha256:36e0c79512ddcd3c284c1d2710b146b18fe8c4258f1b2597d06cdf7b0e1ee471. The initial identity-only fix preserved helper 896a1984...; the later Rust strict-decoding fix intentionally rebuilt it. Stock compiler is tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go, set through both GOMAD3_STOCK_GO and GOMAD_WASM_STOCK_GO; TMPDIR=/tmp.

The final expanded task Quick command plus identity controls passed in 97.387 seconds. Affected backend identity/evidence consumers passed in 6.887 seconds. Rust passed 196 unit and 150 integration tests with two existing ignored tests. Volume conformance, configured native lint, cargo fmt, conformance clippy and make -C tools/gomad validate passed. Final logs are .flow/tmp/fn176.5-review-final-go.log, fn176.5-review-consumers.log, fn176.5-review-rust-full.log, fn176.5-review-conformance.log, fn176.5-review-lint.log, fn176.5-review-clippy.log and fn176.5-review-validate.log.

GATE_SKIPPED:unittest:green-receipt 55c65054 - current expanded task and identity gate pass reused after review

Defect route:
- prior fixes: local history and bug-memory search found no earlier correction for missing source binding, evidence sizing or nested crash-limit admission; unchecked: GitHub PRs/issues (HTTP 401 Bad credentials); tracker inactive.
- diagnosis: source mutation left both storage codecs' model identity unchanged; the real helper exceeded Go's unrelated image-derived evidence cap; a nested unknown field reached the unreachable guest and returned trap before strict serde. Source-bound hashing, explicit encoded allowances and nested deny_unknown_fields close these mechanisms.
- introduced by: helper-storage path 0127791360 and inherited volume limits type; bisect skipped because no known-good version of these new paths exists.
- base: bd539d703e reproduces both unbound codec identities; f7217179d0 reproduces helper evidence capacity and guest execution on an unknown crash-limit member. The escaped-metadata test also failed under the restored old receiver line (.flow/tmp/fn176.5-escaped-metadata-red.log). Head 55c6505464 passes all controls; red logs are fn176.5-identity-red.log, fn176.5-enumeration-red.log and fn176.5-unknown-limits-red.log.
- live: no interactive application surface; real helper processes through Run and both execution owners supply execution proof.

All 18 external fn-180/fn-181/fn-182/fn-183 planning files retain their original SHA256 hashes and remain unstaged. Final frozen qualification, Temporal committed-workflow acceptance, workload success/canary cells, and all four WASM SQLite selected states/catalogue remain with tasks .6/.7 and the conductor. Native execution remains deferred under fn-128/fn-149.

Tier: session (jev-unavailable(no_key))

stage: impl-review - ran [2026-10-15T06:48:10Z..2026-10-15T07:08:01Z] (Codex gpt-6.1-sol at high; first three-axis fanout NEEDS_WORK, same-receipt re-review SHIP with both findings fixed; fresh reviewer context, writer/reviewer from the same GPT/Codex family; receipt /tmp/impl-review-receipt-657da2bc4466-fn-176-gomad-durable-volume-crash-model-in-the.5.json)

stage: memory-capture - ran (bug/integration/bound-encoded-storage-evidence-2026-10-15)

stage: plan-sync - skipped(config: planSync.enabled != true)

Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests:
- PRs:
