I’ll read the task specification first, then review the diff and relevant code paths for correctness and spec coverage.
The agreement theorem still ties the compiler-evaluated literals back to the original tables through kernel-checked equalities. I’m checking whether grouping rows and changing the state reader preserve behavior for all supported compositions, then I’ll run the focused tests available in this read-only environment.
The grouped checker retains the soundness and completeness proof, and its flattening check matches the row order emitted by the composition walker. I haven’t found a concrete regression so far. The focused Lean test is still running; I haven’t located timing evidence for the full-sized composition, so that performance target remains unverified.
The focused `ComposeProofs` test passed with Lean invoked directly against the existing build artifacts. That covers the rejection cases and axiom checks. I’m now re-elaborating the composition fixtures and `workerOutage` to check the generated state reader and agreement theorem.
`workerOutage` re-elaborated successfully, and the original `lake env lean` proof test also passed. The broader composition fixture exceeded my 60-second limit without diagnostics. I’m checking a full-sized caller composition from standard input, including all protocol actions and four timers, to measure the task’s main performance target without changing files.
The full-sized probe is still running and hasn’t produced diagnostics yet. Code review is complete with no blocking finding so far; the remaining uncertainty is performance at the caller’s full size.
The full-sized probe passed: 316 states and 1,468 rows, about 129 seconds of profiled type checking, 161 seconds total, and 4.66 GB peak memory. Its agreement theorem uses only `propext`, `Classical.choice`, and `Quot.sound`. I’m giving the composition fixture a longer limit to resolve its earlier timeout.
No blocking findings.

Verified `ComposeProofs`, composition fixtures, and `workerOutage` elaboration. A full-sized caller composition passed with **316 states, 1,468 rows, 129 seconds of profiled type checking, and 4.66 GB peak memory**. Its agreement theorem retains only the permitted axioms.

The intermediate benchmarks and `workerOutage` timing comparison were not repeated.

## Requirements coverage

Coverage is scoped to task .8; other epic deliverables remain assigned to their respective tasks.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Task .1 |
| R2 | deferred | Task .1 |
| R3 | met | Composition fixtures passed |
| R4 | met | Grouping preserves the walk’s emitted state and row order |
| R5 | partial | Kernel trust and full-sized performance verified; remaining timing comparisons unverified |
| R6 | deferred | Tasks .4–.5 |
| R7 | partial | `workerOutage` elaborates; shipped caller Query explicitly belongs to .5 |
| R8 | deferred | Tasks .3 and .5 |
| R9 | met | No protected fixtures or goldens changed |
| R10 | deferred | Tasks .4–.5 |
| R11 | met | Composition rejection fixtures passed |
| R12 | deferred | Task .6 |
| R13 | deferred | Task .6 |
| R14 | deferred | Differential coverage belongs to companion tasks |
| R15 | deferred | Task .7 |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>