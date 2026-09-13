import Temporal.Case.Evidence
import Temporal.Case.Support

/-!
# The `nexusOperation` realization

A controller-started workflow schedules one Nexus operation on the Case's endpoint role; the
handler responds synchronously, or asynchronously with the controller completing it; the controller
reads the history the Case's evidence is lifted from.

The two response forms differ in what sequences the history read. In the asynchronous form the
controller completes the operation itself, so the read simply depends on that completion. In the
synchronous form nothing in the controller observes the workflow at all, and
`GetWorkflowExecutionHistory` with `wait_new_event` returns the first page as soon as any event
exists -- so the read that the Projection consumes is preceded by a read with the close-event
filter, which cannot resolve until the workflow closes.
-/

namespace Temporal.Case.Template

open Umpire
open Temporal.Case.Support
open Temporal.Testpilot.CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

/-- How the Nexus handler answers the operation. -/
inductive Response where
  | sync
  | async
  deriving BEq, DecidableEq, Repr

namespace NexusOperation

def workerNamespaceBinding := "temporal.worker.namespace"
def taskQueueBinding := "temporal.task-queue.resource"
def nexusEndpointBinding := "temporal.nexus-endpoint.resource"

/-! ### Coordinates

These Definition IDs name the coordinates every Case on this template reads its evidence by: the
Run scope, the operation key, the projection, the evidence source, and one kind per admitted
history event. They are the template's, not one feature's, and they are the values the checked-in
async-Nexus fixture already carries. -/

def projectionId : DefinitionId := .of "temporal.nexus.success.projection"
def evidenceSourceId : DefinitionId := .of "temporal.nexus.success.source.history"
def runFieldId : DefinitionId := .of "temporal.nexus.success.scope.run"
def operationFieldId : DefinitionId := .of "temporal.nexus.success.scope.operation"
def startedEvidenceKindId : DefinitionId := .of "temporal.nexus.success.evidence.started"
def completedEvidenceKindId : DefinitionId := .of "temporal.nexus.success.evidence.completed"

/-- A started or a completed Nexus event records the scheduled event it answers and nothing else
that names its operation, so that scheduled event id is the operation key on every side. -/
private def scheduledEventKey : String := "scheduled_event_id"

private def historySource (kind : String) (kindId : DefinitionId) :
    Umpire.Case.Producer.EvidenceSource :=
  let attributes := (EventKind.attributesField? kind).getD kind
  { eventKind := kind
    attributesField := attributes
    operationKeyPath := historyAttribute attributes scheduledEventKey
    kindId
    sourceId := evidenceSourceId }

def startedSource : Umpire.Case.Producer.EvidenceSource :=
  historySource "nexusOperationStarted" startedEvidenceKindId

def completedSource : Umpire.Case.Producer.EvidenceSource :=
  historySource "nexusOperationCompleted" completedEvidenceKindId

/-! ### The Program -/

private def textOutcome : InstructionOutcomeDefinition :=
  Program.outcome #[
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_STATUS statusType,
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_VALUE textType]

private def rpc
    (id method : String)
    (assignments : Array RequestAssignment)
    (projections : Array ResponseRead)
    (reservations : Array ActivationReservationDefinition := #[]) : InstructionNode :=
  Program.node id (Program.invokeRpc workflowServiceRole method assignments projections)
    (Program.instructionLimits 10000 1) (outcome := some statusOutcome) (reservations := reservations)

private def historyAssignments : Array RequestAssignment := #[
  Program.environmentAssignment (field "namespace") workerNamespaceBinding,
  assign (nested ["execution", "workflow_id"]) runId,
  assign (field "maximum_page_size") (signedInteger 64),
  assign (field "wait_new_event") (boolean true)
]

private def closeReadAssignments : Array RequestAssignment :=
  historyAssignments.push (assign (field "history_event_filter_type") closeEventFilter)

private def startWorkflowNode (workflowType : String) : InstructionNode :=
  rpc "start-workflow" startWorkflowMethod #[
    Program.environmentAssignment (field "namespace") workerNamespaceBinding,
    assign (field "workflow_id") runId,
    assign (nested ["workflow_type", "name"]) (text workflowType),
    Program.environmentAssignment (nested ["task_queue", "name"]) taskQueueBinding,
    assign (field "request_id") runId
  ] #[] #[
    Program.reservation "workflow" 1,
    Program.reservation "handler" 1
  ]

/-- The full history read, run once the instruction before it succeeded. -/
private def historyNode
    (identity : Umpire.Case.Producer.Identity)
    (resolved : List Umpire.Case.Producer.EvidenceRule) : InstructionNode :=
  rpc "history" getHistoryMethod historyAssignments #[
    Program.responseRead historyEvents .READ_CARDINALITY_EMIT_EACH
      #[Program.observationTarget historyObservation,
        Evidence.target runFieldId correlatedObservation identity resolved]
  ]

private def workflowEntrypoint
    (workflowType service operation : String) : Entrypoint :=
  Program.workflow "workflow" workflowType workerRole taskQueueRole #[
    Program.node "start-nexus-operation"
      (Program.startNexusOperation nexusEndpointRole service operation (text "request"))
      (Program.instructionLimits 10000 1) (outcome := some statusOutcome),
    -- The await runs whether or not the start succeeded, so the operation's outcome is recorded.
    Program.node "await-nexus-operation"
      (Program.awaitInstruction (Ref.instruction "workflow" "start-nexus-operation"))
      (Program.instructionLimits 10000 1) (guard := some (boolean true)) (outcome := some textOutcome),
    Program.node "finish-workflow"
      (Program.finish (Expr.outcome
        (Ref.instruction "workflow" "await-nexus-operation")
        .INSTRUCTION_OUTCOME_FIELD_VALUE))
      (Program.instructionLimits 5000 1) (outcome := some statusOutcome)]

