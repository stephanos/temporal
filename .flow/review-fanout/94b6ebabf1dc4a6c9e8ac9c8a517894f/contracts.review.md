I’ll read the task spec first, then trace the diff and its callers to check that the implementation, tests, and documented contracts agree.
The diff includes an opcode-table refactor as well as the server changes. I’m tracing both, with particular attention to unary-call authorization and the claim check after `Accepts` releases the lock.
The authority checks and both claim-validation sites appear preserved, and the new tests cover resolved bounds and claim replacement during `Accepts`. Verification is limited: race tests cannot create a build directory in this read-only sandbox, and lint stopped because another golangci-lint process is running. I’m finishing the static check of opcode dispatch.
No blocking findings. The server preserves per-call authority checks, reads resolved instruction bounds, and rechecks claims after `Accepts`. The opcode table preserves the previous binding order and dispatch behavior.

Verification: `git diff --check` passed. Race tests could not start because the read-only sandbox prevents build-directory creation. Lint stopped because another golangci-lint process is running.

## Requirements coverage

Statuses cover this task and the additional opcode-table changes; other campaign work is deferred.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Correlated-state work outside this task. |
| R2 | deferred | Campaign dead-code removal. |
| R3 | deferred | Campaign primitive consolidation. |
| R4 | met | Shared ceiling check, `InstructionPlan` index, resolved bounds, authority rejection tests. |
| R5 | partial | Opcode table drives binding and dispatch; rejection-order tests retained. Other D1 work deferred. |
| R6 | deferred | Workflow-binding consolidation outside this task. |
| R7 | deferred | Shared test-helper campaign. |
| R8 | partial | Verification attempted; execution limitations noted above. |
| R9 | deferred | Protocol-removal lane. |
| R10 | deferred | Final campaign measurement. |
| R11 | met | No golden, fixture, or protocol files changed. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>