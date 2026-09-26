import Temporal.Case.Evidence
import Temporal.Case.FieldPath
import Temporal.Case.ReadKind
import Temporal.Case.Support
import Temporal.DynamicConfig

/-!
# The Nexus caller-side realization

A controller-started workflow schedules one Nexus operation on the Case's endpoint role; the handler
answers it, and the controller completes it when the handler answered asynchronously; the controller
reads the history the Case's evidence is lifted from. This realization writes no Program: it declares
that scaffolding once and binds each **action class** of a Model to the instruction that performs it,
so the Producer puts those instructions where a Query's path took them.

Every side effect that settles the operation is one action class, realized as the typed worker
instruction that carries the Temporal API message the class names (fn-85 R10):

| class | party | realized as | entrypoint |
| --- | --- | --- | --- |
| `schedule (…)` | the workflow that calls the operation | a workflow command carrying `ScheduleNexusOperationCommandAttributes` on the Case's endpoint role, each deadline the class sets at the realization's duration | `workflow` |
| `handlerReply (async)` | the handler | a handler reply carrying `StartOperationResponse.async_success`, publishing the completion authority | `handler` |
| `handlerReply (syncSuccess)` | the handler | a handler reply carrying `StartOperationResponse.sync_success` with its payload | `handler` |
| `handlerReply (operationFailed)` | the handler | a handler reply carrying `StartOperationResponse.failure` | `handler` |
| `handlerReply (handlerError (retryable := r))` | the handler | a handler reply carrying `HandlerError` with that retry behavior | `handler` |
| `complete (succeeded)` | the handler | a completion carrying a `Payload` over the published handle | `controller` |
| `complete (failed)` | the handler | a completion carrying a `Failure` over the published handle | `controller` |
| `workerStop` | the worker | a fault instruction stopping the worker on the handler's own task queue | `controller` |

`complete` is why a realization binds classes rather than parties: the handler's completion is
performed by a controller instruction over a handle slot the handler published, so a per-party
binding would put it on the handler's entrypoint, where no such instruction can run. `workerStop` is
the other: the worker performs nothing; the controller stops it.

Everything else the Program carries is scaffolding no Model declares -- starting the caller workflow,
waiting for the authority, polling the pending operation's attempt count, waiting on the scheduled
operation, finishing the workflow, and reading history back -- and stays a fixed item or one emitted
only when a class on the path needs it. That is what the entrypoint item order says, and it is what
keeps the Producer free of Nexus.

Each binding names its class by the member key a Scenario's path spells it by
(`handlerReply-async`, `complete-succeeded`, `schedule-unset-expires-unset`), which the Producer
resolves against the Model's own vocabulary, so the caller Model's protocol machine and any Model
over the same classes produce through one realization.

The handler polls its own task queue, distinct from the caller workflow's: the fault that stops the
handler's worker must leave the caller's running, or no schedule command would ever be sent.
-/

namespace Temporal.Case.Realization.Nexus

open Umpire
open Testpilot.Authoring
open Temporal.Case.Support
open Temporal.Testpilot.CaseSupport
open temporal.server.api.testpilot.v1 hiding ModelValue

/-! ### Bindings and coordinates

The environment bindings the roles resolve through, and the Definition IDs every Case on this
realization reads its evidence by: the Run scope, the operation key, the projection, the two
evidence sources and one kind per admitted history event or read. -/

def workerNamespaceBinding := "temporal.worker.namespace"
def taskQueueBinding := "temporal.task-queue.resource"
def handlerTaskQueueBinding := "temporal.handler-task-queue.resource"
def nexusEndpointBinding := "temporal.nexus-endpoint.resource"

def projectionId : DefinitionId := .of "temporal.nexus.caller.projection"
def evidenceSourceId : DefinitionId := .of "temporal.nexus.caller.source.history"
def describeSourceId : DefinitionId := .of "temporal.nexus.caller.source.describe"
def scheduledSourceId : DefinitionId := .of "temporal.nexus.caller.source.scheduled"
def runFieldId : DefinitionId := .of "temporal.nexus.caller.scope.run"
def operationFieldId : DefinitionId := .of "temporal.nexus.caller.scope.operation"
def scheduledEvidenceKindId : DefinitionId := .of "temporal.nexus.caller.evidence.scheduled"
def startedEvidenceKindId : DefinitionId := .of "temporal.nexus.caller.evidence.started"
def completedEvidenceKindId : DefinitionId := .of "temporal.nexus.caller.evidence.completed"
def failedEvidenceKindId : DefinitionId := .of "temporal.nexus.caller.evidence.failed"
def canceledEvidenceKindId : DefinitionId := .of "temporal.nexus.caller.evidence.canceled"
def timedOutEvidenceKindId : DefinitionId := .of "temporal.nexus.caller.evidence.timedOut"
def pendingAttemptsEvidenceKindId : DefinitionId :=
  .of "temporal.nexus.caller.evidence.pendingAttempts"

/-! ### The evidence sources

A started, completed, failed, canceled or timed-out Nexus event records the scheduled event it
answers and nothing else that names its operation, so that scheduled event id is the operation key
on every side; the scheduled event is keyed by its own id. A retryable attempt failure writes no
history event; the pending operation's attempt count is read back through
`DescribeWorkflowExecution`, keyed by the same scheduled event id, through the catalog's binding
(`Temporal.Case.ReadKind`).

The scheduled event is read the same way, out of history as soon as it exists, rather than by the
history read that closes the Run: the verifier orders one operation's evidence across sources by
the order it was lifted in, and the read that confirms a retry must find the operation already
scheduled. -/

private def scheduledEventKey : String := "scheduled_event_id"

private def historySource (kind : String) (kindId : DefinitionId) :
    Umpire.Case.Producer.EvidenceSource :=
  let attributes := (EventKind.attributesField? kind).getD kind
  { eventKind := kind
    recorded := .historyEvent attributes
    operationKeyPath := historyAttribute attributes scheduledEventKey
    kindId
    sourceId := evidenceSourceId }

/-- The field of the scheduled event that names which instance's operation it records: the
operation name the schedule command addressed, resolved against the generated schema the way a
relation's operand is. A relation that captures the scheduled event of an earlier step selects
the instance's own by it. -/
def scheduledSelector : Option Umpire.Case.Producer.FieldOperand :=
  match Temporal.Case.FieldPath.resolve
      { kind := .observation, schema := ["nexusOperationScheduled"], segments := ["operation"] } with
  | .ok resolved => some {
      root := .event
      member := ""
      spelling := "operation"
      observed := "nexusOperationScheduled"
      schema := Temporal.Case.FieldPath.getWorkflowExecutionHistorySchema
      side := resolved.side
      steps := resolved.steps
      type := resolved.type
      presence := resolved.presence }
  | .error _ => none

def scheduledSource : Umpire.Case.Producer.EvidenceSource :=
  let binding := ReadKind.scheduledEvent
  { eventKind := binding.name
    recorded := .read binding.method binding.path
    operationKeyPath := binding.operationKey
    kindId := scheduledEvidenceKindId
    sourceId := scheduledSourceId
    selector := scheduledSelector }

def startedSource : Umpire.Case.Producer.EvidenceSource :=
  historySource "nexusOperationStarted" startedEvidenceKindId

def completedSource : Umpire.Case.Producer.EvidenceSource :=
  historySource "nexusOperationCompleted" completedEvidenceKindId

def failedSource : Umpire.Case.Producer.EvidenceSource :=
  historySource "nexusOperationFailed" failedEvidenceKindId

def canceledSource : Umpire.Case.Producer.EvidenceSource :=
  historySource "nexusOperationCanceled" canceledEvidenceKindId

def timedOutSource : Umpire.Case.Producer.EvidenceSource :=
  historySource "nexusOperationTimedOut" timedOutEvidenceKindId

/-- Every history event kind a caller-side operation records. -/
def historySources : List Umpire.Case.Producer.EvidenceSource :=
  [scheduledSource, startedSource, completedSource, failedSource, canceledSource, timedOutSource]

/-- The read exposes no field to the Contract: the attempt count is a signed integer, which the
correlated Contract's field policies cannot type (text, boolean and unsigned only), so the poll's
condition is what fixes the count and the Contract confirms the kind. -/
def pendingAttemptsSource : Umpire.Case.Producer.EvidenceSource :=
  let binding := ReadKind.pendingAttempts
  { eventKind := binding.name
    recorded := .read binding.method binding.path
    operationKeyPath := binding.operationKey
    kindId := pendingAttemptsEvidenceKindId
    sourceId := describeSourceId }

