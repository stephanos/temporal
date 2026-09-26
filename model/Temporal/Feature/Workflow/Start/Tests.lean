import Temporal.Feature.Workflow.Start.Model
import Temporal.Shared
import Temporal.Testpilot.CaseSupport
import Umpire.Operation.Parameterized

/-!
# What the workflow-start Model says

The relation `submittedTypeIsRecorded` is the typed unary example of fn-83 written as one line, and
the pins here are the ones its hand-written Tests carried, read off the command's output instead of
declared beside it: the operands the platform resolved (their steps, presence reads and type), the
three forms the command accepts and the four mistakes it rejects in place, the produced Case's one
monitor rule and its read, and the independent field requirement evaluated over real admitted
payloads, where a crossed pairing is a Property violation and missing evidence is no answer at all.
-/

namespace Temporal.Feature.Workflow.Start.Tests

open Umpire
open Umpire.Operation
open Umpire.Value
open Umpire.Case
open Umpire.Command
open Umpire.Case.Producer (FieldRelationOperator)
open Temporal.Feature.Workflow.Start
open Temporal.Testpilot.CaseSupport
open temporal.server.api.testpilot.v1 hiding ModelValue SourceLocation

/-! ### The machine -/

#guard workflowStart.table.states.length == 2
#guard workflowStart.actionKeys == #["startWorkflow"]
#guard workflowStart.stuck == none

/-- info: 'Temporal.Feature.Workflow.Start.workflowStart' depends on axioms: [propext] -/
#guard_msgs in
#print axioms workflowStart

/-! ### The relation the platform resolved

Each operand's coordinates are read off the generated descriptors while the Model compiles: the
request's `workflow_type` is an optional message, so the read establishes it; the recorded event is
one arm of the `attributes` oneof of the history response's first event, so the read selects it. -/

private def startRequestRoot := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest"
private def historyResponseRoot :=
  "temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryResponse"
private def workflowTypeNode := "temporal.api.common.v1.WorkflowType"
private def historyNode := "temporal.api.history.v1.History"
private def startedAttributesNode :=
  "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes"

private def submittedSteps : List Field.Step :=
  [.field startRequestRoot 3, .establish, .field workflowTypeNode 1]

private def startedSteps : List Field.Step :=
  [.field historyResponseRoot 1, .establish, .field historyNode 1, .index 0,
    .field historyEventNode 6, .select "attributes", .field startedAttributesNode 1, .establish,
    .field workflowTypeNode 1]

#guard submittedTypeIsRecorded.action == "startWorkflow"
#guard submittedTypeIsRecorded.operator == .equal
#guard submittedTypeIsRecorded.id.value ==
  "temporal.workflow.start.property.submittedTypeIsRecorded"

#guard submittedTypeIsRecorded.left.root == .request
#guard submittedTypeIsRecorded.left.member == "startWorkflow"
#guard submittedTypeIsRecorded.left.spelling == "workflow_type.name"
#guard submittedTypeIsRecorded.left.side == .request
#guard submittedTypeIsRecorded.left.steps == submittedSteps
#guard submittedTypeIsRecorded.left.type == .text
#guard submittedTypeIsRecorded.left.presence == [[.field startRequestRoot 3, .present]]

#guard (submittedTypeIsRecorded.right.map (·.root)) == some .event
#guard (submittedTypeIsRecorded.right.map (·.member)) == some "workflowExecutionStarted"
#guard (submittedTypeIsRecorded.right.map (·.spelling)) == some "workflow_type.name"
#guard (submittedTypeIsRecorded.right.map (·.side)) == some .response
#guard (submittedTypeIsRecorded.right.map (·.steps)) == some startedSteps
#guard (submittedTypeIsRecorded.right.map (·.type)) == some .text
#guard (submittedTypeIsRecorded.right.map (·.presence)) == some [
  [.field historyResponseRoot 1, .present],
  startedSteps.take 5 ++ [.present],
  startedSteps.take 7 ++ [.present]]

/- The schemas an operand carries are the generated methods' own, not copies beside them. -/
private abbrev startMethod := Temporal.Api.Workflowservice.V1.WorkflowService.startWorkflowExecution
private abbrev historyMethod :=
  Temporal.Api.Workflowservice.V1.WorkflowService.getWorkflowExecutionHistory
private def startReference : Temporal.API.MethodReference startMethod := by constructor
private def historyReference : Temporal.API.MethodReference historyMethod := by constructor
private def startWitness : Temporal.API.rpcOwner.Witness
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionRequest
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionResponse := ⟨startMethod, startReference⟩
private def historyWitness : Temporal.API.rpcOwner.Witness
    Temporal.Api.Workflowservice.V1.GetWorkflowExecutionHistoryRequest
    Temporal.Api.Workflowservice.V1.GetWorkflowExecutionHistoryResponse :=
  ⟨historyMethod, historyReference⟩

#guard submittedTypeIsRecorded.left.schema == Temporal.API.rpcOwner.schema startWitness
#guard (submittedTypeIsRecorded.right.map (·.schema)) ==
  some (Temporal.API.rpcOwner.schema historyWitness)
#guard submittedTypeIsRecorded.left.schema.fullName ==
  "temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution"

/-! ### The three forms

Equality is the Model's. Inequality and presence are written here, on fields of the same
schemas: a relation is a Property of the machine, and one written in a test file is admitted the
same way. -/

/- Two fields of one scalar type may be required to differ. -/
property idIsNotType
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.input.workflow_id ≠ workflowExecutionStarted.workflow_type.name

#guard idIsNotType.operator == .notEqual
#guard idIsNotType.left.steps == [.field startRequestRoot 2]
#guard idIsNotType.left.presence == []
#guard (idIsNotType.right.map (·.steps)) == some startedSteps

/- `present` relates a field the read establishes or selects: here a member of the
`versioning_override` oneof, whose read ends at the selection. -/
property overrideIsAutoUpgrade
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.input.versioning_override.auto_upgrade present

#guard overrideIsAutoUpgrade.operator == .present
#guard overrideIsAutoUpgrade.right == none
#guard overrideIsAutoUpgrade.left.type == .boolean
#guard overrideIsAutoUpgrade.left.steps.getLast? == some (.select "override")
#guard overrideIsAutoUpgrade.left.presence.length == 2

/- A result field reads the action's response schema. -/
property runIdIsRecorded
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.result.run_id = workflowExecutionStarted.original_execution_run_id

#guard runIdIsRecorded.left.root == .outcome
#guard runIdIsRecorded.left.side == .response
#guard runIdIsRecorded.left.type == .text
#guard runIdIsRecorded.left.schema == submittedTypeIsRecorded.left.schema

/-! ### What the command rejects, where the author wrote it -/

/- A segment the schema has no field for. -/
/--
error: 'workflow_kind' is not a field of temporal.api.workflowservice.v1.StartWorkflowExecutionRequest
-/
#guard_msgs in
property unknownSegment
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.input.workflow_kind.name = workflowExecutionStarted.workflow_type.name

/- Two fields of different scalar types. -/
/--
error: the compared fields differ in type: string and int32; a relation compares fields of one scalar type
-/
#guard_msgs in
property typeMismatch
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.input.workflow_id = workflowExecutionStarted.attempt

/- A path through a repeated field, which no correlation key selects from. -/
/--
error: 'links' is a repeated field; a path through a repeated field selects by a correlation key, which no relation declares
-/
#guard_msgs in
property repeatedPath
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.input.links.workflow_event.namespace = workflowExecutionStarted.workflow_type.name

/- An observation that is not a recorded history event carries no schema: a Run Event kind such as
`faultInjected` is evidence a machine may name, but it has no fields to relate. -/
machine faultStart
  for: workflow
  state: StartState
  starts: [pending]
  ends: [started]
  evidence:
    workflowExecutionStarted: faultInjected
  steps:
    startWorkflow: startStep

/--
error: 'faultInjected' is not a recorded history event kind, so it carries no schema to name a field of
-/
#guard_msgs in
property faultField
  machine: faultStart
  when: startWorkflow
  relates: startWorkflow.input.workflow_id = faultInjected.workflow_id

/- The operand's action is the one the claim is about. -/
/--
error: an input or result field belongs to the action the claim is about ('startWorkflow'), not 'workflowExecutionStarted'
-/
#guard_msgs in
property wrongAction
  machine: workflowStart
  when: startWorkflow
  relates: workflowExecutionStarted.input.workflow_id = workflowExecutionStarted.workflow_type.name

/- A kind no `evidence:` line of the machine names. -/
/--
error: 'workflowExecutionCompleted' is not a recorded event kind this machine's `evidence:` lines name; named: workflowExecutionStarted
-/
#guard_msgs in
property unnamedKind
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.input.workflow_id = workflowExecutionCompleted.workflow_type.name

/- A path that ends on a message, or on a field that is always present under `present`. -/
/--
error: 'workflow_type' is a message (temporal.api.common.v1.WorkflowType); name one of its fields
-/
#guard_msgs in
property messageEnd
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.input.workflow_type = workflowExecutionStarted.workflow_type.name

/--
error: 'startWorkflow.input.workflow_id' is always present; `present` relates an optional field or a oneof member
-/
#guard_msgs in
property alwaysPresent
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.input.workflow_id present

/-! ### The Case

The set's one Query produces one Case, and the relation is its one monitor rule: the safety rule
`relation`, whose read is the recorded event's `workflow_type.name` and whose literal is the
workflow type the realization's start binding assigns. -/

#guard workflowStartCases.started.identity.fixture == "workflowStartTests-started"
#guard workflowStartCases.started.identity.caseId == "temporal.case.workflowStartTests.started"

private def produced : Option temporal.server.api.testpilot.v1.Case :=
  workflowStartCases.started.toOption

#guard produced.isSome

/-- The states and transitions of the one monitor rule, as the runtime reads them. -/
private def producedRule :
    Option (String × ContractRuleKind × List (String × ContractStateStatus) ×
      List (String × String)) := do
  let output ← produced
  let contract ← output.contract
  let [rule] := contract.rules.toList | none
  pure (rule.rule_id, rule.kind,
    rule.states.toList.map fun state => (state.state_id, state.status),
    rule.transitions.toList.map fun transition =>
      (transition.transition_id, transition.target_state_id))

/- The runtime rule separates the same three answers the Property does: a recorded type that
disagrees is a violation, and an event that never establishes the field leaves the rule pending. -/
#guard producedRule == some ("relation", .CONTRACT_RULE_KIND_SAFETY,
  [("pending", .CONTRACT_STATE_STATUS_PENDING),
   ("satisfied", .CONTRACT_STATE_STATUS_SATISFIED),
   ("violated", .CONTRACT_STATE_STATUS_VIOLATED)],
  [("match-relation", "satisfied"), ("reject-relation", "violated")])

/-- The controller's instructions: the start, then the scaffolding the realization adds. -/
private def instructionIds (entrypointId : String) : List String :=
  ((produced.bind fun output => output.program.bind fun (program : Program) =>
    program.entrypoints.find? (·.entrypoint_id == entrypointId)).map fun entrypoint =>
      entrypoint.instructions.toList.map (·.instruction_id)).getD []

#guard instructionIds "controller" == ["start-workflow", "await-close", "history"]
#guard instructionIds "workflow" == ["finish-workflow"]

/- The Case carries no Known Gap: the one step records what confirms it. -/
#guard (produced.bind fun output => output.provenance.map fun provenance =>
  provenance.known_gaps.size) == some 0

/-- The relation admitted against the Query's Model, with its references resolved through the
vocabulary. -/
private def checkedRelation : Option CheckedFieldProperty :=
  (Umpire.Command.checkedRelation started submittedTypeIsRecorded).toOption

#guard checkedRelation.isSome

private def vocabulary : Option Umpire.Case.Producer.Vocabulary :=
  started.toOption.map fun checked => (Umpire.Command.producerInput checked).vocabulary

private def startActionId : Option DefinitionId :=
  vocabulary.map fun values => (values.namedAction "startWorkflow").definitionId
private def startedFactId : Option DefinitionId :=
  vocabulary.map fun values => (values.namedFact "workflowExecutionStarted").definitionId

#guard startActionId.map (·.value) ==
  some "temporal.workflow.start.action.workflowStart.startWorkflow"
#guard startedFactId.map (·.value) ==
  some "temporal.workflow.start.fact.workflowStart.workflowExecutionStarted"

/-- The literal the start binding assigns under the produced identity. -/
private def submittedWorkflowType : String :=
  Temporal.Case.Realization.Workflow.workflowTypeOf workflowStartCases.started.identity

#guard submittedWorkflowType == "umpire-workflowStartTests-started-workflow"

/-- The Contract lowering of the admitted relation under the realization's literal. -/
private def lowered : Option (Option Projection.Shape × Coverage.Request) := do
  let checked ← checkedRelation
  let actionId ← startActionId
  let observation := Testpilot.Authoring.Program.observation
    Temporal.Case.Support.historyObservation historyEventType
  let realization : Projection.Realization := {
    literals := [{ path := submittedTypeIsRecorded.left.path actionId
                   value := .text submittedWorkflowType
                   entrypointId := "controller", instructionId := "start-workflow" }]
    ruleSuffix := Umpire.Case.Producer.FieldRelation.ruleSuffix }
  let result ← (Projection.lower checked observation realization).toOption
  pure (result.rule.map (·.shape), result.coverage)

/- The derived rule is the safety rule over the recorded workflow type, matched against the
workflow type the Program submits, and its read is the relation's right operand. -/
#guard match lowered with
  | some (some (.safety false read literal), _) =>
      (submittedTypeIsRecorded.right.map fun right =>
        read.path == right.path (startedFactId.getD (.of ""))) == some true &&
      literal.scalar == .text submittedWorkflowType &&
      read.segments == "attributes<workflow_execution_started_event_attributes>.workflow_type.name"
  | _ => false

/- The coverage the relation implies is exactly the submitted field the binding assigns. -/
#guard (lowered.map fun (_, coverage) => coverage.inputs.map fun input =>
    (input.path.steps, input.value, input.entrypointId, input.instructionId)) ==
  some [(submittedSteps, .text submittedWorkflowType, "controller", "start-workflow")]

/-! ### The independent field requirement, over the actual correlated evidence

The Property the relation denotes is evaluated here the way the hand-written one was: over a
request the checked Action template admitted and a started event read through the history schema,
each operand a projection built by a real cursor walk. -/

private def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Workflow/Start/Tests.lean"

private def valueLimits : Value.Limits := ⟨16, 20000000, 262144, 512⟩
private def runtimeBounds : RuntimeBounds := ⟨12, 8192, 64⟩
private def taskQueueNode := "temporal.api.taskqueue.v1.TaskQueue"

/-- The alternate workflow type a crossed sample submits. -/
private def alternateWorkflowType := "umpire-workflowStartTests-alternate-workflow"

private def sampledWorkflowTypes : List String := [submittedWorkflowType, alternateWorkflowType]

private def requestValue (workflowType : String) : Raw :=
  Value.message startRequestRoot [
    (2, Value.literal (.text "umpire-workflow-start")),
    (3, Value.message workflowTypeNode [(1, Value.literal (.text workflowType))]),
    (4, Value.message taskQueueNode [(1, Value.literal (.text "umpire-workflow-start-queue"))])]

private def startedEvidenceValue (workflowType : String) : Raw :=
  Value.message historyResponseRoot [
    (1, Value.message historyNode [
      (1, Value.repeated [
        Value.message historyEventNode [
          (1, Value.literal (.integer .int64 1)),
          (6, Value.message startedAttributesNode [
            (1, Value.message workflowTypeNode [(1, Value.literal (.text workflowType))])])]])])]

private def fieldError (reason : String) : Field.Error := ⟨source, "workflow-start", reason⟩

private abbrev StartTemplate := ActionTemplate Temporal.API.rpcOwner
  Temporal.Api.Workflowservice.V1.StartWorkflowExecutionRequest
  Temporal.Api.Workflowservice.V1.StartWorkflowExecutionResponse Empty

/-- The Action template over the generated Start declaration, identified as the Model's action. -/
private def startTemplate : Option StartTemplate := do
  let actionId ← startActionId
  let binding ← (Temporal.API.bindUnary startMethod startReference).toOption
  (ActionTemplate.check (rpc Empty binding) actionId).toOption

/-- The submitted `workflow_type` cursor and its presence read, over the selected Action's own
immutable arguments. -/
private def submittedProjections {template : StartTemplate}
    (action : ActionInstance template valueLimits) :
    Except Field.Error (List (PropertyFieldProjection Temporal.API.rpcOwner
      template.declaration.reference)) := do
  let typeReference ← Field.reference Temporal.API.rpcOwner template.declaration.reference .request
    startRequestRoot 3 source
  let nameReference ← Field.reference Temporal.API.rpcOwner template.declaration.reference .request
    workflowTypeNode 1 source
  let root ← (Field.root action.arguments).refine (.message startRequestRoot) .singular .available
    source
  let workflowType ← root.field typeReference source
  let workflowType ← workflowType.refine (.message workflowTypeNode) .singular .optional source
  let presence ← workflowType.present source
  let established ← workflowType.establish source
  let name ← established.field nameReference source
  let name ← name.refine .text .singular .available source
  if presenceOrigin : presence.origin.value = action.arguments.value then
    if nameOrigin : name.origin.value = action.arguments.value then
      pure [← PropertyFieldProjection.ofAction action presence presenceOrigin source,
        ← PropertyFieldProjection.ofAction action name nameOrigin source]
    else throw (fieldError "submitted read left the selected arguments")
  else throw (fieldError "submitted presence read left the selected arguments")

