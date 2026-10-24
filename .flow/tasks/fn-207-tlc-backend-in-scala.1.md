---
satisfies: [R1]
---
# fn-207-tlc-backend-in-scala.1 TLA+ emitter: fairness, liveness and deadlock sections

## Description
Extend the shared TLA+ emitter (fn-206.1; create it here if fn-207 lands first) with weak fairness on the classes each progress claim names, one liveness property per progress claim, and the deadlock predicate aligned with `end`. A progress claim over an unbound class refuses naming it.

**Files:** model/framework/export/tla/*.scala, tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/export/tla --require-tests`

## Acceptance
- Samples with progress claims emit fairness and liveness sections.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: