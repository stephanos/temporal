---
satisfies: [R2]
---
# fn-199-native-refinement-replacement-and.2 Queries and claims through a refinement, cross-owner claims

## Description
Replace the three native `unknown` sites that read through a refinement with calls into the refinement runtime: evaluate the refined Model's claim on the mapped state and step. A cross-owner claim over an unrelated owner refuses with both owner names.

**Files:** model/framework/checker/NativeCheck.scala, model/framework/checker/ClaimEvaluation.scala, tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/checker --require-tests`

## Acceptance
- No sample or control answers unknown with the "no native check reads" detail.
- Cross-owner refusal names both owners.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
