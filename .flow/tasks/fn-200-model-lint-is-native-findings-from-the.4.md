---
satisfies: [R4]
---
# fn-200-model-lint-is-native-findings-from-the.4 API-value kind over ScalaPB descriptors

## Description
Walk poll and Run Event guard operands against ScalaPB descriptors and report unmodeled enum values and oneof members; `unmapped(...)` is the acceptance subject. An operand naming an absent field is a finding.

**Files:** model/framework/lint/Native.scala, model/framework/realize (operand access), tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/lint --require-tests`

## Acceptance
- Unmodeled value and absent-field cases reproduced on a synthetic realization.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
