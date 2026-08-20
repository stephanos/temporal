import Temporal.Targets.NexusCancellationFencingFirstOrder

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

example : soundFirstOrderOracle.length = soundSearch.statistics.visited := by
  decide

example : mutatedFirstOrderOracle.length = mutatedReachabilitySearch.statistics.visited := by
  decide

example : soundFirstOrderArtifact.initial.eval (encodeState initial) = true :=
  soundFirstOrderView.initial_preserved initial rfl

example : mutatedFirstOrderArtifact.invariant.eval (encodeState staleSuccess) =
    noStaleCompletion staleSuccess :=
  mutatedFirstOrderView.property_preserved staleSuccess (by decide)

end Umpire3.Tests.FirstOrderView
