I’ll read the task spec first, then trace the diff through its callers and shared state for integration regressions.
The changes mostly move existing behavior into the shared helper. I’m checking the nonmechanical parts: lock cancellation, empty map results, and error selection.
I found a concrete mismatch with the task’s preservation rule: collecting map keys now returns nil for empty maps, where the old helpers returned non-nil empty slices. I’m tracing how those results are consumed.

Verification is blocked here: Go cannot create its temporary build directory in the read-only sandbox, and lint reports another golangci-lint process is running.
- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `common/testing/testpilot/temporal/worker/driver.go:492`
- **R-IDs:** [R3]
- **Problem:** `slices.Collect(maps.Keys(...))` returns nil for empty maps; the removed helpers returned non-nil empty slices. A workflow-only queue now gets nil Nexus registrations, and unchanged `canonical()` preserves that difference. The same change occurs in `worker/routing.go:98` when no candidates match. This violates the task’s explicit requirement to preserve nil-versus-empty results. Current consumers tolerate the difference, so no downstream runtime failure was established.
- **Suggestion:** Use `slices.AppendSeq(make([]T, 0, len(values)), maps.Keys(values))` at these sites to preserve empty results while retaining the standard-library replacement.

Verification: the focused race suite could not start because the sandbox denied temporary-directory creation. Lint stopped with “parallel golangci-lint is running.” Neither passed in this review.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R3 | partial | Shared primitives and callers consolidated; empty-result preservation needs correction. |
| R8 | partial | Focused gates attempted but blocked by the environment. |

Other epic requirements belong to separate tasks.

Unaddressed R-IDs: [R3]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R3"]}
```

<verdict>NEEDS_WORK</verdict>