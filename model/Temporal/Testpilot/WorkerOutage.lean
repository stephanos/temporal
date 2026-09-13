import Temporal.Testpilot.CaseSupport
import Umpire.Case.Compiler
import Umpire.Variations.Lowering

/-!
# A deliberate worker outage, end to end

A Producer-neutral functional Case that asks the Driver for one real outage and then requires the
work to survive it. The controller stops the SDK worker of the Case's own activation queue, starts
the workflow while nothing is polling that queue, resumes the worker, and reads the history the
workflow then records.

Two independent requirements sit on that Run:

* The **outage order** is a bounded-liveness rule over the recorded `FAULT_INJECTED` events. It
  reaches its satisfied state only after a stop on this Case's task-queue role is followed by a
  resume on the same role. Its deadline is an **event count**, not an elapsed-time deadline: a
  Driver's outage window is measured in what the Run recorded, and a slow CI runner must not be able
  to turn a healthy outage into a violated one.
* The **workflow completion** is a safety rule over the history the controller reads back. The
  workflow task that completes the workflow can only have been dispatched after the resume, because
  no worker was polling the queue before it, so a completed workflow is what proves the queued task
  survived the outage rather than being lost with the worker.

The stop precedes `start-workflow`, so no workflow task is in flight when the worker stops. The
reservation ledger mints activation handles from the prepared Program rather than from SDK polling,
so the workflow entrypoint's handle exists before any worker could have claimed it.

This Case is realization only: it carries no checked model, exactly like the system-info fixture, so
EVD-18's six conformance classes are untouched.
-/

namespace Temporal.Testpilot

open CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

def workerOutageServiceRole := "temporal.workflow-service"
def workerOutageWorkerRole := "temporal.worker"
def workerOutageQueueRole := "temporal.task-queue"
def workerOutageNamespaceBinding := "temporal.worker-outage.namespace"
def workerOutageQueueBinding := "temporal.worker-outage.task-queue"
def workerOutageWorkflowType := "umpire-worker-outage-workflow"
def workerOutageObservation := "history-event"

private def startWorkflowMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
private def getHistoryMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

/-- The completed-workflow attribute the success rule reads. Every completed workflow records the
workflow task that completed it, so its presence is exactly "this workflow completed". -/
private def completedAttributes := "workflow_execution_completed_event_attributes"
private def completedTaskField := "workflow_task_completed_event_id"

/-! ### The two fault intents, and the instructions they lower to

The Case's outage is authored as `Umpire` fault intent, not as a hand-written instruction: the
declaration says which occurrence is being disturbed and which capability the Driver realizes, and
`FaultIntentDeclaration.lower` produces the instruction. Only the placement is realization. -/

private def faultSource : Umpire.SourceLocation := {
  path := "Temporal/Testpilot/WorkerOutage.lean", line := 1, column := 1
  provenance := "worker-outage-fault"
}

private def outageOccurrence : Umpire.DefinitionId :=
  .of "temporal.case.worker-outage.occurrence.start-workflow"
private def outageAction : Umpire.DefinitionId :=
  .of "temporal.case.worker-outage.action.start-workflow"

def stopIntent : Umpire.FaultIntentDeclaration :=
  Umpire.FaultIntentDeclaration.atOccurrence
    (.of "temporal.case.worker-outage.fault.stop") faultSource
    outageOccurrence outageAction Umpire.workerStopCapabilityId

def resumeIntent : Umpire.FaultIntentDeclaration :=
  Umpire.FaultIntentDeclaration.atOccurrence
    (.of "temporal.case.worker-outage.fault.resume") faultSource
    outageOccurrence outageAction Umpire.workerResumeCapabilityId

def stopRealization : Umpire.FaultRealization :=
  { instructionId := "stop-worker", roleId := workerOutageQueueRole }

def resumeRealization : Umpire.FaultRealization :=
  { instructionId := "resume-worker", roleId := workerOutageQueueRole }

/-- `HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT`. The read blocks until the workflow closes and returns
only the closing event, so the success rule reads the one event that proves the queued task ran --
and the read cannot resolve before the resumed worker executed it. -/
private def closeEventFilter : Expression :=
  Expr.literal (Value.enumeration "HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT")

private def historyAssignments : Array RequestAssignment := #[
  Program.environmentAssignment (field "namespace") workerOutageNamespaceBinding,
  assign (nested ["execution", "workflow_id"]) runId,
  assign (field "maximum_page_size") (signedInteger 64),
  assign (field "wait_new_event") (boolean true),
  assign (field "history_event_filter_type") closeEventFilter
]

private def program (stop resume : InstructionNode) : Program :=
  Program.make "temporal.case.worker-outage.program"
    #[Program.role workerOutageServiceRole .ROLE_KIND_ENDPOINT,
      Program.role workerOutageWorkerRole .ROLE_KIND_WORKER
        (namespaceBindingId := workerOutageNamespaceBinding),
      Program.role workerOutageQueueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := workerOutageNamespaceBinding)
        (resourceBindingId := workerOutageQueueBinding)]
    #[]
    #[Program.observation workerOutageObservation historyEventType]
    #[Program.controller "controller" #[
        stop,
        Program.node "start-workflow"
          (Program.invokeRpc workerOutageServiceRole startWorkflowMethod #[
            Program.environmentAssignment (field "namespace") workerOutageNamespaceBinding,
            assign (field "workflow_id") runId,
            assign (nested ["workflow_type", "name"]) (text workerOutageWorkflowType),
            Program.environmentAssignment (nested ["task_queue", "name"]) workerOutageQueueBinding,
            assign (field "request_id") runId]),
        resume,
        Program.node "history"
          (Program.invokeRpc workerOutageServiceRole getHistoryMethod historyAssignments
            #[project historyEvents workerOutageObservation .READ_CARDINALITY_EMIT_EACH])
          (Program.instructionLimits (timeoutMilliseconds := some 20000))],
      Program.workflow "workflow" workerOutageWorkflowType workerOutageWorkerRole
        workerOutageQueueRole #[
        Program.node "finish-workflow" (Program.finish (text "completed"))]]
    (Program.cleanup "cleanup" #[])

/-! ### The Contract -/

/-- One field of the recorded fault, read through the Run Event payload. Both transitions filter on
`RUN_EVENT_KIND_FAULT_INJECTED`, which always carries the fault, so the read needs no presence
guard. -/
private def faultField (name : String) : Expression :=
  Expr.path Expr.runEventPayload (nested ["fault_injected", name])

private def faultRoleIs : Expression :=
  Expr.equal (faultField "role_id")
    (Expr.literal (Value.text workerOutageQueueRole))

/-- The value name one fault kind carries. The generated Lean enum is a bare inductive with no name
accessor, so the constructor and its name are paired here once and read from here twice; an unknown
number is spelled in decimal, as the runtime spells a number its enum does not declare. The match
names every declared kind: a vocabulary addition is a Lean error here rather than a predicate
silently comparing against an unnamed kind. -/
private def faultKindName : FaultKind → String
  | .FAULT_KIND_WORKER_STOP => "FAULT_KIND_WORKER_STOP"
  | .FAULT_KIND_WORKER_RESUME => "FAULT_KIND_WORKER_RESUME"
  | .FAULT_KIND_UNSPECIFIED => "FAULT_KIND_UNSPECIFIED"
  | .«Unknown.Value» number => toString number

private def faultKindIs (kind : FaultKind) : Expression :=
  Expr.equal (faultField "kind")
    (Expr.literal (Value.enumeration (faultKindName kind)))

/-- The outage window, counted in evaluated Run Events rather than on the host's clock. The counter
restarts when the stop transitions the rule, so what it bounds is the distance from the stop to the
resume: `start-workflow` started and completed, the resumed workflow activation's own records, and
`resume-worker` started, completed and its fault event. That is about ten on the traced path, and
expiry outranks the satisfying event, so the real requirement is a resume within fifteen. Sixteen is
that bound with deliberate slack, not a measured maximum. -/
def workerOutageDeadline : Int64 := 16

private def outageOrderRule : ContractRule :=
  Contract.rule "worker-outage-order" .CONTRACT_RULE_KIND_BOUNDED_LIVENESS "awaiting-stop"
    #[Contract.state "awaiting-stop" .CONTRACT_STATE_STATUS_PENDING,
      Contract.state "stopped" .CONTRACT_STATE_STATUS_PENDING,
      Contract.state "resumed" .CONTRACT_STATE_STATUS_SATISFIED,
      Contract.state "expired" .CONTRACT_STATE_STATUS_VIOLATED]
    #[Contract.transition "observe-stop" "awaiting-stop" "stopped"
        #[.RUN_EVENT_KIND_FAULT_INJECTED]
        (Expr.all #[faultRoleIs, faultKindIs .FAULT_KIND_WORKER_STOP])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
      Contract.transition "observe-resume" "stopped" "resumed"
        #[.RUN_EVENT_KIND_FAULT_INJECTED]
        (Expr.all #[faultRoleIs, faultKindIs .FAULT_KIND_WORKER_RESUME])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]
    (deadline := some (Contract.deadline (.rule_events workerOutageDeadline) "expired"))

private def workflowCompletedRule : ContractRule :=
  Contract.rule "worker-outage-workflow-completed" .CONTRACT_RULE_KIND_SAFETY "pending"
    #[Contract.state "pending" .CONTRACT_STATE_STATUS_PENDING,
      Contract.state "completed" .CONTRACT_STATE_STATUS_SATISFIED]
    #[Contract.transition "observe-workflow-completed" "pending" "completed"
        #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
        (Expr.all #[
          Expr.present (observed workerOutageObservation),
          Expr.present (projected (observed workerOutageObservation)
            (historyAttribute completedAttributes completedTaskField))])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]

private def outageProperty :=
  binding "temporal.case.worker-outage.property.outage-survived"
    "temporal-case-worker-outage-property/v1" .property

/-- The checked-in `rule_events` Case: one deliberate outage, ordered and survived. -/
def workerOutageCase : Except Umpire.Case.Compiler.Error Case := do
  let stop ← stopIntent.lower stopRealization
  let resume ← resumeIntent.lower resumeRealization
  Umpire.Case.Compiler.compile {
    version := { major := 1 }
    caseId := "temporal.case.worker-outage"
    producerId := "temporal.case.compiler"
    producerVersion := "1"
    definitions := [
      binding "temporal.workflow-service" "temporal-workflow-service/v1" .target,
      outageProperty]
    sources := [source, faultSource]
    knownGaps := []
    program := program stop resume
    contractId := "temporal.case.worker-outage.contract"
    properties := [.monitor outageProperty outageOrderRule,
      .monitor outageProperty workflowCompletedRule]
  }

end Temporal.Testpilot
