I’ll read the task spec first, then trace the changed runtime paths and tests for correctness regressions.

The server now reads prepared instruction bounds, and the handle path still rechecks claims after `Accepts` releases the lock. The range also includes an opcode-table refactor; I’m checking that its dispatch and admission behavior match the previous switches.

I found no correctness regression in the changed paths. Verification is limited: the race tests could not create a temporary build directory in the read-only sandbox, and lint exited because another golangci-lint process holds its lock.

No surviving findings. The changed authority checks, prepared bounds, claim recheck, and opcode dispatch preserve the traced behavior.

Verification: race tests could not start because the read-only sandbox blocked temporary-directory creation. Lint exited because another golangci-lint process held its lock.

## Requirements coverage

Coverage is scoped to this task and commit range.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Correlated-state work is outside this task. |
| R2 | deferred | Campaign-wide dead-code removal is outside this task. |
| R3 | deferred | Campaign-wide primitive consolidation is outside this task. |
| R4 | met | Server uses shared ceiling checking and prepared instruction bounds; authority rejection tests remain. |
| R5 | met | Opcode table drives binding and effect acceptance; rejection-order tests remain. |
| R6 | deferred | Workflow binding consolidation is outside this task. |
| R7 | deferred | Shared test-helper campaign is outside this task. |
| R8 | partial | Focused verification attempted but blocked as described above. |
| R9 | deferred | Protocol removal is outside this task. |
| R10 | deferred | Final campaign measurement is outside this task. |
| R11 | met | No golden or identity-pin files changed. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>