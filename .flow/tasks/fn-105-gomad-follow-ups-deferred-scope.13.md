---
satisfies: [R1, R2]
---
# fn-105-gomad-follow-ups-deferred-scope.13 D13: choice-trace capacity for the transcript-heavy suites

## Description
Origin: fn-106.3 (2026-09-30). TestTaskQueueStats_Pri_Suite, TestVersioning3FunctionalSuite, TestVersioning3QueryFunctionalSuite, and TestWorkerDeploymentSuite exceed the 64 MiB choice-trace maximum, so they qualify without a choice trace: same-seed repeatability is proven but no exact-replay artifact is retained. Raising the choice-trace bound (as the I/O transcript bound was) or streaming the trace would restore replay. Deferred because repeatability is already proven. Revive when a failure in one of these suites needs an exact replay to diagnose.

## Acceptance
- the four suites qualify with choice traces and exact replay


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
