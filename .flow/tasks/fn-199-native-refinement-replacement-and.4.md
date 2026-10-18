---
satisfies: [R4, R5]
---
# fn-199-native-refinement-replacement-and.4 Composed fairness, composition ceilings and end

## Description
Accept sync vals and member-qualified actions in `fair` on compositions and resolve them to composed classes; run progress questions over the product graph under the caller's budget with resource-limit on cut; `end` is the override or `true`.

**Files:** model/framework/dsl/Compose.scala, model/framework/checker/CompositionRuntime.scala, model/framework/checker/NativeCheck.scala (progress), tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/checker model/framework/dsl --require-tests`

## Acceptance
- Deadlock, fair cycle and missed deadline verdicts with lasso witnesses on a composed sample.
- Unbound or ambiguous fairness references refuse at declaration.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
