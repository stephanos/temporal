import Temporal.Monitors
import Temporal.Families.WorkflowOwnership.Refinement

namespace Umpire3.Tests.Families.WorkflowOwnership

set_option maxRecDepth 100000

def ownershipActions : List Umpire3.Temporal.Feature.WorkflowOwnership.Action := [
  .bootstrap .primary .first,
  .dispatch .primary .primary .first,
  .fail .primary,
  .rotate .primary .first .second,
  .dispatch .secondary .primary .second,
  .rejectStale .primary,
  .complete .secondary,
]

def staleCompletionActions : List Umpire3.Temporal.Feature.WorkflowOwnership.Action := [
  .bootstrap .primary .first,
  .dispatch .primary .primary .first,
  .fail .primary,
  .rotate .primary .first .second,
  .complete .primary,
]

example : Umpire3.Temporal.Feature.WorkflowOwnership.executable.follow
    [Umpire3.Temporal.Feature.WorkflowOwnership.initial] ownershipActions =
    [Umpire3.Temporal.Feature.WorkflowOwnership.fencedFinal] := by decide

example : Umpire3.Temporal.Feature.WorkflowOwnership.OwnershipQualified
    Umpire3.Temporal.Feature.WorkflowOwnership.fencedFinal := by decide

example : Umpire3.Temporal.Feature.WorkflowOwnership.executable.follow
    [Umpire3.Temporal.Feature.WorkflowOwnership.initial] staleCompletionActions = [] := by decide

example : Umpire3.Temporal.Feature.WorkflowOwnership.weakenedExecutable.follow
    [Umpire3.Temporal.Feature.WorkflowOwnership.initial] staleCompletionActions =
    [Umpire3.Temporal.Feature.WorkflowOwnership.staleCompletionFinal] := by decide

example : ¬Umpire3.Temporal.Feature.WorkflowOwnership.OwnershipFencing
    Umpire3.Temporal.Feature.WorkflowOwnership.staleCompletionFinal := by decide

example :
    (Umpire3.Temporal.Feature.WorkflowOwnership.bounded.explore
      { maxDepth := 3, maxResults := 5000 }).all
      (fun execution => Umpire3.Temporal.Feature.WorkflowOwnership.ownershipFencingB execution.2) = true := by decide

def ownershipSystemActions :
    List Umpire3.Temporal.System.WorkflowOwnership.Action := [
  .dispatchCurrent,
  .failCurrent,
  .rotateOwner,
  .rejectStale,
  .completeCurrent,
]

def staleCompletionSystemActions :
    List Umpire3.Temporal.System.WorkflowOwnership.Action := [
  .dispatchCurrent,
  .failCurrent,
  .rotateOwner,
  .completeStale,
]

example : Umpire3.Temporal.System.WorkflowOwnership.executable.follow ()
    [Umpire3.Temporal.System.WorkflowOwnership.initial] ownershipSystemActions =
    [.currentCompleted] := by decide

example : Umpire3.Temporal.System.WorkflowOwnership.executable.follow ()
    [Umpire3.Temporal.System.WorkflowOwnership.initial]
    staleCompletionSystemActions = [] := by decide

example : Umpire3.Temporal.System.WorkflowOwnership.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.WorkflowOwnership.initial]
    staleCompletionSystemActions = [.staleCompleted] := by decide

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.WorkflowOwnership.System.mutatedBehavior
      Umpire3.Temporal.Refinement.WorkflowOwnership.Feature.behavior
      Umpire3.Temporal.Refinement.WorkflowOwnership.Projects
      Umpire3.Temporal.Refinement.WorkflowOwnership.actionMap :=
  Umpire3.Temporal.Refinement.WorkflowOwnership.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.workflowOwnershipFencing.Holds
    (Umpire3.Temporal.Monitors.workflowOwnershipObservations
      Umpire3.Temporal.Feature.WorkflowOwnership.fencedFinal) :=
  (Umpire3.Temporal.Monitors.workflowOwnership_monitor_equivalent
    Umpire3.Temporal.Feature.WorkflowOwnership.fencedFinal).2 (by decide)

end Umpire3.Tests.Families.WorkflowOwnership
