---
satisfies: [R5, R9]
---
# fn-161-facets-action-owning-behaviour-areas-of.2 Stage 1: standalone activity effects and claims into facets

## Description
Stage 1 for the standalone activity: move its effects and claims into facet files at both levels. Rules stay grouped by actor in each machine. Proposed facets: Dispatch, Completion, RetryFailures, Cancellation, Pausing, Reset, Heartbeat, Timeouts, Lifecycle. Each level gets only the facets that have something there.

**Size:** M
**Files:** `model/temporal/features/activity/standalone/product/*.scala`, `system/*.scala` (new facet files; existing claim files `Completion`, `RetryFailures`, `Cancellation`, `Pausing`, `Timeouts`, `Heartbeat`, `ResponseByID`, `Reset` absorb their facet's effects), regenerated `model/ir/activity-standalone*.json`
**Touches:** [model/temporal/features/activity/standalone/**/*.scala, model/ir/activity-standalone*.json, model/ir/*.lint.json]

### Approach
- **Declare the facet vocabulary** in the standalone activity's feature file (R9): Dispatch, Completion, RetryFailures, Cancellation, Pausing, Reset, Heartbeat, Timeouts, Lifecycle. Both levels draw their facet names from it.
- **Assign each effect to a facet.** Today's proposed split after fn-159.7:
  - product: Dispatch `schedule`; RetryFailures `serviceRetry`, `failOrReset`; Pausing `pauseHeld`, `unpauseHeld`, `pauseKeptPending`; Reset `resetHeld`, `resetHeldKeepingPause`; Heartbeat `heartbeat`, `heartbeatExpires`, `heartbeatTimeOut`
  - system: Dispatch `schedule`, `startAttempt`, `keep`, `dispatchNow`; Completion `complete`; RetryFailures `fail`, `backOff*`; Cancellation `cancel`, `requestCancel`; Pausing `pause`, `requestPause`, `resume`, `withdrawPause`; Reset `reset*`, `requestReset*`, `applyReset*`; Heartbeat `heartbeat*`; Timeouts `timeOut`; Lifecycle `terminate`
  
  Re-survey against the tree first; fn-159.7 removes the phase-only ones.
- **Keep choice tokens with their decision** inside the facet. Make cross-facet step use explicit with self-types, e.g. RetryFailures depends on Reset for `failOrReset`. Shared steps go in Dispatch or Lifecycle.
- **Claims:** each facet file holds its facet and the existing derived claim object, unchanged (its name and Queries keep their identities). If one-machine-per-file (fn-160.3) has landed, follow its flat layout: one object per file, grouped by the facet's name prefix (`Heartbeat.scala` for the facet, `HeartbeatCompletion.scala`, `HeartbeatRetry.scala`, … for its claim objects), no subfolders.
- **IR:** regenerate. Rows, outcomes, facts, Cases and Query answers must stay the same; differences limited to function bodies, hashes and source positions. Record them.

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now. Checked-in IR, Cases and Go consumers may go stale. Verify with the Scala build, framework and model tests, and fn-160.5's step-table comparison under this task's rename map. Declare the IR change this task will cause; the IR is regenerated once, in Batch 5b, whose shared close supplies the gate evidence for the `make umpire-check-*` criteria. fn-161.2 and fn-161.3 are done in one pass on the activity, after fn-159.7 and fn-159.2.

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
```
## Acceptance
- [ ] Every effect of the standalone activity's product and system machines lives in a facet file; the machines' rules remain grouped by actor; claim objects sit beside their facet with unchanged identities.
- [ ] Cross-facet step use is declared in facet types; choice tokens sit in their decision's facet.
- [ ] Regenerated IR keeps every row, outcome, fact, Case and Query answer; `make umpire-check-model` and `make umpire-check-lint` pass.


## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
