---
satisfies: [R1, R2, R3, R5]
---
# fn-218-test-space-patterns-cover-and-the-full.2 cover and sample in umpire-suite with expansion and coverage goldens

## Description
Implement full-grammar find/Run::along plus cover(&space, Cases(n), path) and seeded finite random-walk sample in umpire-suite. Cases mismatch must fail before Model stepping and list alternatives; count over budget and unwitnessed alternatives fail explicitly. Reuse one explored Model graph, retaining exact edge outcomes/facts and per-alternative shortestness. Cover fails on a cut; along rejects alternatives before search. Covered exposes runs/len/all/coverage/expansion. Expansion prints count, choices, every action with outcome and final state; coverage shows touched/untouched declared actions, outcomes, faults and claims by typed/registration identity, honestly labeling undeclared metadata. Sample must implement actual bounded seeded random walks in the admitted pattern, not just pick cover shortest witnesses, and report seed/experiment size. Register expansion/coverage golden kinds in testing and pin mismatch, impossible alternative, count/permutation limits, cut, predicate failures and reproducible seed controls. **Touches:** `model/umpire/suite/**`, `model/umpire/testing/**`. **Quick:** `cd model && mise exec -- cargo test -p umpire-suite -p umpire-testing`; full gates `make model-check` and `make model-time`.

Leading structural action patterns must preserve the existing action-first actor-to-state inference before later untyped state predicates. Repair the confirmed integration E0282 without requiring an explicit state prefix, duplicating/moving expressions or weakening zero-repeat behavior. The additional implementation surface is narrowly `model/umpire/dsl/src/path_build.rs`, `model/umpire/dsl/macros/src/path.rs` and DSL regression fixtures/tests; do not otherwise reopen the grammar task. Include focused DSL/macros tests for this repair in addition to the suite/testing Quick commands.

**Additional Touches:** `model/umpire/dsl/src/path_build.rs`, `model/umpire/dsl/macros/src/path.rs`, `model/umpire/dsl/tests/**`.

## Acceptance
- [ ] Action-first structural alternatives followed by bare state closures compile through actual find/cover consumers; expression order/single evaluation, zero repeats and legacy/manual-world routes stay intact.
- [ ] Model-state, alternative-admission and finite random-walk budgets are independently controlled; a one-state graph covers two alternatives with Model-state budget one. Sampling reports the walk bound and explicitly refuses a selected alternative that cannot finish within it.
- [ ] Cases mismatch is first, with count and alternatives; impossible alternatives and over-budget/permutation counts refuse without vacuity.
- [ ] One shortest witness per alternative uses one retained Model graph, with explicit cut handling; along rejects multiple alternatives.
- [ ] Typed coverage and expansion goldens account for actual outcomes/actions/faults/claims and honestly report undeclared catalogs.
- [ ] Sample implements bounded pattern-admitted random walks reproducibly by seed and reports its finite experiment.
- [ ] Focused suites, make model-check and make model-time pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
