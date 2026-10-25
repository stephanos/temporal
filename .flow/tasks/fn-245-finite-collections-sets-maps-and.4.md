---
satisfies: [R3, R4, R5, R6]
---
# fn-245-finite-collections-sets-maps-and.4 Shared fixtures finite_sets, finite_maps and bounded_queue with goldens, differential and TLC agreement, the fn-238.6 sets measurement, and the DESIGN/README write-up

## Description

After .2 and .3. Add exactly the three shared fixture directories `model/umpire/dev/fixtures/fixtures/{finite_sets,finite_maps,bounded_queue}` with model.rs, tests.rs and expected/**. Select all three for lift, native/interpreter differential and exhaustive TLC agreement. Each includes a false claim whose typed counterexample is replayed through compiled checked_step, plus the spec's negative control. Keep other edge-case controls in existing unit/macro/backend test suites; do not create additional shared fixture families.

Extend the actual agreement/replay consumer `model/umpire/dev/testing/src/backend.rs` and related shared adapters only as needed: its state_text/debug_value must render typed set/map values exactly as the native collection state Display does, and it must pass shared Schema-decoded values to the fn-250 replay core. Cover empty/nonempty collections and malformed/forged values. Do not build a parallel replay or admission path.

Measure sets using the fn-238.6 recipe (`.flow/tmp/fn-238.6/bench.sh` if present; otherwise reconstruct from that task's committed evidence and model/DESIGN.md, with the executable recipe preserved in this task's evidence). On a quiet machine record source/toolchain/World/bounds, equal complete work, setup including Interpreted::new, five paired samples, medians and interpreted/compiled ratio. A ratio >1.25 fails the sets part; <=1.25 discharges only that part. Compiled remains default unless the complete fn-238.6 gate (differential plus Kettle/wide <=2x plus sets <=1.25) is established; do not silently restore the preserved switch branch. Report the deferred sets obligation as measured either way, retaining any unmet full-gate decision.

Write model/DESIGN.md and model/README.md's finite collection subset: exact methods, canonical values, shared Literal/Schema seams, typed replay, sentinel and ordered short-circuit encoding, guarded queue semantics, fn-240 symbolic word encoding and the named later unbounded checkpoint. Record make model-time output and reviewed golden differences. MILESTONES remains conductor-owned.

Verification at implementation time: focused task controls first; one final `UMPIRE_BACKENDS=require make model-check` covers the full required gate (do not repeat a separate default full run without a distinct reason), followed by `make model-time`. Verify unchanged Temporal Model checking meaning. No full gates run for this plan-only review.

Touches: model/umpire/dev/fixtures/**, model/umpire/dev/testing/**, model/DESIGN.md, model/README.md.

## Acceptance

- [ ] Exactly finite_sets, finite_maps and bounded_queue are registered shared fixtures with reviewed native/lift/TLC goldens, complete interpreter/native/TLC agreement, one negative control each and replayed false claims.
- [ ] Current typed formatting and replay adapters accept canonical collection values; malformed values, forged successors/outcomes/facts and incomplete exploration cannot establish agreement.
- [ ] The reproducible five-pair sets measurement includes setup and equal work, records the 1.25 ratio decision, and leaves compiled default unless every fn-238.6 gate condition is proved.
- [ ] DESIGN/README document the exact finite subset and future handoffs; required model-check and model-time evidence is recorded with no old-byte/guest compatibility obligation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
