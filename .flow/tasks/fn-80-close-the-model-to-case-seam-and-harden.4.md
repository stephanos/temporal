---
satisfies: [R1]
---
# fn-80-close-the-model-to-case-seam-and-harden.4 Route the Nexus3 Producer through checked scoped lowering

## Description
Implements R1 (spec §R1). Replaces the hand-written Program-and-monitor Producer and its equality gate with `produce` over one `CheckedModel`, lowering the three success clauses through `Umpire.Case.Scoped.lower` with explicit coverage, and regenerates both fixture trees. Starts after fn-77.10 lands (shared fixture bytes and Nexus3 files).

**Size:** M
**Files:** `model/Temporal/Feature/Nexus3/Testpilot.lean`, `model/Temporal/Feature/Nexus3/Nexus.lean` (clauses as scoped forms if needed), `model/Temporal/Feature/Nexus3/Tests.lean`, `model/Temporal/Feature/Nexus3/Integration.md`, `model/Temporal/Tool/Testpilot.lean`, `tests/testcore/testpilot/testdata/async-nexus-case.json`, `common/testing/testpilot/testdata/case-runtime-conformance/**/expected.json`, `tests/testpilot_async_nexus_case_test.go` (verdict shape asserts), `tests/testcore/testpilot/artifact_test.go`
**Touches:** [model/Temporal/Feature/Nexus3/**, model/Temporal/Tool/Testpilot.lean, tests/testcore/testpilot/**, common/testing/testpilot/testdata/**, tests/testpilot_async_nexus_case_test.go]

### Approach
- Delete `successRule` (`Testpilot.lean:149-203`), `supportsSuccessProperty` (`:204-217`), and the equality gate (`:263-280`); invert the scoped-clause rejection at `:258-262`.
- New `produce (checked : Authoring.CheckedModel lifecycle)`: keep the Program realization as an Action-keyed map; build the `Projection.Checked` plan for the history-event Observation; call `Umpire.Case.Scoped.lower` passing coverage explicitly (`Scoped.lean:242-244` defaults to empty; never rely on the default); pass `Lowered.contractLowering` (`:309-319`) and a populated `coverage` into `Compiler.compile` (`Compiler.lean:60-127`).
- Express the three `require` clauses as scoped trigger and response predicates through `bounded_response%` (`Umpire/Property/Authoring.lean:93-97`; examples at `Property/Tests/TemporalAuthoring.lean:88-120`); `PropertyPredicate.resultingStateIs`, `selectedActionIs`, and `PropertyPattern.fact` already exist.
- Worked example of `Projection.check` + `Scoped.lower` on a real plan: `model/Umpire/Case/Tests/ScopedFixtures.lean:8-30`; field-operand lowering tests: `model/Umpire/Case/Tests/FieldLowering.lean`.
- Tests: rewrite the five `#guard rejected (produceWith ...)` at `Nexus3/Tests.lean:85-89`: witness-absent stays a rejection; changed Property/Behavior/Query/witness now produce different bytes (assert inequality of the canonical JSON); add a Known-Gap-cannot-waive guard and an emptied-coverage rejection guard; restate `correlationShape` (`:99-119`) as structural invariants rather than fixed counts.
- Regenerate with `make umpire-gen-case-runtime-conformance`; verify with `make umpire-check-case-runtime-conformance` (diffs both trees, `Makefile:1072-1088`). Update `tests/testpilot_async_nexus_case_test.go` rule/supporting-sequence assertions to the scoped Verdict shape.
- Audit axioms on every changed trust-bearing declaration with `#print axioms` (LEAN_GUIDELINES §5).

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus3/Testpilot.lean:90-305` — everything being replaced
- `model/Umpire/Case/Scoped.lean:227-360` — `Lowered`, `lower`, `contractLowering`, theorems to preserve
- `model/Umpire/Case/Compiler.lean:43-127` — `LoweringError`, `ContractLowering`, `Input.coverage`, `compile`
- `model/Umpire/Case/Coverage.lean` and `model/Umpire/Observation/Projection/Coverage.lean` — fn-77.8 coverage admission

**Optional** (reference as needed):
- `model/Umpire/Case/Tests/ScopedFixtures.lean` — worked scoped lowering
- `.plans/LEAN_GUIDELINES.md:182-238` — trust boundary and incremental verification

### Key context
- The Case declares no classic horizon after this task; liveness comes from the scoped operation-transition clock.
- fn-77 R8 asserts the same "one edit, one derived Contract" claim; this task is the load-bearing half for the shipped Case.

## Acceptance
- [ ] `Nexus3/Testpilot.lean` contains no monitor rule constructor and no equality comparison of checked values; `produce` is the renderer entry point in `Temporal/Tool/Testpilot.lean`
- [ ] Regenerated `async-nexus-case.json` carries `contract.scoped`; `make umpire-check-case-runtime-conformance` passes on both trees
- [ ] Live `TestTestpilotAsyncNexusCase` (integration tag) is satisfied in both environments with the scoped Verdict shape; `make umpire-check-live-tests` passes
- [ ] `#guard`s: witness-absent rejects; unexpressible clause rejects naming it; Known Gap does not admit it; emptied coverage rejects at `compile`; changed Property produces different Contract bytes
- [ ] Editing one `require` clause and regenerating changes no Lean file under `model/Temporal/Feature/Nexus3/` other than `Nexus.lean` (documented in the task receipt)
- [ ] `#print axioms` on changed declarations matches the approved baseline; `make lint-model` passes

## Done summary
Blocked:
Serialize behind fn-77-typed-operations-parameterized-actions.10: it modifies the Nexus3 Producer, the async-nexus fixture bytes, and the live test. Unblock when fn-77.10 is done.

Blocked:
BLOCKED: DEPENDENCY_BLOCKED — R1's scoped route has no runtime evidence path.

R1 requires the shipped async-nexus Case to (a) carry `contract.scoped`, (b) drop the
hand-written correlated-history monitor rule, and (c) still be **satisfied live**. Those three
cannot hold together today, and forcing them through would turn a real regression test into a
vacuous one.

Evidence gathered this session:

1. `bindScoped` (`common/testing/testpilot/internal/verification/scoped_prepare.go:71-73`)
   rejects any Case whose `Contract.scoped` names an `evidence_observation_id` that is not a
   Program Observation typed **exactly** `temporal.server.api.testpilot.v1.ScopedEvidence`
   (`PreparationTypeMismatch`, "scoped evidence requires exact declared ScopedEvidence
   Observation"). The async-nexus Program declares only `history-event`
   (`temporal.api.history.v1.HistoryEvent`), so the regenerated Case would not even `Prepare`.

2. An Observation is filled only by an `InvokeRPC` response projection
   (`PROJECTION_KIND_ONE` / `PROJECTION_KIND_EMIT_EACH` are the only two kinds), and the
   version-one `Instruction` table (`instruction.proto:60-70`) has no instruction that can
   produce a `ScopedEvidence` value. `run.proto:125` states the rule outright: "ScopedEvidence
   is supplied only through the capability's declared typed Observation." No Temporal
   WorkflowService RPC returns that message.

3. Consequently a scoped clause on this Program receives zero evidence, and
   `scopedRun.answer` (`internal/verification/scoped.go:622-642`) returns **SATISFIED**
   vacuously for a clause with no operations and no accepted evidence. Shipping that would
   delete the correlated-history rule — the only thing in the Case that today proves the Nexus
   operation actually scheduled, started and completed — and replace it with a clause that is
   green because it never ran.

4. fn-77 already banked this exact conclusion in-tree, as the Known Gap
   `temporal.nexus3.typed-nexus.bounded-completion-is-model-only`
   (`model/Temporal/Feature/Nexus3/TypedNexus.lean:759-768`): "the Driver reads a scoped
   capability only from declared `ScopedEvidence` Observations and no instruction of this
   Program emits one."

Finding on the question the spec asked to settle first (the clause form):

**The scoped clause form CAN carry state, outcome and fact predicates.**
`Umpire.Case.Scoped.pattern` (`model/Umpire/Case/Scoped.lean:44-56`) maps
`.resultingState → SCOPED_PREDICATE_FIELD_RESULTING_STATE`,
`.modelOutcome → SCOPED_PREDICATE_FIELD_OUTCOME`,
`.observation → SCOPED_PREDICATE_FIELD_FACT` and
`.selectedAction → SCOPED_PREDICATE_FIELD_ACTION`, so the three Nexus3 `require` clauses are
expressible as three `bounded_response%` clauses sharing an `awaitSuccess` trigger. The blocker
is the evidence path, not the clause form — so R2 (task .5) and R8 (task .6), which generalize
the macros over the same `Authoring.check` owners, are unaffected and stay startable.

Suggested resolution: land the follow-up fn-77's completion review already named — a
`ScopedEvidence`-emitting projection (a Program-declared source that lifts recorded Nexus
history into `ScopedEvidence` with `identity`, `operation`, `kind` and `fields`) — then re-open
this task. Until then R1's own Goal-§1 defect (the clause-for-clause equality gate in
`produceCompletionCase` that turns a model edit into a lowering error) is still worth closing on
its own, but that is a smaller task than this one's acceptance and should be re-planned rather
than silently substituted here.

Impact: tasks .5, .7 and .8 carried a `depends_on` edge to this task only for file-overlap and
live-assertion-churn reasons that no longer exist; the edges were dropped so the rest of the
spec can proceed. Task .9's R1-related doc bullets will need the same re-planning.
## Evidence
- Commits:
- Tests:
- PRs:
