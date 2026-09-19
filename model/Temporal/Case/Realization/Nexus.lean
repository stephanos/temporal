import Temporal.Case.Template.NexusOperation
import Temporal.DynamicConfig

/-!
# The Nexus caller-side realization

The `nexusOperation` template writes one Program per response form. This realization writes none: it
declares the same scaffolding and binds each **action class** of a Model to the instruction that
performs it, so the Producer puts those instructions where a Query's path took them.

Three side effects decide an asynchronous operation's outcome, and each is one action class,
realized as the typed worker instruction that carries the Temporal API message the class names
(fn-85 R10):

| class | party | realized as | entrypoint |
| --- | --- | --- | --- |
| `schedule` | the workflow that calls the operation | a workflow command carrying `ScheduleNexusOperationCommandAttributes` on the Case's endpoint role | `workflow` |
| `handlerReply (async)` | the handler | a handler reply carrying `StartOperationResponse.async_success`, publishing the completion authority | `handler` |
| `handlerReply (syncSuccess)` | the handler | a handler reply carrying `StartOperationResponse.sync_success` with its payload | `handler` |
| `handlerReply (operationFailed)` | the handler | a handler reply carrying `StartOperationResponse.failure` | `handler` |
| `handlerReply (handlerError (retryable := r))` | the handler | a handler reply carrying `HandlerError` with that retry behavior | `handler` |
| `complete (succeeded)` | the handler | a completion carrying a `Payload` over the published handle | `controller` |
| `complete (failed)` | the handler | a completion carrying a `Failure` over the published handle | `controller` |

`complete` is why a realization binds classes rather than parties: the handler's completion is
performed by a controller instruction over a handle slot the handler published, so a per-party
binding would put it on the handler's entrypoint, where no such instruction can run.

Everything else the Program carries is scaffolding no Model declares -- starting the caller workflow,
waiting for the authority, waiting on the scheduled operation, finishing the workflow, and reading
history back -- and stays a fixed item. That is what the entrypoint item order says, and it is what
lets this realization produce the same Program shape the template does while the Producer stays free
of Nexus.

The scaffolding node builders are the template's own, reused rather than copied; fn-85 .11 deletes
the template and they land here.

Each binding names its class by the member key a Scenario's path spells it by
(`handlerReply-async`, `complete-succeeded`), which the Producer resolves against the Model's own
vocabulary, so the caller Model's protocol machine and any Model over the same classes produce
through one realization. The hand-stated Definition IDs beside the keys are read only by the success
slice, whose actions are waits and whose path the realization states (`asyncPath`), until fn-85 .11
retires it.
-/

namespace Temporal.Case.Realization.Nexus

open Umpire
open Testpilot.Authoring
open Temporal.Case.Template
open Temporal.Testpilot.CaseSupport
open temporal.server.api.testpilot.v1 hiding ModelValue

/-! ### Action classes

The Definition IDs a Model's `action` declarations carry. fn-85 .2 derives them from an `Origin`;
this proof states them, because the proof point runs before any command syntax exists. A class of an
action is stated as its own ID, because a binding is per class: the Producer's path names classes,
and each is realized as a different message. -/

/-- The caller workflow schedules the operation. -/
def scheduleAction : DefinitionId := .of "temporal.nexus.caller.action.schedule"

/-- The handler answers the start request asynchronously, publishing the completion authority. -/
def handlerReplyAction : DefinitionId := .of "temporal.nexus.caller.action.handlerReply"

/-- The handler answers synchronously with the operation's result. -/
def handlerReplySyncAction : DefinitionId :=
  .of "temporal.nexus.caller.action.handlerReply.syncSuccess"

/-- The handler answers that the operation failed at its start. -/
def handlerReplyFailedAction : DefinitionId :=
  .of "temporal.nexus.caller.action.handlerReply.operationFailed"

/-- The handler answers with a retryable handler error. -/
def handlerErrorRetryableAction : DefinitionId :=
  .of "temporal.nexus.caller.action.handlerReply.handlerError.retryable"

/-- The handler answers with a non-retryable handler error. -/
def handlerErrorNonRetryableAction : DefinitionId :=
  .of "temporal.nexus.caller.action.handlerReply.handlerError.nonRetryable"

/-- The handler completes an accepted operation successfully, through the authority it published. -/
def completeAction : DefinitionId := .of "temporal.nexus.caller.action.complete"

/-- The handler completes an accepted operation as failed. -/
def completeFailedAction : DefinitionId := .of "temporal.nexus.caller.action.complete.failed"

/-- The path an asynchronously answered operation takes: the caller schedules, the handler accepts
asynchronously, the handler completes. A `case` block over the success slice realizes this path,
because that slice's own actions are waits that realize nothing; fn-85 .10's protocol machine makes
the Query's own path this one. -/
def asyncPath : List DefinitionId := [scheduleAction, handlerReplyAction, completeAction]

/-- The handle slot the handler's asynchronous reply publishes and the completion consumes. Naming it
once is what keeps the two bindings agreeing about which authority they mean. -/
private def completionAuthority : String := "completion-authority"

/-- The failure a failed reply or completion carries: an application failure the handler names. -/
private def handlerFailure : temporal.api.failure.v1.Failure :=
  { «message» := "operation failed"
    failure_info := some (.application_failure_info {
      «type» := "OperationFailed", non_retryable := true }) }

/-! ### The bindings

Each binding builds its node from the id the Producer supplies, so the same class performed twice on
one path produces two distinct nodes. The messages each carries are the ones the class names; the
concrete values (the request and result payloads, the error types) are the realization's, until a
Model's `examples:` supply them. -/

private def scheduleBinding (service operation : String) : Umpire.Case.Producer.ActionBinding := {
  action := scheduleAction
  key := "schedule-unset-unset-unset"
  instructionId := "start-nexus-operation"
  node := fun _ instructionId =>
    Program.node instructionId
      (Program.scheduleNexusOperation Support.nexusEndpointRole service operation
        (Payload.text "request")) }

private def handlerReplyBinding : Umpire.Case.Producer.ActionBinding := {
  action := handlerReplyAction
  key := "handlerReply-async"
  instructionId := "respond-async"
  node := fun _ instructionId =>
    Program.node instructionId (Program.nexusAsyncReply completionAuthority)
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
  node := fun _ instructionId =>
    Program.node instructionId
      (Program.nexusOperationCompletion completionAuthority (Payload.text "completed")) }

private def completeFailedBinding : Umpire.Case.Producer.ActionBinding := {
  action := completeFailedAction
  key := "complete-failed"
  instructionId := "fail-nexus-operation"
  node := fun _ instructionId =>
    Program.node instructionId (Program.nexusOperationFailure completionAuthority handlerFailure) }

/-! ### The plan

The controller's sequence is the one place the interleaving matters: it starts the workflow, waits for
the authority the handler publishes when a completion is on the path, performs the completion, waits
for the workflow to close, and only then reads history. Writing those as ordered items is what
reproduces the template's dependency edges from path order alone.

One plan serves every Query of a set. A synchronous reply and a handler error publish no authority,
so the wait for it is emitted only when a completion class is on the path; the close-event read is
what orders the history read behind the operation's outcome on every path, and the workflow finishes
whether or not the operation succeeded, so its history closes on every path.
-/

/-- The classes whose instruction consumes the completion authority. -/
private def completionKeys : List String := ["complete-succeeded", "complete-failed"]

