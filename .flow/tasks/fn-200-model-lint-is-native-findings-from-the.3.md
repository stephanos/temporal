---
satisfies: [R3, R5]
---
# fn-200-model-lint-is-native-findings-from-the.3 @gap acceptance judge and coverage tally

## Description
Resolve `@gap` annotations to `(kind, owner, subject)`, judge findings against them (unaccepted finding and stale gap both fail, each named), accept capability waivers without judging, and print population-minus-findings coverage per kind per Model.

**Files:** model/framework/dsl/Gaps.scala, model/framework/export/Gaps.scala (resolution moves to lint), model/framework/lint/Native.scala, tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/lint model/framework/dsl --require-tests`

## Acceptance
- Unknown kind or foreign owner in a gap refuses at compile time.
- Coverage never fails a run.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
