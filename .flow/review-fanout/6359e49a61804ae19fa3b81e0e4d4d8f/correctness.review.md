I’ll read the task spec first, then review the diff and affected callers for correctness and coverage.

The new helpers appear to preserve the existing assertions and add binding checks to the pair fixture. I’m checking run capture, concurrent execution, and the immutability checks, then I’ll attempt the focused verification commands.

No blocking correctness or logic findings in the changed paths.

The consolidated helpers preserve sequential/concurrent execution, namespace isolation, distinct Run IDs, Case immutability, and frozen-binding assertions. No live test or subtest identities changed.

Verification:
- `git diff --check` passed.
- Focused Go tests could not start: sandbox denied temporary-directory creation.
- Live checks stopped during Lean compilation: sandbox denied artifact writes.
- Lint stopped because another golangci-lint process held the lock.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R7 | partial | Task-scoped consolidation is implemented: one scripted session, one preparation helper, and one repeated-run helper serving all three callers. The baseline of 45 passing live identities could not be reverified. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>