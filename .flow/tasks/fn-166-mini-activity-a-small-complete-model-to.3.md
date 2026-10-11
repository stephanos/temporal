---
satisfies: [R3]
---
# fn-166-mini-activity-a-small-complete-model-to.3 Mini activity: protocol level with composition, channel, derivations and claims

## Description
The protocol level with its composition, channel, derivations and claims (R3).

**Size:** M
**Files:** `mini/protocol/Protocol.scala` (phases including `backingOff` reporting the same status as `scheduled`; attempts as `UpTo[2]`; refinement to the product with `query verify` through it), `mini/protocol/Dispatching.scala` and `Completion.scala` (facets), `mini/protocol/Queue.scala` (composition over the task queue foundation with one `sync`; the worker's answers on one bounded lossy channel), `mini/protocol/Faults.scala` (one `FailureModel` with `NegativeControl`: a dispatch that trusts a stale answer; one `assume.fair` progress claim; one monitor; one hole; one assumption), capabilities `Retryable` and `Expirable` with one `overriding` and its reason
**Touches:** [model/temporal/features/activity_mini/protocol/**, model/temporal/features/activity_mini/MiniActivity.scala, model/temporal/features/activity_mini/generated/**, model/temporal/Bounds.scala, model/testdata/binding-order.tsv, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/layout/mini/**]

### Approach
- Subtract from `activity/standalone/protocol/` as task 2 did from the product; keep the shared-input case by letting the queue's `schedule` take `scheduleToStart` as `start` does.
- The channel carries the worker's answers so that a stale answer after a retry is representable; the failure model binds the dispatch that accepts it and its `NegativeControl` finds the violation.
- One `leadsTo` under `assume.fair(worker)` with a bound from `Bounds`; one monitor over attempts; one hole where the mini feature does not say what happens (for example a cancel during back-off), named in the header comment.
- Check: the refinement is closed; the `verify` through it holds; the negative control's `find` finds; the IR stays in the tens of kilobytes.
- Independently derive binding-order inventory rows for new mini exports from their declarations. Update only mini-feature rows in `model/testdata/binding-order.tsv`; preserve every pre-existing real-feature and shared-package row byte-identically. Task 2's scoped source-law helper already supplies the selected-package inventory contract; do not relax its refusal assertions or change table pins or equivalence baselines.
- Follow the current owner-held boundary: no framework semantic changes, shared schema generation, real-feature artifact regeneration, deferred Go changes or live runs. Use the spec's direct Scala fallback when canonical Make prerequisites are held; record red canonical gates without credit. Lint is not run under the owner's current fn-166 override.
- Repair the mini layout fixture proof to lift the isolated fixture, not production mini machines discovered through the production jar. Keep all three variants and exact positive/refusal assertions; add an independent isolation control so production inventory changes cannot masquerade as fixture coverage. Do not repair it by expanding the expected production-machine list. Preserve unrelated fixture assertions and all protected payloads.
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
