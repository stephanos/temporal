import Temporal.Feature.Nexus3.Nexus
import Temporal.Testpilot.CaseSupport
import Umpire.Case.Correlated
import Umpire.Case.Compiler
import Umpire.Case.Projection.Coverage

/-!
# Nexus3 Testpilot producer

The checked model is carried into a Case; it is never compared against an expected one. The
Nexus-specific Program is fixed realization, and everything the Contract says is derived:

* Each `require` clause of the checked Property becomes one operation-correlated bounded-response
  clause. The clause the model wrote decides the clause the Case carries, so editing a `require`
  line changes the Case bytes rather than any file here.
* The Contract carries no monitor rule at all. Its whole content is the correlated capability that
  `Umpire.Case.Correlated.lower` produced, whose `Lowered` value is the correspondence certificate
  between the checked model and the wire bytes.
* The evidence those clauses read is lifted out of the recorded Nexus history by a declaration on
  this Program's history read: one rule per recorded event kind, keyed by the scheduled event the
  operation was scheduled at, which is the only identity a started or completed Nexus event
  records.

What still rejects, and why: a witness the Query did not select (a Case realizes one selected
trace, so a verify-form Query has none), a clause whose shape no correlated predicate can carry, and a
requested clause the lowering did not produce. None of those is waivable by a Known Gap.
-/

namespace Temporal.Feature.Nexus3.Testpilot

open Umpire
open Umpire.Case.Compiler
open Temporal.Testpilot.CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

private def workflowServiceRole := "temporal.workflow-service"
private def workerRole := "temporal.worker"
private def taskQueueRole := "temporal.task-queue"
private def nexusEndpointRole := "temporal.nexus-endpoint"
private def workerNamespaceBinding := "temporal.worker.namespace"
private def taskQueueBinding := "temporal.task-queue.resource"
private def nexusEndpointBinding := "temporal.nexus-endpoint.resource"
private def startWorkflowMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
private def getHistoryMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

def historyObservation := "history-event"
def correlatedObservation := "correlated-evidence"

/-! ### The declared evidence source

The correlated capability reads `CorrelatedEvidence` this Program lifts out of the same history read the
Case already performs. A started or a completed Nexus event records the scheduled event it answers
and nothing else that names its operation, so that scheduled event id is the operation key on every
side. -/

def projectionId : DefinitionId := .of "temporal.nexus3.projection"
def evidenceSourceId : DefinitionId := .of "temporal.nexus3.source.history"
def runFieldId : DefinitionId := .of "temporal.nexus3.scope.run"
def operationFieldId : DefinitionId := .of "temporal.nexus3.scope.operation"
def startedEvidenceKindId : DefinitionId := .of "temporal.nexus3.evidence.started"
def completedEvidenceKindId : DefinitionId := .of "temporal.nexus3.evidence.completed"

/-- The one Run coordinate recorded history does not carry: every event this Case lifts belongs to
the single Run it executes, so the Case declares that scope rather than reading it. -/
def runScopeValue := "async-nexus"

private def startedAttributesField := "nexus_operation_started_event_attributes"
private def completedAttributesField := "nexus_operation_completed_event_attributes"

private def evidenceRule (selected : String) (kind : DefinitionId) : CorrelatedEvidenceRule :=
  Program.correlatedEvidenceRule
    (guard := Path.make #[Path.oneofSelector "attributes" selected])
    (source := evidenceSourceId.value)
    (kind := kind.value)
    (operation := historyAttribute selected "scheduled_event_id")
    (scope := #[Program.correlatedEvidenceLiteral runFieldId.value runScopeValue])

private def evidenceTarget : ProjectionTarget :=
  Program.correlatedEvidenceTarget correlatedObservation #[
    evidenceRule startedAttributesField startedEvidenceKindId,
    evidenceRule completedAttributesField completedEvidenceKindId]

private def textOutcome : InstructionOutcomeDefinition :=
  Program.outcome #[
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_STATUS statusType,
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_VALUE textType]

private def rpc
    (id method : String)
    (dependencies : Array InstructionRef)
    (assignments : Array RequestAssignment)
    (projections : Array ResponseProjection)
    (guard : Option ProgramExpression := none)
    (reservations : Array ActivationReservationDefinition := #[]) : InstructionDefinition :=
  Program.node id (Program.invokeRPC workflowServiceRole method assignments projections)
    (bounds 10000 128) dependencies guard (some statusOutcome) reservations

private def historyAssignments : Array RequestAssignment := #[
  Program.environmentAssignment (field "namespace") workerNamespaceBinding,
  assign (nested ["execution", "workflow_id"]) runId,
  assign (field "maximum_page_size") (signedInteger 64),
  assign (field "wait_new_event") (boolean true)
]

