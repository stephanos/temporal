import Temporal.Monitors
import Temporal.Families

namespace Umpire3.Tests.TemporalWorkflowProgress

set_option maxRecDepth 100000

def progressedActions : List Umpire3.Temporal.Feature.WorkflowProgress.Action := [
  .enqueue .primary .primary,
  .makeWorkerAvailable .primary,
  .wait .primary,
  .dispatch .primary .primary,
  .complete .primary .primary,
]

def starvedActions : List Umpire3.Temporal.Feature.WorkflowProgress.Action := [
  .enqueue .primary .primary,
  .makeWorkerAvailable .primary,
  .wait .primary,
  .wait .primary,
]

def wrongEntityActions : List Umpire3.Temporal.Feature.WorkflowProgress.Action := [
  .enqueue .primary .primary,
  .makeWorkerAvailable .primary,
  .dispatch .primary .primary,
  .complete .primary .secondary,
]

example : Umpire3.Temporal.Feature.WorkflowProgress.executable.follow
    [Umpire3.Temporal.Feature.WorkflowProgress.initial] progressedActions =
    [Umpire3.Temporal.Feature.WorkflowProgress.progressedFinal] := by decide

example : Umpire3.Temporal.Feature.WorkflowProgress.StarvationQualified
    Umpire3.Temporal.Feature.WorkflowProgress.progressedFinal := by decide

example : Umpire3.Temporal.Feature.WorkflowProgress.ProgressQualified
    Umpire3.Temporal.Feature.WorkflowProgress.progressedFinal := by decide

example : Umpire3.Temporal.Feature.WorkflowProgress.executable.follow
    [Umpire3.Temporal.Feature.WorkflowProgress.initial] starvedActions = [] := by decide

example : Umpire3.Temporal.Feature.WorkflowProgress.weakenedExecutable.follow
    [Umpire3.Temporal.Feature.WorkflowProgress.initial] starvedActions =
    [Umpire3.Temporal.Feature.WorkflowProgress.starvedFinal] := by decide

example : ¬Umpire3.Temporal.Feature.WorkflowProgress.WorkflowTaskStarvation
    Umpire3.Temporal.Feature.WorkflowProgress.starvedFinal := by decide

example : Umpire3.Temporal.Feature.WorkflowProgress.executable.follow
    [Umpire3.Temporal.Feature.WorkflowProgress.initial] wrongEntityActions = [] := by decide

example : Umpire3.Temporal.Feature.WorkflowProgress.weakenedExecutable.follow
    [Umpire3.Temporal.Feature.WorkflowProgress.initial] wrongEntityActions =
    [Umpire3.Temporal.Feature.WorkflowProgress.wrongEntityFinal] := by decide

example : ¬Umpire3.Temporal.Feature.WorkflowProgress.EntityProgress
    Umpire3.Temporal.Feature.WorkflowProgress.wrongEntityFinal := by decide

example :
    (Umpire3.Temporal.Feature.WorkflowProgress.bounded.explore
      { maxDepth := 4, maxResults := 10000 }).all
      (fun execution => Umpire3.Temporal.Feature.WorkflowProgress.invariantB execution.2) = true := by decide

def systemActions :
    List Umpire3.Temporal.System.WorkflowProgress.Action := [
  .enqueue,
  .makeWorkerAvailable,
  .wait,
  .dispatch,
  .complete,
]

def starvedSystemActions :
    List Umpire3.Temporal.System.WorkflowProgress.Action := [
  .enqueue,
  .makeWorkerAvailable,
  .wait,
  .waitAgain,
]

def wrongEntitySystemActions :
    List Umpire3.Temporal.System.WorkflowProgress.Action := [
  .enqueue,
  .makeWorkerAvailable,
  .wait,
  .dispatch,
  .completeWrongEntity,
]

example : Umpire3.Temporal.System.WorkflowProgress.executable.follow ()
    [Umpire3.Temporal.System.WorkflowProgress.initial] systemActions =
    [.completed] := by decide

example : Umpire3.Temporal.System.WorkflowProgress.executable.follow ()
    [Umpire3.Temporal.System.WorkflowProgress.initial]
    starvedSystemActions = [] := by decide

example : Umpire3.Temporal.System.WorkflowProgress.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.WorkflowProgress.initial]
    starvedSystemActions = [.starved] := by decide

example : Umpire3.Temporal.System.WorkflowProgress.executable.follow ()
    [Umpire3.Temporal.System.WorkflowProgress.initial]
    wrongEntitySystemActions = [] := by decide

example : Umpire3.Temporal.System.WorkflowProgress.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.WorkflowProgress.initial]
    wrongEntitySystemActions = [.wrongEntityCompleted] := by decide

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.WorkflowProgress.System.mutatedBehavior
      Umpire3.Temporal.Refinement.WorkflowProgress.Feature.behavior
      Umpire3.Temporal.Refinement.WorkflowProgress.Projects
      Umpire3.Temporal.Refinement.WorkflowProgress.actionMap :=
  Umpire3.Temporal.Refinement.WorkflowProgress.starvationMutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.workflowTaskStarvation.Holds
    (Umpire3.Temporal.Monitors.workflowTaskStarvationObservations
      Umpire3.Temporal.Feature.WorkflowProgress.progressedFinal) :=
  (Umpire3.Temporal.Monitors.workflowTaskStarvation_monitor_equivalent
    Umpire3.Temporal.Feature.WorkflowProgress.progressedFinal).2 (by decide)

example : Umpire3.Temporal.Monitors.entityProgress.Holds
    (Umpire3.Temporal.Monitors.entityProgressObservations
      Umpire3.Temporal.Feature.WorkflowProgress.progressedFinal) :=
  (Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent
    Umpire3.Temporal.Feature.WorkflowProgress.progressedFinal).2 (by decide)

end Umpire3.Tests.TemporalWorkflowProgress
