import Temporal.Case.Evidence
import Temporal.Case.Support

/-!
# The workflow-start realization

A controller starts one workflow on the Case's task queue and reads its history back once it
closed; the workflow itself does nothing but finish. It is the realization of a Model whose one
side effect is the start command: the `startWorkflow` action class is bound to the
`StartWorkflowExecution` call, and the literal the call assigns to `workflow_type.name` is stated
on the binding, so a field relation over that input field compares the value the Program
constructs. Everything else -- the close-event read, the history read, the finish -- is
scaffolding no Model declares.
-/

namespace Temporal.Case.Realization.Workflow

open Umpire
open Testpilot.Authoring
open Temporal.Case.Support
open Temporal.Testpilot.CaseSupport
open temporal.server.api.testpilot.v1 hiding ModelValue

def workerNamespaceBinding := "temporal.worker.namespace"
def taskQueueBinding := "temporal.task-queue.resource"

def projectionId : DefinitionId := .of "temporal.workflow.start.projection"
def evidenceSourceId : DefinitionId := .of "temporal.workflow.start.source.history"
def runFieldId : DefinitionId := .of "temporal.workflow.start.scope.run"
def workflowFieldId : DefinitionId := .of "temporal.workflow.start.scope.workflow"
def startedEvidenceKindId : DefinitionId :=
  .of "temporal.workflow.start.evidence.workflowExecutionStarted"

/-- The action class the realization binds, stated as the fallback a binding is read by where the
Model's vocabulary has no member of the binding's key. -/
def startAction : DefinitionId := .of "temporal.workflow.start.action.startWorkflow"

/-- The started event names its workflow by the run that first executed it, which is what keys
one workflow across every event of its history. -/
private def startedAttributes : String := "workflow_execution_started_event_attributes"

def startedSource : Umpire.Case.Producer.EvidenceSource := {
  eventKind := "workflowExecutionStarted"
  recorded := .historyEvent startedAttributes
  operationKeyPath := historyAttribute startedAttributes "first_execution_run_id"
  kindId := startedEvidenceKindId
  sourceId := evidenceSourceId }

def sharedRoles : Array Role := #[
  Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
  Program.role workerRole .ROLE_KIND_WORKER (namespaceBindingId := workerNamespaceBinding),
  Program.role taskQueueRole .ROLE_KIND_TASK_QUEUE
    (namespaceBindingId := workerNamespaceBinding) (resourceBindingId := taskQueueBinding)]

def sharedObservations : Array Observation := #[
  Program.observation historyObservation historyEventType,
  Program.observation correlatedObservation correlatedEvidenceType]

/-- The workflow type a Case's fixture name derives. -/
def workflowTypeOf (identity : Umpire.Case.Producer.Identity) : String :=
  "umpire-" ++ identity.fixture ++ "-workflow"

/-- The dotted path of the request field the start assigns the workflow type at, as a relation
names it. -/
def workflowTypeField : String := "workflow_type.name"

private def rpc (id method : String) (assignments : Array RequestAssignment)
    (projections : Array ResponseRead) : InstructionNode :=
  Program.node id (Program.invokeRpc workflowServiceRole method assignments projections)

private def historyAssignments : Array RequestAssignment := #[
  Program.environmentAssignment (field "namespace") workerNamespaceBinding,
  assign (nested ["execution", "workflow_id"]) runId,
  assign (field "maximum_page_size") (signedInteger 64),
  assign (field "wait_new_event") (boolean true)]

def startWorkflowNode (instructionId workflowType : String) : InstructionNode :=
  rpc instructionId startWorkflowMethod #[
    Program.environmentAssignment (field "namespace") workerNamespaceBinding,
    assign (field "workflow_id") runId,
    assign (nested ["workflow_type", "name"]) (text workflowType),
    Program.environmentAssignment (nested ["task_queue", "name"]) taskQueueBinding,
    assign (field "request_id") runId] #[]

/-- The close-event read, which resolves once the workflow closed. -/
def awaitCloseNodeWith (instructionId : String) : InstructionNode :=
  rpc instructionId getHistoryMethod
    (historyAssignments.push (assign (field "history_event_filter_type") closeEventFilter)) #[]

def awaitCloseNode : InstructionNode := awaitCloseNodeWith "await-close"

/-- The full history read, run once the workflow closed, lifting the declared events. -/
def historyNode (resolved : List Umpire.Case.Producer.EvidenceRule) : InstructionNode :=
  rpc "history" getHistoryMethod historyAssignments #[
    Program.responseRead historyEvents .READ_CARDINALITY_EMIT_EACH
      #[Program.observationTarget historyObservation,
        Evidence.target correlatedObservation resolved]]

/-- The start command: the one action class. Its literal for `workflow_type.name` is stated
beside the node, so a relation over that field reads the value the node assigns. -/
def startBinding : Umpire.Case.Producer.ActionBinding := {
  action := startAction
  key := "startWorkflow"
  instructionId := "start-workflow"
  node := fun placement instructionId =>
    startWorkflowNode instructionId (workflowTypeOf placement.identity)
  literals := fun placement => [(workflowTypeField, .text (workflowTypeOf placement.identity))] }

/-- The plan every workflow Case is assembled from: a controller whose items the realization
orders around the start, and a workflow that does nothing but finish with `finished`. -/
def plan (controller : List Umpire.Case.Producer.EntrypointItem) (finished : String) :
    Umpire.Case.Producer.ProgramPlan := {
  roles := sharedRoles
  observations := sharedObservations
  entrypoints := [
    { activate := fun _ nodes => Program.controller "controller" nodes
      items := controller },
    { activate := fun placement nodes =>
        Program.workflow "workflow" (workflowTypeOf placement.identity) workerRole taskQueueRole
          nodes
      items := [
        .fixed fun _ _ =>
          Program.node "finish-workflow" (Program.finish (text finished))
            (Program.instructionLimits (timeoutMilliseconds := some 5000))] }]
  cleanup := Program.cleanup "cleanup" #[] }

/-! ### The outage

The same scaffolding, disturbed: the worker of the Case's own task queue is stopped before the
start and resumed after it, and the close read waits for the workflow the resumed worker then
completes. The two faults are the `worker` party's action classes, each bound to a fault
instruction on the task-queue role; the wait for completion is the caller's, bound to the
close-event read. Whether the two faults happen in order and the resume is not long in coming is
the outage-order rule the Producer derives from the assembled Program. -/

def outageProjectionId : DefinitionId := .of "temporal.workflow.outage.projection"
def outageEvidenceSourceId : DefinitionId := .of "temporal.workflow.outage.source.history"
def outageRunFieldId : DefinitionId := .of "temporal.workflow.outage.scope.run"
def outageWorkflowFieldId : DefinitionId := .of "temporal.workflow.outage.scope.workflow"
def completedEvidenceKindId : DefinitionId :=
  .of "temporal.workflow.outage.evidence.workflowExecutionCompleted"

def workerStopAction : DefinitionId := .of "temporal.workflow.outage.action.workerStop"
def workerResumeAction : DefinitionId := .of "temporal.workflow.outage.action.workerResume"
def awaitCompletionAction : DefinitionId := .of "temporal.workflow.outage.action.awaitCompletion"

/-- The completed event names its workflow by the workflow task that completed it: the one task
that can only have been dispatched once the worker was back, which is what an outage the work
survived leaves behind. -/
private def completedAttributes : String := "workflow_execution_completed_event_attributes"

def completedSource : Umpire.Case.Producer.EvidenceSource := {
  eventKind := "workflowExecutionCompleted"
  recorded := .historyEvent completedAttributes
  operationKeyPath := historyAttribute completedAttributes "workflow_task_completed_event_id"
  kindId := completedEvidenceKindId
  sourceId := outageEvidenceSourceId }

private def faultBinding (action : DefinitionId) (key instructionId : String) (kind : FaultKind) :
    Umpire.Case.Producer.ActionBinding := {
  action
  key
  instructionId
  node := fun _ instructionId =>
    Program.node instructionId (Program.injectFault taskQueueRole kind) }

/-- The worker of the Case's task queue stops polling. -/
def workerStopBinding : Umpire.Case.Producer.ActionBinding :=
  faultBinding workerStopAction "workerStop" "stop-worker" .FAULT_KIND_WORKER_STOP

/-- The worker of the Case's task queue polls again. -/
def workerResumeBinding : Umpire.Case.Producer.ActionBinding :=
  faultBinding workerResumeAction "workerResume" "resume-worker" .FAULT_KIND_WORKER_RESUME

/-- The caller waits for the workflow to close: the close-event read, which resolves only once the
resumed worker completed it. -/
def awaitCompletionBinding : Umpire.Case.Producer.ActionBinding := {
  action := awaitCompletionAction
  key := "awaitCompletion"
  instructionId := "await-close"
  node := fun _ instructionId => awaitCloseNodeWith instructionId }

end Temporal.Case.Realization.Workflow

namespace Temporal.Case.Realization

open Umpire

/-- One workflow started by a controller and read back once it closed, realized from a Model's
`startWorkflow` class. -/
def workflowStart : Umpire.Case.Producer.Realization := {
  plan := Workflow.plan [
    .actions [Workflow.startAction],
    .fixed fun _ _ => Workflow.awaitCloseNode,
    .fixed fun _ resolved => Workflow.historyNode resolved] "started"
  actions := [Workflow.startBinding]
  producerId := "temporal.workflow.start.testpilot"
  producerVersion := "1"
  projectionId := Workflow.projectionId
  scopeField := Workflow.runFieldId
  operationKey := Workflow.workflowFieldId
  historyObservation := Support.historyObservation
  correlatedObservation := Support.correlatedObservation
  sources := [Workflow.startedSource]
  projectionLimits := {
    events := 32, buffered := 16, keys := 8, support := 128
    work := 1000000000, eventSize := 512 }
  runLimits := {
    «transitions» := 16, obligations := 16, work := 100000000, captures := 0 } }

/- The one class is bound to the start call, and the literal it assigns is the derived workflow
type. -/
#guard (workflowStart.actions.map (·.key)) == ["startWorkflow"]
#guard (workflowStart.actions.map fun binding =>
    binding.literals { identity := Umpire.Case.Producer.Identity.ofFixture "temporal.case" "x" }) ==
  [[("workflow_type.name", .text "umpire-x-workflow")]]

/-- One workflow started while the worker of its own task queue is stopped, the worker resumed,
and the workflow read back once the resumed worker completed it: realized from a Model's
`workerStop`, `startWorkflow`, `workerResume` and `awaitCompletion` classes, in the controller
order the classes are named here, whatever order the path performs them in. -/
def workflowOutage : Umpire.Case.Producer.Realization := {
  plan := Workflow.plan [
    .actions [Workflow.workerStopAction],
    .actions [Workflow.startAction],
    .actions [Workflow.workerResumeAction],
    .actions [Workflow.awaitCompletionAction],
    .fixed fun _ resolved => Workflow.historyNode resolved] "completed"
  actions := [Workflow.workerStopBinding, Workflow.startBinding, Workflow.workerResumeBinding,
    Workflow.awaitCompletionBinding]
  producerId := "temporal.workflow.outage.testpilot"
  producerVersion := "1"
  projectionId := Workflow.outageProjectionId
  scopeField := Workflow.outageRunFieldId
  operationKey := Workflow.outageWorkflowFieldId
  historyObservation := Support.historyObservation
  correlatedObservation := Support.correlatedObservation
  sources := [Workflow.completedSource]
  projectionLimits := {
    events := 32, buffered := 16, keys := 8, support := 128
    work := 1000000000, eventSize := 512 }
  runLimits := {
    «transitions» := 16, obligations := 16, work := 100000000, captures := 0 } }

/- The two faults are bound to fault instructions on the Case's task-queue role, the wait to the
close-event read. -/
#guard (workflowOutage.actions.map (·.key)) ==
  ["workerStop", "startWorkflow", "workerResume", "awaitCompletion"]
#guard (workflowOutage.actions.map (·.instructionId)) ==
  ["stop-worker", "start-workflow", "resume-worker", "await-close"]
#guard (Umpire.Case.Producer.injectedFaults
    ((workflowOutage.program (Umpire.Case.Producer.Identity.ofFixture "temporal.case" "x")
      (path := [Workflow.workerStopAction, Workflow.startAction, Workflow.workerResumeAction,
        Workflow.awaitCompletionAction])).toOption.getD (Testpilot.Authoring.Program.make "" #[] #[] #[] #[]
      (Testpilot.Authoring.Program.cleanup "" #[])))) ==
  [(Support.taskQueueRole, .FAULT_KIND_WORKER_STOP), (Support.taskQueueRole, .FAULT_KIND_WORKER_RESUME)]

end Temporal.Case.Realization
