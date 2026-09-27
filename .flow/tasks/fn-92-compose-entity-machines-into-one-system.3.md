---
satisfies: [R5, R7, R8]
---
# fn-92-compose-entity-machines-into-one-system.3 Agreement theorem, per-composition kernel check, and the workerOutage proof point

## Description
Prove that `composedTableAgrees members literal = true` implies soundness and completeness over reachable sources, make `compose` emit that check per composition by `decide +kernel`, then declare the `workerOutage` composition over the unchanged Outage machine and the worker machine as the proof point: bound, kernel time, and the stopped-worker claim. This is the stop/re-plan gate.

**Size:** M
**Files:** `model/Umpire/Command/ComposeProofs.lean` (new), `model/Umpire/Command/Compose.lean` (emit the check), `model/Umpire/Command/Tests/ComposeProofs.lean` (new), `model/Umpire/Command/Tests.lean`, `model/Temporal/Feature/Workflow/Outage/Model.lean` (add `compose workerOutage`, its Property, Scenario, Limits, `verify` Query), `model/Temporal/Feature/Workflow/Outage/Tests.lean`
**Touches:** [model/Umpire/Command/ComposeProofs.lean, model/Umpire/Command/Compose.lean, model/Umpire/Command/Tests/ComposeProofs.lean, model/Umpire/Command/Tests.lean, model/Temporal/Feature/Workflow/Outage/**]

### Approach
- Define `composedTableAgrees` as a decidable check that every literal row is a composition-function row from a reachable source and every such row is in the literal; prove the generic implication to soundness and completeness following `CheckedTable`'s closure proofs (`Table.lean:93-120`); pin axioms.
- In `Compose.lean`, emit `by decide +kernel` on the check for each composition, as the refinement decision does (`ImplementationLink/Refinement.lean:251,274`); a failing check is a located error.
- `workerOutage`: members `workflow: workflowOutage`, `worker: polling`; `sync: workerStop: workflow.workerStop ∥ worker.workerStop`, `workerResume` likewise, `awaitCompletion: workflow.awaitCompletion ∥ worker.serve`; `starts:` and `ends:` over the structure; Property: no `awaitCompletion` step while `worker.phase == stopped`; `verify` Query with Limits that complete on `reference`; pin the reachable count and the outcome. The Outage functional set and Case are untouched.
- Record in evidence: reachable count, kernel seconds of the agreement check and the law proof, elaboration seconds of the module. If the checks exceed budget, stop and report; `native_decide` only with the policy approval `LEAN_GUIDELINES.md` §5 requires.

### Investigation targets
**Required:**
- `model/Umpire/Model/Table.lean:53-120, 426-560, 703`; `model/Umpire/Core.lean:367-390`
- `model/Umpire/ImplementationLink/Refinement.lean:251,274`; `model/Umpire/Command/Syntax.lean:2379-2387, 2411-2457`
- `model/Temporal/Feature/Workflow/Outage/Model.lean`
- `.plans/LEAN_GUIDELINES.md` §5

### Key context
- The theorems are about the literal the command emitted; the kernel reduces a value, not the BFS.

## Acceptance
- [ ] Generic agreement theorem with axiom pins passing; `compose` emits the per-composition check and refuses a failing one (negative test pinned)
- [ ] `workerOutage` elaborates; reachable count pinned and below the bound; stopped-worker `verify` Query pinned
- [ ] Kernel and elaboration seconds recorded in task evidence with the seam used
- [ ] Outage fixture and goldens byte-identical; `make lint-model` passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
