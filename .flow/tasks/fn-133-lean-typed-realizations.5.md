---
satisfies: [R7, R8]
---
# fn-133-lean-typed-realizations.5 Name collisions and the activity realization's local fixes

## Description
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
Part B, R7, R8.

Apply the activity fixes in `model/temporal/features/activity/standalone/system/Realization.scala`, preserving the System placement introduced by fn-126.11 and carried through fn-132. Do not recreate the historical root realization file.

- **Collisions.** Rename the kit's `deadline` operand, and settle how a realization names the shared worker party next to a feature's `worker` section (consistent with fn-126.8's names), so no realization imports `deadline as requestDeadline` or `worker as process`.
- **Activity realization fixes:**
  - name `stopWorkerUntilReleased` for what it is (the pause path's stop);
  - `attemptFailure` (or `applicationFailure`) takes `retryable`;
  - `lostAdmissionResponse` gets its own section;
  - floating comments attach to their declarations;
  - rewrite the header and the `status` doc plainly;
  - `perCase` and the script stop sharing the string "activity".

## Acceptance
- [ ] No realization imports a collision alias.
- [ ] Each local fix is done.
- [ ] A before/after projection is identical apart from listed renamed command ids.
- [ ] The spec's Verification gates pass.

## Done summary
Removed the collision aliases and made the activity realization's local fixes (R7, R8). The activity realization stays in `model/temporal/features/activity/standalone/system/Realization.scala`, at its System placement.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: IMPLEMENTER claude-opus-5-5 at high

### What changed
- **Collisions (R7).**
  - The kit's `deadline` operand is renamed `deadlineOperand`, on its one line in Kit.scala, so no kit position moves. The activity imports `temporal.realize.*` without hiding anything.
  - The shared worker party is named by its qualified name, `shared.worker.worker.stop`, beside the feature's `worker` section, so `worker as process` is gone. Decision: the party keeps its fn-126.8 name, and a realization qualifies it. Renaming the party object would rename its action ids in every Model.
  - lifts/Scripts.scala now uses `deadlineOperand`.
- **Activity local fixes (R8):**
  - `stopWorkerUntilReleased` is renamed `stopWorkerBeforePause` (the pause path's stop), each fault with its own comment.
  - `attemptFailure` takes `retryable`: done in fn-133.1, and now named `failed(retryable)` because fn-133.3 took `attemptFailure` for the kit's lower-case form.
  - The unreachable `scheduleToClose` await: removed in fn-133.4.
  - `lostAdmissionResponse` has its own section, `### The lost admission answer`, with `loseAdmissionResponse`.
  - Floating comments are attached to their declarations: the scheduling paragraph goes to `standalone`, the "machine starts scheduled" paragraph to `heldDelivery`, and the "server refuses a start" note to `startUnreached`.
  - The header and the described-status doc are rewritten plainly.
  - The script `"activity"` is now `"attempts"`, so it no longer shares the string with `perCase("activity")`.

### Declared IR delta (batch regeneration)
- **Renamed command id:** `stop-worker-until-released` → `stop-worker-before-pause`, in activity-standalone.json (`standalone`, `controller`). The generated Cases carrying it change only by this id.
- **Renamed script id:** `activity` → `attempts`, in activity-standalone.json (`standalone`), together with the two `delivered` evidence kinds' `runEvent.attempt.script` that name it.
- Everything else: positions only.
- Checked with a scratch lift: the evidence differs only in `script: activity → attempts`, and the scripts only in those two ids, besides fn-133.4's declared deltas.

### Tests
- `mise exec -- scala-cli test model/irgen`: 96 passed, 0 failed.
- A scratch lift of model IR shows the two renames and nothing else new.

### Line counts
After .5: activity 308.

### For later tasks
- Package-level helpers of the activity realization will move out of `object ActivityRealization` in fn-133.6. Their command names do not change, and evidence ids stay keyed by package.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1e36538992
- Tests: mise exec -- scala-cli test model/irgen, scratch lift --ir of model IR; activity realizations differ only by the declared renames (plus fn-133.4 deltas)
- PRs: