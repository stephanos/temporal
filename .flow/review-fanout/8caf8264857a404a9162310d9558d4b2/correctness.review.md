I’ll read the task specification, review the commit range, and check the affected replay and selection paths against the acceptance criteria.
The replay gate shares the reference path’s endpoint evaluation and checks setup, initial-state and transition membership. Selection has a tested `cutover = false` switch pending the later golden-update task. I’ll now check the finalization edge cases and run the focused checks available here.
The replay tests pass with the installed Lean 4.32 toolchain. I found a possible edge case where replay rejects a witness but finalization replaces the error with `unsatisfiable`; I’m running a small reproduction to confirm it.
The reproduction confirms the bug: a faulty witness for a statically unsatisfiable Scenario returns `unsatisfiable`, losing the required `invalid/unreplayableWitness` diagnostic. Direct Lean checks passed for Search, Selection, Admission, Replay and visibility. The lint check stopped because its subprocess tried to use an unavailable elan toolchain.
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/Umpire/Search.lean:1358`
- **R-IDs**: [R10]
- **Problem**: Replay failures lose their diagnostic when `query.behavior.isUnsatisfiable` is true. The new gate produces `.invalid unreplayableWitness`, but downstream `finalizePlanning` replaces it with `.unsatisfiable`. Reproduced using `fixtureQuery` with `spaceStatus := .unsatisfiable` and a faulty `.violationFound` result: finalization returns `some "unsatisfiable"` instead of the required invalid outcome.
- **Suggestion**: Preserve replay rejection through finalization, retaining the error and rendered witness while keeping ordinary unsatisfiable-result handling unchanged. Add this case to the negative-control tests.

Direct Lean checks passed for Search, Selection, Admission, Replay, visibility, and outcome tests. Lint could not complete because its subprocess attempted to use an unavailable elan toolchain.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R10 | partial | Replay and negative controls implemented; unsatisfiable case loses the error. |
| R12 | met | Public-surface checks and 1,426-line pin pass. |
| R16 | met | Seeded fallback tested; completeness and admitted-endpoint gates retained. |

Unaddressed R-IDs: [R10]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R10"]}
```

<verdict>NEEDS_WORK</verdict>