---
satisfies: [R22]
---
# fn-105-gomad-follow-ups-deferred-scope.22 D22: fix parallel Nexus outcome endpoint collisions

## Description
Required fix approved on 2026-09-30. The start/cancel outcome tests in TestNexusApiTestSuiteWithLegacyErrorPaths and TestNexusApiTestSuiteWithTemporalFailures reuse testcase endpoint names between parallel ByNamespaceAndTaskQueue and ByEndpoint subtests on a shared cluster. Assign independent endpoint identities per subtest while preserving both dispatch paths, outcome assertions, and parallel execution. The fix applies to ordinary tests and does not change production endpoint uniqueness behavior.

## Acceptance
- Independent outcome subtests register distinct endpoint identities, and handler/request endpoint assertions remain consistent with each subtest's registered identity.
- Preserve start/cancel operations, both dispatch paths, both error-handling variants, existing outcome assertions, and parallel execution; no Gomad-only source rewrite or serialization workaround.
- Demonstrate the collision with a retained regression reproducer or focused test execution, then verify the corrected outcome tests under native Go and Gomad on seeds 11 and 17 with recorded commands, platform identity, and outcomes.
- Remove the four corresponding start/cancel outcome skips from the source qualification generator and regenerate its manifest only after verification passes.
- Production endpoint uniqueness behavior remains unchanged; diagnosis or a still-skipped test cannot close this required fix.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
