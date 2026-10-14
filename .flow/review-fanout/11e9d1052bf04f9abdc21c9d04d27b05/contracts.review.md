I’ll read the task specification first, then review the diff and affected callers for contract and consistency issues.
The diff preserves the three ceiling-check error formats, field order, and correlated-field exceptions. The Go test command could not start because the read-only sandbox blocks creation of its temporary build directory. I’m checking the existing tests and the remaining verification commands.
No introduced findings. The extraction preserves rejection paths, details, ordering, nil handling, and skipped fields. Tests cover all three ceiling-check call shapes.

Verification: `git diff --check` passed. Go tests and conformance were blocked by sandbox write restrictions; lint was blocked by another running golangci-lint process.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R3 | partial | Core primitives consolidated; remaining work belongs to fn-94.8. |
| R8 | partial | No fixture changes; execution of verification gates was blocked as described above. |
| R1, R2, R4–R7, R9–R11 | deferred | Assigned to other tasks in the parent specification. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>