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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: