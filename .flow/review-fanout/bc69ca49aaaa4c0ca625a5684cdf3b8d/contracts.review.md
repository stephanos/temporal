I’ll read the task specification first, then review the diff and focused checks for agreement between the implementation, tests, and stated contracts.
The agreement theorem still connects the generated literals to the original tables through kernel-checked equalities. I’m checking whether grouping changes any accepted behavior and whether the lint mirror isolates the build artifacts as its comment promises.
The grouping preserves the row order emitted by the composition walk, and the existing negative fixtures cover missing rows, invalid results, unreachable states, and invalid starts. Lake applies the lint options only to the root package, so sharing dependency packages does not defeat the new build isolation. The focused Lean checks are still running.
The performance commit records the full-size result: 316 states and 1,468 rows, about 60 seconds of kernel time, and 6.6 GB peak memory. That meets the task’s stated limit. I haven’t found an introduced contract regression; I’m finishing the focused verification before issuing the verdict.
No introduced findings.

Focused Lean checks passed: `Compose`, `ComposeProofs`, and `Workflow.Outage.Tests`. The theorem retains kernel-checked equality bridges. Commit `7c2392c51d` records full-size performance at approximately 60 seconds and 6.6 GB; I did not independently repeat that benchmark.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R7 | partial | Scaling prerequisite implemented and focused tests pass. The `nexusCaller` Query remains assigned to task .5, explicitly outside this task. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>