import Temporal.Feature.Nexus.Success.Model
import Temporal.Testpilot.CaseSupport
import Umpire.Case.Producer

/-!
# Nexus.Success Testpilot producer

The checked model is carried into a Case; it is never compared against an expected one. Everything
this file still owns is Temporal: the Program the Case runs, the history attributes its evidence is
lifted from, and the Definition IDs those coordinates carry. The derivation itself — correlated
clauses from `require` lines, the projection from the checked Machine along the witness trace,
provenance, and coverage — lives in `Umpire.Case.Producer` and is shared by every Case.

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
-/

namespace Temporal.Feature.Nexus.Success.Producer

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

def projectionId : DefinitionId := .of "temporal.nexus.success.projection"
def evidenceSourceId : DefinitionId := .of "temporal.nexus.success.source.history"
def runFieldId : DefinitionId := .of "temporal.nexus.success.scope.run"
def operationFieldId : DefinitionId := .of "temporal.nexus.success.scope.operation"
def startedEvidenceKindId : DefinitionId := .of "temporal.nexus.success.evidence.started"
def completedEvidenceKindId : DefinitionId := .of "temporal.nexus.success.evidence.completed"

private def startedAttributesField := "nexus_operation_started_event_attributes"
private def completedAttributesField := "nexus_operation_completed_event_attributes"

/-- The two history event kinds this realization admits. An `evidence` line naming anything else
rejects against this list. -/
def startedSource : Case.Producer.EvidenceSource := {
  eventKind := "nexusOperationStarted"
  attributesField := startedAttributesField
  operationKeyPath := historyAttribute startedAttributesField "scheduled_event_id"
  kindId := startedEvidenceKindId
  sourceId := evidenceSourceId }

def completedSource : Case.Producer.EvidenceSource := {
  eventKind := "nexusOperationCompleted"
  attributesField := completedAttributesField
  operationKeyPath := historyAttribute completedAttributesField "scheduled_event_id"
  kindId := completedEvidenceKindId
  sourceId := evidenceSourceId }

private def evidenceRule
    (identity : Case.Producer.Identity) (rule : Case.Producer.EvidenceRule) :
    CorrelatedEvidenceRule :=
  Program.correlatedEvidenceRule
    (guard := Path.make #[Path.oneofSelector "attributes" rule.source.attributesField])
    (source := rule.source.sourceId.value)
    (kind := rule.source.kindId.value)
    (operation := rule.source.operationKeyPath)
    (scope := #[Program.correlatedEvidenceLiteral runFieldId.value identity.runScope])

private def evidenceTarget
    (identity : Case.Producer.Identity) (rules : List Case.Producer.EvidenceRule) :
    ProjectionTarget :=
  Program.correlatedEvidenceTarget correlatedObservation
    (rules.map (evidenceRule identity)).toArray

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

private def historyNode
    (identity : Case.Producer.Identity) (rules : List Case.Producer.EvidenceRule) :
    InstructionDefinition :=
  rpc "history" getHistoryMethod #[Ref.instruction "controller" "complete-nexus-operation"]
    historyAssignments #[
    Program.responseProjection historyEvents .PROJECTION_KIND_EMIT_EACH
      #[Program.observationTarget historyObservation, evidenceTarget identity rules]
  ] (some (succeeded "controller" "complete-nexus-operation"))

private def program
    (identity : Case.Producer.Identity) (rules : List Case.Producer.EvidenceRule) : Program :=
  Program.make identity.programId
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
        historyNode identity rules],
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

/-- The async Nexus realization: one controller-started workflow schedules one Nexus operation the
handler answers asynchronously, and the controller reads the history back. -/
def realization : Case.Producer.Realization := {
  program
  producerId := "temporal.nexus.success.testpilot"
  producerVersion := "1"
  projectionId
  scopeField := runFieldId
  operationKey := operationFieldId
  historyObservation
  correlatedObservation
  taskQueueRole
  faultRuleId := "async-nexus-order"
  hooks := [
    { name := "start", instruction := Ref.instruction "controller" "start-workflow" },
    { name := "completion",
      instruction := Ref.instruction "controller" "complete-nexus-operation" }]
  sources := [startedSource, completedSource]
  contractLimits := asyncNexusContractLimits
  projectionLimits
  runLimits }

/-- The identity today's checked-in fixture carries. The Program ID predates the fixture-derived
convention, so it is stated rather than derived; every other identity is the derivation. -/
def identity : Case.Producer.Identity := {
  caseId := "temporal.case.async-nexus-success"
  fixture := "async-nexus"
  programId := "temporal.case.async-nexus.program" }

/-- Which recorded history event confirms each Action the Scenario selects. Both Actions are named
by their declared spelling, so a Model that renames one rejects at production rather than silently
mapping the wrong event. -/
def evidence (vocabulary : Case.Producer.Vocabulary) : List Case.Producer.EvidenceMapping := [
  { «action» := vocabulary.namedAction "awaitStart", eventKind := startedSource.eventKind },
  { «action» := vocabulary.namedAction "awaitSuccess",
    eventKind := completedSource.eventKind }]

/-- The checked Nexus.Success authoring bundle as the Umpire-owned Producer input. A `.umpire`
module may not import this namespace, so the conversion lives here. -/
def producerInput {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : Authoring.SuccessModel Setup State Action Outcome Fact}
    (checked : Authoring.CheckedModel «model») :
    Case.Producer.Input «model».lawStatement := {
  target := checked.target
  vocabulary := {
    «states» := checked.vocabulary.states
    «actions» := checked.vocabulary.actions
    «outcomes» := checked.vocabulary.outcomes
    «facts» := checked.vocabulary.facts }
  «property» := checked.property
  «scenario» := checked.behavior
  «witness» := checked.witness
  operationRole := «model».operationRoleId
  queryId := checked.query.id
  querySource := checked.query.source
  queryFingerprint := checked.query.behaviorFingerprint.render
  knownGaps := checked.query.authoredKnownGaps
  source := Authoring.source }

/-- Lower one checked Nexus.Success model into a Case. The checked values are carried, never compared
against an expected model: a different Target, Behavior, Query or Property produces different Case
bytes. Only a claim this Producer cannot realize rejects.

`required` names clauses the caller requires the Case to carry, beyond the ones the checked Property
already names. Coverage is always requested explicitly, never left to a default. -/
def produce {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : Authoring.SuccessModel Setup State Action Outcome Fact}
    (checked : Authoring.CheckedModel «model»)
    (required : List DefinitionId := []) :
    Except Error temporal.server.api.testpilot.v1.Case :=
  let input := producerInput checked
  Case.Producer.produce input identity realization (evidence input.vocabulary) required

private def compilerError (definitionId construct : String) : Error := {
  sourceDefinitionId := definitionId
  source := Authoring.source
  construct
}

private def checkedCompletion : Except Error (Authoring.CheckedModel lifecycle) :=
  completion.mapError fun _ =>
    compilerError "temporal.nexus.success.query.completion" "checked-completion"

/-- The checked Nexus.Success completion declaration lowered to the closed Case format. -/
def completionCase : Except Error temporal.server.api.testpilot.v1.Case := do
  produce (← checkedCompletion)

end Temporal.Feature.Nexus.Success.Producer