private def historyNode : InstructionDefinition :=
  rpc "history" getHistoryMethod #[Ref.instruction "controller" "complete-nexus-operation"]
    historyAssignments #[
    Program.responseProjection historyEvents .PROJECTION_KIND_EMIT_EACH
      #[Program.observationTarget historyObservation, evidenceTarget]
  ] (some (succeeded "controller" "complete-nexus-operation"))

private def program : Program :=
  Program.make "temporal.case.async-nexus.program"
    #[
      Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
      Program.role workerRole .ROLE_KIND_WORKER
        (namespaceBindingId := workerNamespaceBinding),
      Program.role taskQueueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := workerNamespaceBinding) (resourceBindingId := taskQueueBinding),
      Program.role nexusEndpointRole .ROLE_KIND_ENDPOINT
        (resourceBindingId := nexusEndpointBinding)]
    #[Program.capabilitySlot "completion-authority"]
    #[Program.observation historyObservation historyEventType,
      Program.observation correlatedObservation (Types.singular
        (Types.messageType "temporal.server.api.testpilot.v1.CorrelatedEvidence"))]
    #[
      Program.controller "controller" #[
        rpc "start-workflow" startWorkflowMethod #[] #[
          Program.environmentAssignment (field "namespace") workerNamespaceBinding,
          assign (field "workflow_id") runId,
          assign (nested ["workflow_type", "name"]) (text "umpire-async-nexus-workflow"),
          Program.environmentAssignment (nested ["task_queue", "name"]) taskQueueBinding,
          assign (field "request_id") runId
        ] #[] none #[
          Program.reservation "workflow" 1,
          Program.reservation "handler" 1
        ],
        Program.node "await-completion-authority" (Program.awaitSlot "completion-authority")
          (bounds 10000) #[Ref.instruction "controller" "start-workflow"]
          (some (succeeded "controller" "start-workflow")) (some statusOutcome),
        Program.node "complete-nexus-operation"
          (Program.completeNexusOperation "completion-authority" (text "completed"))
          (bounds 10000) #[Ref.instruction "controller" "await-completion-authority"]
          (some (succeeded "controller" "await-completion-authority")) (some statusOutcome),
        historyNode],
      Program.workflow "workflow" "umpire-async-nexus-workflow" workerRole taskQueueRole #[
        Program.node "start-nexus-operation"
          (Program.startNexusOperation nexusEndpointRole "umpire.case.service" "complete"
            (text "request"))
          (bounds 10000) #[] none (some statusOutcome),
        Program.node "await-nexus-operation"
          (Program.awaitInstruction (Ref.instruction "workflow" "start-nexus-operation"))
          (bounds 10000) #[Ref.instruction "workflow" "start-nexus-operation"]
          none (some textOutcome),
        Program.node "finish-workflow"
          (Program.finish (ProgramExpr.outcome
            (Ref.instruction "workflow" "await-nexus-operation")
            .INSTRUCTION_OUTCOME_FIELD_VALUE))
          bounds #[Ref.instruction "workflow" "await-nexus-operation"]
          (some (succeeded "workflow" "await-nexus-operation")) (some statusOutcome)],
      Program.nexusHandler "handler" "umpire.case.service" "complete" workerRole taskQueueRole #[
        Program.node "respond-async"
          (Program.respondNexus .NEXUS_RESPONSE_KIND_ASYNCHRONOUS
            (text "accepted") "completion-authority")
          bounds #[] none (some statusOutcome)]]
    (Program.cleanup "cleanup" #[])
    programLimits
    (environment := #[
      Program.environment workerNamespaceBinding,
      Program.environment taskQueueBinding,
      Program.environment nexusEndpointBinding])

/-! ### The derived correlated Property

A `require` clause says what must hold at the step that selects one Action. The checked Behavior
says where that Action sits in the operation's own sequence. Together they are a bounded response:
from the operation's first selected Action, the required value is due within exactly as many
semantic transitions as the Behavior places between them.

That is what makes the derived Contract discriminating rather than vacuous. A same-step clause
triggered on its own Action would answer satisfied for an operation that never reached the Action
at all, because nothing triggered; triggering on the operation's first Action instead leaves the
obligation open until the operation either reaches the required value or the window closes. -/

private def compilerError (definitionId construct : String) : Error := {
  sourceDefinitionId := definitionId
  source := Authoring.source
  construct
}

/-- The predicate field one modeled trace field names in a same-step predicate environment. A trace
field with no same-step predicate is a clause this lowering cannot express. -/
private def predicateField : PropertyTraceField → Option PropertyPredicateField
  | .priorState => some .priorState
  | .selectedAction => some .selectedAction
  | .resultingState => some .resultingState
  | .outcome => some .outcome
  | .observation => some .expectationFact
  | .state | .relation => none

private def predicateOf (pattern : PropertyPattern) : Option PropertyPredicate := do
  let field ← predicateField pattern.field
  let constraint ← match pattern.constraint with
    | .present => some PropertyAtomConstraint.present
    | .equals value => some (.equals (.text value))
    | _ => none
  pure (.atom { field, reference := pattern.reference, constraint })

/-- Whether one modeled step already carries a pattern's value. -/
private def patternHolds
    (pattern : PropertyPattern)
    (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) : Bool :=
  let carries := fun (value : ModelValue) =>
    pattern.reference == value.definitionId &&
      match pattern.constraint with
      | .present => true
      | .equals expected => expected == value.value
      | _ => false
  match pattern.field with
  | .selectedAction => carries step.selectedAction
  | .resultingState => carries step.state
  | .outcome => carries step.outcome
  | .observation => step.facts.any carries
  | _ => false

/-- One `require` clause as an operation-correlated clause, placed by the checked Behavior. A same-step
clause is the only form with a trigger and a response to carry across; a value constraint the
portable predicate vocabulary has no spelling for, a trigger that is not an Action, and an Action
the Behavior never selects each reject by clause name rather than being narrowed or guessed.

The window is inclusive of its trigger step, so a required value that the selected trace already
carries somewhere before the Action the clause names would answer the clause without that Action
ever being observed. That is the vacuity this trigger choice exists to avoid, so a Property whose
response holds earlier rejects rather than lowering a clause a shorter trace could satisfy. -/
private def correlatedRuleOf
    (occurrences : List DefinitionId) (opening : ModelValue)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (clause : CheckedPropertyClause) : Except Error PropertyCorrelatedClause :=
  let unexpressible := fun construct => Except.error (compilerError clause.id.value construct)
  match clause with
  | .transitionContract id trigger response
  | .inputOutput id trigger response =>
      if trigger.field != .selectedAction then
        unexpressible "property.clause-shape"
      else
        match occurrences.idxOf? trigger.reference, predicateOf response with
        | some bound, some lowered =>
            if (steps.take bound).any (patternHolds response) then
              unexpressible "property.clause-early-response"
            else
              .ok {
                id, source := Authoring.source
                trigger := .selectedActionIs opening, response := lowered
                scope := [runFieldId], key := operationFieldId
                bound, ending := .«partial» }
        | none, _ => unexpressible "property.clause-occurrence"
        | _, none => unexpressible "property.clause-shape"
  | _ => unexpressible "property.clause-form"

/-- The correlated capability's own retention shares the Contract's capture budget, and it retains one
admitted evidence value per obligation rather than one captured message per rule, so this Case
declares its own ceilings instead of the shared single-capture ones. -/
def asyncNexusContractLimits : ContractLimits :=
  { contractLimits with max_captures := 64, max_capture_bytes := 65536 }

/-- Evaluation ceilings for the correlated consumer, separate from the semantic window above. -/
def runLimits : Property.Correlated.Limits :=
  { «transitions» := 16, obligations := 16, work := 100000000, captures := 0 }

def projectionLimits : Case.Projection.Limits := {
  events := 32, buffered := 16, keys := 8, support := 128
  work := 1000000000, eventSize := 512 }

/-- The declared projection. A started Nexus event confirms the model's `awaitStart` step and a
completed one confirms its `awaitSuccess` step; the scheduled event names no step, because the
model's operation is already scheduled when it starts. -/
private def projectionDeclaration
    (vocabulary : Authoring.ModelVocabulary) :
    Case.Projection.Declaration ModelValue ModelValue ModelValue ModelValue := {
  id := projectionId
  scopeFields := [runFieldId]
  operationField := operationFieldId
  sources := [evidenceSourceId]
  rules := [
    { kind := startedEvidenceKindId
      meaning := .confirmed none [(vocabulary.actionAt 0,
        { «state» := vocabulary.stateAt 1, «outcome» := vocabulary.outcomeAt 0
          «facts» := [vocabulary.factAt 0] })] },
    { kind := completedEvidenceKindId
      meaning := .confirmed none [(vocabulary.actionAt 1,
        { «state» := vocabulary.stateAt 2, «outcome» := vocabulary.outcomeAt 1
          «facts» := [vocabulary.factAt 1] })] }]
  «limits» := projectionLimits }

