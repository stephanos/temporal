# Periodic task3 audit reports

Pinned candidate: 8f1b4e7201b6c840d8062063634275d94c9aa359. Reports below retained verbatim; correction decisions belong to the conductor.

### Correctness axis

## Quality Audit — Correctness axis: fn-211.3

### Summary

- Files changed: 31 model files · Critical 0 · Should Fix 1 · Consider 0 · Ship: ⚠️ Fix first

### Should Fix

- **model/umpire/backends/apalache/src/replay.rs:178** (Conf 75): **walked**: `ArithmeticSafe` without a native refusal falls through to authored-claim lookup by name. A failing invariant named `arithmetic` therefore accepts an `ERROR=false` trace as `Confirmation::Violation`; `ValidStep`/`valid_step` collide likewise, and equal invariant/negative names select the invariant first. Dispatch by the selected property's metadata category, require arithmetic confirmation for `ArithmeticSafe`, and isolate invariant/negative lookup.

### Test Gaps

- Add collision controls for generated properties and equal invariant/negative names, asserting rejection when the specifically selected property holds.

### Test Budget

- Added test/implementation lines: 848:1,301, excluding fixtures.
- Existing tests inspected retain their assertions; additions expand replay controls and metadata checks.

### What's Good

- **walked**: strict declarations, typed structural identity, canonical ordinals, stutter history and arithmetic prefix replay have discriminating controls.
- Reviewed task-base→`8f1b4e7` with cumulative spec context. No tests executed during this audit; fresh guest/live proofs and final required gate remain pending.

### Standards axis

## Quality Audit — Standards axis: fn-211.3

### Summary

- Files changed: 77 · Should Fix 1 · Consider 1 · Blocking: none possible (standards axis).
- Scope: `218901134e4f..8f1b4e7201b6`; 31 source/config/fixture files plus retained evidence metadata reviewed read-only.

### Should Fix

- **model/umpire/backends/apalache/tests/fixtures/kettle-0.62.4.itf.json:1001** (Conf 100): Executed line-count comparison confirms this new fixture crosses the charter’s mechanical 1,000-line threshold (0 → 1,807). Preserve byte-exact raw evidence under `.flow/artifacts/` and compact the test copy.

### Consider

- **model/README.md:297** (Conf 100): Cited documentation still says `imbl::Vector` cannot export or bind, although `dsl/src/value.rs:95` now implements native projection. Update this statement and `model/DESIGN.md:578` to describe the supported native-only boundary.

### What's Good

- Walked changes preserve owning-module boundaries, reuse typed TLC assignments, and share one start-domain generator; macro mirrors contain projection additions.
- Cited evidence distinguishes historical live proof from the candidate’s focused checks. Fresh guest/live/full-gate checks remain pending; this audit executed no build or runtime tests.

Correctness: 1 finding, worst Should Fix; 0 Critical.
Standards: 2 findings, worst Should Fix (1 Should Fix, 1 Consider).

Conductor decision: reproduce and fix the property-category/name collisions; compact only the test copy after retaining and verifying byte-exact raw ITF evidence; also update the stale native Vector projection documentation. Preserve original normal review session before applying source corrections. Fresh source-scoped guest/live/full gates remain pending.

