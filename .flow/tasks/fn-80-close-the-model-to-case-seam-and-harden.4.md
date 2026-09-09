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
- [ ] Live `TestTestpilotAsyncNexusCase` (integration tag) is satisfied in both environments with the scoped Verdict shape; `umpire-check-live-tests` expected-failure list unchanged
- [ ] `#guard`s: witness-absent rejects; unexpressible clause rejects naming it; Known Gap does not admit it; emptied coverage rejects at `compile`; changed Property produces different Contract bytes
- [ ] Editing one `require` clause and regenerating changes no Lean file under `model/Temporal/Feature/Nexus3/` other than `Nexus.lean` (documented in the task receipt)
- [ ] `#print axioms` on changed declarations matches the approved baseline; `make lint-model` passes

## Done summary
Blocked:
Serialize behind fn-77-typed-operations-parameterized-actions.10: it modifies the Nexus3 Producer, the async-nexus fixture bytes, and the live test. Unblock when fn-77.10 is done.
## Evidence
- Commits:
- Tests:
- PRs:
