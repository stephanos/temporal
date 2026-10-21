---
satisfies: [R3, R4]
---
# fn-227-parameterized-models-declared-worlds.3 After .2: lifter admits impl Phased and reject! arms; activity Model gains maximum_attempts and RespondFailed { retryable }

## Description
Implement R3/R4's Phased/reject source admission and retry checkpoint in `model/temporal/feature/activity/model.rs`, with native/TLC agreement. Add attempt, retryable failure and stale-Poll rejection while preserving terminal laws and existing deadline behavior at this checkpoint. Use the approved R4 lifecycle-refinement contract, not literal equality of the new rendered state/actions with old bytes. Retain the independent old graph/goldens, state the state/action/fact projection, and compare the projected one-attempt lifecycle exactly against that reference. Pin a deliberately changed old edge as a failing refinement control. The default three-attempt World gets new goldens and fn-218 coverage with its Cases(n). Keep existing realization/live controls compiling, with Fail mapped to retryable=false until .5.

The owner approved the compatibility amendment on 2026-10-21. The timeout task completes R4's final one-tick compatibility World and re-runs this proof after replacing direct deadline actions with Tick. This task neither removes deadline edges nor claims an all-timeouts-absent World has them. Preserve task prerequisites; refine Files/Touches/sizing and pass the parent plan review before activity admission.

## Acceptance
- [ ] Activity Model lifts and agrees with TLC
- [ ] The one-attempt retry checkpoint passes an explicit lifecycle-refinement comparison against the retained old graph; a changed old edge fails. Old evidence remains byte-identical. The final timeout-era compatibility witness remains assigned to .4.
- [ ] Default world goldens, cover test and terminal controls updated
- [ ] Live slice still passes; gate and model-time pasted

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
