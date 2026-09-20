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
def awaitCloseNode : InstructionNode :=
  rpc "await-close" getHistoryMethod
    (historyAssignments.push (assign (field "history_event_filter_type") closeEventFilter)) #[]

/-- The full history read, run once the workflow closed, lifting the started event. -/
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
  node := fun identity instructionId => startWorkflowNode instructionId (workflowTypeOf identity)
  literals := fun identity => [(workflowTypeField, .text (workflowTypeOf identity))] }

def plan : Umpire.Case.Producer.ProgramPlan := {
  roles := sharedRoles
  observations := sharedObservations
  entrypoints := [
    { activate := fun _ nodes => Program.controller "controller" nodes
      items := [
        .actions [startAction],
        .fixed fun _ _ => awaitCloseNode,
        .fixed fun _ resolved => historyNode resolved] },
    { activate := fun identity nodes =>
        Program.workflow "workflow" (workflowTypeOf identity) workerRole taskQueueRole nodes
      items := [
        .fixed fun _ _ =>
          Program.node "finish-workflow" (Program.finish (text "started"))
            (Program.instructionLimits (timeoutMilliseconds := some 5000))] }]
  cleanup := Program.cleanup "cleanup" #[] }

end Temporal.Case.Realization.Workflow

namespace Temporal.Case.Realization

open Umpire

/-- One workflow started by a controller and read back once it closed, realized from a Model's
`startWorkflow` class. -/
def workflowStart : Umpire.Case.Producer.Realization := {
  plan := Workflow.plan
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
    binding.literals (Umpire.Case.Producer.Identity.ofFixture "temporal.case" "x")) ==
  [[("workflow_type.name", .text "umpire-x-workflow")]]

end Temporal.Case.Realization