/-! ### The scaffolding

The roles, slot and observations every Case declares, and the nodes no Model declares. -/

def sharedRoles : Array Role := #[
  Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
  Program.role workerRole .ROLE_KIND_WORKER
    (namespaceBindingId := workerNamespaceBinding),
  Program.role taskQueueRole .ROLE_KIND_TASK_QUEUE
    (namespaceBindingId := workerNamespaceBinding) (resourceBindingId := taskQueueBinding),
  Program.role handlerTaskQueueRole .ROLE_KIND_TASK_QUEUE
    (namespaceBindingId := workerNamespaceBinding)
    (resourceBindingId := handlerTaskQueueBinding),
  Program.role nexusEndpointRole .ROLE_KIND_ENDPOINT
    (resourceBindingId := nexusEndpointBinding)]

def sharedObservations : Array Observation := #[
  Program.observation historyObservation historyEventType,
  Program.observation correlatedObservation correlatedEvidenceType]

/-- The workflow type a Case's fixture name derives. -/
def workflowTypeOf (identity : Umpire.Case.Producer.Identity) : String :=
  "umpire-" ++ identity.fixture ++ "-workflow"

private def rpc
    (id method : String)
    (assignments : Array RequestAssignment)
    (projections : Array ResponseRead) : InstructionNode :=
  Program.node id (Program.invokeRpc workflowServiceRole method assignments projections)

private def historyAssignments : Array RequestAssignment := #[
  Program.environmentAssignment (field "namespace") workerNamespaceBinding,
  assign (nested ["execution", "workflow_id"]) runId,
  assign (field "maximum_page_size") (signedInteger 64),
  assign (field "wait_new_event") (boolean true)
]

def startWorkflowNode (workflowType : String) : InstructionNode :=
  rpc "start-workflow" startWorkflowMethod #[
    Program.environmentAssignment (field "namespace") workerNamespaceBinding,
    assign (field "workflow_id") runId,
    assign (nested ["workflow_type", "name"]) (text workflowType),
    Program.environmentAssignment (nested ["task_queue", "name"]) taskQueueBinding,
    assign (field "request_id") runId
  ] #[]

/-- The close-event read: `GetWorkflowExecutionHistory` with the close-event filter resolves only
once the workflow closes, so a read placed after it observes the whole history. It is what orders
the history read behind the operation's outcome on every path, whichever party settled it. -/
def awaitCloseNode : InstructionNode :=
  rpc "await-close" getHistoryMethod
    (historyAssignments.push (assign (field "history_event_filter_type") closeEventFilter)) #[]

/-- The full history read, run once the instruction before it succeeded. It lifts the history
kinds among the resolved rules; a path that records none -- a schedule alone, whose evidence is the
pending-operation read -- lifts nothing, because a lift with no rule is a Case Prepare rejects. -/
def historyNode (resolved : List Umpire.Case.Producer.EvidenceRule) : InstructionNode :=
  rpc "history" getHistoryMethod historyAssignments #[
    Program.responseRead historyEvents .READ_CARDINALITY_EMIT_EACH
      (#[Program.observationTarget historyObservation] ++
        if resolved.any Evidence.readsHistory then #[Evidence.target correlatedObservation resolved]
        else #[])
  ]

/-- The controller's bounded poll of the pending operation, run until `condition` holds of one
element -- `Expr.projectedValue` is one `PendingNexusOperationInfo` -- or the node times out. -/
def pendingAttemptsNode (condition : Expression) (pollIntervalMilliseconds : Int64 := 250) :
    InstructionNode :=
  Program.node "pending-attempts"
    (Program.readEvidence pendingAttemptsEvidenceKindId.value workflowServiceRole
      #[Program.environmentAssignment (field "namespace") workerNamespaceBinding,
        assign (nested ["execution", "workflow_id"]) runId]
      condition pollIntervalMilliseconds)

/-- The controller's bounded poll of the history for the scheduled event, run until the event
exists. -/
def awaitScheduledNode (pollIntervalMilliseconds : Int64 := 250) : InstructionNode :=
  Program.node "await-scheduled"
    (Program.readEvidence scheduledEvidenceKindId.value workflowServiceRole
      #[Program.environmentAssignment (field "namespace") workerNamespaceBinding,
        assign (nested ["execution", "workflow_id"]) runId]
      (Expr.present (projected Expr.projectedValue
        (Path.make #[Path.oneofMember "attributes" "nexus_operation_scheduled_event_attributes"])))
      pollIntervalMilliseconds)

/-! ### Action classes

The Definition IDs a Model's `action` declarations carry, stated here as the fallback a binding is
read by where the Model's vocabulary has no member of the binding's key. -/

def scheduleAction : DefinitionId := .of "temporal.nexus.caller.action.schedule"
def scheduleToStartAction : DefinitionId :=
  .of "temporal.nexus.caller.action.schedule.scheduleToStart"
def startToCloseAction : DefinitionId := .of "temporal.nexus.caller.action.schedule.startToClose"
def handlerReplyAction : DefinitionId := .of "temporal.nexus.caller.action.handlerReply"
def handlerReplySyncAction : DefinitionId :=
  .of "temporal.nexus.caller.action.handlerReply.syncSuccess"
def handlerReplyFailedAction : DefinitionId :=
  .of "temporal.nexus.caller.action.handlerReply.operationFailed"
def handlerErrorRetryableAction : DefinitionId :=
  .of "temporal.nexus.caller.action.handlerReply.handlerError.retryable"
def handlerErrorNonRetryableAction : DefinitionId :=
  .of "temporal.nexus.caller.action.handlerReply.handlerError.nonRetryable"
def completeAction : DefinitionId := .of "temporal.nexus.caller.action.complete"
def completeFailedAction : DefinitionId := .of "temporal.nexus.caller.action.complete.failed"
def workerStopAction : DefinitionId := .of "temporal.nexus.caller.action.workerStop"

/-- The handle slot the handler's asynchronous reply publishes and the completion consumes, one
per instance of the operation. Naming it once is what keeps the two bindings agreeing about which
authority they mean. -/
private def completionAuthority (placement : Umpire.Case.Producer.Placement) : String :=
  "completion-authority" ++ placement.suffix

/-- The operation one instance addresses: the realization's on a Case over one instance, and one
name per instance on a Case over several, so each instance's handler registers its own. -/
def operationOf (operation : String) (placement : Umpire.Case.Producer.Placement) : String :=
  operation ++ placement.suffix

/-- The failure a failed reply or completion carries: an application failure the handler names. -/
private def handlerFailure : temporal.api.failure.v1.Failure :=
  { «message» := "operation failed"
    failure_info := some (.application_failure_info {
      «type» := "OperationFailed", non_retryable := true }) }

/-! ### Timers

A deadline the schedule command sets realizes as a concrete duration: short enough for a live test
to wait out, long enough for the operation to be dispatched first. The backoff a retryable failure
starts is the server's own retry interval (`component.nexusoperations.retryPolicy.initialInterval`
and its CHASM counterpart, one second by default), which the realization reads through the pending
operation and does not set. -/

def timers : List Umpire.Case.Producer.TimerBinding := [
  { name := "scheduleToStart", milliseconds := 2000 },
  { name := "startToClose", milliseconds := 2000 }]

private def timerDuration (name : String) : Option google.protobuf.Duration :=
  (timers.find? (·.name == name)).map fun timer => Duration.milliseconds timer.milliseconds

/-! ### The bindings

Each binding builds its node from the id the Producer supplies, so the same class performed twice on
one path produces two distinct nodes. The messages each carries are the ones the class names; the
concrete values (the request and result payloads, the error types) are the realization's, until a
Model's `examples:` supply them. -/

/-- The schedule command for one class of the `schedule` action: the deadlines the class sets, at
the realization's durations. -/
private def scheduleBinding (action : DefinitionId) (key service operation : String)
    (scheduleToStart startToClose : Option google.protobuf.Duration := none) :
    Umpire.Case.Producer.ActionBinding := {
  action
  key
  instructionId := "start-nexus-operation"
  node := fun placement instructionId =>
    Program.node instructionId
      (Program.scheduleNexusOperation Support.nexusEndpointRole service
        (operationOf operation placement) (Payload.text "request")
        (scheduleToStart := scheduleToStart) (startToClose := startToClose))
  -- The operation name the command assigns, which the scheduled event records: what selects the
  -- instance's own scheduled event where a relation captures it.
  literals := fun placement => [("operation", .text (operationOf operation placement))] }

private def handlerReplyBinding : Umpire.Case.Producer.ActionBinding := {
  action := handlerReplyAction
  key := "handlerReply-async"
  instructionId := "respond-async"
  node := fun placement instructionId =>
    Program.node instructionId (Program.nexusAsyncReply (completionAuthority placement))
      (Program.instructionLimits (timeoutMilliseconds := some 5000)) }

private def handlerReplySyncBinding : Umpire.Case.Producer.ActionBinding := {
  action := handlerReplySyncAction
  key := "handlerReply-syncSuccess"
  instructionId := "respond-sync"
  node := fun _ instructionId =>
    Program.node instructionId (Program.nexusSyncReply (Payload.text "completed"))
      (Program.instructionLimits (timeoutMilliseconds := some 5000)) }

private def handlerReplyFailedBinding : Umpire.Case.Producer.ActionBinding := {
  action := handlerReplyFailedAction
  key := "handlerReply-operationFailed"
  instructionId := "respond-failed"
  node := fun _ instructionId =>
    Program.node instructionId (Program.nexusFailedReply handlerFailure)
      (Program.instructionLimits (timeoutMilliseconds := some 5000)) }

private def handlerErrorBinding (action : DefinitionId) (key instructionId errorType : String)
    (retryBehavior : temporal.api.enums.v1.NexusHandlerErrorRetryBehavior) :
    Umpire.Case.Producer.ActionBinding := {
  action
  key
  instructionId
  node := fun _ instructionId =>
    Program.node instructionId (Program.nexusHandlerError errorType "handler error" retryBehavior)
      (Program.instructionLimits (timeoutMilliseconds := some 5000)) }

private def completeBinding : Umpire.Case.Producer.ActionBinding := {
  action := completeAction
  key := "complete-succeeded"
  instructionId := "complete-nexus-operation"
  node := fun placement instructionId =>
    Program.node instructionId
      (Program.nexusOperationCompletion (completionAuthority placement)
        (Payload.text "completed")) }

private def completeFailedBinding : Umpire.Case.Producer.ActionBinding := {
  action := completeFailedAction
  key := "complete-failed"
  instructionId := "fail-nexus-operation"
  node := fun placement instructionId =>
    Program.node instructionId
      (Program.nexusOperationFailure (completionAuthority placement) handlerFailure) }

/-- The handler's worker stops polling its queue: a deliberate outage of the handler's task queue
and no other, so the caller workflow keeps running. -/
private def workerStopBinding : Umpire.Case.Producer.ActionBinding := {
  action := workerStopAction
  key := "workerStop"
  instructionId := "stop-handler-worker"
  node := fun _ instructionId =>
    Program.node instructionId
      (Program.injectFault Support.handlerTaskQueueRole .FAULT_KIND_WORKER_STOP) }

/-! ### The plan

The controller's sequence is the one place the interleaving matters: it stops the handler's worker
when the path says so, starts the workflow, reads the scheduled event as soon as it exists, polls
the attempt count when a retryable failure is on the path, waits for the authority the handler
publishes when a completion is on the path, performs the completion, waits for the workflow to
close, and only then reads history. Writing those as
ordered items is what reproduces the dependency edges from path order alone.

One plan serves every Query of a set. A synchronous reply and a handler error publish no authority,
so the wait for it is emitted only when a completion class is on the path; the close-event read is
what orders the history read behind the operation's outcome on every path, and the workflow finishes
whether or not the operation succeeded, so its history closes on every path.

The worker stop is placed before the workflow starts although the Model's path may place it after
the schedule: the stop changes nothing about the operation, and a stop that raced the dispatch of
the start request would sometimes lose.
-/

/-- The classes whose instruction consumes the completion authority. -/
private def completionKeys : List String := ["complete-succeeded", "complete-failed"]

/-- The classes that schedule the operation, one per assignment of its deadlines. -/
private def scheduleKeys : List String :=
  ["schedule-unset-unset-unset", "schedule-unset-expires-unset", "schedule-unset-unset-expires"]

/-- The class whose reply backs the operation off, after which the attempt count is read. -/
private def retryKeys : List String := ["handlerReply-handlerError-true"]

/-- The attempt count a retryable failure leaves the pending operation at: its first attempt failed,
and the read confirms the backoff before the retry settles the operation. -/
private def firstAttemptFailed : Expression :=
  Expr.equal (projected Expr.projectedValue (field "attempt")) (signedInteger 1)

/-- The plan every caller-side Case is assembled from. -/
def asyncPlan (service operation : String) : Umpire.Case.Producer.ProgramPlan := {
  roles := sharedRoles
  -- One completion authority per instance of the operation.
  instanceSlots := fun placement => #[Program.handleSlot (completionAuthority placement)]
  observations := sharedObservations
  entrypoints := [
    { activate := fun _ nodes => Program.controller "controller" nodes
      items := [
        .actions [workerStopAction],
        .fixed fun placement _ => startWorkflowNode (workflowTypeOf placement.identity),
        .fixed fun _ _ => awaitScheduledNode,
        .whenOnPath retryKeys fun _ _ => pendingAttemptsNode firstAttemptFailed,
        -- Each instance's completion follows its own wait for the authority its handler
        -- publishes.
        .perInstance [
          .whenOnPath completionKeys fun placement _ =>
            Program.node ("await-completion-authority" ++ placement.suffix)
              (Program.awaitSlot (completionAuthority placement)),
          .actions [completeAction, completeFailedAction]],
        .fixed fun _ _ => awaitCloseNode,
        .fixed fun _ resolved => historyNode resolved] },
    { activate := fun placement nodes =>
        Program.workflow "workflow" (workflowTypeOf placement.identity) workerRole taskQueueRole
          nodes
      items := [
        -- The schedule is the Model's; the await and the finish are scaffolding, because no Model
        -- declares them. Each instance's schedule is awaited by its own node.
        .actions [scheduleAction, scheduleToStartAction, startToCloseAction],
        .whenOnPath scheduleKeys fun placement _ =>
          Program.node ("await-nexus-operation" ++ placement.suffix)
            (Program.awaitInstruction
              (Ref.instruction "workflow" ("start-nexus-operation" ++ placement.suffix)))
            (guard := some (boolean true)),
        -- The workflow closes on every path: a failed or timed-out operation is the await's
        -- recorded outcome, not a reason to leave the workflow open, so the finish runs regardless
        -- and returns a literal rather than the awaited payload a failed operation has none of.
        .fixed fun _ _ =>
          Program.node "finish-workflow" (Program.finish (text "done"))
            (Program.instructionLimits (timeoutMilliseconds := some 5000))
            (guard := some (boolean true))] },
    -- One handler per instance: each answers the operation its instance addresses, and a handler
    -- entrypoint ends at its first reply.
    { activate := fun placement nodes =>
        Program.nexusHandler ("handler" ++ placement.suffix) service
          (operationOf operation placement) Support.workerRole Support.handlerTaskQueueRole nodes
      items := [.actions [handlerReplyAction, handlerReplySyncAction, handlerReplyFailedAction,
        handlerErrorRetryableAction, handlerErrorNonRetryableAction]]
      perInstance := true }]
  cleanup := Program.cleanup "cleanup" #[] }

/-! ### The switch

The realization declares the switch a functional set repeats over. It names keys of the generated
dynamic-config catalog, so a key that leaves the registry fails here rather than at a Run. -/

/-- The value a `Bool` setting takes, spelled the way the Profile records it. -/
private def flag (setting : Temporal.DynamicConfig.Setting) (enabled : Bool) : String × String :=
  (setting.key, toString enabled)

/-- The Nexus implementation switch: the same Model runs under the HSM implementation and under
CHASM, and which one is the three settings the upstream Nexus suites set at environment
construction. The value names are what a `repeat:` names and what a live test spells its
environments after. -/
def implementationSwitch : Umpire.Case.Producer.SwitchBinding := {
  name := "implementation"
  values := [
    { name := "hsm", configuration := implementation false },
    { name := "chasm", configuration := implementation true }] }
where
  implementation (chasm : Bool) : List (String × String) := [
    flag Temporal.DynamicConfig.Settings.history_enablechasm chasm,
    flag Temporal.DynamicConfig.Settings.history_enablechasmcallbacks chasm,
    flag Temporal.DynamicConfig.Settings.nexusoperation_enablechasmworkflowoperations chasm]

/- The switch is declared once, with its two values. -/
#guard implementationSwitch.values.map (·.name) == ["hsm", "chasm"]
#guard (implementationSwitch.values.map fun value => value.configuration.map (·.1)) ==
  [["history.enablechasm", "history.enablechasmcallbacks",
    "nexusoperation.enablechasmworkflowoperations"],
   ["history.enablechasm", "history.enablechasmcallbacks",
    "nexusoperation.enablechasmworkflowoperations"]]

/-- Every binding the realization carries, the schedule command once per class of deadline a path
of the caller Model sets. -/
def bindings (service operation : String) : List Umpire.Case.Producer.ActionBinding := [
  scheduleBinding scheduleAction "schedule-unset-unset-unset" service operation,
  scheduleBinding scheduleToStartAction "schedule-unset-expires-unset" service operation
    (scheduleToStart := timerDuration "scheduleToStart"),
  scheduleBinding startToCloseAction "schedule-unset-unset-expires" service operation
    (startToClose := timerDuration "startToClose"),
  handlerReplyBinding,
  handlerReplySyncBinding,
  handlerReplyFailedBinding,
  handlerErrorBinding handlerErrorRetryableAction "handlerReply-handlerError-true"
    "respond-error-retryable" "INTERNAL" .NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE,
  handlerErrorBinding handlerErrorNonRetryableAction "handlerReply-handlerError-false"
    "respond-error" "BAD_REQUEST" .NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE,
  completeBinding,
  completeFailedBinding,
  workerStopBinding]

end Temporal.Case.Realization.Nexus

namespace Temporal.Case.Realization

open Umpire

/-- One Nexus operation scheduled by a controller-started workflow and answered by a handler that
runs inside the Case's own worker, realized from a Model's action classes. -/
def asyncNexus (service operation : String) : Umpire.Case.Producer.Realization := {
  plan := Nexus.asyncPlan service operation
  actions := Nexus.bindings service operation
  timers := Nexus.timers
  switches := [Nexus.implementationSwitch]
  producerId := "temporal.nexus.caller.testpilot"
  producerVersion := "1"
  projectionId := Nexus.projectionId
  scopeField := Nexus.runFieldId
  operationKey := Nexus.operationFieldId
  historyObservation := Support.historyObservation
  correlatedObservation := Support.correlatedObservation
  sources := Nexus.historySources ++ [Nexus.pendingAttemptsSource]
  projectionLimits := {
    events := 32, buffered := 16, keys := 8, support := 128
    work := 1000000000, eventSize := 512 }
  runLimits := {
    «transitions» := 16, obligations := 16, work := 100000000, captures := 0 } }

/- Every class the design names is bound, and each to the instruction that carries its message: the
schedule commands to workflow commands, the replies to handler replies, the completions to
completions, the worker stop to a fault. -/
#guard ((asyncNexus "service" "operation").actions.map (·.action.value)) == [
  "temporal.nexus.caller.action.schedule",
  "temporal.nexus.caller.action.schedule.scheduleToStart",
  "temporal.nexus.caller.action.schedule.startToClose",
  "temporal.nexus.caller.action.handlerReply",
  "temporal.nexus.caller.action.handlerReply.syncSuccess",
  "temporal.nexus.caller.action.handlerReply.operationFailed",
  "temporal.nexus.caller.action.handlerReply.handlerError.retryable",
  "temporal.nexus.caller.action.handlerReply.handlerError.nonRetryable",
  "temporal.nexus.caller.action.complete",
  "temporal.nexus.caller.action.complete.failed",
  "temporal.nexus.caller.action.workerStop"]
#guard ((asyncNexus "service" "operation").actions.map fun binding =>
    match (binding.node { identity := Umpire.Case.Producer.Identity.ofFixture "temporal.case" "x" }
        "n").instruction.bind (·.instruction) with
    | some (.workflow_command _) => "command"
    | some (.nexus_handler_reply _) => "reply"
    | some (.nexus_operation_completion _) => "completion"
    | some (.inject_fault _) => "fault"
    | _ => "other") ==
  ["command", "command", "command", "reply", "reply", "reply", "reply", "reply", "completion",
    "completion", "fault"]

