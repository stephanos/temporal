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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
