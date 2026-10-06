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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
