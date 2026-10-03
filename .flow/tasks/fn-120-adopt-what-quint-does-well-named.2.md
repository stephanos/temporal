---
satisfies: [R2, R3]
---
# fn-120-adopt-what-quint-does-well-named.2 Finish named-choice rollout and refuse unnamed branches

Touches: [model/temporal/**, model/lifter/**, model/umpire/**, model/ir/**]

## Description
Complete Part A only after fn-114 has migrated all Models and fixtures to the settled syntax. Fn-112 and fn-114 own their conversions; this task audits every remaining consumer and turns on the final refusal. Before work, verify fn-114 is closed. Flowctl permits only same-spec task dependencies, so this is an explicit cross-spec entry gate.

**Size:** M
**Files:** remaining model/temporal and fixture branch declarations, model/lifter refusal fixtures, model/umpire author surface.

### Approach
- Inventory branches after fn-114's rollout; identify any alternative nobody can name for owner disposition without changing its behavior.
- Refuse a multi-result unnamed list at its Scala line only after all current consumers use named choices.
- Reuse task 1's original-baseline harness and its single choice-name-only allowance; do not define another projection or enlarge the accepted delta during retirement.

## Acceptance
- [ ] fn-114 is closed before this task starts; every current Model and lifter fixture with multiple results names each branch.
- [ ] The lifter refuses an unnamed multi-result list at the source; one-result steps remain valid without choose.
- [ ] Task 1's unchanged harness proves all behavioral, identity and Case outputs remain at the original baseline under its precise choice-name metadata allowance.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
