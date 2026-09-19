import Temporal.Case.Syntax

/-!
Pins for the realization templates: the hooks and evidence sources each declares, the ordering the
synchronous Nexus form depends on, and the event kinds the generated schema admits.
-/

namespace Temporal.Case.Tests.Template

open Temporal.Case
open temporal.server.api.testpilot.v1

private def identity : Umpire.Case.Producer.Identity :=
  Umpire.Case.Producer.Identity.ofFixture Case.caseIdRoot "example"

private def hookNames (realization : Umpire.Case.Producer.Realization) : List String :=
  realization.hooks.map (·.name)

private def sourceKinds (realization : Umpire.Case.Producer.Realization) : List String :=
  realization.sources.map (·.eventKind)

private def asyncNexus := Case.Template.nexusOperation "umpire.case.service" "complete" .async
private def syncNexus := Case.Template.nexusOperation "umpire.case.service" "complete" .sync
private def workflowTemplate := Case.Template.workflow "umpire-example-workflow"

/-! Every template names exactly the two hooks a `fault` line may be placed against. -/

#guard hookNames asyncNexus == ["start", "completion"]
#guard hookNames syncNexus == ["start", "completion"]
#guard hookNames workflowTemplate == ["start", "completion"]

/-! Each declares the evidence kinds its Cases may map evidence to, and nothing else: the Nexus
templates carry the two history events and the `pendingAttempts` read, the workflow template one
history event. -/

#guard sourceKinds asyncNexus == ["nexusOperationStarted", "nexusOperationCompleted", "pendingAttempts"]
#guard sourceKinds syncNexus == ["nexusOperationStarted", "nexusOperationCompleted", "pendingAttempts"]
#guard sourceKinds workflowTemplate == ["workflowExecutionCompleted"]

/-! The recorded data each source reads: a history source's generated attributes field, resolved
from the schema rather than spelled beside the kind, and the read source's method, field and key,
which are the catalog's. -/

private def attributesFields (realization : Umpire.Case.Producer.Realization) : List String :=
  realization.sources.filterMap fun source =>
    match source.recorded with
    | .historyEvent attributesField => some attributesField
    | _ => none

#guard attributesFields asyncNexus ==
  ["nexus_operation_started_event_attributes", "nexus_operation_completed_event_attributes"]
#guard attributesFields workflowTemplate == ["workflow_execution_completed_event_attributes"]

#guard (match Case.Template.NexusOperation.pendingAttemptsSource.recorded with
  | .read method path =>
      method == Case.ReadKind.describeWorkflowExecutionMethod && path == "pending_nexus_operations"
  | _ => false)
#guard Case.Template.NexusOperation.pendingAttemptsSource.operationKeyPath == "scheduled_event_id"
#guard Case.Template.NexusOperation.pendingAttemptsSource.fields == [("attempts", "attempt")]

/-! ### Ordering in the synchronous Nexus form

Nothing in the controller observes the workflow, so the read the Projection consumes must be
ordered behind a close-event read. -/

private def controllerInstructions (realization : Umpire.Case.Producer.Realization) :
    Array InstructionNode :=
  match realization.program identity with
  | .error _ => #[]
  | .ok program =>
      match program.entrypoints.find? (·.entrypoint_id == "controller") with
      | some entrypoint => entrypoint.instructions
      | none => #[]

/-- The instructions an instruction runs after: its `after` set, or the instruction before it. -/
private def dependencies (realization : Umpire.Case.Producer.Realization) (instructionId : String) :
    List String :=
  let instructions := (controllerInstructions realization).toList
  match instructions.findIdx? (·.instruction_id == instructionId) with
  | some index => match instructions[index]?.bind (·.after) with
    | some after => after.instructions.toList.map (·.instruction_id)
    | none => if index == 0 then [] else (instructions[index - 1]?.map (·.instruction_id)).toList
  | none => []

private def instructionIds (realization : Umpire.Case.Producer.Realization) : List String :=
  (controllerInstructions realization).toList.map (·.instruction_id)

#guard instructionIds syncNexus == ["start-workflow", "await-close", "history"]
#guard dependencies syncNexus "await-close" == ["start-workflow"]
#guard dependencies syncNexus "history" == ["await-close"]

/-- Whether one instruction's RPC request assigns the close-event filter. The close-event read is
what makes the ordering above meaningful: it carries the filter, and the full read does not. -/
private def assignsCloseFilter
    (realization : Umpire.Case.Producer.Realization) (instructionId : String) : Bool :=
  let assignsFilter := fun (assignment : RequestAssignment) =>
    assignment.target == "history_event_filter_type"
  match (controllerInstructions realization).find? (·.instruction_id == instructionId) with
  | some declared =>
      match declared.instruction with
      | some carried =>
          match carried.instruction with
          | some (.invoke_rpc rpc) => rpc.request_assignments.any assignsFilter
          | _ => false
      | none => false
  | none => false

#guard assignsCloseFilter syncNexus "await-close"
#guard !assignsCloseFilter syncNexus "history"

/-! The asynchronous form keeps its completion pair and needs no extra read. -/

#guard instructionIds asyncNexus ==
  ["start-workflow", "await-completion-authority", "complete-nexus-operation", "history"]
#guard dependencies asyncNexus "history" == ["complete-nexus-operation"]

/-! ### Event kind resolution -/

#guard EventKind.attributesField? "nexusOperationStarted" ==
  some "nexus_operation_started_event_attributes"
#guard EventKind.attributesField? "nexusOperationCompleted" ==
  some "nexus_operation_completed_event_attributes"
#guard EventKind.attributesField? "workflowExecutionCompleted" ==
  some "workflow_execution_completed_event_attributes"

/-! An unknown kind rejects against what the generated schema carries, listing it. -/

private def rejection : String :=
  match EventKind.resolve "nexusOperationSucceeded" with
  | .error reported => reported
  | .ok resolved => resolved

#guard rejection.startsWith "unknown history event kind 'nexusOperationSucceeded'; admitted: "
#guard (rejection.splitOn "nexusOperationStarted").length == 2
#guard (rejection.splitOn "workflowExecutionCompleted").length == 2

end Temporal.Case.Tests.Template
