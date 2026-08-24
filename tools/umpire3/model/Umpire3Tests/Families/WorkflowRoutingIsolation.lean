import Temporal.Monitors
import Temporal.Families

namespace Umpire3.Tests.Families.WorkflowRoutingIsolation

set_option maxRecDepth 100000

def continuationActions : List Umpire3.Temporal.Feature.WorkflowLineage.Action := [
  .observe .secondary .continuation .primary .secondary .primary,
]

def resetActions : List Umpire3.Temporal.Feature.WorkflowLineage.Action := [
  .observe .secondary .reset .primary .primary .primary,
]

def invalidContinuationActions : List Umpire3.Temporal.Feature.WorkflowLineage.Action := [
  .observe .secondary .continuation .primary .primary .primary,
]

example : Umpire3.Temporal.Feature.WorkflowLineage.executable.follow
    [Umpire3.Temporal.Feature.WorkflowLineage.initial] continuationActions =
    [Umpire3.Temporal.Feature.WorkflowLineage.continuationFinal] := by decide

example : Umpire3.Temporal.Feature.WorkflowLineage.ContinuationQualified
    Umpire3.Temporal.Feature.WorkflowLineage.continuationFinal := by decide

example : Umpire3.Temporal.Feature.WorkflowLineage.executable.follow
    [Umpire3.Temporal.Feature.WorkflowLineage.initial] resetActions =
    [Umpire3.Temporal.Feature.WorkflowLineage.resetFinal] := by decide

example : Umpire3.Temporal.Feature.WorkflowLineage.ResetQualified
    Umpire3.Temporal.Feature.WorkflowLineage.resetFinal := by decide

example : Umpire3.Temporal.Feature.WorkflowLineage.executable.follow
    [Umpire3.Temporal.Feature.WorkflowLineage.initial] invalidContinuationActions = [] := by decide

example : Umpire3.Temporal.Feature.WorkflowLineage.weakenedExecutable.follow
    [Umpire3.Temporal.Feature.WorkflowLineage.initial] invalidContinuationActions =
    [Umpire3.Temporal.Feature.WorkflowLineage.invalidContinuationFinal] := by decide

example : ¬Umpire3.Temporal.Feature.WorkflowLineage.ContinuationLineage
    Umpire3.Temporal.Feature.WorkflowLineage.invalidContinuationFinal := by decide

example :
    (Umpire3.Temporal.Feature.WorkflowLineage.bounded.explore
      { maxDepth := 1, maxResults := 1000 }).all
      (fun execution => Umpire3.Temporal.Feature.WorkflowLineage.lineageConsistencyB execution.2) = true := by decide

def systemActions :
    List Umpire3.Temporal.System.WorkflowLineage.Action := [
  .observeContinuation,
]

def invalidLineageSystemActions :
    List Umpire3.Temporal.System.WorkflowLineage.Action := [
  .observeInvalidContinuation,
]

example : Umpire3.Temporal.System.WorkflowLineage.executable.follow ()
    [Umpire3.Temporal.System.WorkflowLineage.initial] systemActions =
    [.continuationObserved] := by decide

example : Umpire3.Temporal.System.WorkflowLineage.executable.follow ()
    [Umpire3.Temporal.System.WorkflowLineage.initial]
    invalidLineageSystemActions = [] := by decide

example : Umpire3.Temporal.System.WorkflowLineage.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.WorkflowLineage.initial]
    invalidLineageSystemActions = [.invalidContinuation] := by decide

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.WorkflowLineage.System.mutatedBehavior
      Umpire3.Temporal.Refinement.WorkflowLineage.Feature.behavior
      Umpire3.Temporal.Refinement.WorkflowLineage.Projects
      Umpire3.Temporal.Refinement.WorkflowLineage.actionMap :=
  Umpire3.Temporal.Refinement.WorkflowLineage.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.continuationLineage.Holds
    (Umpire3.Temporal.Monitors.continuationLineageObservations
      Umpire3.Temporal.Feature.WorkflowLineage.continuationFinal) :=
  (Umpire3.Temporal.Monitors.continuationLineage_monitor_equivalent
    Umpire3.Temporal.Feature.WorkflowLineage.continuationFinal).2 (by decide)

example : Umpire3.Temporal.Monitors.resetLineage.Holds
    (Umpire3.Temporal.Monitors.resetLineageObservations
      Umpire3.Temporal.Feature.WorkflowLineage.resetFinal) :=
  (Umpire3.Temporal.Monitors.resetLineage_monitor_equivalent
    Umpire3.Temporal.Feature.WorkflowLineage.resetFinal).2 (by decide)

def matchingRoutingActions : List Umpire3.Temporal.Feature.WorkflowRouting.Action := [
  .assignTask .primary .primary,
  .registerPoller .primary .primary,
  .reserve .primary .primary .primary,
]

def crossingRoutingActions : List Umpire3.Temporal.Feature.WorkflowRouting.Action := [
  .assignTask .primary .primary,
  .registerPoller .primary .secondary,
  .reserve .primary .primary .primary,
]

example : Umpire3.Temporal.Feature.WorkflowRouting.executable.follow
    [Umpire3.Temporal.Feature.WorkflowRouting.initial] matchingRoutingActions =
    [Umpire3.Temporal.Feature.WorkflowRouting.matchingFinal] := by decide

example : Umpire3.Temporal.Feature.WorkflowRouting.RoutingQualified
    Umpire3.Temporal.Feature.WorkflowRouting.matchingFinal := by decide

example : Umpire3.Temporal.Feature.WorkflowRouting.executable.follow
    [Umpire3.Temporal.Feature.WorkflowRouting.initial] crossingRoutingActions = [] := by decide

example : Umpire3.Temporal.Feature.WorkflowRouting.weakenedExecutable.follow
    [Umpire3.Temporal.Feature.WorkflowRouting.initial] crossingRoutingActions =
    [Umpire3.Temporal.Feature.WorkflowRouting.crossingFinal] := by decide

example : ¬Umpire3.Temporal.Feature.WorkflowRouting.RoutingIsolation
    Umpire3.Temporal.Feature.WorkflowRouting.crossingFinal := by decide

example :
    (Umpire3.Temporal.Feature.WorkflowRouting.bounded.explore
      { maxDepth := 3, maxResults := 5000 }).all
      (fun execution => Umpire3.Temporal.Feature.WorkflowRouting.routingIsolationB execution.2) = true := by decide

def routingSystemActions :
    List Umpire3.Temporal.System.WorkflowRouting.Action := [
  .assignTask,
  .registerMatchingPoller,
  .reserveMatching,
]

def crossingRoutingSystemActions :
    List Umpire3.Temporal.System.WorkflowRouting.Action := [
  .assignTask,
  .registerCrossingPoller,
  .reserveCrossing,
]

example : Umpire3.Temporal.System.WorkflowRouting.executable.follow ()
    [Umpire3.Temporal.System.WorkflowRouting.initial] routingSystemActions =
    [.matchingReservation] := by decide

example : Umpire3.Temporal.System.WorkflowRouting.executable.follow ()
    [Umpire3.Temporal.System.WorkflowRouting.initial]
    crossingRoutingSystemActions = [] := by decide

example : Umpire3.Temporal.System.WorkflowRouting.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.WorkflowRouting.initial]
    crossingRoutingSystemActions = [.crossingReservation] := by decide

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.WorkflowRouting.System.mutatedBehavior
      Umpire3.Temporal.Refinement.WorkflowRouting.Feature.behavior
      Umpire3.Temporal.Refinement.WorkflowRouting.Projects
      Umpire3.Temporal.Refinement.WorkflowRouting.actionMap :=
  Umpire3.Temporal.Refinement.WorkflowRouting.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.workflowRoutingIsolation.Holds
    (Umpire3.Temporal.Monitors.workflowRoutingObservations
      Umpire3.Temporal.Feature.WorkflowRouting.matchingFinal) :=
  (Umpire3.Temporal.Monitors.workflowRouting_monitor_equivalent
    Umpire3.Temporal.Feature.WorkflowRouting.matchingFinal).2 (by decide)

end Umpire3.Tests.Families.WorkflowRoutingIsolation
