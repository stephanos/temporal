---
satisfies: [R5, R6]
---
# fn-206-apalache-backend-in-scala.5 checkedBy(Checker.apalache) and confirm select backend runs; opt-in backend stage

## Description
Add the `UMPIRE_BACKENDS=require` backend stage to the framework check; route claims with `checkedBy(Checker.apalache)` to this backend and run their `confirm` list; `Checker.umpire` stays the default; an unrunnable backend is unsupported with the reason.

**Files:** model/framework/check/FrameworkLoop.scala, model/framework/dsl/Claims.scala (readers), tests.

**Quick commands:** `UMPIRE_BACKENDS=require make check-framework`

## Acceptance
- Stage runs only when opted in; default run looks for no tool.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: