I’ll read the task spec first, then inspect the diff and trace its integration with the existing command code.
The new field clauses use the state-field values already exposed to both evaluators, and the query paths include the planned instance and refinement refusals. I’m checking sparse reachable-state catalogs next, where a field change may have no matching catalog state.
The focused build passed. I reproduced an integration defect in the new refinement fixture: a property written directly on `crankMachine` that fixes `phase = down` is refused as `fixesNothing`. The field-mutation check holds the generated `leverMachine` projection constant, even though changing `phase` must change that projection too.

`lint-model` stopped at the inventory check because the read-only sandbox prevented it from creating a temporary file.
- **Severity:** P2
- **Confidence:** 75
- **Classification:** introduced
- **File:Line:** `model/Umpire/Command/Predicate.lean:108`
- **R-IDs:** [R15]
- **Problem:** The comparison `candidate == held.map (...)` requires every other exposed field to remain unchanged, including derived refinement projections. Existing machine elaboration adds the abstract state as a synthetic field. Consequently, changing a concrete field also changes that projection, and no mutation matches. Reproduced using this change’s `crankMachine` fixture:

  ```lean
  property directCrankField
    machine: crankMachine
    when: pull
    holds: fun step => step.state.phase == .down
  ```

  This incorrectly fails with `fixesNothing`: `phase` cannot change while the synthetic `leverMachine` field remains constant. This is a property on the refining machine itself, distinct from the intentionally unsupported lifting of an abstract machine’s field claim.
- **Suggestion:** Distinguish structural fields from derived projections when constructing field mutations. Allow projections to change consistently with the concrete state, and add the direct refining-machine property as a regression test.

The four focused build targets passed. `LEAN_NUM_THREADS=1 make lint-model` stopped because the read-only sandbox prevented the inventory check from creating a temporary file.

## Requirements coverage

Coverage is scoped to task .7’s assigned requirements.

| R-ID | Status | Evidence |
|---|---|---|
| R7 | partial | Classed field-claim fixture verifies; shipped caller integration belongs to task .5. |
| R14 | met | New `repliesRunning` Query appears in the passing differential block. |
| R15 | partial | Field lowering and evaluator integration pass; direct claims on refining machines fail as described above. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>