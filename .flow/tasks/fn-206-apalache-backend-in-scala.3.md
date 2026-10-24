---
satisfies: [R3]
---
# fn-206-apalache-backend-in-scala.3 Apalache runner with pinned artifact and deterministic lifecycle

## Description
Implement `Apalache.check(module, invariant, steps): Verdict` by the chosen route: in-process, or a server on an ephemeral port started and stopped by the framework; artifact pinned in the framework dependency list or fetched to model/build; nothing in the user's home; missing artifact fails before any test; tool and inliner failures are not-run with the error text.

**Files:** new model/framework/backends/apalache/*.scala, model/framework/check/Tools.scala, mise/toolchain pins.

**Quick commands:** `UMPIRE_BACKENDS=require mise exec -- scala-cli test --server=false model/framework/backends/apalache --require-tests`

## Acceptance
- Verdicts for the monitored samples; not-run reproduced on a planted failure.
- No leftover process after the test.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: