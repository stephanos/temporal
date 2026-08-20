import Temporal.Monitors
import Temporal.Refinement.WorkflowLineage
import Temporal.Refinement.WorkflowRouting
import Temporal.Refinement.WorkflowOwnership

namespace Umpire3.Tests.TemporalWorkflowEvidence

set_option maxRecDepth 100000

def continuationActions : List Umpire3.Temporal.Product.WorkflowLineage.Action := [
  .observe .secondary .continuation .primary .secondary .primary,
]

def resetActions : List Umpire3.Temporal.Product.WorkflowLineage.Action := [
  .observe .secondary .reset .primary .primary .primary,
]

def invalidContinuationActions : List Umpire3.Temporal.Product.WorkflowLineage.Action := [
  .observe .secondary .continuation .primary .primary .primary,
]

example : Umpire3.Temporal.Product.WorkflowLineage.executable.follow
    [Umpire3.Temporal.Product.WorkflowLineage.initial] continuationActions =
    [Umpire3.Temporal.Product.WorkflowLineage.continuationFinal] := by decide

example : Umpire3.Temporal.Product.WorkflowLineage.ContinuationQualified
    Umpire3.Temporal.Product.WorkflowLineage.continuationFinal := by decide

example : Umpire3.Temporal.Product.WorkflowLineage.executable.follow
    [Umpire3.Temporal.Product.WorkflowLineage.initial] resetActions =
    [Umpire3.Temporal.Product.WorkflowLineage.resetFinal] := by decide

example : Umpire3.Temporal.Product.WorkflowLineage.ResetQualified
    Umpire3.Temporal.Product.WorkflowLineage.resetFinal := by decide

example : Umpire3.Temporal.Product.WorkflowLineage.executable.follow
    [Umpire3.Temporal.Product.WorkflowLineage.initial] invalidContinuationActions = [] := by decide

example : Umpire3.Temporal.Product.WorkflowLineage.weakenedExecutable.follow
    [Umpire3.Temporal.Product.WorkflowLineage.initial] invalidContinuationActions =
    [Umpire3.Temporal.Product.WorkflowLineage.invalidContinuationFinal] := by decide

example : ¬Umpire3.Temporal.Product.WorkflowLineage.ContinuationLineage
    Umpire3.Temporal.Product.WorkflowLineage.invalidContinuationFinal := by decide

example :
    (Umpire3.Temporal.Product.WorkflowLineage.bounded.explore
      { maxDepth := 1, maxResults := 1000 }).all
      (fun execution => Umpire3.Temporal.Product.WorkflowLineage.lineageConsistencyB execution.2) = true := by decide

def systemActions : List Umpire3.Temporal.System.WorkflowLineage.Action := [
  .observe .secondary .continuation .primary .secondary .primary,
  .redeliver .secondary,
]

example : Umpire3.Temporal.System.WorkflowLineage.executable.follow
    [Umpire3.Temporal.System.WorkflowLineage.initial] systemActions =
    [Umpire3.Temporal.System.WorkflowLineage.continuationFinal] := by decide

example : Umpire3.Temporal.Monitors.continuationLineage.Holds
    (Umpire3.Temporal.Monitors.continuationLineageObservations
      Umpire3.Temporal.Product.WorkflowLineage.continuationFinal) :=
  (Umpire3.Temporal.Monitors.continuationLineage_monitor_equivalent
    Umpire3.Temporal.Product.WorkflowLineage.continuationFinal).2 (by decide)

example : Umpire3.Temporal.Monitors.resetLineage.Holds
    (Umpire3.Temporal.Monitors.resetLineageObservations
      Umpire3.Temporal.Product.WorkflowLineage.resetFinal) :=
  (Umpire3.Temporal.Monitors.resetLineage_monitor_equivalent
    Umpire3.Temporal.Product.WorkflowLineage.resetFinal).2 (by decide)

def matchingRoutingActions : List Umpire3.Temporal.Product.WorkflowRouting.Action := [
  .assignTask .primary .primary,
  .registerPoller .primary .primary,
  .reserve .primary .primary .primary,
]

def crossingRoutingActions : List Umpire3.Temporal.Product.WorkflowRouting.Action := [
  .assignTask .primary .primary,
  .registerPoller .primary .secondary,
  .reserve .primary .primary .primary,
]

example : Umpire3.Temporal.Product.WorkflowRouting.executable.follow
    [Umpire3.Temporal.Product.WorkflowRouting.initial] matchingRoutingActions =
    [Umpire3.Temporal.Product.WorkflowRouting.matchingFinal] := by decide

example : Umpire3.Temporal.Product.WorkflowRouting.RoutingQualified
    Umpire3.Temporal.Product.WorkflowRouting.matchingFinal := by decide

example : Umpire3.Temporal.Product.WorkflowRouting.executable.follow
    [Umpire3.Temporal.Product.WorkflowRouting.initial] crossingRoutingActions = [] := by decide

example : Umpire3.Temporal.Product.WorkflowRouting.weakenedExecutable.follow
    [Umpire3.Temporal.Product.WorkflowRouting.initial] crossingRoutingActions =
    [Umpire3.Temporal.Product.WorkflowRouting.crossingFinal] := by decide

example : ¬Umpire3.Temporal.Product.WorkflowRouting.RoutingIsolation
    Umpire3.Temporal.Product.WorkflowRouting.crossingFinal := by decide

example :
    (Umpire3.Temporal.Product.WorkflowRouting.bounded.explore
      { maxDepth := 3, maxResults := 5000 }).all
      (fun execution => Umpire3.Temporal.Product.WorkflowRouting.routingIsolationB execution.2) = true := by decide

def routingSystemActions : List Umpire3.Temporal.System.WorkflowRouting.Action := [
  .observeTaskRoute .primary .primary,
  .observePollerRoute .primary .primary,
  .observeReservation .primary .primary .primary,
  .redeliverReservation .primary,
]

example : Umpire3.Temporal.System.WorkflowRouting.executable.follow
    [Umpire3.Temporal.System.WorkflowRouting.initial] routingSystemActions =
    [Umpire3.Temporal.System.WorkflowRouting.matchingFinal] := by decide

example : Umpire3.Temporal.Monitors.workflowRoutingIsolation.Holds
    (Umpire3.Temporal.Monitors.workflowRoutingObservations
      Umpire3.Temporal.Product.WorkflowRouting.matchingFinal) :=
  (Umpire3.Temporal.Monitors.workflowRouting_monitor_equivalent
    Umpire3.Temporal.Product.WorkflowRouting.matchingFinal).2 (by decide)

def ownershipActions : List Umpire3.Temporal.Product.WorkflowOwnership.Action := [
  .bootstrap .primary .first,
  .dispatch .primary .primary .first,
  .fail .primary,
  .rotate .primary .first .second,
  .dispatch .secondary .primary .second,
  .rejectStale .primary,
  .complete .secondary,
]

def staleCompletionActions : List Umpire3.Temporal.Product.WorkflowOwnership.Action := [
  .bootstrap .primary .first,
  .dispatch .primary .primary .first,
  .fail .primary,
  .rotate .primary .first .second,
  .complete .primary,
]

example : Umpire3.Temporal.Product.WorkflowOwnership.executable.follow
    [Umpire3.Temporal.Product.WorkflowOwnership.initial] ownershipActions =
    [Umpire3.Temporal.Product.WorkflowOwnership.fencedFinal] := by decide

example : Umpire3.Temporal.Product.WorkflowOwnership.OwnershipQualified
    Umpire3.Temporal.Product.WorkflowOwnership.fencedFinal := by decide

example : Umpire3.Temporal.Product.WorkflowOwnership.executable.follow
    [Umpire3.Temporal.Product.WorkflowOwnership.initial] staleCompletionActions = [] := by decide

example : Umpire3.Temporal.Product.WorkflowOwnership.weakenedExecutable.follow
    [Umpire3.Temporal.Product.WorkflowOwnership.initial] staleCompletionActions =
    [Umpire3.Temporal.Product.WorkflowOwnership.staleCompletionFinal] := by decide

example : ¬Umpire3.Temporal.Product.WorkflowOwnership.OwnershipFencing
    Umpire3.Temporal.Product.WorkflowOwnership.staleCompletionFinal := by decide

example :
    (Umpire3.Temporal.Product.WorkflowOwnership.bounded.explore
      { maxDepth := 3, maxResults := 5000 }).all
      (fun execution => Umpire3.Temporal.Product.WorkflowOwnership.ownershipFencingB execution.2) = true := by decide

def ownershipSystemActions : List Umpire3.Temporal.System.WorkflowOwnership.Action := [
  .observeBootstrap .primary .first,
  .observeDispatch .primary .primary .first,
  .observeFailure .primary,
  .observeRotation .primary .first .second,
  .observeDispatch .secondary .primary .second,
  .observeStaleRejection .primary,
  .observeCompletion .secondary,
  .redeliverCompletion .secondary,
]

example : Umpire3.Temporal.System.WorkflowOwnership.executable.follow
    [Umpire3.Temporal.System.WorkflowOwnership.initial] ownershipSystemActions =
    [Umpire3.Temporal.System.WorkflowOwnership.fencedFinal] := by decide

example : Umpire3.Temporal.Monitors.workflowOwnershipFencing.Holds
    (Umpire3.Temporal.Monitors.workflowOwnershipObservations
      Umpire3.Temporal.Product.WorkflowOwnership.fencedFinal) :=
  (Umpire3.Temporal.Monitors.workflowOwnership_monitor_equivalent
    Umpire3.Temporal.Product.WorkflowOwnership.fencedFinal).2 (by decide)

end Umpire3.Tests.TemporalWorkflowEvidence
