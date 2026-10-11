---
satisfies: [R3]
---
# fn-166-mini-activity-a-small-complete-model-to.3 Mini activity: protocol level with composition, channel, derivations and claims

## Description
The protocol level with its composition, channel, derivations and claims ((R3)).

**Size:** M
**Files:** `mini/protocol/Protocol.scala` (phases including `backingOff` reporting the same status as `scheduled`; attempts as `UpTo[2]`; refinement to the product with `query verify` through it), `mini/protocol/Dispatching.scala` and `Completion.scala` (facets), `mini/protocol/Queue.scala` (composition over the task queue foundation with one `sync`; the worker's answers on one bounded lossy channel), `mini/protocol/Faults.scala` (one `FailureModel` with `NegativeControl`: a dispatch that trusts a stale answer; one `assume.fair` progress claim; one monitor; one hole; one assumption), capabilities `Retryable` and `Expirable` with one `overriding` and its reason
**Touches:** [model/temporal/features/activity/mini/protocol/**, model/temporal/features/activity/mini/MiniActivity.scala, model/temporal/Bounds.scala, model/ir/activity-mini.*, model/cases/activity-mini-*, model/cases/manifest.json]

### Approach
- Subtract from `standalone/protocol/` as task 2 did from the product; keep the shared-input case by letting the queue's `schedule` take `scheduleToStart` as `start` does.
- The channel carries the worker's answers so that a stale answer after a retry is representable; the failure model binds the dispatch that accepts it and its `NegativeControl` finds the violation.
- One `leadsTo` under `assume.fair(worker)` with a bound from `Bounds`; one monitor over attempts; one hole where the mini feature does not say what happens (for example a cancel during back-off), named in the header comment.
- Check: the refinement is closed; the `verify` through it holds; the negative control's `find` finds; the IR stays in the tens of kilobytes.

## Acceptance
- [ ] The protocol refines the product, a product Property verifies through the refinement, and the refinement check passes.
- [ ] `scheduleToStart` is an input of both `start` and the queue's `schedule`; `backingOff` reports the same status fact as `scheduled`.
- [ ] One composition with one `sync`, one bounded lossy channel, one `FailureModel` with a `NegativeControl` whose `find` finds, one `leadsTo` under `assume.fair`, one monitor, one hole, one assumption, `Retryable` and `Expirable` with one `overriding` and a reason.
- [ ] `make model-mini` passes and the header comment lists every hard instance with its location.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
