---
satisfies: [R3, R4]
---
# fn-166-mini-activity-a-small-complete-model-to.4 Mini activity: realization level and live witnesses

## Description
The realization level and the live witnesses, so the mini feature's third level is real ((R3), R3).

**Size:** M
**Files:** `mini/realization/MiniActivityRun.scala` (`Realizes` with controller, workers, evidence for every fact, one `deadlines` use), `mini/realization/Kit.scala` (only what the mini feature's actions need, by subtraction from the standalone kit), witnesses in the protocol or product level: one `.live` witness with a `satisfied` expectation and one with an `inconclusive` expectation and its reason
**Touches:** [model/temporal/features/activity/mini/realization/**, model/temporal/features/activity/mini/**, model/ir/activity-mini.*, model/cases/activity-mini-*, model/cases/manifest.json]

### Approach
- Subtract from `standalone/realization/`; keep the kit's typed API reads and the evidence kinds the mini feature's facts need; drop every by-ID, pause and heartbeat helper.
- Lower the mini feature's Cases with the existing commands and run one live once; record the verdict and the Run on the task. If the kit lacks something the mini feature needs, name it in the done summary rather than stubbing it.
- Do not pin the mini feature's Cases in the functional tests or the canary.

## Acceptance
- [ ] The realization lifts, every fact has evidence, one `deadlines` use exists, and the Cases lower.
- [ ] One `.live` witness expects `satisfied` and one expects `inconclusive` with its reason on the expectation.
- [ ] One Case ran live to a verdict once, recorded on the task with the Run.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
