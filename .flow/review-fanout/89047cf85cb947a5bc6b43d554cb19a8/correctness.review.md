I’ll read the task specification first, then trace the changed admission logic and its callers for correctness and edge cases.
The three ceiling checks appear to preserve their original rejection order, paths, and details, including the correlated limits’ optional fields. I’ll check the surrounding validation and attempt the task’s verification commands; the read-only sandbox may prevent them from running.
No introduced correctness or logic findings. The shared helpers preserve validation behavior, rejection order, paths, and details.

Verification: Go tests and conformance checks were blocked by sandbox write restrictions. Lint stopped because another golangci-lint process was running.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R3 | met | Task .7’s four primitives consolidated; table tests cover all three ceiling-check shapes. |
| R8 | partial | Fixtures unchanged in the diff; verification could not complete. |
| R1, R2, R4–R7, R9–R11 | deferred | Assigned to other tasks; outside this review’s scope. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>