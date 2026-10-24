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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: