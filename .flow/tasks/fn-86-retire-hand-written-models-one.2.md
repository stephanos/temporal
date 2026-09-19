---
satisfies: [R2, R3]
---
# fn-86-retire-hand-written-models-one.2 Field relations in the property command; the typed unary example migrated (early proof point)

## Description
Add the field-relation clause to `property` (R2): `require:` compares an action's input field, an action's result field or an observation's field through their schemas, checked while the file compiles and lowered onto the existing `PropertyFieldPath` and Contract field reads. Prove it on the typed unary example (R3, the smaller half): re-author it as a Model file with the commands, a field relation and an RPC-backed realization, and show its Case lowers to the same Contract reads as the task .1 baseline. Stop if the typed unary Property cannot be expressed as a field relation that lowers to the same reads.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (the `property` command at `:404-476` gains the field-relation forms `<path> = <path>`, `<path> ≠ <path>`, `<path> present` under a key of this task's choosing; `modelRequirement` at `:40-44` is the retired `require:` form and cannot carry them, see the dated note), `model/Umpire/Command/Authoring.lean` (`authoredProperty` at `:455` lowers a field relation to `PropertyFieldComparison` over `PropertyFieldPath`; schema resolution through the realization's catalog for observations), `model/Temporal/Case/Schema.lean` (field-path resolution against a message schema; repeated fields select through the row entity's key; optional submessages present only when every segment is set), `model/Temporal/Feature/Workflow/Start/Model.lean` (new; the typed unary Model: `startWorkflow` action with `schema: temporal.api.workflowservice.v1.StartWorkflowExecutionRequest`, `workflowExecutionStarted` observation, the `submittedTypeIsRecorded` relation), `model/Temporal/Case/Realization/Workflow.lean` (RPC-backed action binding), `model/Temporal/Feature/Workflow/Start/Tests.lean` (`#guard_msgs` for the four R2 rejections; the crossed-pairing mutation as a realization mutation yielding `some false`), `tests/testcore/testpilot/testdata/<set>-<query>-case.json` (replaces `typed-unary-case.json`), `tests/testcore/testpilot/typed_unary_artifact_test.go` and `tests/testpilot_typed_unary_case_test.go` (re-pointed), `model/Temporal/Feature/Nexus/Success/TypedUnary.lean` and `Success/Tests/TypedUnary.lean` (deleted), `model/Temporal/Tool/Testpilot.lean` (its `register_case` line removed), `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Command/**, model/Temporal/Case/**, model/Temporal/Feature/Workflow/**, model/Temporal/Feature/Nexus/Success/TypedUnary.lean, model/Temporal/Feature/Nexus/Success/Tests/TypedUnary.lean, model/Temporal/Tool/Testpilot.lean, model/TemporalModelTests.lean, tests/testcore/testpilot/**, tests/testpilot_typed_unary_case_test.go]

### Approach
- Plan review round 1 (F1): the last step deletes `.1`'s
  `tests/testcore/testpilot/testdata/baseline/typed-unary-contract.json`. It is a scaffold for this
  comparison; leaving it in the tree would be a generated Contract no generator writes.
- No new clause kind: `PropertyFieldPath` already has `.index`, `.select`, `.cardinality`, `.establish` steps and a `capture` key; the command elaborates the grammar in the spec's API Contracts to that structure and reuses `PropertyFieldComparison.check` for type mismatch and ordering rejections.
- Rejections to pin: a path segment not in the schema; a type mismatch between compared fields; an observation without a schema; a repeated-field path with no correlation key.
- Proof: regenerate the migrated Case, extract its Contract, and compare its field reads (paths, operators, literals) structurally with the task .1 baseline; identity and provenance differ, reads must not.
- Crossed pairing: the typed unary test's `evaluation 0 1 == some false` versus `none` distinction is kept by mutating the realization's key binding, not by deleting the test.
- Adjusted 2026-09-19 after fn-85 .15, .5 and .7 landed. **`require:` is retired.** fn-85 .15
  made `property` take `machine:`, an optional `when:` (bare or classed action) and `holds:` with
  a `Step → Bool` or `Step → Step → Bool` predicate, enumerated over the machine's table into
  clause records; the keyed form is rejected at its key (`Syntax.lean:466-476`, "the keyed form is
  retired"), and `require`/`model` are bare words SEM-20 keeps out of the vocabulary gate. The
  spec's field-relation grammar (`when:` + `require: <path> = <path>`) therefore needs a new key
  beside `holds:` (a `relates:` line, or field paths admitted inside a predicate the command
  recognizes); this task decides the spelling and records it, keeping R2's three forms and four
  rejections and the lowering onto `PropertyFieldPath` and `PropertyFieldComparison.check`. A
  Property's fingerprint reads through the machine's state fields since fn-85 .4, so the migrated
  Property's fingerprint will not equal the hand-written one; the comparison is the Contract's
  field reads, as written. **Realization and identity:** the realization shape to follow is
  `Temporal.Case.Realization.Nexus` (`Realization` with `plan`, `actions : List ActionBinding`,
  `setup`, `switches`; an `ActionBinding` builds its node from the instruction id the Producer
  supplies), so the typed unary action is an `ActionBinding` whose node is `Program.invokeRpc` on
  `StartWorkflowExecution`, and the Case is produced by the Temporal `case <name> realizes <set> as
  <template> evidence <lines>` block (`Temporal/Case/Syntax.lean:183-260`), whose `as` clause today
  parses only the `nexusOperation …` and `<ident> type <str>` template arms until fn-85 .11 replaces
  them with a realization reference; an RPC-only realization needs that reference or a third arm.
  The fixture is `<set>-<query>-case.json` and the artifact table (`fixture_table_test.go`) picks it
  up without a Go table entry. **Schema:** `Temporal.Case.Schema` resolves `schema:` by message
  name only (fn-85 .2); fn-85 .8 adds the first member-against-descriptor check, which field-path
  resolution should build on rather than duplicate.

### Investigation targets
**Required:**
- `model/Umpire/Property.lean:18-103` — `PropertyFieldPath`, `PropertyFieldComparison.check`
- `model/Umpire/Case/Projection/Lowering.lean:3-47,58-89,93-137` — the rejection reasons and what a checked field Property compares
- `model/Temporal/Feature/Nexus/Success/TypedUnary.lean:44` and `Success/Tests/TypedUnary.lean:226-236` — the binding and the crossed-pairing pins
- `model/Umpire/Command/Syntax.lean:40-44,359,404-476` and `Authoring.lean:455-520,605-660` — `modelRequirement` (retired form), `propertyWhen`, the `property` command, `authoredProperty`, `producerInput`/`produce`/`produceCase`
- `model/Temporal/Case/Realization/Nexus.lean` and `model/Temporal/Case/Syntax.lean:39-47,183-260` — the realization shape and the set-realizing `case` block with its template arms
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