/- Each binding names its class by the member key a protocol Scenario's path spells. -/
#guard ((asyncNexus "service" "operation").actions.map (·.key)) == [
  "schedule-unset-unset-unset", "schedule-unset-expires-unset", "schedule-unset-unset-expires",
  "handlerReply-async", "handlerReply-syncSuccess", "handlerReply-operationFailed",
  "handlerReply-handlerError-true", "handlerReply-handlerError-false", "complete-succeeded",
  "complete-failed", "workerStop"]

/- The two deadlines a path sets realize as the realization's durations, and the backoff is the
server's. -/
#guard ((asyncNexus "service" "operation").timers.map fun timer => (timer.name, timer.milliseconds))
  == [("scheduleToStart", 2000), ("startToClose", 2000)]
#guard ((asyncNexus "service" "operation").timer? "backoff").isNone

/- Every history event kind of the operation is an admitted source, keyed by the scheduled event;
the scheduled event itself by its own id; and the read observation beside them. -/
#guard ((asyncNexus "service" "operation").sources.map (·.eventKind)) == [
  "nexusOperationScheduled", "nexusOperationStarted", "nexusOperationCompleted",
  "nexusOperationFailed", "nexusOperationCanceled", "nexusOperationTimedOut", "pendingAttempts"]

/- The recorded data each source reads: a history source's generated attributes field, resolved
from the schema rather than spelled beside the kind, and a read source's method, field and key,
which are the catalog's. The scheduled event is read out of history by a poll of its own. -/
#guard ((asyncNexus "service" "operation").sources.filterMap fun source =>
    match source.recorded with
    | .historyEvent attributesField => some attributesField
    | _ => none) ==
  ["nexus_operation_started_event_attributes",
    "nexus_operation_completed_event_attributes", "nexus_operation_failed_event_attributes",
    "nexus_operation_canceled_event_attributes", "nexus_operation_timed_out_event_attributes"]
