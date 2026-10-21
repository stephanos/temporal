---
satisfies: [R1, R3]
---
# fn-227-parameterized-models-declared-worlds.14 Simplify: derived value handoff replaces the augmented-copy binding route

## Description
Owner decision 2026-10-21 (`.flow/artifacts/fn-227-parameterized-models-declared-worlds/simplify-report.md`, Task B). After the previous simplification task.

**Why:** fn-227.7 passes the selected World's values and start states to the lifter by generating an augmented copy of `model.rs` in OUT_DIR with a projection spliced into the machine module, plus discovery overrides, `tests.rs` rewriting, selection thread-locals and enumeration counters. The owner dropped the "no native reflection/serialization interface" rule for values only: derives expand in the type's own module and may export private field values. The lifter still admits and translates `model.rs` independently; only values cross.

**Size:** M.

### Approach

1. `umpire-dsl`: a small `Value` type mirroring `umpire_lift::Literal` (so Model crates never depend on the lifter), `Values`/`IntoValue` traits; hidden `values()` from `derive(World)` and `derive(State)` (private fields readable, visibility unchanged); hidden concrete-catalog/domain export from `derive(Action)`; fieldless-enum `IntoValue`.
2. `umpire-testing`: `bound(world, budget, negative) -> Bound` selects one World, enumerates `Machine::starts` once and the checked catalog once, converts `Value` to `Literal`; `fixture!` emits `bound()` for declared-World fixtures. Fold `explore_from`/`refuted_from` into this route where they become redundant.
3. `lift/src/build.rs`: delete `Bound::read`, the projection/identity/domain generators, prefix search and `generate_bound`; a declared-World fixture's lift runs `Lift::source(path)?.bind(&bound().binding())` and emission at test time. Unit fixtures keep eager OUT_DIR artifacts and `check_machine`.
4. `testing/src/discovery.rs`: delete per-input overrides and their tests; `tests/build.rs` returns to plain discovery.
5. Fixtures: replace thread-local selection with direct `bound(World::new(..), ..)` calls; drop enumeration counters.
6. `Lifted::bind` and every schema/range/catalog-order refusal stay exactly as they are; no lifter semantic file (`shape.rs`, `body/*`, `emit.rs`, `tree.rs`) changes beyond what A already did.
7. Record the decision in README/DESIGN.md (derived values, same exposure as derived `Debug`).

### Verification

As the previous task, plus `UMPIRE_BACKENDS=require` on `umpire-tests` for `parameterized_world` and `action_inputs` across all five CFG families; `tlc.txt`, `*.tla`, `*.cfg`, checker/report goldens byte-identical. Report line deltas (target about -250 source, -220 test).

## Acceptance
- [ ] Derives export typed values (private fields included, visibility unchanged); Model crates do not depend on `umpire-lift`.
- [ ] One `bound()` route selects one World, one start enumeration and one checked catalog; binding validation and all refusals in `Lifted::bind` are unchanged and still tested.
- [ ] Augmented-copy generation, discovery overrides, selection thread-locals and counters are deleted.
- [ ] All checker/report/TLA+/cfg goldens and TLC agreement byte-identical under `UMPIRE_BACKENDS=require`.
- [ ] Docs record the values-only exception; Quick gates pass, Full gates after review; line deltas reported.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
