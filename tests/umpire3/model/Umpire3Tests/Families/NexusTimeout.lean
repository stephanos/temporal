import Temporal.Monitors
import Temporal.Families.NexusTimeout.Refinement

namespace Umpire3.Tests.Families.NexusTimeout

def validTimeoutActions : List Umpire3.Temporal.Feature.NexusTimeout.Action := [
  .configure .primary,
  .recordTimeout .primary .primary .startToClose .operationTimedOut,
]

def invalidTimeoutActions : List Umpire3.Temporal.Feature.NexusTimeout.Action := [
  .configure .primary,
  .recordTimeout .primary .primary .scheduleToClose .operationTimedOut,
]

example : Umpire3.Temporal.Feature.NexusTimeout.executable.follow
    [Umpire3.Temporal.Feature.NexusTimeout.initial] validTimeoutActions =
    [Umpire3.Temporal.Feature.NexusTimeout.permittedFinal] := by decide

example : Umpire3.Temporal.Feature.NexusTimeout.TimeoutSemantics
    Umpire3.Temporal.Feature.NexusTimeout.permittedFinal := by decide

example : Umpire3.Temporal.Feature.NexusTimeout.executable.follow
    [Umpire3.Temporal.Feature.NexusTimeout.initial] invalidTimeoutActions = [] := by decide

example : Umpire3.Temporal.Feature.NexusTimeout.unsafeExecutable.follow
    [Umpire3.Temporal.Feature.NexusTimeout.initial] invalidTimeoutActions =
    [Umpire3.Temporal.Feature.NexusTimeout.unsafeInvalidFinal] := by decide

example : ¬Umpire3.Temporal.Feature.NexusTimeout.TimeoutSemantics
    Umpire3.Temporal.Feature.NexusTimeout.unsafeInvalidFinal := by decide

example :
    (Umpire3.Temporal.Feature.NexusTimeout.bounded.explore
      { maxDepth := 2, maxResults := 1000 }).all
      (fun execution => Umpire3.Temporal.Feature.NexusTimeout.timeoutSemanticsB execution.2) = true := by decide

def timeoutSystemActions :
    List Umpire3.Temporal.System.NexusTimeout.Action := [
  .configure,
  .recordTimeout,
]

def malformedTimeoutSystemActions :
    List Umpire3.Temporal.System.NexusTimeout.Action := [
  .configure,
  .recordMalformedTimeout,
]

example : Umpire3.Temporal.System.NexusTimeout.executable.follow ()
    [Umpire3.Temporal.System.NexusTimeout.initial] timeoutSystemActions =
    [.timedOut] := by decide

example : Umpire3.Temporal.System.NexusTimeout.executable.follow ()
    [Umpire3.Temporal.System.NexusTimeout.initial]
    malformedTimeoutSystemActions = [] := by decide

example : Umpire3.Temporal.System.NexusTimeout.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.NexusTimeout.initial]
    malformedTimeoutSystemActions = [.malformedTimeout] := by decide

example : Umpire3.Temporal.Refinement.NexusTimeout.soundSimulation.Relates
    (world := ())
    Umpire3.Temporal.System.NexusTimeout.initial
    Umpire3.Temporal.Feature.NexusTimeout.initial := rfl

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.NexusTimeout.System.mutatedBehavior
      Umpire3.Temporal.Refinement.NexusTimeout.Feature.behavior
      Umpire3.Temporal.Refinement.NexusTimeout.Projects
      Umpire3.Temporal.Refinement.NexusTimeout.actionMap :=
  Umpire3.Temporal.Refinement.NexusTimeout.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.nexusOperationTimeoutSemantics.Holds
    (Umpire3.Temporal.Monitors.nexusTimeoutObservations
      Umpire3.Temporal.Feature.NexusTimeout.permittedFinal) :=
  (Umpire3.Temporal.Monitors.nexusTimeout_monitor_equivalent
    Umpire3.Temporal.Feature.NexusTimeout.permittedFinal).2 (by decide)

end Umpire3.Tests.Families.NexusTimeout