private def asyncProgram
    (service operation : String)
    (identity : Umpire.Case.Producer.Identity)
    (resolved : List Umpire.Case.Producer.EvidenceRule) : Program :=
  let workflowType := "umpire-" ++ identity.fixture ++ "-workflow"
  Program.make identity.programId
    #[
      Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
      Program.role workerRole .ROLE_KIND_WORKER
        (namespaceBindingId := workerNamespaceBinding),
      Program.role taskQueueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := workerNamespaceBinding) (resourceBindingId := taskQueueBinding),
      Program.role nexusEndpointRole .ROLE_KIND_ENDPOINT
        (resourceBindingId := nexusEndpointBinding)]
    #[Program.handleSlot "completion-authority"]
    #[Program.observation historyObservation historyEventType,
      Program.observation correlatedObservation correlatedEvidenceType]
    #[
      Program.controller "controller" #[
        startWorkflowNode workflowType,
        Program.node "await-completion-authority" (Program.awaitSlot "completion-authority")
          (Program.instructionLimits 10000 1) (outcome := some statusOutcome),
        Program.node "complete-nexus-operation"
          (Program.completeNexusOperation "completion-authority" (text "completed"))
          (Program.instructionLimits 10000 1) (outcome := some statusOutcome),
        historyNode identity resolved],
      workflowEntrypoint workflowType service operation,
      Program.nexusHandler "handler" service operation workerRole taskQueueRole #[
        Program.node "respond-async"
          (Program.respondNexus .NEXUS_RESPONSE_KIND_ASYNCHRONOUS
            (text "accepted") "completion-authority")
          (Program.instructionLimits 5000 1) (outcome := some statusOutcome)]]
    (Program.cleanup "cleanup" #[])
    (environment := #[
      Program.environment workerNamespaceBinding,
      Program.environment taskQueueBinding,
      Program.environment nexusEndpointBinding])

private def syncProgram
    (service operation : String)
    (identity : Umpire.Case.Producer.Identity)
    (resolved : List Umpire.Case.Producer.EvidenceRule) : Program :=
  let workflowType := "umpire-" ++ identity.fixture ++ "-workflow"
  Program.make identity.programId
    #[
      Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
      Program.role workerRole .ROLE_KIND_WORKER
        (namespaceBindingId := workerNamespaceBinding),
      Program.role taskQueueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := workerNamespaceBinding) (resourceBindingId := taskQueueBinding),
      Program.role nexusEndpointRole .ROLE_KIND_ENDPOINT
        (resourceBindingId := nexusEndpointBinding)]
    #[]
    #[Program.observation historyObservation historyEventType,
      Program.observation correlatedObservation correlatedEvidenceType]
    #[
      Program.controller "controller" #[
        startWorkflowNode workflowType,
        -- Nothing else in the controller observes the workflow, so this close-event read is what
        -- orders the full read after the operation completed.
        rpc "await-close" getHistoryMethod closeReadAssignments #[],
        historyNode identity resolved],
      workflowEntrypoint workflowType service operation,
      Program.nexusHandler "handler" service operation workerRole taskQueueRole #[
        Program.node "respond-sync"
          (Program.respondNexus .NEXUS_RESPONSE_KIND_SYNCHRONOUS (text "completed"))
          (Program.instructionLimits 5000 1) (outcome := some statusOutcome)]]
    (Program.cleanup "cleanup" #[])
    (environment := #[
      Program.environment workerNamespaceBinding,
      Program.environment taskQueueBinding,
      Program.environment nexusEndpointBinding])

end NexusOperation

/-- One Nexus operation, scheduled by a controller-started workflow and answered by a handler that
runs inside the Case's own worker. -/
def nexusOperation (service operation : String) (responds : Response) :
    Umpire.Case.Producer.Realization :=
  let completionHook := match responds with
    | .async => "complete-nexus-operation"
    | .sync => "await-close"
  { program := match responds with
      | .async => NexusOperation.asyncProgram service operation
      | .sync => NexusOperation.syncProgram service operation
    producerId := "temporal.nexus.success.testpilot"
    producerVersion := "1"
    projectionId := NexusOperation.projectionId
    scopeField := NexusOperation.runFieldId
    operationKey := NexusOperation.operationFieldId
    historyObservation := Support.historyObservation
    correlatedObservation := Support.correlatedObservation
    taskQueueRole := Support.taskQueueRole
    -- Named per response form. It reaches a Case only through the ordering rule the Producer adds
    -- when a Scenario carries `fault` lines; a Case with none never carries it.
    faultRuleId := match responds with
      | .async => "async-nexus-order"
      | .sync => "sync-nexus-order"
    hooks := [
      { name := "start", instruction := Ref.instruction "controller" "start-workflow" },
      { name := "completion", instruction := Ref.instruction "controller" completionHook }]
    sources := [NexusOperation.startedSource, NexusOperation.completedSource]
    projectionLimits := {
      events := 32, buffered := 16, keys := 8, support := 128
      work := 1000000000, eventSize := 512 }
    runLimits := {
      «transitions» := 16, obligations := 16, work := 100000000, captures := 0 } }

end Temporal.Case.Template
