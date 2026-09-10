import Temporal.Testpilot.CaseSupport
import Umpire.Case.Compiler
import Umpire.Space.Lowering

/-!
# A deliberate worker outage, end to end

A Producer-neutral functional Case that asks the Driver for one real outage and then requires the
work to survive it. The controller stops the SDK worker of the Case's own activation queue, starts
the workflow while nothing is polling that queue, resumes the worker, and reads the history the
workflow then records.

Two independent requirements sit on that Run:

* The **outage order** is a bounded-liveness rule over the recorded `FAULT_INJECTED` events. It
  reaches its satisfied state only after a stop on this Case's task-queue role is followed by a
  resume on the same role. Its horizon is an **event count**, not an elapsed-time deadline: a
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
  { instructionId := "stop-worker", roleId := workerOutageQueueRole, limits := bounds 10000
    outcome := some statusOutcome }

def resumeRealization : Umpire.FaultRealization :=
  { instructionId := "resume-worker", roleId := workerOutageQueueRole, limits := bounds 10000
    dependencies := #[Ref.instruction "controller" "start-workflow"]
    guard := some (succeeded "controller" "start-workflow")
    outcome := some statusOutcome }

/-- `HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT`. The read blocks until the workflow closes and returns
only the closing event, so the success rule reads the one event that proves the queued task ran --
and the read cannot resolve before the resumed worker executed it. -/
private def closeEventFilter : ProgramExpression :=
  ProgramExpr.literal (Value.enumeration 2)

private def historyAssignments : Array RequestAssignment := #[
  Program.environmentAssignment (field "namespace") workerOutageNamespaceBinding,
  assign (nested ["execution", "workflow_id"]) runId,
  assign (field "maximum_page_size") (signedInteger 64),
  assign (field "wait_new_event") (boolean true),
  assign (field "history_event_filter_type") closeEventFilter
]

private def program (stop resume : InstructionDefinition) : Program :=
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
          (Program.invokeRPC workerOutageServiceRole startWorkflowMethod #[
            Program.environmentAssignment (field "namespace") workerOutageNamespaceBinding,
            assign (field "workflow_id") runId,
            assign (nested ["workflow_type", "name"]) (text workerOutageWorkflowType),
            Program.environmentAssignment (nested ["task_queue", "name"]) workerOutageQueueBinding,
            assign (field "request_id") runId])
          (bounds 10000) #[Ref.instruction "controller" "stop-worker"]
          (some (succeeded "controller" "stop-worker")) (some statusOutcome)
          #[Program.reservation "workflow" 1],
        resume,
        Program.node "history"
          (Program.invokeRPC workerOutageServiceRole getHistoryMethod historyAssignments
            #[project historyEvents workerOutageObservation .PROJECTION_KIND_EMIT_EACH])
          (Program.instructionLimits 20000 1 64 8192)
          #[Ref.instruction "controller" "resume-worker"]
          (some (succeeded "controller" "resume-worker")) (some statusOutcome)],
      Program.workflow "workflow" workerOutageWorkflowType workerOutageWorkerRole
        workerOutageQueueRole #[
        Program.node "finish-workflow" (Program.finish (text "completed"))
          (bounds 10000) #[] none (some statusOutcome)]]
    (Program.cleanup "cleanup" #[])
    { programLimits with max_run_events := 512, max_response_bytes := 8192 }
    (environment := #[
      Program.environment workerOutageNamespaceBinding,
      Program.environment workerOutageQueueBinding])

/-! ### The Contract -/

private def faultRoleIs : ContractExpression :=
  ContractExpr.equals (ContractExpr.runEvent .RUN_EVENT_FIELD_FAULT_ROLE_ID)
    (ContractExpr.literal (Value.text workerOutageQueueRole))

/-- The wire number one fault kind carries. The generated Lean enum is a bare inductive with no
number accessor, so the constructor and its number are paired here once and read from here twice. -/
private def faultKindNumber : FaultKind → Int32
  | .FAULT_KIND_WORKER_STOP => 1
  | .FAULT_KIND_WORKER_RESUME => 2
  | _ => 0

private def faultKindIs (kind : FaultKind) : ContractExpression :=
  ContractExpr.equals (ContractExpr.runEvent .RUN_EVENT_FIELD_FAULT_KIND)
    (ContractExpr.literal (Value.enumeration (faultKindNumber kind)))

/-- The outage window, counted in evaluated Run Events rather than on the host's clock. Sixteen is
the whole recorded distance between the stop and the resume this Program can produce: the start,
its activation records, and the fault events themselves. -/
def workerOutageHorizon : Int64 := 16

private def outageOrderRule : ContractRuleDefinition :=
  Monitor.rule "worker-outage-order" .CONTRACT_RULE_KIND_BOUNDED_LIVENESS "awaiting-stop"
    #[Monitor.state "awaiting-stop" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Monitor.state "stopped" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Monitor.state "resumed" .CONTRACT_STATE_STATUS_SATISFIED,
      Monitor.state "expired" .CONTRACT_STATE_STATUS_VIOLATED]
    #[Monitor.transition "observe-stop" "awaiting-stop" "stopped"
        #[.RUN_EVENT_KIND_FAULT_INJECTED]
        (ContractExpr.all #[faultRoleIs, faultKindIs .FAULT_KIND_WORKER_STOP])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
      Monitor.transition "observe-resume" "stopped" "resumed"
        #[.RUN_EVENT_KIND_FAULT_INJECTED]
        (ContractExpr.all #[faultRoleIs, faultKindIs .FAULT_KIND_WORKER_RESUME])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]
    (horizon := some (Monitor.horizonEvents workerOutageHorizon "expired"))

private def workflowCompletedRule : ContractRuleDefinition :=
  Monitor.rule "worker-outage-workflow-completed" .CONTRACT_RULE_KIND_SAFETY "pending"
    #[Monitor.state "pending" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Monitor.state "completed" .CONTRACT_STATE_STATUS_SATISFIED]
    #[Monitor.transition "observe-workflow-completed" "pending" "completed"
        #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
        (ContractExpr.all #[
          ContractExpr.present (observed workerOutageObservation),
          ContractExpr.present (projected (observed workerOutageObservation)
            (historyAttribute completedAttributes completedTaskField))])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]

private def outageProperty :=
  binding "temporal.case.worker-outage.property.outage-survived"
    "temporal-case-worker-outage-property/v1" .property

/-- The checked-in `rule_events` Case: one deliberate outage, ordered and survived. -/
def workerOutageCase : Except Umpire.Case.Compiler.LoweringError Case := do
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
    contractLimits
  }

end Temporal.Testpilot
