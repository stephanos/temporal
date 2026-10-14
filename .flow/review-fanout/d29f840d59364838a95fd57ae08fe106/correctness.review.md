I’ll read the task spec first, then review the diff and affected callers for correctness and spec coverage.
The diff removes only `Reference.model_value`. Keeping the other two candidate arms is justified: Lean producers still emit them. I’m checking the pinned Run changes next, since the schema change also changes the Driver catalog identity.
No blocking correctness findings.

Focused Go tests could not start because the read-only sandbox prevents creating Go’s build directory. `git diff --check` passed.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R9 | met | Unused arm removed from schema, generated Go, Lean renamer and test tables; retirement gates and removal checklist added. Retained arms have active Lean producers. |
| R11 | met | Catalog golden, both pinned Runs and receipt goldens updated consistently; records contain fresh execution IDs and timestamps. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>