I’ll read the task spec first, then review the diff and affected contracts directly, focusing on agreement between the schema, docs, tests, and runtime behavior.

The commit removes only `Reference.model_value`. It retains the other two candidate arms because Lean producers emit them, as R9 requires. I’m checking the regenerated schema, retired-name checks, and pinned Run identities for consistency.

The schema, generated Go, Lean renamer, and test tables agree on the removal. Both pinned Runs contain fresh run IDs and event timestamps, and their catalog identity matches the updated golden and receipts.

The focused Go tests could not start because the read-only sandbox prevents Go from creating its build directory. The verdict will therefore rely on static review, with that verification limit noted.

No blocking findings in the reviewed changes.

The removal is consistent across the schema, generated Go, Lean renamer, tests, and documentation. Retaining `evidence_field_id` and `correlated_capture` follows R9 because Lean producers emit both.

Focused Go tests could not start: the read-only sandbox denied creation of Go’s build directory. Lean and live-cluster gates were not rerun.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R9 | met | Removed arm has no remaining consumers; retirement checks and removal checklist added. |
| R11 | met | Catalog golden, both pinned Runs, and dependent receipts updated together; Runs contain fresh execution IDs and timestamps. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>