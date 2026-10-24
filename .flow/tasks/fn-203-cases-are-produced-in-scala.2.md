---
satisfies: [R2]
---
# fn-203-cases-are-produced-in-scala.2 Generic producer: Program placement, correlated Contract, localized IDs, fingerprints

## Description
Build `Cases.lower(suite, query): Either[Standing, Case]` in framework/export/testpilot: place path actions and entrypoints into a Program, build correlated clauses and projection plans, localize Definition IDs, fingerprint over `Lifted.canonical`. Standings: realizable, no realization, unsupported, resource limit.

**Files:** new model/framework/export/testpilot/*.scala, model/framework/realize/*.scala, tests against the samples' realizations.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/export/testpilot --require-tests`

## Acceptance
- Each standing produced by a control.
- Program and Contract shapes match the schema.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: