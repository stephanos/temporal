import Temporal.Monitors
import Temporal.Families.NexusProgress.Refinement

namespace Umpire3.Tests.TemporalNexusProgress

open Umpire3.Temporal.Feature.NexusProgress

def conformingActions : List Action := [
  .schedule,
  .observeRetryableFailure,
  .wait,
  .settle,
]

def stuckActions : List Action := [
  .schedule,
  .observeRetryableFailure,
  .wait,
  .wait,
]

example : executable.follow [initial] conformingActions = [settledFinal] := by decide

example : NexusOperationProgress settledFinal := by decide

example : executable.follow [initial] stuckActions = [] := by decide

example : weakenedExecutable.follow [initial] stuckActions = [stuckFinal] := by decide

example : ¬NexusOperationProgress stuckFinal := by decide

example :
    Umpire3.Temporal.System.NexusProgress.executable.follow ()
      [Umpire3.Temporal.System.NexusProgress.initial]
      Umpire3.Temporal.System.NexusProgress.conformingActions = [.settled] := by decide

example :
    Umpire3.Temporal.System.NexusProgress.executable.follow ()
      [Umpire3.Temporal.System.NexusProgress.initial]
      Umpire3.Temporal.System.NexusProgress.stuckActions = [] := by decide

example :
    Umpire3.Temporal.System.NexusProgress.mutatedExecutable.follow ()
      [Umpire3.Temporal.System.NexusProgress.initial]
      Umpire3.Temporal.System.NexusProgress.stuckActions = [.stuckAfterDeadline] := by decide

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.NexusProgress.System.mutatedBehavior
      Umpire3.Temporal.Refinement.NexusProgress.Feature.behavior
      Umpire3.Temporal.Refinement.NexusProgress.Projects
      Umpire3.Temporal.Refinement.NexusProgress.actionMap :=
  Umpire3.Temporal.Refinement.NexusProgress.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.nexusOperationProgress.Holds
    (Umpire3.Temporal.Monitors.nexusProgressObservations settledFinal) :=
  (Umpire3.Temporal.Monitors.nexusOperationProgress_monitor_equivalent settledFinal).2 (by decide)

example : Umpire3.Temporal.Monitors.nexusOperationProgress.expression.eval
    (Umpire3.Temporal.Monitors.nexusProgressObservations stuckFinal) = some false := by
  change (MonitorExpression.observation "nexus-operation-progressed" true).eval
    [{ identifier := "nexus-operation-progressed", value := false }] = some false
  simp [MonitorExpression.eval, lookupObservation]

end Umpire3.Tests.TemporalNexusProgress