/-- Lower one checked Nexus3 model into a Case. The checked values are carried, never compared
against an expected model: a different Target, Behavior, Query or Property produces different Case
bytes. Only a claim this Producer cannot realize rejects.

`required` names clauses the caller requires the Case to carry, beyond the ones the checked Property
already names. Coverage is always requested explicitly, never left to a default. -/
def produce {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : Authoring.SuccessModel Setup State Action Outcome Fact}
    (checked : Authoring.CheckedModel «model»)
    (required : List DefinitionId := []) :
    Except Error temporal.server.api.testpilot.v1.Case := do
  -- A Case realizes one selected trace, so a Query that verifies rather than selects has no
  -- witness to realize and rejects here. A Known Gap does not admit it.
  let selected ← match checked.witness with
    | some selected => pure selected
    | none => throw (compilerError checked.query.id.value "witness.absent")
  -- The operation's own sequence in trace order: what it does first, and where each later Action
  -- sits after it. Only an `exactly` Behavior fixes that order, and the derivation needs it, so a
  -- Behavior that only bounds occurrences rejects rather than being read in canonical key order.
  let occurrences ← match checked.behavior.actionsExactly with
    | some occurrences => pure occurrences
    | none => throw (compilerError checked.behavior.id.value "behavior.sequence.absent")
  let opening ← match occurrences.head? with
    | some first =>
        match checked.vocabulary.actions.find? fun value => value.definitionId == first with
        | some opening => pure opening
        | none => throw (compilerError first.value "behavior.action.undeclared")
    | none => throw (compilerError checked.behavior.id.value "behavior.sequence.absent")
  let correlatedRules ← checked.property.clauses.mapM
    (correlatedRuleOf occurrences opening selected.trace.steps)
  if correlatedRules.isEmpty then
    throw (compilerError checked.property.id.value "property.clauses.absent")
  let correlatedProperty ← (Property.check (.ofTarget checked.target) ({
      id := checked.property.id
      source := checked.property.source
      version := checked.property.version
      requires := checked.property.requires
      clauses := []
      correlatedRules })).mapError fun _ =>
    compilerError checked.property.id.value "property.correlated-admission"
  let compiled ← (Property.Correlated.compile checked.target correlatedProperty [runFieldId] operationFieldId
    runLimits).mapError fun _ =>
    compilerError checked.property.id.value "property.correlated-compile"
  let setup : List RoleBinding :=
    [{ «role» := «model».operationRoleId, value := checked.vocabulary.stateAt 0 }]
  let plan ← (Case.Projection.check checked.target
    (projectionDeclaration checked.vocabulary) setup (checked.vocabulary.stateAt 0)).mapError
    fun _ => compilerError projectionId.value "projection.admission"
  let lowered ← Umpire.Case.Correlated.lower plan compiled correlatedObservation
    (Case.Projection.Coverage.empty plan)
  compile {
    version := { major := 1 }
    caseId := "temporal.case.async-nexus-success"
    producerId := "temporal.nexus3.testpilot"
    producerVersion := "1"
    definitions := [
      binding checked.target.id.value checked.target.behaviorFingerprint.render .target,
      binding checked.behavior.id.value checked.behavior.behaviorFingerprint.render .«scenario»,
      binding checked.query.id.value checked.query.behaviorFingerprint.render .«query»,
      binding correlatedProperty.id.value correlatedProperty.behaviorFingerprint.render .«property»]
    sources := [checked.target.source, checked.behavior.source, checked.query.source,
      checked.property.source]
    knownGaps := checked.query.authoredKnownGaps.toProvenanceGaps
    program
    contractId := "temporal.case.async-nexus-success.contract"
    properties := [lowered.contractLowering]
    contractLimits := asyncNexusContractLimits
    -- Every clause the model wrote must appear among the lowered ones, so a clause silently lost
    -- between the checked Property and the Contract rejects here, before any Driver I/O. A caller
    -- may name further clauses it requires; one this Case does not carry rejects the same way.
    coverage := { clauses := (correlatedRules.map (·.id) ++ required).eraseDups }
  }

private def checkedCompletion : Except Error (Authoring.CheckedModel lifecycle) :=
  completion.mapError fun _ =>
    compilerError "temporal.nexus3.query.completion" "checked-completion"

/-- The checked Nexus3 completion declaration lowered to the closed Case format. -/
def completionCase : Except Error temporal.server.api.testpilot.v1.Case := do
  produce (← checkedCompletion)

end Temporal.Feature.Nexus3.Testpilot
