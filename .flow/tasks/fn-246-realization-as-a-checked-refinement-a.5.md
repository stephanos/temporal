---
satisfies: [R6]
---
# fn-246-realization-as-a-checked-refinement-a.5 Worker strategies enumerated from the Model replace hand-named scenarios; program rows derived from strategies and fillers

## Description
After .3. Enumerate deterministic worker strategies by exploring the activity Model's worker choices within the admitted World; derive canonical, stable scenario names; produce program rows as (strategy, task key) filled by the action's filler; stamp the derived name into the `umpire-scenario` header. Remove `SCENARIOS` and `response()`; the coverage test becomes a construction. fn-243's live slice and controls (mutant executors, empty program, replay) pass with derived names; wire goldens for `ResetResponse.program` change once, listed. Unbounded worker choices refuse enumeration naming the state. `WorkerProgram` and the executors are unchanged.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
