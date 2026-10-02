I’ll read the task spec first, then trace the changed code and its callers for correctness and edge cases.

The shared admission helpers preserve the existing check order and replay caches. I’m tracing the new binding extractor through the composite driver and checking the concurrency test. The workspace has unrelated local edits; I’m keeping those outside the review scope.

No blocking findings in the changed code. The shared admission paths preserve rejection order, replay behavior, and locking; binding consolidation preserves the route’s JSON layout.

Verification was limited: the race suite could not create its build directory in the read-only sandbox; lint stopped with “parallel golangci-lint is running.” Live tests were not run because their build and temporary-file steps require filesystem writes.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R6 | met | One binding type and shared extractor; serialization round trip removed; route golden unchanged. |
| R8 | partial | Focused verification attempted but blocked as described above. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>