---
satisfies: [R3]
---
# fn-203-cases-are-produced-in-scala.3 Property to Contract clauses by finite enumeration

## Description
Lower each Property the samples' realizations declare into trigger plus state, outcome and fact requirements by enumerating the claim over `Finite` domains; a Property outside the enumerable form is unsupported with the reason. Record the claim-seam fallback in the spec if enumeration proves insufficient.

**Files:** model/framework/export/testpilot/Clauses.scala, tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/export/testpilot --require-tests`

## Acceptance
- Every sample Property with a realization lowers or is unsupported with a reason.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: