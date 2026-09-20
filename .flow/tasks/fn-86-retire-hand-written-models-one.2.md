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
  `tests/testcore/testpilot/baseline/typed-unary-contract.json`. It is a scaffold for this
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
- Adjusted 2026-09-20 by fn-86 .1 after fn-85 closed. **The `as` clause takes a `Realization`
  value** (fn-85 .11): `case <name> realizes <set> as (<term>)` with an optional `evidence` block,
  so an RPC-only realization is a `Umpire.Case.Producer.Realization` value named in parentheses,
  no third arm. `Temporal.Case.Syntax` also admits a canary set (fn-85 .12). `Realization` carries
  `plan`, `actions` (keyed `ActionBinding`s), `timers`, `switches`, `sources` and `limits`; the
  template-era `hooks`, `taskQueueRole` and `faultRuleId` are gone. `Temporal.Case.Schema`
  resolves `schema:` by message name and fn-85 .8 checks members against the descriptor. The
  baseline to compare against is `tests/testcore/testpilot/baseline/typed-unary-contract.json`
  (fn-86 .1), the fixture's `contract` block dedented; delete the directory once the comparison
  passes. `register_case` still has four lines in `model/Temporal/Tool/Testpilot.lean`.

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
- [x] `property` accepts the three field-relation forms and lowers them onto `PropertyFieldPath`; the four R2 rejections reject in place, pinned by `#guard_msgs`
- [x] the typed unary Model file replaces the hand-written module; its Case's Contract field reads equal the baseline structurally (a Lean or Go test prints the first differing read); the artifact and live tests pass on the regenerated fixture
- [x] the crossed-pairing mutation yields a Property violation (`some false`) and the missing-evidence case `none`, pinned
- [x] `make umpire-check-regression` exit 0; `make lint-model` green; the inventory row for the typed unary example is marked migrated


## Done summary

Done 2026-09-20; self-review. Commit 7b85de2.

### The spelling: `relates:`

