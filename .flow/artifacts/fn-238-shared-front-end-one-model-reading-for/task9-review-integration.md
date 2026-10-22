I’ll read the task specification first, then trace the changed code through its callers and check the focused validation evidence.
The derive preserves the existing constructor paths and field order, and the unchanged DSL re-exports still resolve those paths. I’m checking the shared scalar/container printer and the golden comparisons, then rerunning the task’s Quick command.
No surviving findings. The derived printer preserves constructor paths, field order, scalar handling, and integration with the unchanged DSL and lifter.

The Quick command was blocked by the read-only Cargo build lock; existing Quick test binaries passed. Independently verified 650 golden hashes and the retained IR dump checksum. Committed evidence records passing expansion, required-TLC, guest slice, and gaps checks.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Existing `model/design/shared-front-end.md` records the comparison and recommendation. |
| R2 | met | Printing is derived from IR declarations; the added-field/variant control passes; shared front-end consumers remain compatible. |
| R3 | met | No golden changes; constructor equivalence, IR comparison, and TLC evidence retained. |
| R4 | met | DESIGN and MILESTONES describe the boundary; source/test deltas and timing are recorded. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>