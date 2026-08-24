import Temporal.Monitors
import Temporal.Families.NexusActivityLink.Refinement

namespace Umpire3.Tests.Families.NexusActivityLink

def matchingLinkActions : List Umpire3.Temporal.Feature.NexusActivityLink.Action := [
  .observeOperation .primary (some .primary),
  .observeActivity .primary (some .primary),
]

def oneSidedLinkActions : List Umpire3.Temporal.Feature.NexusActivityLink.Action := [
  .observeOperation .primary (some .primary),
  .observeActivity .primary none,
]

example : Umpire3.Temporal.Feature.NexusActivityLink.executable.follow
    [Umpire3.Temporal.Feature.NexusActivityLink.initial] matchingLinkActions =
    [Umpire3.Temporal.Feature.NexusActivityLink.matchingFinal] := by decide

example : Umpire3.Temporal.Feature.NexusActivityLink.LinkConsistency
    Umpire3.Temporal.Feature.NexusActivityLink.matchingFinal := by decide

example : Umpire3.Temporal.Feature.NexusActivityLink.executable.follow
    [Umpire3.Temporal.Feature.NexusActivityLink.initial] oneSidedLinkActions = [] := by decide

example : Umpire3.Temporal.Feature.NexusActivityLink.unsafeExecutable.follow
    [Umpire3.Temporal.Feature.NexusActivityLink.initial] oneSidedLinkActions =
    [Umpire3.Temporal.Feature.NexusActivityLink.oneSidedFinal] := by decide

example : ¬Umpire3.Temporal.Feature.NexusActivityLink.LinkConsistency
    Umpire3.Temporal.Feature.NexusActivityLink.oneSidedFinal := by decide

example :
    (Umpire3.Temporal.Feature.NexusActivityLink.bounded.explore
      { maxDepth := 2, maxResults := 1000 }).all
      (fun execution => Umpire3.Temporal.Feature.NexusActivityLink.linkConsistencyB execution.2) = true := by decide

def linkSystemActions :
    List Umpire3.Temporal.System.NexusActivityLink.Action := [
  .observeOperation,
  .observeLinkedActivity,
]

def oneSidedLinkSystemActions :
    List Umpire3.Temporal.System.NexusActivityLink.Action := [
  .observeOperation,
  .observeOneSidedActivity,
]

example : Umpire3.Temporal.System.NexusActivityLink.executable.follow ()
    [Umpire3.Temporal.System.NexusActivityLink.initial] linkSystemActions =
    [.linked] := by decide

example : Umpire3.Temporal.System.NexusActivityLink.executable.follow ()
    [Umpire3.Temporal.System.NexusActivityLink.initial]
    oneSidedLinkSystemActions = [] := by decide

example : Umpire3.Temporal.System.NexusActivityLink.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.NexusActivityLink.initial]
    oneSidedLinkSystemActions = [.oneSided] := by decide

example : Umpire3.Temporal.Refinement.NexusActivityLink.soundSimulation.Relates
    (world := ())
    Umpire3.Temporal.System.NexusActivityLink.initial
    Umpire3.Temporal.Feature.NexusActivityLink.initial := rfl

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.NexusActivityLink.System.mutatedBehavior
      Umpire3.Temporal.Refinement.NexusActivityLink.Feature.behavior
      Umpire3.Temporal.Refinement.NexusActivityLink.Projects
      Umpire3.Temporal.Refinement.NexusActivityLink.actionMap :=
  Umpire3.Temporal.Refinement.NexusActivityLink.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.nexusActivityLinkConsistency.Holds
    (Umpire3.Temporal.Monitors.nexusActivityLinkObservations
      Umpire3.Temporal.Feature.NexusActivityLink.matchingFinal) :=
  (Umpire3.Temporal.Monitors.nexusActivityLink_monitor_equivalent
    Umpire3.Temporal.Feature.NexusActivityLink.matchingFinal).2 (by decide)

end Umpire3.Tests.Families.NexusActivityLink
