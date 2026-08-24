import Temporal.Monitors
import Temporal.Families.CallbackResponse.Refinement

namespace Umpire3.Tests.Families.CallbackResponse

set_option maxRecDepth 100000

def consistentResponseActions : List Umpire3.Temporal.Feature.CallbackResponse.Action := [
  .register .primary .primary .primary,
  .settleOperation .primary .second true,
  .recordResponse .primary .asyncSuccess .accepted .third,
  .recordResponse .primary .asyncSuccess .accepted .third,
]

def conflictingResponseActions : List Umpire3.Temporal.Feature.CallbackResponse.Action := [
  .register .primary .primary .primary,
  .recordResponse .primary .asyncSuccess .accepted .second,
  .recordResponse .primary .failure .conflicting .third,
]

example : Umpire3.Temporal.Feature.CallbackResponse.executable.follow
    [Umpire3.Temporal.Feature.CallbackResponse.initial] consistentResponseActions =
    [Umpire3.Temporal.Feature.CallbackResponse.consistentFinal] := by decide

example : Umpire3.Temporal.Feature.CallbackResponse.ResponseQualified
    Umpire3.Temporal.Feature.CallbackResponse.consistentFinal := by decide

example : Umpire3.Temporal.Feature.CallbackResponse.executable.follow
    [Umpire3.Temporal.Feature.CallbackResponse.initial] conflictingResponseActions = [] := by decide

example : Umpire3.Temporal.Feature.CallbackResponse.weakenedExecutable.follow
    [Umpire3.Temporal.Feature.CallbackResponse.initial] conflictingResponseActions =
    [Umpire3.Temporal.Feature.CallbackResponse.conflictingFinal] := by decide

example : ¬Umpire3.Temporal.Feature.CallbackResponse.ResponseConsistency
    Umpire3.Temporal.Feature.CallbackResponse.conflictingFinal := by decide

example :
    (Umpire3.Temporal.Feature.CallbackResponse.bounded.explore
      { maxDepth := 2, maxResults := 2500 }).all
      (fun execution => Umpire3.Temporal.Feature.CallbackResponse.responseConsistencyB execution.2) = true := by decide

def responseSystemActions :
    List Umpire3.Temporal.System.CallbackResponse.Action := [
  .register,
  .settle,
  .respond,
]

def conflictingResponseSystemActions :
    List Umpire3.Temporal.System.CallbackResponse.Action := [
  .register,
  .settle,
  .respond,
  .conflict,
]

example : Umpire3.Temporal.System.CallbackResponse.executable.follow ()
    [Umpire3.Temporal.System.CallbackResponse.initial] responseSystemActions =
    [.responded] := by decide

example : Umpire3.Temporal.System.CallbackResponse.executable.follow ()
    [Umpire3.Temporal.System.CallbackResponse.initial]
    conflictingResponseSystemActions = [] := by decide

example : Umpire3.Temporal.System.CallbackResponse.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.CallbackResponse.initial]
    conflictingResponseSystemActions = [.conflictingResponse] := by decide

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.CallbackResponse.System.mutatedBehavior
      Umpire3.Temporal.Refinement.CallbackResponse.Feature.behavior
      Umpire3.Temporal.Refinement.CallbackResponse.Projects
      Umpire3.Temporal.Refinement.CallbackResponse.actionMap :=
  Umpire3.Temporal.Refinement.CallbackResponse.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.callbackResponseConsistency.Holds
    (Umpire3.Temporal.Monitors.callbackResponseObservations
      Umpire3.Temporal.Feature.CallbackResponse.consistentFinal) :=
  (Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent
    Umpire3.Temporal.Feature.CallbackResponse.consistentFinal).2 (by decide)

end Umpire3.Tests.Families.CallbackResponse
