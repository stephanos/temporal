---
satisfies: [R24]
---
# fn-105-gomad-follow-ups-deferred-scope.24 D24: synchronize the Nexus reset test before signalling

## Description
Required test fix approved on 2026-09-30 for TestNexusWorkflowTestSuiteHSM/TestNexusOperationSurvivesResetCrossTree. The expected history assumes the first post-reset workflow task completes before the done signal, but the test sends the signal immediately. Gomad can buffer the signal into that first task and produce a valid shorter history. Establish the intended post-reset state explicitly before signalling, preserving the pending-operation survival and HSM/CHASM creation-policy transition assertions. Production reset/signal semantics remain unchanged.

## Acceptance
- Wait for an explicit observable predicate that establishes completion of the required post-reset workflow task before sending the done signal, using existing test/history observation patterns and bounded waiting.
- Preserve the reset-run identity, pending Nexus operation survival assertions, HSM/CHASM unchanged-mode and upgrade/downgrade cases, and the intended final history ordering.
- Preserve ordinary production reset/signal behavior; no Gomad-only source rewrite, arbitrary sleep, relaxed history assertion, or extra workflow event solely to hide the ordering race.
- Verify all four creation-policy cases under native Go and Gomad on seeds 11 and 17 with retained commands, platform identity, and outcomes.
- Remove the reset-cross-tree skip from the source qualification generator and regenerate its manifest after verification passes. Diagnosis or a still-skipped test cannot close this required fix.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
