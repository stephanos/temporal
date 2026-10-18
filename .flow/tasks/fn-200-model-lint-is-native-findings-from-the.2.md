---
satisfies: [R2]
---
# fn-200-model-lint-is-native-findings-from-the.2 Graph lint kinds from the native Search; choice token on Step

## Description
Carry the chosen choice token on `Step`. From the `Search` result compute unreachable value, never-enabled per class, stuck state, unproduced outcome and fact, unreachable binding, uncovered class, unfired verify and untaken choice. A cut exploration suppresses graph kinds with one warning. Compositions lint their own classes and syncs.

**Files:** model/framework/dsl/Machine.scala (Step), model/framework/checker/NativeCheck.scala (Search accessors), model/framework/lint/Native.scala, tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/lint model/framework/checker --require-tests`

## Acceptance
- Each graph kind reproduced on a synthetic Model; a hole-reaching state is never stuck.
- Cut exploration yields the warning and no graph findings.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