#guard (match Nexus.scheduledSource.recorded with
  | .read method path =>
      method == ReadKind.getWorkflowExecutionHistoryMethod &&
        path == Temporal.Testpilot.CaseSupport.historyEvents
  | _ => false)
#guard Nexus.scheduledSource.operationKeyPath == "event_id"

/- The scheduled event is selected, where a relation captures it, by the operation name it records,
read off the generated schema: the arm's `operation` field, a string. -/
#guard (Nexus.scheduledSource.selector.map fun selector =>
    (selector.spelling, selector.type, selector.steps.getLast?, selector.presence.length)) ==
  some ("operation", .text,
    some (.field "temporal.api.history.v1.NexusOperationScheduledEventAttributes" 3), 2)

/- On a Case over one instance every name is the realization's own; over several, each instance's
operation, handler, slot and await carry the instance. -/
#guard Nexus.operationOf "complete" { identity := Umpire.Case.Producer.Identity.ofFixture "t" "x" }
  == "complete"
#guard Nexus.operationOf "complete"
    { identity := Umpire.Case.Producer.Identity.ofFixture "t" "x", number := 2, count := 2 }
  == "complete-2"
#guard (match Nexus.pendingAttemptsSource.recorded with
  | .read method path =>
      method == ReadKind.describeWorkflowExecutionMethod && path == "pending_nexus_operations"
  | _ => false)
#guard Nexus.pendingAttemptsSource.operationKeyPath == "scheduled_event_id"
#guard Nexus.pendingAttemptsSource.fields == []

/- The handler polls its own queue, bound apart from the caller workflow's. -/
#guard ((asyncNexus "service" "operation").plan.roles.toList.filterMap fun role =>
    if role.kind == .ROLE_KIND_TASK_QUEUE then some role.resource_binding_id else none) ==
  ["temporal.task-queue.resource", "temporal.handler-task-queue.resource"]

/- A `repeat:` resolves against the switches the realization declares: the implementation switch is
one, and an undeclared name is none, which is what rejects the `set`. -/
#guard ((asyncNexus "service" "operation").switch? "implementation").map
  (·.values.map (·.name)) == some ["hsm", "chasm"]
#guard ((asyncNexus "service" "operation").switch? "rollout").isNone

end Temporal.Case.Realization
