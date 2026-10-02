I’ll read the task spec first, then trace the diff against its contracts and focused test coverage.
The shared helpers preserve the three callers’ rejection paths, details, and check order. The new table test covers each call shape, including skipped fields and multiple invalid fields. Go tests could not start because the read-only sandbox prevents creating a build directory; I’m checking the remaining gates and test coverage.
No blocking findings. The extraction preserves nil handling, identity validation, path truncation, and ceiling rejection order. Tests cover all three ceiling call shapes.

Verification could not complete: Go and conformance builds hit read-only sandbox restrictions; lint stopped because another golangci-lint process was running.

## Requirements coverage

Coverage is scoped to task `.7`; other tasks’ requirements remain deferred from this review.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Owned by tasks `.1`, `.3`. |
| R2 | deferred | Owned by tasks `.4`, `.5`, `.17`. |
| R3 | met | Four shared primitives extracted; targeted callers migrated; three-shape table test added. |
| R4 | deferred | Owned by tasks `.6`, `.9`, `.10`. |
| R5 | deferred | Owned by tasks `.11`, `.12`. |
| R6 | deferred | Owned by task `.13`. |
| R7 | deferred | Owned by tasks `.14`, `.15`. |
| R8 | partial | No fixture changes; verification blocked as described above. |
| R9 | deferred | Owned by task `.16`. |
| R10 | deferred | Owned by tasks `.2`, `.17`. |
| R11 | deferred | Owned by tasks `.2`, `.16`; no golden changes here. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>