/-- The plan every caller-side Case is assembled from: the template's scaffolding, with the side
effects moved from fixed nodes to the places their classes land. -/
def asyncPlan (service operation : String) : Umpire.Case.Producer.ProgramPlan := {
  roles := NexusOperation.sharedRoles
  slots := #[Program.handleSlot completionAuthority]
  observations := NexusOperation.sharedObservations
  entrypoints := [
    { activate := fun _ nodes => Program.controller "controller" nodes
      items := [
        .fixed fun identity _ =>
          NexusOperation.startWorkflowNode (NexusOperation.workflowTypeOf identity),
        .whenOnPath completionKeys fun _ _ =>
          Program.node "await-completion-authority" (Program.awaitSlot completionAuthority),
        .actions [completeAction, completeFailedAction],
        .fixed fun _ _ => NexusOperation.awaitCloseNode,
        .fixed fun identity resolved => NexusOperation.historyNode identity resolved] },
    { activate := fun identity nodes =>
        NexusOperation.workflowEntrypointWith (NexusOperation.workflowTypeOf identity)
          service operation nodes
      items := [
        -- The schedule is the Model's; the await and the finish are scaffolding, because no Model
        -- declares them. They are written out rather than taken from the template's item list, so a
        -- change to that list cannot silently drop one here.
        .actions [scheduleAction],
        .fixed fun _ _ =>
          Program.node "await-nexus-operation"
            (Program.awaitInstruction (Ref.instruction "workflow" "start-nexus-operation"))
            (guard := some (boolean true)),
        -- The workflow closes on every path: a failed operation is the await's recorded outcome,
        -- not a reason to leave the workflow open, so the finish runs regardless and returns a
        -- literal rather than the awaited payload a failed operation has none of.
        .fixed fun _ _ =>
          Program.node "finish-workflow" (Program.finish (text "done"))
            (Program.instructionLimits (timeoutMilliseconds := some 5000))
            (guard := some (boolean true))] },
    { activate := fun _ nodes =>
        Program.nexusHandler "handler" service operation
          Support.workerRole Support.taskQueueRole nodes
      items := [.actions [handlerReplyAction, handlerReplySyncAction, handlerReplyFailedAction,
        handlerErrorRetryableAction, handlerErrorNonRetryableAction]] }]
  cleanup := Program.cleanup "cleanup" #[] }

/-! ### The switch and the setup parameters

The realization declares the switch a functional set repeats over and binds the machine's setup
parameters to the configuration keys the Profile sets them through. Both name keys of the generated
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

end Temporal.Case.Realization.Nexus

namespace Temporal.Case.Realization

open Umpire

/-- One Nexus operation answered asynchronously, realized from a Model's action classes rather than
from a whole-Program template. Every coordinate a Contract reads it back through is the template's,
so a Case produced here and a Case produced there read the same recorded history. -/
def asyncNexus (service operation : String) : Umpire.Case.Producer.Realization :=
  let template := Template.nexusOperation service operation .async
  { template with
    producerId := "temporal.nexus.caller.testpilot"
    plan := Nexus.asyncPlan service operation
    actions := [
      Nexus.scheduleBinding service operation,
      Nexus.handlerReplyBinding,
      Nexus.handlerReplySyncBinding,
      Nexus.handlerReplyFailedBinding,
      Nexus.handlerErrorBinding Nexus.handlerErrorRetryableAction "handlerReply-handlerError-true"
        "respond-error-retryable" "INTERNAL" .NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE,
      Nexus.handlerErrorBinding Nexus.handlerErrorNonRetryableAction
        "handlerReply-handlerError-false" "respond-error" "BAD_REQUEST"
        .NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE,
      Nexus.completeBinding,
      Nexus.completeFailedBinding]
    sources := Template.NexusOperation.historySources ++
      [Template.NexusOperation.pendingAttemptsSource]
    switches := [Nexus.implementationSwitch] }

/- Every class the design names is bound, and each to the instruction that carries its message: the
schedule to a workflow command, the replies to handler replies, the completions to completions. -/
#guard ((asyncNexus "service" "operation").actions.map (·.action.value)) == [
  "temporal.nexus.caller.action.schedule",
  "temporal.nexus.caller.action.handlerReply",
  "temporal.nexus.caller.action.handlerReply.syncSuccess",
  "temporal.nexus.caller.action.handlerReply.operationFailed",
  "temporal.nexus.caller.action.handlerReply.handlerError.retryable",
  "temporal.nexus.caller.action.handlerReply.handlerError.nonRetryable",
  "temporal.nexus.caller.action.complete",
  "temporal.nexus.caller.action.complete.failed"]
#guard ((asyncNexus "service" "operation").actions.map fun binding =>
    match (binding.node (Umpire.Case.Producer.Identity.ofFixture "temporal.case" "x") "n").instruction.bind (·.instruction) with
    | some (.workflow_command _) => "command"
    | some (.nexus_handler_reply _) => "reply"
    | some (.nexus_operation_completion _) => "completion"
    | _ => "other") ==
  ["command", "reply", "reply", "reply", "reply", "reply", "completion", "completion"]

/- Each binding names its class by the member key a protocol Scenario's path spells. -/
#guard ((asyncNexus "service" "operation").actions.map (·.key)) == [
  "schedule-unset-unset-unset", "handlerReply-async", "handlerReply-syncSuccess",
  "handlerReply-operationFailed", "handlerReply-handlerError-true",
  "handlerReply-handlerError-false", "complete-succeeded", "complete-failed"]

/- Every history event kind of the operation is an admitted source, keyed by the scheduled event;
the scheduled event itself by its own id; and the read observation beside them. -/
#guard ((asyncNexus "service" "operation").sources.map (·.eventKind)) == [
  "nexusOperationScheduled", "nexusOperationStarted", "nexusOperationCompleted",
  "nexusOperationFailed", "nexusOperationCanceled", "nexusOperationTimedOut", "pendingAttempts"]

/- A `repeat:` resolves against the switches the realization declares: the implementation switch is
one, and an undeclared name is none, which is what rejects the `set`. -/
#guard ((asyncNexus "service" "operation").switch? "implementation").map
  (·.values.map (·.name)) == some ["hsm", "chasm"]
#guard ((asyncNexus "service" "operation").switch? "rollout").isNone

end Temporal.Case.Realization