/-- The recorded `workflow_type` cursor and the three presence facts its read traverses. -/
private def startedProjections (factId : DefinitionId) (workflowType : String) :
    Except Field.Error (List (PropertyFieldProjection Temporal.API.rpcOwner historyWitness)) := do
  let value ← (Value.check Temporal.API.rpcOwner historyWitness .response valueLimits
    (startedEvidenceValue workflowType)).mapError fun error =>
      Field.Error.mk source error.path error.reason
  let historyReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyResponseRoot 1 source
  let eventsReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyNode 1 source
  let attributesReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyEventNode 6 source
  let typeReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    startedAttributesNode 1 source
  let nameReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    workflowTypeNode 1 source
  let root ← (Field.root value).refine (.message historyResponseRoot) .singular .available source
  let history ← root.field historyReference source
  let history ← history.refine (.message historyNode) .singular .optional source
  let historyPresence ← history.present source
  let history ← history.establish source
  let events ← history.field eventsReference source
  let events ← events.refine (.message historyEventNode) .repeated .available source
  let event ← events.index 0 source
  let attributes ← event.field attributesReference source
  let attributes ← attributes.refine (.message startedAttributesNode) .singular
    (.oneof "attributes") source
  let attributesPresence ← attributes.present source
  let attributes ← attributes.select "attributes" source
  let recordedType ← attributes.field typeReference source
  let recordedType ← recordedType.refine (.message workflowTypeNode) .singular .optional source
  let typePresence ← recordedType.present source
  let recordedType ← recordedType.establish source
  let name ← recordedType.field nameReference source
  let name ← name.refine .text .singular .available source
  pure [
    ← PropertyFieldProjection.ofCursor .event factId historyPresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .event factId attributesPresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .event factId typePresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .event factId name (by decide) source]

/- Every operand of the relation is exactly a projection a real cursor walk builds: the presence
reads the operands traverse and the reads they end on. -/
#guard (do
  let factId ← startedFactId
  let right ← submittedTypeIsRecorded.right
  let projections ← (startedProjections factId submittedWorkflowType).toOption
  pure (projections.map (·.evidence.path) == right.presencePaths factId ++ [right.path factId])) ==
  some true

/-- The two admitted Start requests, in sample order. -/
private def admittedActions : Option (Σ template : StartTemplate,
    List (ActionInstance template valueLimits)) := do
  let template ← startTemplate
  let domain ← (ParameterDomain.check template valueLimits (sampledWorkflowTypes.map requestValue)
    .sampled (.schema runtimeBounds)).toOption
  pure ⟨template, domain.actions⟩

#guard (admittedActions.map fun ⟨_, actions⟩ => actions.length) == some 2

#guard (do
  let actionId ← startActionId
  let ⟨_, actions⟩ ← admittedActions
  let action ← actions[0]?
  let submitted ← (submittedProjections action).toOption
  pure (submitted.map (·.evidence.path) ==
    submittedTypeIsRecorded.left.presencePaths actionId ++
      [submittedTypeIsRecorded.left.path actionId])) == some true

/-- Evaluate the admitted relation over one modeled step: the Action at `actionIndex` paired with
the started evidence recorded for `evidenceIndex`. -/
private def evaluation (actionIndex evidenceIndex : Nat) (withEvidence : Bool := true) :
    Option Bool := do
  let checked ← checkedRelation
  let values ← vocabulary
  let factId ← startedFactId
  let ⟨_, actions⟩ ← admittedActions
  let action ← actions[actionIndex]?
  let workflowType ← sampledWorkflowTypes[evidenceIndex]?
  let submitted ← (submittedProjections action).toOption
  let started ← (startedProjections factId workflowType).toOption
  let recorded ← started.getLast?
  let trace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
    initialState := values.namedState "pending"
    steps := [{ selectedAction := action.modelValue
                outcome := values.namedOutcome "accepted"
                state := values.namedState "started"
                facts := [recorded.modelValue] }] }
  let evidence := submitted.map (·.evidence) ++
    (if withEvidence then started.map (·.evidence) else [])
  let input ← (checked.checkInput trace [evidence]).toOption
  pure (evaluateProperty checked.property input).satisfied

/- Each admitted request is paired with the workflow type its own execution recorded. -/
#guard evaluation 0 0 == some true
#guard evaluation 1 1 == some true

/- Crossing the pairing violates the relation: a product violation, not a rejection. -/
#guard evaluation 0 1 == some false
#guard evaluation 1 0 == some false

/- Missing evidence never satisfies the comparison; the input is rejected instead. -/
#guard evaluation 0 0 (withEvidence := false) == none

end Temporal.Feature.Workflow.Start.Tests
