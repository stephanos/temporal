I’ll read the task specification first, then trace the diff through its callers and focused tests for integration regressions.

The three ceiling checks preserve their existing nil guards, field order, error paths, and correlated-field exceptions. The shared ID and nil helpers also match the removed implementations. I’m checking the existing rejection tests and attempting the task’s verification commands.

No blocking findings. The shared helpers preserve caller behavior, including typed-nil rejection, ceiling-check order, error paths/details, and correlated-field exceptions.

Verification: `git diff --check` passed. Go tests and corpus verification were blocked by filesystem permissions; lint stopped because another golangci-lint process was running. These checks are not claimed as passing.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Owned by fn-94.1/.3. |
| R2 | deferred | Owned by fn-94.4/.5/.17. |
| R3 | met | Task-scoped primitives consolidated in `ir`; all three ceiling call shapes covered by table tests. Driver consolidation belongs to fn-94.8. |
| R4 | deferred | Owned by fn-94.6/.9/.10. |
| R5 | deferred | Owned by fn-94.11/.12. |
| R6 | deferred | Owned by fn-94.13. |
| R7 | deferred | Owned by fn-94.14/.15. |
| R8 | partial | No fixture changes; verification attempts blocked as described above. |
| R9 | deferred | Owned by fn-94.16. |
| R10 | deferred | Owned by fn-94.2/.17. |
| R11 | deferred | Owned by fn-94.2/.16; no golden changes here. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>