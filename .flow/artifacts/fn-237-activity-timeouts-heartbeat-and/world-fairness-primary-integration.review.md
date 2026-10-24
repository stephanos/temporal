I’ll read the task specification first, then trace the changed code across its callers and backends for integration issues.

The implementation retains conditional branches for rebinding and selects a scope before checking exact catalog membership. I’m checking name reservation and the new native-to-TLA metadata comparison against existing emitters and progress overrides.

Cargo cannot acquire its build lock in this read-only environment. I found existing focused test binaries and will try those while continuing the source review; I won’t claim a fresh build.

The existing binaries passed 6 front-end tests, 9 binding tests, and the fixture’s 7 tests in default mode. The two required-TLC fixture tests returned early because `UMPIRE_BACKENDS=require` was unset. I’m finishing the review of empty scopes, duplicate catalog entries, and rebinding.

No blocking findings in the reviewed range. Selection, rebinding, name reservation, catalog checks, and metadata comparison integrate consistently with existing consumers.

Existing focused binaries passed 6 front-end, 9 binding, and 7 fixture tests. Cargo rebuilding was blocked by the read-only filesystem; required-TLC branches were not rerun.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R2 | met | Task .6’s fairness prerequisite: conditional admission, typed World selection, independent binding, metadata checks, and mutation controls. Activity integration remains assigned to subsequent tasks. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>