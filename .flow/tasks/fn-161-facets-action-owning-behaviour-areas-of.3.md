---
satisfies: [R3, R5, R9]
---
# fn-161-facets-action-owning-behaviour-areas-of.3 Stage 2: standalone activity rules and capabilities into owning facets

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

Stage 2 for the standalone activity: move rules and capability declarations into their owning facets, and turn on the one-owner-per-action check for these machines.

**Size:** M
**Files:** the standalone activity's product and system facet files and machine files, regenerated `model/ir/activity-standalone*.json`
**Touches:** [model/temporal/features/activity/standalone/**/*.scala, model/ir/activity-standalone*.json, model/ir/*.lint.json]

### Approach
- **Assign each action to its owning facet:**
  - Dispatch: `client.start`, `activityWorker.poll`
  - Completion: `activityWorker.respondCompleted`, `completionClient.respondCompletedByID`
  - RetryFailures: `activityWorker.respondFailed`, `completionClient.respondFailedByID`
  - Cancellation: `client.requestCancel`, `activityWorker.respondCanceled`, `completionClient.respondCanceledByID`
  - Pausing: `client.pause`, `client.unpause`
  - Reset: `client.reset`
  - Heartbeat: `activityWorker.heartbeat`, `deadline.heartbeat`
  - Timeouts: `timers.timeout` and the system's terminal deadlines
  - Lifecycle: `client.terminate`, `process.stop`, the closed-state `notFound` blocks
  
  (Names assume fn-159.1 and fn-159.2 have landed.) The By-ID actions sit with what they do, so the worker and By-ID rows that must match sit side by side.
- **Within a facet,** group rules by actor with `from(<actor>) { … }`, using fn-159.1's qualified forms.
- **Capabilities:** move each declaration into the facet it describes (`Pausable` into Pausing, `Pollable` into Dispatch, `Closable` into Lifecycle, `Retries`/`Retryable` into RetryFailures, `Deadline`/`Expirable` into Timeouts or Heartbeat). Waivers and overrides move with them.
- **Interaction facet:** create one only if a case has no natural owning action, and record why.
- **IR:** regenerate. Rows, outcomes, facts, Cases and Query answers must stay the same. Each action's step function case order follows facet order, so function bodies and hashes change. Record them.

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now. Checked-in IR, Cases and Go consumers may go stale. Verify with the Scala build, framework and model tests, and fn-160.5's step-table comparison under this task's rename map. Declare the IR change this task will cause; the IR is regenerated once, in Batch 5b, whose shared close supplies the gate evidence for the `make umpire-check-*` criteria. fn-161.2 and fn-161.3 are done in one pass on the activity, after fn-159.7 and fn-159.2.

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
```
## Acceptance
- [ ] Every action of both standalone activity machines is owned by exactly one facet holding all its rules; capability declarations sit in their facets; the ownership check passes.
- [ ] Regenerated IR keeps every row, outcome, fact, Case and Query answer; `make umpire-check-model` and `make umpire-check-lint` pass.
- [ ] The done summary records any interaction facet created and why, or that none was needed.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: