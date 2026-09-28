---
satisfies: [R5, R7, R8, R14]
---
# fn-92-compose-entity-machines-into-one-system.3 Agreement theorem, per-composition kernel check, and the workerOutage proof point

## Description
Prove that `composedTableAgrees members literal = true` implies soundness and completeness over reachable sources, make `compose` emit that check per composition by `decide +kernel`, then declare the `workerOutage` composition over the unchanged Outage machine and the worker machine as the proof point: bound, kernel time, backend, and the stopped-worker claim (R5, R7, R8, R14). This is the stop/re-plan gate.

**Size:** M
**Files:** `model/Umpire/Command/ComposeProofs.lean` (new), `model/Umpire/Command/Compose.lean` (emit the check), `model/Umpire/Command/Tests/ComposeProofs.lean` (new), `model/UmpireTests.lean` (import), `model/Temporal/Feature/Workflow/Outage/Model.lean` (add `compose workerOutage`, its Property, Scenario, Limits, `verify` Query), `model/Temporal/Feature/Workflow/Outage/Tests.lean`, `model/TemporalModelTests/SearchDifferential.lean` (expected line; the Outage module is already imported at :9)
**Touches:** [model/Umpire/Command/ComposeProofs.lean, model/Umpire/Command/Compose.lean, model/Umpire/Command/Tests/ComposeProofs.lean, model/UmpireTests.lean, model/Temporal/Feature/Workflow/Outage/**, model/TemporalModelTests/SearchDifferential.lean]

### Approach
- Define `composedTableAgrees` as a decidable `Bool` over the literal, comparing constructor indices as `Nat` and never sorting inside the proof: every literal row is a composition-function row from a reachable source and every such row is in the literal; prove the generic implication to soundness and completeness following `CheckedTable`'s closure fields (`Table.lean:93-119`) and the `machine_*_mem` theorems (`:514-548`); pin axioms as `Caller/Tests.lean:112-114` does.
- In `Compose.lean`, emit `by decide +kernel` on the check for each composition inside `elabGenerated` (`Syntax.lean:1803`), as the refinement decision at `Syntax.lean:2459` does, with the `sorryAx` check of `:2409-2417`; a failing check is a located error pinned with `substring := true` (`Search/VisibilityTests.lean:54` shape).
- `workerOutage`: members `workflow: workflowOutage`, `worker: polling`; `sync: workerStop: workflow.workerStop ∥ worker.workerStop`, `workerResume` likewise, `awaitCompletion: workflow.awaitCompletion ∥ worker.serve`; `starts:` and `ends:` over member-qualified values; Property `when: awaitCompletion` with `holds` naming the unique whole composed state a completion leaves, `step.state == { workflow := .completed, worker := .polling } && step.facts.contains .workflowExecutionCompleted` (a whole-state requirement the enumeration fixes today, `Predicate.lean:91-135`, so this task does not wait on .7's field requirement; no `branches` or guarded form, so it lowers to a monitor); a `scenario` declaration and `limits` sized so both backends answer verified within limits; `verify` Query; pin the reachable count, the outcome, and the differential line `Temporal.Feature.Workflow.Outage.<query>: veil default, …` in `SearchDifferential.lean:30-104`. The Outage functional set, Property `completes`, and Case are untouched.
- Record in evidence: reachable count, kernel seconds of the agreement check and the law proof, predicate-enumeration seconds, elaboration seconds of the module. If the checks exceed budget, first change the proof shape; stop and report before `native_decide`, which needs the policy approval `LEAN_GUIDELINES.md` §5 requires and mints a per-call axiom the R5 pin rejects.

### Investigation targets
**Required:**
- `model/Umpire/Model/Table.lean:93-119, 426-548, 703-727`; `model/Umpire/Core.lean:367-387`
- `model/Umpire/Command/Syntax.lean:1803, 2409-2465`; `model/Umpire/ImplementationLink/Refinement.lean:251, 274` (theorem shape)
- `model/Temporal/Feature/Workflow/Outage/Model.lean:87-135`; `Outage/Tests.lean`
- `model/Umpire/Search/Product/Monitor.lean:69-102`; `model/TemporalModelTests/SearchDifferential.lean:1-104`
- `.plans/LEAN_GUIDELINES.md` §5

### Key context
- The theorems are about the literal the command emitted; the kernel reduces a value, not the BFS.
- The kernel honours `maxHeartbeats` and ignores `maxRecDepth`; WF-recursive `Decidable` instances get stuck under `+kernel`.
- Pins are taken with `Selection.cutover` true, after fn-88 closes.

### Quick commands
```bash
cd model && lake build Umpire.Command.Tests.ComposeProofs Temporal.Feature.Workflow.Outage.Tests TemporalModelTests.SearchDifferential
make umpire-check-case-runtime-conformance && make umpire-check-goldens
LEAN_NUM_THREADS=1 make lint-model
```
## Acceptance
- [ ] Generic agreement theorem with axiom pins passing; `compose` emits the per-composition check and refuses a failing one (negative test pinned)
- [ ] `workerOutage` elaborates; reachable count pinned and below the bound; stopped-worker `verify` Query pinned; its Temporal differential line reads `veil default` with both backends verified within limits
- [ ] Kernel, predicate-enumeration, and elaboration seconds recorded in task evidence with the seam used
- [ ] Outage fixture and goldens byte-identical; `make lint-model` passes
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
