import Temporal.Monitors
import Temporal.Refinement.WorkflowProgress

namespace Umpire3.Tests.TemporalWorkflowProgress

set_option maxRecDepth 100000

def progressedActions : List Umpire3.Temporal.Product.WorkflowProgress.Action := [
  .enqueue .primary .primary,
  .makeWorkerAvailable .primary,
  .wait .primary,
  .dispatch .primary .primary,
  .complete .primary .primary,
]

def starvedActions : List Umpire3.Temporal.Product.WorkflowProgress.Action := [
  .enqueue .primary .primary,
  .makeWorkerAvailable .primary,
  .wait .primary,
  .wait .primary,
]

def wrongEntityActions : List Umpire3.Temporal.Product.WorkflowProgress.Action := [
  .enqueue .primary .primary,
  .makeWorkerAvailable .primary,
  .dispatch .primary .primary,
  .complete .primary .secondary,
]

example : Umpire3.Temporal.Product.WorkflowProgress.executable.follow
    [Umpire3.Temporal.Product.WorkflowProgress.initial] progressedActions =
    [Umpire3.Temporal.Product.WorkflowProgress.progressedFinal] := by decide

example : Umpire3.Temporal.Product.WorkflowProgress.StarvationQualified
    Umpire3.Temporal.Product.WorkflowProgress.progressedFinal := by decide

example : Umpire3.Temporal.Product.WorkflowProgress.ProgressQualified
    Umpire3.Temporal.Product.WorkflowProgress.progressedFinal := by decide

example : Umpire3.Temporal.Product.WorkflowProgress.executable.follow
    [Umpire3.Temporal.Product.WorkflowProgress.initial] starvedActions = [] := by decide

example : Umpire3.Temporal.Product.WorkflowProgress.weakenedExecutable.follow
    [Umpire3.Temporal.Product.WorkflowProgress.initial] starvedActions =
    [Umpire3.Temporal.Product.WorkflowProgress.starvedFinal] := by decide

example : ¬Umpire3.Temporal.Product.WorkflowProgress.WorkflowTaskStarvation
    Umpire3.Temporal.Product.WorkflowProgress.starvedFinal := by decide

example : Umpire3.Temporal.Product.WorkflowProgress.executable.follow
    [Umpire3.Temporal.Product.WorkflowProgress.initial] wrongEntityActions = [] := by decide

example : Umpire3.Temporal.Product.WorkflowProgress.weakenedExecutable.follow
    [Umpire3.Temporal.Product.WorkflowProgress.initial] wrongEntityActions =
    [Umpire3.Temporal.Product.WorkflowProgress.wrongEntityFinal] := by decide

example : ¬Umpire3.Temporal.Product.WorkflowProgress.EntityProgress
    Umpire3.Temporal.Product.WorkflowProgress.wrongEntityFinal := by decide

example :
    (Umpire3.Temporal.Product.WorkflowProgress.bounded.explore
      { maxDepth := 4, maxResults := 10000 }).all
      (fun execution => Umpire3.Temporal.Product.WorkflowProgress.invariantB execution.2) = true := by decide

def systemActions : List Umpire3.Temporal.System.WorkflowProgress.Action := [
  .observeEnqueue .primary .primary,
  .observeWorkerAvailability .primary,
  .observeWait .primary,
  .observeDispatch .primary .primary,
  .observeCompletion .primary .primary,
  .redeliverCompletion .primary,
]

example : Umpire3.Temporal.System.WorkflowProgress.executable.follow
    [Umpire3.Temporal.System.WorkflowProgress.initial] systemActions =
    [Umpire3.Temporal.System.WorkflowProgress.progressedFinal] := by decide

example : Umpire3.Temporal.Monitors.workflowTaskStarvation.Holds
    (Umpire3.Temporal.Monitors.workflowTaskStarvationObservations
      Umpire3.Temporal.Product.WorkflowProgress.progressedFinal) :=
  (Umpire3.Temporal.Monitors.workflowTaskStarvation_monitor_equivalent
    Umpire3.Temporal.Product.WorkflowProgress.progressedFinal).2 (by decide)

example : Umpire3.Temporal.Monitors.entityProgress.Holds
    (Umpire3.Temporal.Monitors.entityProgressObservations
      Umpire3.Temporal.Product.WorkflowProgress.progressedFinal) :=
  (Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent
    Umpire3.Temporal.Product.WorkflowProgress.progressedFinal).2 (by decide)

end Umpire3.Tests.TemporalWorkflowProgress
