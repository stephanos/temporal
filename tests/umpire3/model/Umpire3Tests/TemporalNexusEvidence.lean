import Temporal.Monitors
import Temporal.Refinement.MigratedFamilies

namespace Umpire3.Tests.TemporalNexusEvidence

def validTimeoutActions : List Umpire3.Temporal.Product.NexusTimeout.Action := [
  .configure .primary,
  .recordTimeout .primary .primary .startToClose .operationTimedOut,
]

def invalidTimeoutActions : List Umpire3.Temporal.Product.NexusTimeout.Action := [
  .configure .primary,
  .recordTimeout .primary .primary .scheduleToClose .operationTimedOut,
]

example : Umpire3.Temporal.Product.NexusTimeout.executable.follow
    [Umpire3.Temporal.Product.NexusTimeout.initial] validTimeoutActions =
    [Umpire3.Temporal.Product.NexusTimeout.permittedFinal] := by decide

example : Umpire3.Temporal.Product.NexusTimeout.TimeoutSemantics
    Umpire3.Temporal.Product.NexusTimeout.permittedFinal := by decide

example : Umpire3.Temporal.Product.NexusTimeout.executable.follow
    [Umpire3.Temporal.Product.NexusTimeout.initial] invalidTimeoutActions = [] := by decide

example : Umpire3.Temporal.Product.NexusTimeout.unsafeExecutable.follow
    [Umpire3.Temporal.Product.NexusTimeout.initial] invalidTimeoutActions =
    [Umpire3.Temporal.Product.NexusTimeout.unsafeInvalidFinal] := by decide

example : ¬Umpire3.Temporal.Product.NexusTimeout.TimeoutSemantics
    Umpire3.Temporal.Product.NexusTimeout.unsafeInvalidFinal := by decide

example :
    (Umpire3.Temporal.Product.NexusTimeout.bounded.explore
      { maxDepth := 2, maxResults := 1000 }).all
      (fun execution => Umpire3.Temporal.Product.NexusTimeout.timeoutSemanticsB execution.2) = true := by decide

def timeoutSystemActions :
    List Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.Action := [
  .configure,
  .recordTimeout,
]

def malformedTimeoutSystemActions :
    List Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.Action := [
  .configure,
  .recordMalformedTimeout,
]

example : Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.executable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.initial] timeoutSystemActions =
    [.timedOut] := by decide

example : Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.executable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.initial]
    malformedTimeoutSystemActions = [] := by decide

example : Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.initial]
    malformedTimeoutSystemActions = [.malformedTimeout] := by decide

example : Umpire3.Temporal.Refinement.MigratedFamilies.NexusTimeout.soundSimulation.Relates
    (world := ())
    Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.initial
    Umpire3.Temporal.Product.NexusTimeout.initial := rfl

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusTimeout.System.mutatedBehavior
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusTimeout.Feature.behavior
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusTimeout.Projects
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusTimeout.actionMap :=
  Umpire3.Temporal.Refinement.MigratedFamilies.NexusTimeout.mutationBreaksDeclaredSimulation

def matchingLinkActions : List Umpire3.Temporal.Product.NexusActivityLink.Action := [
  .observeOperation .primary (some .primary),
  .observeActivity .primary (some .primary),
]

def oneSidedLinkActions : List Umpire3.Temporal.Product.NexusActivityLink.Action := [
  .observeOperation .primary (some .primary),
  .observeActivity .primary none,
]

example : Umpire3.Temporal.Product.NexusActivityLink.executable.follow
    [Umpire3.Temporal.Product.NexusActivityLink.initial] matchingLinkActions =
    [Umpire3.Temporal.Product.NexusActivityLink.matchingFinal] := by decide

example : Umpire3.Temporal.Product.NexusActivityLink.LinkConsistency
    Umpire3.Temporal.Product.NexusActivityLink.matchingFinal := by decide

example : Umpire3.Temporal.Product.NexusActivityLink.executable.follow
    [Umpire3.Temporal.Product.NexusActivityLink.initial] oneSidedLinkActions = [] := by decide

example : Umpire3.Temporal.Product.NexusActivityLink.unsafeExecutable.follow
    [Umpire3.Temporal.Product.NexusActivityLink.initial] oneSidedLinkActions =
    [Umpire3.Temporal.Product.NexusActivityLink.oneSidedFinal] := by decide

example : ¬Umpire3.Temporal.Product.NexusActivityLink.LinkConsistency
    Umpire3.Temporal.Product.NexusActivityLink.oneSidedFinal := by decide

example :
    (Umpire3.Temporal.Product.NexusActivityLink.bounded.explore
      { maxDepth := 2, maxResults := 1000 }).all
      (fun execution => Umpire3.Temporal.Product.NexusActivityLink.linkConsistencyB execution.2) = true := by decide

def linkSystemActions :
    List Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.Action := [
  .observeOperation,
  .observeLinkedActivity,
]

def oneSidedLinkSystemActions :
    List Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.Action := [
  .observeOperation,
  .observeOneSidedActivity,
]

example : Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.executable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.initial] linkSystemActions =
    [.linked] := by decide

example : Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.executable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.initial]
    oneSidedLinkSystemActions = [] := by decide

example : Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.initial]
    oneSidedLinkSystemActions = [.oneSided] := by decide

example : Umpire3.Temporal.Refinement.MigratedFamilies.NexusActivityLink.soundSimulation.Relates
    (world := ())
    Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.initial
    Umpire3.Temporal.Product.NexusActivityLink.initial := rfl

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusActivityLink.System.mutatedBehavior
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusActivityLink.Feature.behavior
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusActivityLink.Projects
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusActivityLink.actionMap :=
  Umpire3.Temporal.Refinement.MigratedFamilies.NexusActivityLink.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.nexusOperationTimeoutSemantics.Holds
    (Umpire3.Temporal.Monitors.nexusTimeoutObservations
      Umpire3.Temporal.Product.NexusTimeout.permittedFinal) :=
  (Umpire3.Temporal.Monitors.nexusTimeout_monitor_equivalent
    Umpire3.Temporal.Product.NexusTimeout.permittedFinal).2 (by decide)

example : Umpire3.Temporal.Monitors.nexusActivityLinkConsistency.Holds
    (Umpire3.Temporal.Monitors.nexusActivityLinkObservations
      Umpire3.Temporal.Product.NexusActivityLink.matchingFinal) :=
  (Umpire3.Temporal.Monitors.nexusActivityLink_monitor_equivalent
    Umpire3.Temporal.Product.NexusActivityLink.matchingFinal).2 (by decide)

end Umpire3.Tests.TemporalNexusEvidence
