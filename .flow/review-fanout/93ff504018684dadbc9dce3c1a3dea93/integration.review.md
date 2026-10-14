I’ll read the task spec first, then trace the diff’s integration with the existing Lean modules and test runner.

The new imports reach both validation hooks. The added schema root covers the existing workflow-start actions, and the negative fixture tests both checks through `Temporal.Case.Syntax` alone. I’m checking the remaining consumers and focused build for regressions.

I haven’t found an introduced integration defect. The focused Lake build produced no output and was interrupted; I’m trying the fixture directly against the existing compiled dependencies. Golden and lint checks require writable output paths that this read-only session does not provide.

No surviving findings. The imports install both checks, and the added schema root admits the existing workflow-start actions.

Direct Lean checks passed for `ProductionImports.lean` and `TemporalModelTests.lean` using existing compiled dependencies. Fresh build, golden, and lint verification remain unconfirmed in this read-only session.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Both hooks imported through Syntax; negative fixture passes and is wired into the test aggregator. |

Coverage is scoped to this task’s assigned requirement.

Unaddressed R-IDs: []
Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>