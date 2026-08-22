import Temporal.Families.NexusCancellation.Targets.Finite
import Umpire3.TraceReplay

namespace Umpire3.Tests.TraceReplay

open Umpire3.Temporal.Targets.NexusCancellationFencing

def staleCompletionTrace : List String := [
  "dispatch-task",
  "acquire-ownership",
  "worker-returns-success",
  "persist-success",
]

example : TraceReplay.check mutatedFiniteView noStaleCompletion staleCompletionTrace = true := by
  decide

example : TraceReplay.check soundFiniteView noStaleCompletion staleCompletionTrace = false := by
  decide

example : TraceReplay.check mutatedFiniteView noStaleCompletion
    ["dispatch-task", "unknown-action"] = false := by
  decide

example : ∃ state,
    Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior.Reachable .smoke state ∧
      noStaleCompletion state = false :=
  TraceReplay.checked mutatedFiniteView noStaleCompletion staleCompletionTrace (by decide)

end Umpire3.Tests.TraceReplay
