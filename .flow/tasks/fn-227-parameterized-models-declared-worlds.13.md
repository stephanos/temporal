---
satisfies: [R1]
---
# fn-227-parameterized-models-declared-worlds.13 Simplify: one #[machine] generator, no module generics, method-level #[starts]

## Description
Owner decision 2026-10-21 after the simplification scout (`.flow/artifacts/fn-227-parameterized-models-declared-worlds/simplify-report.md`, Task A; audit context in `audit-report.md`). Runs before fn-227.8's review resumes; .8's commits (3fd1d2f551, 0469334f91) are the baseline.

**Why:** generic `#[machine]` modules are used by no Model and refused by the lifter (`lift/src/shape.rs:171`), yet account for ~560 of `dsl/macros/src/machine/declared.rs`'s 892 lines, all of `readonly.rs` and ~600 test lines, and they block merging the two machine generators. Computed starts use a separate `#[starts] impl Model { fn starts(..) }` block with a forced name, unlike `#[on]`, `#[assumption]` and `#[scope]`.

**Size:** M. Deletion-dominated.

### Approach

1. Refuse generic `Model`/`World`/actor declarations inside `#[machine]` with one located compile error (standalone derives keep their generics). Delete the generic plumbing listed in the report (`declared.rs` generic sections, `readonly.rs`, the parameterized/state-generics branches in `machine.rs`/`registration.rs`) and the generic-only tests; add the refusal trybuild case.
2. Register `#[starts]` as a method attribute inside the main `impl Model { }`: any method name, associated fn with optional trailing `&World`, returning `impl IntoIterator<Item = Self>`. Field-level `#[starts(range)]` stays; declaring both remains the compile error "declare starts once". Refuse the old impl-block spelling with a located error pointing to the method form.
3. Fold `declared::expand` into `machine::expand` behind an optional declared World, keeping unit-World token output identical.
4. Lifter: locate the `#[starts]` method in the method loop (`shape.rs`), dropping the separate impl handling; source semantics otherwise unchanged.
5. Migrate every `#[starts] impl` site (fixtures, tests, trybuild), README (computed starts section), DESIGN.md readiness rows, and the fn-227 spec wording is already amended by the conductor.

### Verification

Quick from `model/`: `mise exec -- cargo test -p umpire-dsl-macros -p umpire-dsl -p umpire-lift -p umpire-tests -p umpire-suite -p umpire-checker`, default and `UMPIRE_BACKENDS=require`; scoped strict clippy and fmt. Record a baseline manifest of every `expected/`, snapshot and `.stderr` first; after the change, only the declared-World expansion snapshots (`dsl/macros/tests/fixtures/{declared_world,world}`) and trybuild stderr for removed/renamed cases may differ, each listed with its reason. Checker/report goldens, TLA+/cfg goldens and TLC agreement stay byte-identical. Full (after review): `UMPIRE_BACKENDS=require make model-check`, `make model-time`; guest slice/gaps if guest bytes change. Report source/test line deltas.

## Acceptance
- [ ] `#[machine]` refuses module generics with one located error; generic-only plumbing and tests are deleted; standalone derive generics unaffected.
- [ ] `#[starts]` is a method attribute in the main impl under any name with optional `&World`; field `#[starts(range)]` stays; both forms and the old impl-block form refuse with located errors; trybuild pins all three.
- [ ] One machine generator; unit-World expansions byte-identical; only declared-World expansion snapshots change, each justified.
- [ ] Lifter finds starts by attribute; all checker/report/TLA+/cfg goldens and TLC agreement byte-identical.
- [ ] README/DESIGN.md updated; Quick gates pass, Full gates after review; line deltas reported (target about -900 source, -600 test).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
