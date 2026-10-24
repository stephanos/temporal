---
satisfies: [R3, R4, R5, R6]
---
# fn-245-finite-collections-sets-maps-and.4 Shared fixtures finite_sets, finite_maps and bounded_queue with goldens, differential and TLC agreement, the fn-238.6 sets measurement, and the DESIGN/README write-up

## Description
After .2 and .3. Add `model/umpire/dev/fixtures/fixtures/{finite_sets,finite_maps,bounded_queue}/{model.rs,tests.rs,expected/**}` as described in the spec, each with one negative control and one false claim replayed by TLC under `UMPIRE_BACKENDS=require`; select them for lift, interpreter differential and TLC. Measure interpreted versus compiled checking on `finite_sets` with the fn-238.6 bench recipe (`.flow/tmp/fn-238.6/bench.sh`, quiet machine, median of five) and record the ratio; if the interpreted checker is more than 25 percent slower the default stays compiled and the deferred obligation closes as measured, otherwise record that too. Write the DESIGN.md collection subset section (types, method table, canonical values, TLA+ encodings including `Absent`, the symbolic word encoding fn-240 will use, the unbounded checkpoint named as Apalache's or SMT arrays' later work) and the `model/README.md` authoring paragraph; paste `make model-time`. MILESTONES remains root-owned.

Full gates: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`; guest bytes should not change (no Temporal Model touched), verify and say so.

Touches: model/umpire/dev/fixtures/**, model/DESIGN.md, model/README.md.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
