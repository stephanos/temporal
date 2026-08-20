import Temporal.Refinement.NexusCancellationFencing
import Temporal.Targets.UpdateLifecycleBehavior

namespace Umpire3.Tests.TemporalNexusFencing

example : Umpire3.Temporal.Feature.NexusCancellationFencing.executable.initials .smoke =
    [Umpire3.Temporal.Feature.NexusCancellationFencing.initial] := rfl

example :
    (Umpire3.Temporal.System.NexusCancellationFencing.Action.dispatchTask,
      { Umpire3.Temporal.System.NexusCancellationFencing.initial with
        task := .dispatched, workerEpoch := some 0 }) ∈
      Umpire3.Temporal.System.NexusCancellationFencing.executable.successors .smoke
        Umpire3.Temporal.System.NexusCancellationFencing.initial := by
  apply (Umpire3.Temporal.System.NexusCancellationFencing.executable.successors_exact
    .smoke Umpire3.Temporal.System.NexusCancellationFencing.initial
    .dispatchTask _).mpr
  change { Umpire3.Temporal.System.NexusCancellationFencing.initial with
      task := .dispatched, workerEpoch := some 0 } ∈
    Umpire3.Temporal.System.NexusCancellationFencing.next
      .smoke Umpire3.Temporal.System.NexusCancellationFencing.initial .dispatchTask
  simp [Umpire3.Temporal.System.NexusCancellationFencing.next,
    Umpire3.Temporal.System.NexusCancellationFencing.initial]

example : Umpire3.SafetySimulation
    Umpire3.Temporal.System.NexusCancellationFencing.behavior
    Umpire3.Temporal.Feature.NexusCancellationFencing.behavior :=
  Umpire3.Temporal.Refinement.NexusCancellationFencing.soundSimulation

example : Umpire3.Runs
    (Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior.at .smoke)
    Umpire3.Temporal.System.NexusCancellationFencing.initial
    Umpire3.Temporal.System.NexusCancellationFencing.mutatedCounterexampleActions
    Umpire3.Temporal.System.NexusCancellationFencing.staleSuccess :=
  Umpire3.Temporal.System.NexusCancellationFencing.mutatedCounterexample

example :
    Umpire3.Temporal.System.NexusCancellationFencing.mutatedExecutable.follow .smoke
      [Umpire3.Temporal.System.NexusCancellationFencing.initial]
      Umpire3.Temporal.System.NexusCancellationFencing.mutatedCounterexampleActions =
    [Umpire3.Temporal.System.NexusCancellationFencing.staleSuccess] := by
  decide

example :
    Umpire3.Temporal.System.NexusCancellationFencing.executable.follow .smoke
      [Umpire3.Temporal.System.NexusCancellationFencing.initial]
      Umpire3.Temporal.System.NexusCancellationFencing.mutatedCounterexampleActions = [] := by
  decide

def ownershipBound : Umpire3.Temporal.System.NexusCancellationFencing.State :=
  { Umpire3.Temporal.System.NexusCancellationFencing.afterCancellationAccepted with
    ownerEpoch :=
      Umpire3.Temporal.NexusCancellationFencing.World.maxOwnerEpoch .smoke }

example : Umpire3.Temporal.System.NexusCancellationFencing.next .smoke ownershipBound
    .acquireOwnership = [] := by
  decide

example : ¬Umpire3.StepSimulation
    Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior
    Umpire3.Temporal.Feature.NexusCancellationFencing.behavior
    Umpire3.Temporal.Refinement.NexusCancellationFencing.Projects
    Umpire3.Temporal.Refinement.NexusCancellationFencing.actionMap :=
  Umpire3.Temporal.Refinement.NexusCancellationFencing.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Targets.UpdateLifecycleBehavior.featureExecutable.initials () =
    [Umpire3.Temporal.Product.Update.initial] := rfl

example : Umpire3.Temporal.Targets.UpdateLifecycleBehavior.systemExecutable.initials () =
    [Umpire3.Temporal.System.UpdateTasks.initial] := rfl

end Umpire3.Tests.TemporalNexusFencing
