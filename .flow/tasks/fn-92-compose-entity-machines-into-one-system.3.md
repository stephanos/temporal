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
- `.plans/lean/LEAN_GUIDELINES.md` §5

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
Added `Umpire.Command.ComposeProofs`: tables read by catalog position, the composition function `stepsFrom`, the decided `composedTableAgrees members candidates literal`, and `ComposedAgreement.ofChecked`, which turns a `true` into the literal's soundness and completeness against the composition of the members' tables over every candidate action and every state its starts reach. `compose` now derives `Finite` on its Action union, generates a position view per composition, and declares `<name>.agrees` by `decide +kernel` through `elabComposedAgreement`; a literal the kernel refuses is a located error. `workerOutage` composes the unchanged Outage machine with the worker entity (six reachable states over four actions, nine rows) and `stoppedWorkerCompletesNothing` verifies that a completion leaves `(completed, polling)` on `veil default`.

Pins: `Umpire.Command.Tests.ComposeProofs` (positive and per-clause negative literals on a two-member lamp/switch example, an enabled action dropped whole, the command's refusal through `elabComposedAgreement`, axioms of the generic theorem `[propext, Classical.choice, Quot.sound]` and of the fixture pipeline's theorem, the fixture and gates candidates); `Outage/Tests.lean` (state and action catalogs, row count, the wait's single row, axioms of `workerOutage` and `workerOutage.agrees`, the Property's enumerated requirements, the Query outcome `verified-within-limits`); `SearchDifferential.lean` (`Temporal.Feature.Workflow.Outage.stoppedWorkerCompletesNothing: veil default, verified-within-limits 5 paths, verified-within-limits 5 states`).

Evidence: agreement check 78 ms kernel; law proof 75 ms elaboration + 76 ms kernel; predicate enumeration 4.3 ms; module elaboration 6.1 s; no `native_decide`, no proof-shape fallback needed. Outage fixture and goldens byte-identical (`umpire-check-case-runtime-conformance`, `umpire-check-goldens` rc=0); `lake build UmpireTests TemporalModelTests` rc=0; `LEAN_NUM_THREADS=1 make lint-model` rc=0.

Deviations: the compose elaborator lives in `Syntax.lean`, not `Compose.lean` as the task's file list says, so the emission edit is there (the prompt names `Syntax.lean` as the decision site). Round 1 of the Codex fan-out (three draws, all NEEDS_WORK on one finding) found that completeness quantified over the literal's own action catalog, so a dropped enabled action escaped; fixed in 6254f47e3f by reading the candidates off the generated Action domain (`candidatesOf ((members (α := Action)).map view.action)`), pinned, and captured to memory. Round 2 was SHIP.

baseline: green (focused lake build; goldens gates rc=0 pre-edit); GATE_SKIPPED:lint-model:green-receipt 05fae56a - baseline reused from prior post-gate pass

stage: impl-review - ran [round 1 NEEDS_WORK (codex fan-out, rid ad05e4f205974fa8a63de5b5891d1496) .. round 2 SHIP]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5b0257cf965c69aee5c36fdbda18292760fbb2cc, 6254f47e3fcd6405351d44be1d2a15a32cb3f219
- Tests: cd model && lake build Umpire.Command.Tests.ComposeProofs Temporal.Feature.Workflow.Outage.Tests TemporalModelTests.SearchDifferential, cd model && lake build UmpireTests TemporalModelTests, make umpire-check-case-runtime-conformance && make umpire-check-goldens, LEAN_NUM_THREADS=1 make lint-model, baseline: green (focused lake build rc=0; goldens gates rc=0 pre-edit); GATE_SKIPPED:lint-model:green-receipt 05fae56a - baseline reused from prior post-gate pass, timing (workerOutage, Lean 4.32.0, mise, profiler): agreement check decide +kernel 78 ms; canonical-table law proof 75 ms elaboration + 76 ms kernel; predicate enumeration of completedByPollingWorker 4.3 ms; Outage/Model.lean elaboration 6.1 s under lake (import 0.88 s, compose command interpretation ~0.5 s); fixture pipeline agreement check 230 ms (12 states, 7 actions); seam: Umpire.Command.Compose.ComposedAgreement.ofChecked over IndexedMember.ofModel / candidatesOf (Finite Action domain) / IndexedLiteral.ofModel, kernel decide, no native_decide, reachable count: workerOutage 6 states over 4 actions (24 evaluations, bound 16384), 9 rows
- PRs: