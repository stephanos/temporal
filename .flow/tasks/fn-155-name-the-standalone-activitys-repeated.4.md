---
satisfies: [R5]
---
# fn-155-name-the-standalone-activitys-repeated.4 Derive HeldDispatch and unify the Dispatch composition capabilities

## Description
Implements spec section E in fn-151's Dispatch* files. HeldDispatch derives from the corrected design only if the equivalence holds. A single generic class serves the two queue compositions. Waiver reasons move into their own section.

**Size:** M
**Files:** `system/Dispatch.scala`, `system/DispatchRaces.scala`, `system/DispatchWithTaskQueue.scala`; `RoleRefinements.test.scala` if HeldDispatch's refinement owner changes
**Touches:** [model/temporal/features/activity/standalone/system/Dispatch.scala, model/temporal/features/activity/standalone/system/DispatchRaces.scala, model/temporal/features/activity/standalone/system/DispatchWithTaskQueue.scala, model/temporal/features/activity/standalone/RoleRefinements.test.scala, model/irgen/test/Fixtures.test.scala]

### Approach
- **HeldDispatch.** Build `ActivityRecord.restrict(history.dispatch, client.pause, worker.poll, history.answerMatching).rebind(...).refining(ActivityProduct)(...)` with the committed admission. `restrict` drops the refinement (`model/framework/Machine.scala:410-412`), so `refining` (`:456`) restores it. Follow `TrustingActivityRecord` (`Dispatch.scala:323-334`) and `ActivityWorker`.
- **Equivalence check.** Compare the interpreter-built step table (task 1's dump), refinement, monitors and evidence with the hand-written machine before replacing it. If anything differs, keep the hand-written one and record why in `mapping.md`.
- **Shared capability class.** Use the form task 1's probe accepted: one class over a shared bound on the composed record, replacing `OverQueueCapabilities` and `OverMatchingCapabilities` (`DispatchWithTaskQueue.scala:53-83`). Map the state-type change, and update `model/irgen/test/Fixtures.test.scala:2279-2284` if the pinned lift changes. If the probe refused it, keep both classes and record why (spec R5).
- **Waiver reasons.** Move `queueStepsOn` and `deliveryAfterClose` into an `object waivers`, or the shared reasons.
- **Run expectation.** Map HeldDispatch's IR identity change. The race realization (`HeldDelivery`) must keep its run expectation.

### Investigation targets
**Required:**
- `system/Dispatch.scala:323-334` — TrustingActivityRecord; `system/DispatchRaces.scala:32` — HeldDispatch
- `model/framework/Machine.scala:413-470` — `restrict`, `rebind`, `unmonitored`
- `system/DispatchWithTaskQueue.scala:53-130`

### Acceptance
- [ ] HeldDispatch is either derived, with an equal step table and run expectation, or kept with the recorded reason
- [ ] One capability class serves both compositions, or both stay with the recorded reason; the `activity-standalone-record` IR on a scratch lift differs only by mapped identities and recorded structural-review entries
- [ ] No `object states` holds only reason strings
- [ ] Focused Scala tests pass
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
