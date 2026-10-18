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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
