I’ll read the task spec first, then trace the changed code through its callers and dependencies, focusing on integration issues introduced by this diff.

The consolidated paths appear to preserve the authority checks and the claim re-check after `Accepts`. I’m checking how prepared plans supply methods and bounds, including cleanup entries, and whether the broader ceiling check affects existing callers.

The prepared-plan APIs supply the expected RPC methods, resolved bounds, and cleanup instructions; I haven’t found an introduced integration defect. Verification is incomplete: the race tests cannot create a temporary build directory in this read-only sandbox, and lint exited because another golangci-lint process holds its lock.

No qualifying findings in the reviewed changeset. The prepared-plan index preserves cleanup handling, RPC authority checks, resolved bounds, and claim validation after reacquiring the lock.

Verification: `git diff --check` passed. Race tests were blocked by sandbox filesystem permissions; lint exited because another golangci-lint process was running.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R4 | met | Task-scoped server changes use `CheckCeilings` and `InstructionPlan`; authority rejection and claim replacement tests cover retained checks. |
| R8 | partial | Required commands attempted; execution blocked as described above. |
| R11 | met | No golden changes in this diff. |
| R1, R2, R3, R5, R6, R7, R9, R10 | deferred | Other epic tasks; outside task .10’s scope. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>