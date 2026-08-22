import Temporal.Monitors
import Temporal.Families

namespace Umpire3.Tests.TemporalSpeculativeTask

set_option maxRecDepth 100000

def committedActions : List Umpire3.Temporal.Feature.SpeculativeTask.Action := [
  .requestUpdate .primary,
  .create .primary .primary,
  .commit .primary,
]

def orphanedActions : List Umpire3.Temporal.Feature.SpeculativeTask.Action := [
  .create .primary .secondary,
]

example : Umpire3.Temporal.Feature.SpeculativeTask.executable.follow
    [Umpire3.Temporal.Feature.SpeculativeTask.initial] committedActions =
    [Umpire3.Temporal.Feature.SpeculativeTask.committedFinal] := by decide

example : Umpire3.Temporal.Feature.SpeculativeTask.SpeculativeQualified
    Umpire3.Temporal.Feature.SpeculativeTask.committedFinal := by decide

example : Umpire3.Temporal.Feature.SpeculativeTask.executable.follow
    [Umpire3.Temporal.Feature.SpeculativeTask.initial] orphanedActions = [] := by decide

example : Umpire3.Temporal.Feature.SpeculativeTask.weakenedExecutable.follow
    [Umpire3.Temporal.Feature.SpeculativeTask.initial] orphanedActions =
    [Umpire3.Temporal.Feature.SpeculativeTask.orphanedFinal] := by decide

example : ¬Umpire3.Temporal.Feature.SpeculativeTask.SpeculativeTaskCreation
    Umpire3.Temporal.Feature.SpeculativeTask.orphanedFinal := by decide

example :
    (Umpire3.Temporal.Feature.SpeculativeTask.bounded.explore
      { maxDepth := 3, maxResults := 5000 }).all
      (fun execution => Umpire3.Temporal.Feature.SpeculativeTask.speculativeCreationB execution.2) = true := by decide

def systemActions :
    List Umpire3.Temporal.System.SpeculativeTask.Action := [
  .requestUpdate,
  .createTask,
  .commitTask,
]

def orphanedSystemActions :
    List Umpire3.Temporal.System.SpeculativeTask.Action := [
  .createOrphan,
]

example : Umpire3.Temporal.System.SpeculativeTask.executable.follow ()
    [Umpire3.Temporal.System.SpeculativeTask.initial] systemActions =
    [.taskCommitted] := by decide

example : Umpire3.Temporal.System.SpeculativeTask.executable.follow ()
    [Umpire3.Temporal.System.SpeculativeTask.initial]
    orphanedSystemActions = [] := by decide

example : Umpire3.Temporal.System.SpeculativeTask.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.SpeculativeTask.initial]
    orphanedSystemActions = [.orphanedTask] := by decide

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.SpeculativeTask.System.mutatedBehavior
      Umpire3.Temporal.Refinement.SpeculativeTask.Feature.behavior
      Umpire3.Temporal.Refinement.SpeculativeTask.Projects
      Umpire3.Temporal.Refinement.SpeculativeTask.actionMap :=
  Umpire3.Temporal.Refinement.SpeculativeTask.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.speculativeTaskCreation.Holds
    (Umpire3.Temporal.Monitors.speculativeTaskObservations
      Umpire3.Temporal.Feature.SpeculativeTask.committedFinal) :=
  (Umpire3.Temporal.Monitors.speculativeTask_monitor_equivalent
    Umpire3.Temporal.Feature.SpeculativeTask.committedFinal).2 (by decide)

end Umpire3.Tests.TemporalSpeculativeTask
