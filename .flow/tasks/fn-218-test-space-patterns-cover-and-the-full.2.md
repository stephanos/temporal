---
satisfies: [R2, R3, R5]
---
# fn-218-test-space-patterns-cover-and-the-full.2 cover and sample in umpire-suite with expansion and coverage goldens

## Description
Implement full-grammar find/Run::along plus cover(&space, Cases(n), path) and seeded finite random-walk sample in umpire-suite. Cases mismatch must fail before Model stepping and list alternatives; count over budget and unwitnessed alternatives fail explicitly. Reuse one explored Model graph, retaining exact edge outcomes/facts and per-alternative shortestness. Cover fails on a cut; along rejects alternatives before search. Covered exposes runs/len/all/coverage/expansion. Expansion prints count, choices, every action with outcome and final state; coverage shows touched/untouched declared actions, outcomes, faults and claims by typed/registration identity, honestly labeling undeclared metadata. Sample must implement actual bounded seeded random walks in the admitted pattern, not just pick cover shortest witnesses, and report seed/experiment size. Register expansion/coverage golden kinds in testing and pin mismatch, impossible alternative, count/permutation limits, cut, predicate failures and reproducible seed controls. **Touches:** `model/umpire/suite/**`, `model/umpire/testing/**`. **Quick:** `cd model && mise exec -- cargo test -p umpire-suite -p umpire-testing`; full gates `make model-check` and `make model-time`.

## Acceptance
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
