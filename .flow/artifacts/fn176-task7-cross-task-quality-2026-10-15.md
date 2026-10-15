# fn176 periodic cross-task quality audit

Pinned source: `cbd7ae3b285976ee242efc16a28bb09b9ff1f50c`.
Review base: `2237e0276f50f3b4b4b244f32a564ec17caca110`.
Reports below are reproduced verbatim and kept separate by axis.

### Correctness axis

## Quality Audit — Correctness axis: fn176 at cbd7ae3b285976ee242efc16a28bb09b9ff1f50c

### Summary

- Files changed: 99 under `tools/gomad` · Critical 0 · Should Fix 0 · Consider 1 · Ship: ⚠️ Fix first — known task6 blockers remain actively owned; no additional shipping blocker found.

### Consider

- **tools/gomad/internal/backends/wasm/wasmhost/src/storage_transport.rs:209** (Conf 75): Post-run enumeration/image failures discard collected storage log, snapshots and operations, whereas Go retains them — preserve partial evidence with the classified failure and add terminal-path parity coverage.

### Evidence

- **walked:** `evidence()` builds the prefix, then returns `Err`; `finish_storage()` retains only classification/message. Go `process.go:470–493` populates results before failure. Seventeen 64-KiB overwrites with a 64-KiB filesystem limit provide an image-overflow scenario. Classification remains correct; impact is missing diagnostic evidence. Actual-helper reproduction remains pending.

### Test Gaps

- Compare partial storage evidence from both owners after final image-capacity failure.

### Test Budget

- Added test/implementation lines ≤0.95:1 including inline Rust tests, before excluding setup; corpus JSON excluded.
- Modified existing assertions preserve behavior checks; output assertions implement the accepted offered-byte contract.

### What's Good

- **walked:** Recovery admission, model identity binding, strict envelope checks and storage routing preserve the documented ownership boundaries.
- Known task6 revocation/context blockers are excluded as duplicate findings. No tests, builds, cache mutations or repository writes performed.

### Standards axis

## Quality Audit — Standards axis: fn176 at `cbd7ae3b285976ee242efc16a28bb09b9ff1f50c`

### Summary

- Files changed: 98 scoped · Should Fix 5 · Consider 0 · Blocking: none possible (standards axis)

No documented repository-rule violations found. Findings below are mandatory charter §9 file-size notices, **executed** through pinned Git line counts. They do not establish behavioral defects.

### Should Fix

- **tools/gomad/internal/backends/wasm/wasmhost/src/volume/state.rs:1001** (Conf 100): Charter size threshold crossed, 0→2,574 lines — separate filesystem operations from persistence-effect machinery.
- **tools/gomad/internal/backends/wasm/wasmhost/src/volume/image.rs:1001** (Conf 100): Charter size threshold crossed, 0→1,148 lines — separate image codec, validation and tests.
- **tools/gomad/internal/backends/wasm/wasi/process.go:1001** (Conf 100): Charter size threshold crossed, 872→1,091 lines — extract helper protocol handling from process supervision.
- **tools/gomad/internal/backends/wasm/wasi/volume_test.go:1001** (Conf 100): Charter size threshold crossed, 280→1,530 lines — separate conformance harness, core vectors and storage evidence cases.
- **tools/gomad/internal/backends/wasm/wasi/testdata/volume-conformance/vectors/core.json:1001** (Conf 100): Charter size threshold crossed, 0→28,381 lines — partition generated vectors into bounded fixtures while preserving comparison coverage.

Suppressed findings: +31 over cap, all additional JSON fixture threshold crossings.

### What's Good

- **Cited:** storage codec and recovery-state sources participate in implementation identities; recorded SQLite evidence documents authenticated provenance and its limited replay scope.
- Read-only inspection only. Final frozen qualification remains task7’s pending requirement.

## Axis counts

Correctness: 1 finding; worst tier Consider. Known task6 blockers are separately owned.
Standards: 5 reported findings; worst tier Should Fix, plus 31 suppressed fixture-size notices.

## Conductor disposition

The task6 review owns the reproducible current-revocation and cancellation defects.
Those fixes and their tests must finish before frozen qualification.

The partial-evidence advisory will be reproduced and assessed under task7 before
freezing. Protocol text promises collection at every attached terminal; the
absence of a shipping-level auditor finding is not proof that this parity
requirement is met. Actual-helper error-path evidence is still needed.

The five standards notices are not applied in this delivery. They identify size
thresholds, not repository-rule violations or behavioral defects. Broad file
splits and repartitioning generated conformance vectors are not required to
establish the fn176 contracts; existing semantic coverage and source identities
remain intact. This disposition does not waive any correctness or execution gate.

stage: quality-audit - ran (two axis-scoped fresh reviewer agents; separate verbatim reports retained)


## Task 7 disposition

The partial-evidence advisory reproduced against the actual helper for image-capacity
and post-run enumeration failure. The corrected reproduction is committed at
`e21d275a9a`; `.flow/tmp/fn176.7-parity-red-corrected.log` preserves both owners'
matching terminal classifications before the collected-prefix mismatch. The first
image fixture hit the unrelated transcript budget, so that initial observation is
inconclusive for image capacity. Its transcript budget was raised before the
corrected red run. Complete v1 evidence stays unchanged; explicit partial v2
envelopes retain the collected prefix only with a matching failure terminal.

The conductor's five size-notice dispositions above remain unchanged. No broad
source or generated-vector split is part of this task.
