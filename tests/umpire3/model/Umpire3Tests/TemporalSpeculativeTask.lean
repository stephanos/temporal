import Temporal.Monitors
import Temporal.Refinement.MigratedFamilies

namespace Umpire3.Tests.TemporalSpeculativeTask

set_option maxRecDepth 100000

def committedActions : List Umpire3.Temporal.Product.SpeculativeTask.Action := [
  .requestUpdate .primary,
  .create .primary .primary,
  .commit .primary,
]

def orphanedActions : List Umpire3.Temporal.Product.SpeculativeTask.Action := [
  .create .primary .secondary,
]

example : Umpire3.Temporal.Product.SpeculativeTask.executable.follow
    [Umpire3.Temporal.Product.SpeculativeTask.initial] committedActions =
    [Umpire3.Temporal.Product.SpeculativeTask.committedFinal] := by decide

example : Umpire3.Temporal.Product.SpeculativeTask.SpeculativeQualified
    Umpire3.Temporal.Product.SpeculativeTask.committedFinal := by decide

example : Umpire3.Temporal.Product.SpeculativeTask.executable.follow
    [Umpire3.Temporal.Product.SpeculativeTask.initial] orphanedActions = [] := by decide

example : Umpire3.Temporal.Product.SpeculativeTask.weakenedExecutable.follow
    [Umpire3.Temporal.Product.SpeculativeTask.initial] orphanedActions =
    [Umpire3.Temporal.Product.SpeculativeTask.orphanedFinal] := by decide

example : ¬Umpire3.Temporal.Product.SpeculativeTask.SpeculativeTaskCreation
    Umpire3.Temporal.Product.SpeculativeTask.orphanedFinal := by decide

example :
    (Umpire3.Temporal.Product.SpeculativeTask.bounded.explore
      { maxDepth := 3, maxResults := 5000 }).all
      (fun execution => Umpire3.Temporal.Product.SpeculativeTask.speculativeCreationB execution.2) = true := by decide

def systemActions :
    List Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.Action := [
  .requestUpdate,
  .createTask,
  .commitTask,
]

def orphanedSystemActions :
    List Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.Action := [
  .createOrphan,
]

example : Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.executable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.initial] systemActions =
    [.taskCommitted] := by decide

example : Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.executable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.initial]
    orphanedSystemActions = [] := by decide

example : Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.initial]
    orphanedSystemActions = [.orphanedTask] := by decide

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.MigratedFamilies.SpeculativeTask.System.mutatedBehavior
      Umpire3.Temporal.Refinement.MigratedFamilies.SpeculativeTask.Feature.behavior
      Umpire3.Temporal.Refinement.MigratedFamilies.SpeculativeTask.Projects
      Umpire3.Temporal.Refinement.MigratedFamilies.SpeculativeTask.actionMap :=
  Umpire3.Temporal.Refinement.MigratedFamilies.SpeculativeTask.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.speculativeTaskCreation.Holds
    (Umpire3.Temporal.Monitors.speculativeTaskObservations
      Umpire3.Temporal.Product.SpeculativeTask.committedFinal) :=
  (Umpire3.Temporal.Monitors.speculativeTask_monitor_equivalent
    Umpire3.Temporal.Product.SpeculativeTask.committedFinal).2 (by decide)

end Umpire3.Tests.TemporalSpeculativeTask
