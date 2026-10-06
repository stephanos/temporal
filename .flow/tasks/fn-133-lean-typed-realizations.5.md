---
satisfies: [R7, R8]
---
# fn-133-lean-typed-realizations.5 Name collisions and the activity realization's local fixes

## Description
Part B, R7, R8.

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
