# Activity coverage

## Goal

Model the standalone activity behaviour that upstream's Go model (`chasm/lib/activity/model`) covers and ours does not. Source: `.plans/ACTIVITY_MODEL_COMPARISON.md` (recommendations P2-5, P2-6, P2-7, P3-13). Owner decision 2026-10-05.

## Current source context

Current source after fn-132.2: `model/temporal/features/activity/standalone/Standalone.scala` owns the form; `product/Product.scala` and `system/System.scala` each own their local `Phase`, `State` and `Fact`. Actions are `client.start` and `client.control` on the `activity` entity. All three System realizations remain in `system/Realization.scala`. The kind header `features/activity/Activity.scala` declares `Timeout`, `TimeoutType`, `AttemptResult`, the worker's `poll` and `respond`, `timers` and `deadline` (fn-132.6); the form binds the worker actions to its `activity` entity. IR stems are `activity-standalone`, `activity-standalone-record` and `activity-standalone-race`.

## Requirements

- **R1 Heartbeat.**
  - A `worker.heartbeat` action, enabled while the worker holds the attempt.
  - A `deadline.heartbeat` timeout, retryable under the precision spec's retry policy.
  - An observation of heartbeat details if the realization can read them.
- **R2 Respond by ID.** An actor (e.g. `service`, decision 26's `Actor`) whose completion, failure and cancel answers go by activity ID rather than task token, and are allowed in the phases the server allows (`model.go`'s `ByID` forms). Realized through the corresponding API calls.
- **R3 Reset.** A `Control.reset(keepPaused)` class, a `resetRequested` phase for a reset deferred while an attempt is held, and its application on the attempt's settlement, as the Go model's `applyDeferredReset` does. Properties cover keep-paused and the Cancel > Reset > Pause precedence, extending the precision spec's precedence Property.
- **R4 Exploration.** `.explore` on the activity's `find` Queries where it adds witnesses the pinned Scenarios miss (today only the Nexus control uses it).
- **R5 Realization and Cases.** Each new behaviour is realized, with at least one live Query per requirement, and the done summary lists the new Cases.

## Boundaries

- The approved activity-batch conductor gate starts implementation after fn-138.3 is done, with fn-138 itself starting after fn-128.5 is done. Flow cannot express cross-spec task dependencies, so these source gates supplement its metadata rather than waiting for either prerequisite spec to close. fn-128.6 and fn-129.5 share regeneration, review and live-run evidence at the batch boundary; no activity spec closes prematurely.
- No workflow-scheduled activity: that stays with fn-119.
- No change to upstream's Go model or its harness. Driving their harness from our IR (comparison P3-12) is an owner decision recorded in MILESTONES.

## Verification

As the precision spec, plus each new Query's live Case run once.
