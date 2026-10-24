I’ll read the task specification first, then trace the changed fairness code and its focused tests against the acceptance criteria.
Both backends select the conditional scope before checking exact catalog membership. I’m now checking helper binding, generated names, and the new metadata comparison for cases the fixture may miss.
Cargo cannot run because the read-only sandbox blocks its build lock. Cached test binaries are available; I’ll use those for focused checks and note that limitation in the verdict.
The cached focused tests passed: 6 frontend tests, 9 binding tests, and the fixture checks. The two required-TLC tests returned early because `UMPIRE_BACKENDS=require` was unset. I’ve traced the changed paths without finding a blocking correctness issue; fresh TLC results remain supported by the committed evidence only.
No surviving correctness findings.

Cached focused tests passed: 6 frontend, 9 binding, and the fixture checks. Cargo rebuilding was blocked by the read-only build lock. Fresh required-TLC checks were not run; their results were assessed from committed evidence.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R2 | met | Task .6’s fairness prerequisite: exhaustive guards, typed selection before membership, conditional name reservation, exact metadata agreement, rebinding, and mutation controls. Activity policy remains assigned to subsequent tasks. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge (no blocking introduced findings; task-scoped requirements met).