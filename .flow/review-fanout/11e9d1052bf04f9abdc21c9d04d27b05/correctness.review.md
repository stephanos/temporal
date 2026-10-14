I’ll read the task specification first, then trace the changed code and its callers for correctness and edge cases.

The three ceiling checks preserve their existing rejection order, paths, details, and skipped fields. The shared ID and nil helpers also match the removed implementations. I’ll check the existing regression coverage and attempt the task’s verification commands.

No introduced correctness findings. The shared helpers preserve validation behavior, rejection order, paths, details, and correlated-field exemptions. The new table test covers all three ceiling-check shapes.

Verification:
- `git diff --check` passed.
- Go tests could not start: the read-only sandbox prevented temporary-directory creation.
- Conformance verification stopped because Lean build artifacts could not be written.
- Lint stopped because another `golangci-lint` process held its lock.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Assigned to tasks .1 and .3. |
| R2 | deferred | Assigned to tasks .4, .5, and .17. |
| R3 | partial | This task’s four primitives consolidated; Driver and standard-library work belongs to .8. |
| R4 | deferred | Assigned to .6, .9, and .10. |
| R5 | deferred | Assigned to .11 and .12. |
| R6 | deferred | Assigned to .13. |
| R7 | deferred | Assigned to .14 and .15. |
| R8 | partial | No fixtures changed; verification attempts blocked as described above. |
| R9 | deferred | Assigned to .16. |
| R10 | deferred | Assigned to .2 and .17. |
| R11 | deferred | Assigned to .2 and .16; this diff changes no goldens. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>