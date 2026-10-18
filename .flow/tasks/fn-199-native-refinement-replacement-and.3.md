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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
