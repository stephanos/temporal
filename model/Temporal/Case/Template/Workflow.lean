import Temporal.Case.Evidence
import Temporal.Case.EventKind

/-!
# The `workflow` realization

A controller-started workflow that finishes; the controller reads the history back. This is the
smallest shape a Case can have, and it is the one a fault line disturbs: with nothing else in
flight, a stop before the start and a resume after it are exactly the outage the Run must survive.

The read is filtered to the close event, so it cannot resolve before the workflow completed --
which, with no worker polling the queue until the resume, is what proves the queued task survived
the outage rather than being lost with the worker.

One Run runs one workflow, so a completed-workflow event names no operation. The operation key is
therefore the event's own id: a stable path on the single close event, because
`ScopedEvidenceRule.operation` is always a path and never a literal.
-/

namespace Temporal.Case.Template

open Umpire
open Temporal.Testpilot.CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

namespace Workflow

def workflowServiceRole := "temporal.workflow-service"
def workerRole := "temporal.worker"
def taskQueueRole := "temporal.task-queue"
def namespaceBinding := "temporal.workflow.namespace"
def taskQueueBinding := "temporal.workflow.task-queue"

private def startWorkflowMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
private def getHistoryMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

def historyObservation := "history-event"
def correlatedObservation := "correlated-evidence"

def projectionId : DefinitionId := .of "temporal.case.workflow.projection"
def evidenceSourceId : DefinitionId := .of "temporal.case.workflow.source.history"
def runFieldId : DefinitionId := .of "temporal.case.workflow.scope.run"
def operationFieldId : DefinitionId := .of "temporal.case.workflow.scope.operation"
def completedEvidenceKindId : DefinitionId := .of "temporal.case.workflow.evidence.completed"

/-- The single close event's own id: one Run runs one workflow, so this names the operation. -/
private def eventKey : FieldPath := field "event_id"

def completedSource : Umpire.Case.Producer.EvidenceSource :=
  let kind := "workflowExecutionCompleted"
  { eventKind := kind
    attributesField := (EventKind.attributesField? kind).getD kind
    operationKeyPath := eventKey
    kindId := completedEvidenceKindId
    sourceId := evidenceSourceId }

/-- `HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT`. -/
private def closeEventFilter : ProgramExpression :=
  ProgramExpr.literal (Value.enumeration 2)

private def historyAssignments : Array RequestAssignment := #[
  Program.environmentAssignment (field "namespace") namespaceBinding,
  assign (nested ["execution", "workflow_id"]) runId,
  assign (field "maximum_page_size") (signedInteger 64),
  assign (field "wait_new_event") (boolean true),
  assign (field "history_event_filter_type") closeEventFilter
]

private def program
    (workflowType : String)
    (identity : Umpire.Case.Producer.Identity)
    (resolved : List Umpire.Case.Producer.EvidenceRule) : Program :=
  Program.make identity.programId
    #[Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
      Program.role workerRole .ROLE_KIND_WORKER
        (namespaceBindingId := namespaceBinding),
      Program.role taskQueueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := namespaceBinding)
        (resourceBindingId := taskQueueBinding)]
    #[]
    #[Program.observation historyObservation historyEventType,
      Program.observation correlatedObservation (Types.singular
        (Types.messageType "temporal.server.api.testpilot.v1.CorrelatedEvidence"))]
    #[Program.controller "controller" #[
        Program.node "start-workflow"
          (Program.invokeRPC workflowServiceRole startWorkflowMethod #[
            Program.environmentAssignment (field "namespace") namespaceBinding,
            assign (field "workflow_id") runId,
            assign (nested ["workflow_type", "name"]) (text workflowType),
            Program.environmentAssignment (nested ["task_queue", "name"]) taskQueueBinding,
            assign (field "request_id") runId])
          (bounds 10000) #[] none (some statusOutcome)
          #[Program.reservation "workflow" 1],
        Program.node "history"
          (Program.invokeRPC workflowServiceRole getHistoryMethod historyAssignments
            #[Program.responseProjection historyEvents .PROJECTION_KIND_EMIT_EACH
              #[Program.observationTarget historyObservation,
                Evidence.target runFieldId correlatedObservation identity resolved]])
          (Program.instructionLimits 20000 1 64 8192)
          #[Ref.instruction "controller" "start-workflow"]
          (some (succeeded "controller" "start-workflow")) (some statusOutcome)],
      Program.workflow "workflow" workflowType workerRole taskQueueRole #[
        Program.node "finish-workflow" (Program.finish (text "completed"))
          (bounds 10000) #[] none (some statusOutcome)]]
    (Program.cleanup "cleanup" #[])
    { programLimits with max_run_events := 512, max_response_bytes := 8192 }
    (environment := #[
      Program.environment namespaceBinding,
      Program.environment taskQueueBinding])

end Workflow

/-- One controller-started workflow that finishes, with the history read the Case's evidence is
lifted from. Its fault rule ID keeps the outage-order spelling the checked-in artifact asserts. -/
def workflow (workflowType : String) : Umpire.Case.Producer.Realization := {
  program := Workflow.program workflowType
  producerId := "temporal.case.workflow"
  producerVersion := "1"
  projectionId := Workflow.projectionId
  scopeField := Workflow.runFieldId
  operationKey := Workflow.operationFieldId
  historyObservation := Workflow.historyObservation
  correlatedObservation := Workflow.correlatedObservation
  taskQueueRole := Workflow.taskQueueRole
  faultRuleId := "worker-outage-order"
  hooks := [
    { name := "start", instruction := Ref.instruction "controller" "start-workflow" },
    { name := "completion", instruction := Ref.instruction "controller" "history" }]
  sources := [Workflow.completedSource]
  contractLimits := { contractLimits with max_captures := 64, max_capture_bytes := 65536 }
  projectionLimits := {
    events := 32, buffered := 16, keys := 8, support := 128
    work := 1000000000, eventSize := 512 }
  runLimits := {
    «transitions» := 16, obligations := 16, work := 100000000, captures := 0 } }

end Temporal.Case.Template
