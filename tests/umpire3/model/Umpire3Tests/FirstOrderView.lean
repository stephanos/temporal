import Temporal.Families.NexusCancellation.Targets.FirstOrder

namespace Umpire3.Tests.FirstOrderView

open Umpire3.Temporal.System.NexusCancellationFencing
open Umpire3.Temporal.Targets.NexusCancellationFencing

example : soundFirstOrderArtifact.actionIdentifiers = [
    "dispatch-task",
    "request-cancellation",
    "acquire-ownership",
    "commit-cancellation",
    "worker-returns-success",
    "persist-success",
  ] := by
  decide

example : soundFirstOrderArtifact.initial.eval (encodeState initial) := by
  decide

example : soundFirstOrderArtifact.next (encodeState staleReturned) "persist-success" = none := by
  decide

example : mutatedFirstOrderArtifact.next (encodeState staleReturned) "persist-success" =
    some (encodeState staleSuccess) := by
  decide

example : soundFirstOrderArtifact.invariant.eval (encodeState staleSuccess) = false := by
  decide

example : soundFirstOrderExport.isSome := by decide

example : mutatedFirstOrderExport.isSome := by decide

def limitedSearch := Exact.explore soundFiniteView (fun _ => true) {
  maxDepth := 16
  maxStates := 1
  maxTransitions := 4096
  maxStateBytes := 16384
}

example : (FirstOrderExport.ofSearch (resolved_first_order% soundFirstOrderView)
    soundFiniteView (fun _ => true) limitedSearch encodeState).isNone := by decide

def counterexampleSearch := Exact.explore mutatedFiniteView noStaleCompletion {
  maxDepth := 16
  maxStates := 256
  maxTransitions := 4096
  maxStateBytes := 16384
}

example : (FirstOrderExport.ofSearch (resolved_first_order% mutatedFirstOrderView)
    mutatedFiniteView noStaleCompletion counterexampleSearch encodeState).isNone := by decide

example : soundFirstOrderArtifact.initial.eval (encodeState initial) = true :=
  soundFirstOrderView.initial_preserved initial rfl

example : mutatedFirstOrderArtifact.invariant.eval (encodeState staleSuccess) =
    noStaleCompletion staleSuccess :=
  mutatedFirstOrderView.property_preserved staleSuccess (by decide)

end Umpire3.Tests.FirstOrderView
