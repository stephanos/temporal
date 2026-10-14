I’ll read the task spec first, then check the diff for agreement between the implementation, tests, and documented contracts.
The diff replaces the rebuilt node and evidence maps with prepared instruction plans and preserves both claim-validity checks around the lock release. I’m tracing how those plans are bound and checking whether the tests cover the stated authority guarantees.
I haven’t found an introduced contract mismatch. The focused race tests could not start because the read-only sandbox blocks Go’s temporary build directory; lint also stopped because another golangci-lint process was running. I’ll report both as unverified.
No surviving findings. The implementation, README, and tests agree on prepared instruction bounds, shared unary authorization, and claim validation after reacquiring the lock.

Verification was incomplete: race tests could not create Go’s build directory in the read-only sandbox; lint stopped because another golangci-lint process was running.

## Requirements coverage

Statuses reflect this task’s scope.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Assigned to other tasks. |
| R2 | deferred | Assigned to other tasks. |
| R3 | deferred | Assigned to other tasks. |
| R4 | met | Shared ceiling check, Profile snapshot, InstructionPlan index, preserved authority checks, and claim re-check regression test. |
| R5 | deferred | Assigned to other tasks. |
| R6 | deferred | Assigned to other tasks. |
| R7 | deferred | Assigned to other tasks. |
| R8 | deferred | Focused verification attempted but blocked as described above. |
| R9 | deferred | Assigned to the wire-removal task. |
| R10 | deferred | Assigned to the final measurement task. |
| R11 | met | No golden or identity-format changes in this range. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>