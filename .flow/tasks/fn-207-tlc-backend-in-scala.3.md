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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: