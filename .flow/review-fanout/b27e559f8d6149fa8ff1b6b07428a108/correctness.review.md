I’ll read the task specification first, then review the diff and relevant callers for correctness and edge cases.
Both resolvers now map member-qualified references through the recorded sync groups before checking the action catalog. I’m checking that class arguments, bare references, and ambiguity errors still behave correctly, then I’ll attempt the task’s verification commands.
I haven’t found a correctness issue in the changed paths. Verification is incomplete: lint failed on sandbox-denied writes, and I stopped the build after it produced no output. I’ll try compiling the focused test file directly against the existing dependencies.
No blocking findings in the changed paths.

The focused Compose test file compiled successfully against existing dependencies. Full build verification remains incomplete; lint failed on sandbox-denied writes, and goldens were not rerun.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R3 | met | Both resolvers share sync-aware resolution; regression pins cover dotted Property and Scenario references, unknown actions, and ambiguity. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>