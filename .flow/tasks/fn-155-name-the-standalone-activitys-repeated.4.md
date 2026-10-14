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
# Dispatch integration

Moved the two unchanged waiver reasons into the shared `dispatchWaivers` section; no reason-only `states` object remains. Retained both permitted R5 fallbacks: derivation loses HeldDispatch refinement/Product StateFields, and the common capability bound is refused by the lifter despite compiling. The sealed refusal evidence and original machine/realization expectations remain intact.

Worker base: 9a146bb59e7ee7ae2fc249d4017bdda77aca488e. Worker source: a35461306bf65d77786b408ca5ed4d62dfddb7d8. Integrated source range: f7726f57ed575ae5efe932743e2ec9353b674fb1..bcf3c09a4c0f32d42c2875b6a6907dfcf696414a. Review metadata: 0d15d24c6a583810421c5c622ea9b1918178d6d0. Only Dispatch.scala and DispatchWithTaskQueue.scala changed in the source range.

The full worker proof preserves all raw IR fields except Position, all 154 Queries, both race realizations, all 21 exact waivers, and 23 complete native tables comprising 1,361,467 ordered records. The primary IR is byte-identical. The immutable foundation and worker inputs were independently hash-checked against the integrated tree.

Fresh-context Codex gpt-6.1-sol/high review returned SHIP on correctness, contracts and integration, with no findings (same family disclosed); receipt `.flow/tmp/fn155-task4-review/impl-review-receipt.json`. Fresh integrated `make lint-model` exited 0; all three RoleRefinements tests passed. Receipts `.flow/tmp/fn155-integration/task4-lint.json` and `task4-scala-after-build.json` pin unchanged source inputs. The initial Scala command failed because this new worktree lacked its generated API jar; its original RED log/receipt remain retained, followed by the normal lint/build and identical successful test command. No production source fix was needed.

stage: memory-capture - skipped(clean first-round SHIP; no review fixes)
stage: plan-sync - skipped(policy: rolling route)

Inherited canonical Model/Case/fixture/Go RED receipts remain RED. This task does not claim production regeneration, full gates or live evidence; task 6 and the named deferred specs/batch retain those obligations. All task-attributable commands have exited naturally.
## Evidence
- Commits: a35461306bf65d77786b408ca5ed4d62dfddb7d8, bcf3c09a4c0f32d42c2875b6a6907dfcf696414a, 0d15d24c6a583810421c5c622ea9b1918178d6d0
- Tests: make lint-model, mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/framework model/temporal --test-only framework.RoleRefinements, Complete worker IR projection, raw Position-only equality, 154-Query/21-waiver inventories and 23-table/1361467-record native comparison; immutable input hashes independently checked, Codex gpt-6.1-sol high implementation review: three axes SHIP, no findings; RID 17a0f604f80a4ccf89ea875e2d60d04a
- PRs: