---
satisfies: [R2]
---
# fn-200-model-lint-is-native-findings-from-the.7 Unbounded-arithmetic lint kind

## Description
Add a lint kind that flags an arithmetic effect (add, subtract, multiply, negate) over a var or field with no declared bound, per the fn-201.5 numeric decision (host Int, wrapping). `@gap` subject `unboundedArithmetic`; accepted on the six samples where their worlds bound the values by guard.

**Files:** model/framework/lint/Native.scala, model/framework/dsl/Gaps.scala, sample sources for acceptances.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/lint --require-tests`

## Acceptance
- Synthetic Model with an unbounded counter is flagged; a guard-bounded one is not.
- Samples pass with gaps where needed.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: