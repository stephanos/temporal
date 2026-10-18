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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
