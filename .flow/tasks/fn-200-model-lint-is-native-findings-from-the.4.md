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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: