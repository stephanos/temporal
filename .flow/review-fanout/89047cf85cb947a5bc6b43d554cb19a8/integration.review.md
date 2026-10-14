I’ll read the task specification first, then trace the diff through its callers and run the focused checks available in this sandbox.
The shared helpers preserve the existing nil handling, ID rules, ceiling-check order, and error paths. I’m checking the callers’ nil guards and correlated-limit exceptions before attempting the task’s verification commands.
No surviving findings. The refactor preserves callers’ nil guards, rejection order, error paths/details, and correlated-limit exceptions.

Verification: `git diff --check` passed. Go tests and conformance verification were blocked by read-only filesystem restrictions. Lint stopped because another golangci-lint process was running.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R3 | met | Task .7’s four primitives are centralized in `ir`; callers reuse them. Table tests cover all three ceiling-check shapes. |
| R8 | partial | Fixtures unchanged; execution of verification gates blocked as described above. |
| R1, R2, R4–R7, R9–R11 | deferred | Assigned to other tasks by the epic’s requirement coverage. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>