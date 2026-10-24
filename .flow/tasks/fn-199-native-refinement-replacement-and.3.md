---
satisfies: [R3]
---
# fn-199-native-refinement-replacement-and.3 Replacement composition proof

## Description
Admit `withMember` only when the replacing member refines the replaced member under its declared or derived map and covers its starts. Construction refuses naming the member and the first failure; a member with no map refuses with "no map".

**Files:** model/framework/dsl/Compose.scala, model/framework/checker/CompositionRuntime.scala, tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/checker model/framework/dsl --require-tests`

## Acceptance
- Refining replacement admitted; non-refining and non-covering replacements refuse with the contracted messages.
- Existing composition tests unchanged.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: