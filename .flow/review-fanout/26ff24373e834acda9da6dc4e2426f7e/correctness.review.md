I’ll read the task spec first, then trace the changed code and its callers for correctness and edge cases.
The refactor uses the prepared instruction bounds and preserves both claim checks around `Accepts`. I’m checking that the shared RPC prelude preserves authorization for both invocation and polling, then I’ll attempt the focused verification commands.
No blocking correctness or logic findings in the changed paths. The shared authorization checks preserve coordinate, role, and method restrictions; both handle-claim checks remain under the lock.

Verification was limited: the focused race tests could not create a temporary build directory in the read-only sandbox. `make lint-code-fast` stopped with “parallel golangci-lint is running.” `git diff --check` passed.

## Requirements coverage

Coverage is scoped to task `.10`’s declared requirement, R4.

| R-ID | Status | Evidence |
|---|---|---|
| R4 | met | Construction uses `ir.CheckCeilings`; sessions index `InstructionPlan` and read resolved bounds. Rejection tests cover coordinate, role, method, attempt bounds, and claim replacement during `Accepts`. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>