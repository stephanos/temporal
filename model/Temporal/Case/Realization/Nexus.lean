import Temporal.Case.Template.NexusOperation

/-!
# The Nexus caller-side realization

The `nexusOperation` template writes one Program per response form. This realization writes none: it
declares the same scaffolding and binds each **action class** of a Model to the instruction that
performs it, so the Producer puts those instructions where a Query's path took them.

Three side effects decide an asynchronous operation's outcome, and each is one action class:

| class | party | realized as | entrypoint |
| --- | --- | --- | --- |
| `schedule` | the workflow that calls the operation | `StartNexusOperation` on the Case's endpoint role | `workflow` |
| `handlerReply` | the handler | `RespondNexus`, asynchronous, publishing the completion authority | `handler` |
| `complete` | the handler | `CompleteNexusOperation` over the published handle | `controller` |

`complete` is why a realization binds classes rather than parties: the handler's completion is
performed by a controller instruction over a handle slot the handler published, so a per-party
binding would put it on the handler's entrypoint, where no such instruction can run.

Everything else the Program carries is scaffolding no Model declares -- starting the caller workflow,
waiting for the authority, waiting on the scheduled operation, finishing the workflow, and reading
history back -- and stays a fixed item. That is what the entrypoint item order says, and it is what
lets this realization produce the same Program the template does while the Producer stays free of
Nexus.

The node builders are the template's own, reused rather than copied; fn-85 .11 deletes the template
and they land here.
-/

namespace Temporal.Case.Realization.Nexus

open Umpire
open Testpilot.Authoring
open Temporal.Case.Template
open Temporal.Testpilot.CaseSupport
open temporal.server.api.testpilot.v1 hiding ModelValue

/-! ### Action classes

The Definition IDs a Model's `action` declarations carry. fn-85 .2 derives them from an `Origin`;
this proof states them, because the proof point runs before any command syntax exists. -/

/-- The caller workflow schedules the operation. -/
def scheduleAction : DefinitionId := .of "temporal.nexus.caller.action.schedule"

/-- The handler answers the start request. The asynchronous class is the one this realization binds;
a synchronous reply is a different class of the same action, bound in fn-85 .10. -/
def handlerReplyAction : DefinitionId := .of "temporal.nexus.caller.action.handlerReply"

/-- The handler completes an accepted operation, through the authority it published. -/
def completeAction : DefinitionId := .of "temporal.nexus.caller.action.complete"

/-- The handle slot the handler's asynchronous reply publishes and the completion consumes. Naming it
once is what keeps the two bindings agreeing about which authority they mean. -/
private def completionAuthority : String := "completion-authority"

/-! ### The bindings

Each binding builds its node from the id the Producer supplies, so the same class performed twice on
one path produces two distinct nodes. -/

private def scheduleBinding (service operation : String) : Umpire.Case.Producer.ActionBinding := {
  action := scheduleAction
  instructionId := "start-nexus-operation"
  node := fun _ instructionId =>
    Program.node instructionId
      (Program.startNexusOperation Support.nexusEndpointRole service operation (text "request")) }

private def handlerReplyBinding : Umpire.Case.Producer.ActionBinding := {
  action := handlerReplyAction
  instructionId := "respond-async"
  node := fun _ instructionId =>
    Program.node instructionId
      (Program.respondNexus .NEXUS_RESPONSE_KIND_ASYNCHRONOUS
        (text "accepted") completionAuthority)
      (Program.instructionLimits (timeoutMilliseconds := some 5000)) }

private def completeBinding : Umpire.Case.Producer.ActionBinding := {
  action := completeAction
  instructionId := "complete-nexus-operation"
  node := fun _ instructionId =>
    Program.node instructionId
      (Program.completeNexusOperation completionAuthority (text "completed")) }

/-! ### The plan

The controller's sequence is the one place the interleaving matters: it starts the workflow, waits for
the authority the handler publishes, performs the completion, and only then reads history. Writing
those four as ordered items is what reproduces the template's dependency edges from path order alone.
-/

/-- The asynchronous form's plan: the template's scaffolding, with the three side effects moved from
fixed nodes to the places their classes land. -/
def asyncPlan (service operation : String) : Umpire.Case.Producer.ProgramPlan := {
  roles := NexusOperation.sharedRoles
  slots := #[Program.handleSlot completionAuthority]
  observations := NexusOperation.sharedObservations
  entrypoints := [
    { activate := fun _ nodes => Program.controller "controller" nodes
      items := [
        .fixed fun identity _ =>
          NexusOperation.startWorkflowNode (NexusOperation.workflowTypeOf identity),
        .fixed fun _ _ =>
          Program.node "await-completion-authority" (Program.awaitSlot completionAuthority),
        .actions [completeAction],
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
        .fixed fun _ _ =>
          Program.node "finish-workflow"
            (Program.finish (Expr.outcome
              (Ref.instruction "workflow" "await-nexus-operation")
              .INSTRUCTION_OUTCOME_FIELD_VALUE))
            (Program.instructionLimits (timeoutMilliseconds := some 5000))] },
    { activate := fun _ nodes =>
        Program.nexusHandler "handler" service operation
          Support.workerRole Support.taskQueueRole nodes
      items := [.actions [handlerReplyAction]] }]
  cleanup := Program.cleanup "cleanup" #[] }

end Temporal.Case.Realization.Nexus

namespace Temporal.Case.Realization

open Umpire

/-- One Nexus operation answered asynchronously, realized from a Model's action classes rather than
from a whole-Program template. Every coordinate a Contract reads it back through is the template's,
so a Case produced here and a Case produced there read the same recorded history. -/
def asyncNexus (service operation : String) : Umpire.Case.Producer.Realization :=
  let template := Template.nexusOperation service operation .async
  { template with
    plan := Nexus.asyncPlan service operation
    actions := [
      Nexus.scheduleBinding service operation,
      Nexus.handlerReplyBinding,
      Nexus.completeBinding] }

end Temporal.Case.Realization
