I’ll read the task spec first, then trace the diff and focused tests against its replay and selection contracts.
The replay checks share the reference search’s endpoint evaluation, and the existing completeness gates remain in place. I’m checking how replay failures reach the final outcome and whether the new tests cover those paths.
The focused Search build and tests pass. I reproduced one contract failure: for an unsatisfiable Scenario, a faulty backend’s witness fails replay, but finalization replaces the required `invalid/unreplayableWitness` result with `unsatisfiable`, losing the diagnostic.
- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/Umpire/Search.lean:1358`
- **R-IDs:** [R10]
- **Problem:** Replay failures lose their diagnostic when the Scenario is already unsatisfiable. The new replay gate produces `.invalid .unreplayableWitness`, but downstream `finalizePlanning` overrides it with `.unsatisfiable`. Reproduced using the fixture Query with `spaceStatus := .unsatisfiable` and a faulty `.violationFound` result: finalization returns `"unsatisfiable"`, contrary to R10’s required `invalid` outcome and offending trace.
- **Suggestion:** Preserve replay failures through finalization while retaining the existing unsatisfiability rules for successful replay and absence results. Add this case to the negative-control tests.

Focused Search builds and tests passed. Golden comparison and lint were blocked by temporary-file creation in the read-only sandbox.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R10 | partial | Replay and negative controls implemented; unsatisfiable Scenarios suppress replay errors. |
| R12 | met | Reference traversal preserved; public-surface and line-count pins pass. |
| R16 | met | Seeded fallback records its reason; completeness and admitted-endpoint gates preserved. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>