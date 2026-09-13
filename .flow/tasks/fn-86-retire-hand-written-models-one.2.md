---
satisfies: [R2, R3]
---
# fn-86-retire-hand-written-models-one.2 Field relations in the property command; the typed unary example migrated (early proof point)

## Description
Add the field-relation clause to `property` (R2): `require:` compares an action's input field, an action's result field or an observation's field through their schemas, checked while the file compiles and lowered onto the existing `PropertyFieldPath` and Contract field reads. Prove it on the typed unary example (R3, the smaller half): re-author it as a Model file with the commands, a field relation and an RPC-backed realization, and show its Case lowers to the same Contract reads as the task .1 baseline. Stop if the typed unary Property cannot be expressed as a field relation that lowers to the same reads.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (`modelRequirement` gains the field-relation forms: `<path> = <path>`, `<path> ≠ <path>`, `<path> present`), `model/Umpire/Command/Authoring.lean` (`authoredProperty` lowers a field relation to `PropertyFieldComparison` over `PropertyFieldPath`; schema resolution through the realization's catalog for observations), `model/Temporal/Case/Schema.lean` (field-path resolution against a message schema; repeated fields select through the row entity's key; optional submessages present only when every segment is set), `model/Temporal/Feature/Workflow/Start/Model.lean` (new; the typed unary Model: `startWorkflow` action with `schema: temporal.api.workflowservice.v1.StartWorkflowExecutionRequest`, `workflowExecutionStarted` observation, the `submittedTypeIsRecorded` relation), `model/Temporal/Case/Realization/Workflow.lean` (RPC-backed action binding), `model/Temporal/Feature/Workflow/Start/Tests.lean` (`#guard_msgs` for the four R2 rejections; the crossed-pairing mutation as a realization mutation yielding `some false`), `tests/testcore/testpilot/testdata/<set>-<query>-case.json` (replaces `typed-unary-case.json`), `tests/testcore/testpilot/typed_unary_artifact_test.go` and `tests/testpilot_typed_unary_case_test.go` (re-pointed), `model/Temporal/Feature/Nexus/Success/TypedUnary.lean` and `Success/Tests/TypedUnary.lean` (deleted), `model/Temporal/Tool/Testpilot.lean` (its `register_case` line removed), `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Command/**, model/Temporal/Case/**, model/Temporal/Feature/Workflow/**, model/Temporal/Feature/Nexus/Success/TypedUnary.lean, model/Temporal/Feature/Nexus/Success/Tests/TypedUnary.lean, model/Temporal/Tool/Testpilot.lean, model/TemporalModelTests.lean, tests/testcore/testpilot/**, tests/testpilot_typed_unary_case_test.go]

### Approach
- Plan review round 1 (F1): the last step deletes `.1`'s
  `tests/testcore/testpilot/testdata/baseline/typed-unary-contract.json`. It is a scaffold for this
  comparison; leaving it in the tree would be a generated Contract no generator writes.
- No new clause kind: `PropertyFieldPath` already has `.index`, `.select`, `.cardinality`, `.establish` steps and a `capture` key; the command elaborates the grammar in the spec's API Contracts to that structure and reuses `PropertyFieldComparison.check` for type mismatch and ordering rejections.
- Rejections to pin: a path segment not in the schema; a type mismatch between compared fields; an observation without a schema; a repeated-field path with no correlation key.
- Proof: regenerate the migrated Case, extract its Contract, and compare its field reads (paths, operators, literals) structurally with the task .1 baseline; identity and provenance differ, reads must not.
- Crossed pairing: the typed unary test's `evaluation 0 1 == some false` versus `none` distinction is kept by mutating the realization's key binding, not by deleting the test.

### Investigation targets
**Required:**
- `model/Umpire/Property.lean:18-103` — `PropertyFieldPath`, `PropertyFieldComparison.check`
- `model/Umpire/Case/Projection/Lowering.lean:3-47,58-89,93-137` — the rejection reasons and what a checked field Property compares
- `model/Temporal/Feature/Nexus/Success/TypedUnary.lean:44` and `Success/Tests/TypedUnary.lean:226-236` — the binding and the crossed-pairing pins
- `model/Umpire/Command/Syntax.lean:59-64,349-428` and `Authoring.lean:331-383` — `modelRequirement`, `property`, `authoredProperty`
- `model/Umpire/Case/Tests/FieldLowering.lean:589-773` — monitor-arm pins and the axiom pins

**Optional:**
- `model/Umpire/Property/Evaluate.lean:696-720` — `CheckedFieldProperty`

### Key context
- fn-85's Realization binds action classes to instructions or RPCs; the typed unary action is a unary RPC (`StartWorkflowExecution`), so no new realization kind is needed.
- Memory: behavior-neutral refactors must not strengthen validation; the migrated Case must reject and accept exactly what the hand-written one did (the `some false`/`none` pins).

## Acceptance
- [ ] `property` accepts the three field-relation forms and lowers them onto `PropertyFieldPath`; the four R2 rejections reject in place, pinned by `#guard_msgs`
- [ ] the typed unary Model file replaces the hand-written module; its Case's Contract field reads equal the baseline structurally (a Lean or Go test prints the first differing read); the artifact and live tests pass on the regenerated fixture
- [ ] the crossed-pairing mutation yields a Property violation (`some false`) and the missing-evidence case `none`, pinned
- [ ] `make umpire-check-regression` exit 0; `make lint-model` green; the inventory row for the typed unary example is marked migrated


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
