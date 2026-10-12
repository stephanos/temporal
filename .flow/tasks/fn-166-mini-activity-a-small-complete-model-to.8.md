---
satisfies: [R3, R4]
---
# fn-166-mini-activity-a-small-complete-model-to.8 Scala mini realization and typed live witness authoring

## Description
Implements the remaining Scala source portion of R3 and prepares R4. Case lowering and live execution remain in held task 4.

**Size:** M
**Files:** mini realization `MiniActivityRun.scala`, mini-local `Kit.scala`, query declarations/header in `MiniActivity.scala` and the existing mini levels; focused mini source/lift fixtures.
**Touches:** [model/temporal/features/activity_mini/**, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/layout/mini/**]

### Approach
- Subtract the existing standalone realization from `model/temporal/features/activity/standalone/realization/StandaloneActivityRun.scala:47`. Bind the committed mini Protocol and its facts; do not import the sibling standalone kit or feature.
- Reuse `Realizes(machine, runs = ...)` and ordered sections from `model/temporal/realize/Objects.scala:27`, shared typed evidence modules and `deadlines` from `model/temporal/realize/Modules.scala:124`. Concrete start presets retain the non-timeout maxAttempts input.
- Declare evidence for every mini fact, a deadlines use, one satisfied live witness and one reasoned inconclusive live witness. Keep the feature header's hard-instance inventory current.
- Prove discovery, typed bindings, evidence coverage, selected Queries and expectation metadata through isolated Scala compile/lift fixtures. Reuse `model/irgen/Claims.scala:651` and `model/irgen/Lifting.scala:201`; no new consumer or schema is required.
- Keep scratch outputs ignored. No checked-in artifact publication, Case lowering, Go changes, pinning or live run is authorized. Source evidence cannot establish executable evidence coverage or a live verdict.

### Verification
Run mini-focused compile/lift and focused tests for changed source/lifter fixtures with --server=false. Assert the inventory independently from declarations and include wrong-owner/missing-evidence/expectation controls. Broad validation and review remain at the owner-approved batch-end gate.

## Acceptance
- [ ] The mini realization compiles and lifts with the expected machine, bindings, Query membership and evidence for every fact.
- [ ] Typed deadlines and satisfied/reasoned-inconclusive live expectations execute in focused positive/refusal fixtures.
- [ ] The header lists the source hard instances; no sibling feature or protected artifact changes. R3 transport and R4 live proof remain explicitly open in held tasks.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