A field relation is a `relates:` line of `property`, beside `holds:`: `property <name> machine: <m>
when: <action> relates: <left> = <right>`, `≠`, or `relates: <operand> present`. The retired
`require:` key stays rejected at its key. An operand is `<action>.input.<path>` or
`<action>.result.<path>` -- the action must be the one `when:` names, and it must declare a
`schema:` -- or `<kind>.<path>`, where `<kind>` is a recorded event kind one of the machine's
`evidence:` lines names (the operand's member is the fact that line confirms). The command hands
each dotted path to the platform through a new hook, `Umpire.Command.installFieldResolver`
(`Umpire/Command/FieldResolver.lean`, an `IO.Ref` installed at import like the schema and catalog
checks), and `Temporal.Case.FieldPath` answers it off the generated descriptors: an action's
`schema:` message is the request of one of the admitted workflow-service methods
(`StartWorkflowExecution`, `GetWorkflowExecutionHistory`; a Model that needs another adds it to
the list), an input path is walked from that request and a result path from the method's
response; a recorded kind is one arm of the `HistoryEvent` `attributes` oneof read back through the
history response (`response.history` established, `events[0]`, the arm selected), then the
author's segments. Each optional message the walk enters is a presence read the relation
establishes, a oneof member a selection; a path must end at a comparable scalar (a repeated field,
a map field, a message end, a float and an unsupported type reject with the reason at the operand).
The answer names the constant that holds the schema (`Temporal.Case.FieldPath.startWorkflowSchema`,
`getWorkflowExecutionHistorySchema`), so the emitted `FieldOperand` carries the generated method's
own `RpcSchema` by reference. The elaborator checks the two operands' scalar types agree, and that a
`present` operand ends on an `.establish` or `.select` step (a oneof member or a proto3-optional
scalar; an optional message's presence is written as a field inside it). It emits
`def <name> : Umpire.Case.Producer.FieldRelation` (`Umpire/Case/Relation.lean`: id, name, action
key, operator, `FieldOperand`s with root, member, spelling, schema, side, steps, type and presence
paths, source) and records it in a new `relationExtension` (`Registry.recordRelation`,
`relationsOf`), which the Temporal `case … realizes` block reads to pass the model's relations to
`produceCase`. `ActionEntry` now records the action's `schema:`.

### Lowering

`FieldRelation.declaration` builds the Property the relation denotes -- one branch, one clause,
`all` of a presence atom per traversed presence path and the comparison (or, for `present`, the
operand's own presence read) -- and `FieldRelation.check` admits it with
`CheckedFieldProperty.check` under `PropertyFieldBinding.ofSchema` bindings (new in
`Umpire/Property.lean`) for the references the Producer resolved: the request/result root to the
action's Definition ID, the event root to the fact's. `Umpire.Case.Producer.produce` takes
`relations : List FieldRelation` (the ones whose action the performed path names), resolves each
through the vocabulary, reads the observation and literal from the realization -- an
`ActionBinding` gains `literals : Identity → List (String × Scalar)`, the request fields its node
assigns by dotted spelling, so the literal a relation compares against is the value the Program
constructs -- and lowers it through `Umpire.Case.Projection.lower` with the rule suffix
`FieldRelation.ruleSuffix = "relation"`: the rule is `<property>.relation` with transitions
`match-relation` and `reject-relation`. Rejections are Producer errors at the relation:
`relation.action-unbound`, `relation.literal-unassigned`, `relation.action-unplaced`,
`relation.observation-undeclared`, `relation.no-rule`. A `present` relation admits and lowers to
no rule (`Projection.lower` has no shape for a presence-only clause), which `Producer` reports as
`relation.no-rule` only where a Case is produced over it; the test file's `overrideIsAutoUpgrade`
is admitted, not produced. The relation's definition and source join the Case's definitions and
sources (plain concatenation, so the seven Caller fixtures are byte-identical), its lowering joins
`properties` and its input mappings the coverage request.

### The typed unary example, re-authored

`Temporal/Feature/Workflow/Start/Model.lean`: `entity workflow key: firstExecutionRunId`, a
two-phase `StartState`, `action startWorkflow party: caller creates: workflow schema:
temporal.api.workflowservice.v1.StartWorkflowExecutionRequest`, `machine workflowStart` with
`evidence: workflowExecutionStarted: workflowExecutionStarted`, `property startRecorded` (holds),
`property submittedTypeIsRecorded … relates: startWorkflow.input.workflow_type.name =
workflowExecutionStarted.workflow_type.name`, `scenario startOnce`, `limits one`, `query started`,
`set workflowStartTests purpose: functional bind: caller: driven queries: [started]` and `case
workflowStartCases realizes workflowStartTests as Temporal.Case.Realization.workflowStart`.
`Temporal/Case/Realization/Workflow.lean` is the RPC-backed realization: the `startWorkflow`
binding's node is `Program.invokeRpc` on `StartWorkflowExecution` with `workflow_type.name :=
"umpire-<fixture>-workflow"` stated as its literal, then a close-event read and the history read
that lifts the started event (keyed by `first_execution_run_id`), and a workflow entrypoint that
finishes. The produced Case is `temporal.case.workflowStartTests.started`, fixture
`workflowStartTests-started-case.json`.

**Baseline comparison: identical up to the rule suffix and the literal.** The migrated Case's
monitor rule, compared read by read with `.1`'s `typed-unary-contract.json`: the same safety rule
(`pending`/`satisfied`/`violated`), the same two transitions on `INSTRUCTION_COMPLETED`, each
predicate the same `present history-event`, the same read
`attributes<workflow_execution_started_event_attributes>.workflow_type.name` on the `history-event`
Observation, the same `Equal` comparison and the same negated form on the reject arm; what differs
is the rule name (`relation` for `recorded-workflow-type`) and the literal
(`umpire-workflowStartTests-started-workflow` for `umpire-typed-unary-workflow`), both the
identity's. `TestWorkflowStartCaseReadsMatchTypedUnaryBaseline`
(`tests/testcore/testpilot/workflow_start_artifact_test.go`) renders the rule's reads in order and
names the first differing read against that list. The hand-written module, its tests, its
`register_case` line, `typed-unary-case.json` and the baseline scaffold are deleted; the Go readers
are re-pointed (`workflow_start_fixture.go`, `workflow_start_artifact_test.go`,
`tests/testpilot_workflow_start_case_test.go`, the profile oracle, the generator's fake registry).
Fixture diff: `typed-unary-case.json` deleted, `workflowStartTests-started-case.json` added, the
seven `nexusCallerTests-*` fixtures unchanged. `model/README.md`'s typed-authoring bullet is
re-pointed at the Model (the section's rewrite stays with `.9`); the ledger row is marked migrated.

### Pins

`Temporal/Feature/Workflow/Start/Tests.lean`: the resolved operands (steps, presence paths, type,
schema equal to `rpcOwner.schema` of the generated witness); the three forms (`≠` and `present`
written in the test file, a `result` operand reading the response schema); nine `#guard_msgs`
rejections -- an unknown segment, a type mismatch (`string` and `int32`), a repeated field
(`links`), an observation without a schema (`faultInjected`, a Run Event kind a test machine names
as evidence), the wrong action, an unnamed kind, a message end, and `present` on an always-present
field; the produced Case (identity, the one rule and its transitions, the controller's three
instructions, no Known Gap); the lowering's shape (safety rule, the right operand's path, the
segments string, the literal) and coverage; and the independent field requirement evaluated over
real admitted payloads with references from the vocabulary: `evaluation 0 0`/`1 1` `some true`,
crossed `0 1`/`1 0` `some false`, missing evidence `none`.

### Gates

`lake build` (all 633 jobs) green; conformance fixtures regenerated (only the diff above);
goldens, inventory, retired vocabulary, protocol, authoring and regression-view checks exit 0;
`LEAN_NUM_THREADS=1 make lint-model` at the `.1` baseline (41 warnings, none new);
`GOLANGCI_LINT_BASE_REV=55580b2 make lint-code-fast` clean; `go test` over
`common/testing/testpilot`, `tests/testcore/testpilot` and `tools/umpire` ok;
`TestTestpilotWorkflowStartCase` passes live; `make umpire-check-regression` exit 0 with
29 passing live identities (29 before; the typed-unary identity is replaced by the
workflow-start one).

## Evidence
- Commits: 7b85de2
- Tests: `lake build`; `make umpire-gen-case-runtime-conformance && make umpire-check-case-runtime-conformance`; `make umpire-check-goldens`; `make umpire-gen-inventory && make umpire-check-inventory`; `make umpire-check-retired-vocabulary`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `make umpire-check-regression-views`; `LEAN_NUM_THREADS=1 make lint-model`; `GOLANGCI_LINT_BASE_REV=55580b2 make lint-code-fast`; `go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/...`; `go test -count=1 -tags test_dep,integration ./tests -run TestTestpilotWorkflowStartCase`; `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-regression`
- PRs:
