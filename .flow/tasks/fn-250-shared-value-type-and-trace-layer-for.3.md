---
satisfies: [R3, R6]
---
# fn-250-shared-value-type-and-trace-layer-for.3 Observer names in one module and one trace-replay core for TLC and Apalache

## Description
Implement **Observers and replay (A5)**: `umpire_tla::observers` owns the `__umpire_*` names and `EdgeObservers::decode(&TraceState)`; one step-replay core (input ordinal → catalog entry with `Concrete` identity, choice, `checked_step`, state/outcome/facts comparison) serves `dev/testing/src/backend.rs` (TLC) and `apalache/src/replay.rs`; each keeps only its stutter/tick handling. The dsl macro that emits the names uses the same constants (or a shared constant crate if the macro crate cannot depend on umpire-tla).

## Acceptance
- [ ] No `__umpire_*` literal outside the observers module (and its generated uses).
- [ ] TLC and Apalache replays share one core; divergences still reported per step.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
