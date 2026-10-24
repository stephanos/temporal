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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: