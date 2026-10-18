---
satisfies: [R1]
---
# fn-203-cases-are-produced-in-scala.1 Public rows and Definition IDs from the checker

## Description
Expose `Checker.rows(instance, state)` (ordered results per class with choice names) and carry Definition IDs from the lift on `PathStep` values. Rows of an unexplored state refuse naming the state.

**Files:** model/framework/checker/Checker.scala, NativeCheck.scala, Result.scala, model/framework/lift (ID source), tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/checker --require-tests`

## Acceptance
- Rows equal the private step-function results on the samples.
- Path values carry IDs equal to the lift's.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
