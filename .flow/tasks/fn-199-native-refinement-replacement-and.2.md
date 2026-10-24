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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: