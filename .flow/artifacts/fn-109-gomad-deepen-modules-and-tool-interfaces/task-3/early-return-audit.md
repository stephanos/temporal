# Seed completion counter characterization before refactor

Source: task-3 preimage of `runner.go` (SHA-256 `358a37dafa538e66b7b02baea2c0c8cff78268fee35f3a0c5cd75d653eee3e0f`). `FinishAttempt` ran immediately on receipt, before every condition below. Thus each received completion decremented active work and incremented `Attempted`, even when a later host/evidence operation failed. `Record*` changed classification only at the three explicit sites.

| Path after receipt | Existing classified counters | Other reported counters/effects |
| --- | --- | --- |
| Overall context error; supervision error; prepared target verification error | None | Supervision may publish a Runner failure and set summary `DistinctFailures` directly. |
| Stopped, cancelled result | `Cancelled` +1 | Even partial preservation, Runner failure publication, or journal append errors do not undo cancellation. |
| World assessment error | None | Runner failure publication may set summary `DistinctFailures` directly. |
| Completion assessment error; diagnostic trace write error; execution evidence mount error; journal classified transition error; context error after transition | None | Existing evidence or partial state may remain. |
| Success retention decision, success artifact publication, or guidance merge error | None | Any previously published success artifact and retention totals remain reported. |
| Success journal append error | None | The success record is not counted. |
| Success journal append succeeds | `Succeeded` +1 | Later partial cleanup error does not undo success. |
| Failure manifest mount/build error; failure artifact publication error; context error after artifact publication; guidance merge error | None | Published artifact may exist, but it is not in the distinct map until after guidance. |
| Failure artifact is accepted into distinct map | `Failures` +1; watchdog/replay counters by domain/reason; `DistinctFailures` set to map size | Later relative-path, journal append, or partial cleanup errors do not undo classification. First policy stops and cancels active; budget stops admission without cancellation; all keeps admitting. |

The added `TestRunEarlyCompletionCountersRemainUnclassified` pins five independently reachable host/evidence returns, including the World error's `Attempted=1`, `Failures=0`, `DistinctFailures=1` combination. Existing `TestCompletionFaultsKeepReasonPrecedenceAndEvidence` covers additional simultaneous faults across seed and exploration strategies; `TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem` and `TestRunBudgetCountsDistinctSignatures` pin policy behavior.
