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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
