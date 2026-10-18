---
satisfies: [R3]
---
# fn-207-tlc-backend-in-scala.3 TLC runner: pinned tla2tools, deadlock on, state dump and JSON trace

## Description
Implement `Tlc.check(module, config): Verdict` and `Tlc.dump(module, config): Iterator[State]` with the jar pinned on the framework classpath, `-deadlock` kept on, `-dump` streamed, `-dumpTrace json` on violation, a reachable-set ceiling yielding resource-limit; missing or mis-versioned jar fails before any test; TLC failure is not-run with the error text.

**Files:** new model/framework/backends/tlc/*.scala, model/framework/check/Tools.scala, toolchain pins.

**Quick commands:** `UMPIRE_BACKENDS=require mise exec -- scala-cli test --server=false model/framework/backends/tlc --require-tests`

## Acceptance
- Runs on the six samples; ceiling and not-run reproduced on controls.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
