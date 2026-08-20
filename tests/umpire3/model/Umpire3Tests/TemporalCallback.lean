import Temporal.Monitors
import Temporal.Refinement.CallbackReference
import Temporal.Refinement.CallbackResponse

namespace Umpire3.Tests.TemporalCallback

set_option maxRecDepth 100000

def matchingReferenceActions : List Umpire3.Temporal.Product.CallbackReference.Action := [
  .observeAttachment .primary .primary .event .workflowStarted .first false,
  .observeOperationStart .primary .primary .primary .event .workflowStarted .second false,
]

def wrongReferenceActions : List Umpire3.Temporal.Product.CallbackReference.Action := [
  .observeAttachment .primary .primary .event .workflowStarted .first false,
  .observeOperationStart .primary .primary .secondary .request .optionsUpdated .second false,
]

example : Umpire3.Temporal.Product.CallbackReference.executable.follow
    [Umpire3.Temporal.Product.CallbackReference.initial] matchingReferenceActions =
    [Umpire3.Temporal.Product.CallbackReference.matchingFinal] := by decide

example : Umpire3.Temporal.Product.CallbackReference.ReferenceQualified
    Umpire3.Temporal.Product.CallbackReference.matchingFinal := by decide

example : Umpire3.Temporal.Product.CallbackReference.executable.follow
    [Umpire3.Temporal.Product.CallbackReference.initial] wrongReferenceActions = [] := by decide

example : Umpire3.Temporal.Product.CallbackReference.weakenedExecutable.follow
    [Umpire3.Temporal.Product.CallbackReference.initial] wrongReferenceActions =
    [Umpire3.Temporal.Product.CallbackReference.wrongReferenceFinal] := by decide

example : ¬Umpire3.Temporal.Product.CallbackReference.ReferenceConsistency
    Umpire3.Temporal.Product.CallbackReference.wrongReferenceFinal := by decide

example :
    (Umpire3.Temporal.Product.CallbackReference.bounded.explore
      { maxDepth := 2, maxResults := 5000 }).all
      (fun execution => Umpire3.Temporal.Product.CallbackReference.referenceConsistencyB execution.2) = true := by decide

def referenceSystemActions : List Umpire3.Temporal.System.CallbackReference.Action := [
  .observeAttachment .primary .primary .event .workflowStarted .first false,
  .redeliverAttachment .primary,
  .observeOperationStart .primary .primary .primary .event .workflowStarted .second false,
]

example : Umpire3.Temporal.System.CallbackReference.executable.follow
    [Umpire3.Temporal.System.CallbackReference.initial] referenceSystemActions =
    [Umpire3.Temporal.System.CallbackReference.matchingFinal] := by decide

def consistentResponseActions : List Umpire3.Temporal.Product.CallbackResponse.Action := [
  .register .primary .primary .primary,
  .settleOperation .primary .second true,
  .recordResponse .primary .asyncSuccess .accepted .third,
  .recordResponse .primary .asyncSuccess .accepted .third,
]

def conflictingResponseActions : List Umpire3.Temporal.Product.CallbackResponse.Action := [
  .register .primary .primary .primary,
  .recordResponse .primary .asyncSuccess .accepted .second,
  .recordResponse .primary .failure .conflicting .third,
]

example : Umpire3.Temporal.Product.CallbackResponse.executable.follow
    [Umpire3.Temporal.Product.CallbackResponse.initial] consistentResponseActions =
    [Umpire3.Temporal.Product.CallbackResponse.consistentFinal] := by decide

example : Umpire3.Temporal.Product.CallbackResponse.ResponseQualified
    Umpire3.Temporal.Product.CallbackResponse.consistentFinal := by decide

example : Umpire3.Temporal.Product.CallbackResponse.executable.follow
    [Umpire3.Temporal.Product.CallbackResponse.initial] conflictingResponseActions = [] := by decide

example : Umpire3.Temporal.Product.CallbackResponse.weakenedExecutable.follow
    [Umpire3.Temporal.Product.CallbackResponse.initial] conflictingResponseActions =
    [Umpire3.Temporal.Product.CallbackResponse.conflictingFinal] := by decide

example : ¬Umpire3.Temporal.Product.CallbackResponse.ResponseConsistency
    Umpire3.Temporal.Product.CallbackResponse.conflictingFinal := by decide

example :
    (Umpire3.Temporal.Product.CallbackResponse.bounded.explore
      { maxDepth := 2, maxResults := 2500 }).all
      (fun execution => Umpire3.Temporal.Product.CallbackResponse.responseConsistencyB execution.2) = true := by decide

def responseSystemActions : List Umpire3.Temporal.System.CallbackResponse.Action := [
  .observeRegistration .primary .primary .primary,
  .observeSettlement .primary .second true,
  .observeResponse .primary .asyncSuccess .accepted .third,
  .redeliverResponse .primary,
]

example : Umpire3.Temporal.System.CallbackResponse.executable.follow
    [Umpire3.Temporal.System.CallbackResponse.initial] responseSystemActions =
    [Umpire3.Temporal.System.CallbackResponse.consistentFinal] := by decide

example : Umpire3.Temporal.Monitors.callbackReferenceConsistency.Holds
    (Umpire3.Temporal.Monitors.callbackReferenceObservations
      Umpire3.Temporal.Product.CallbackReference.matchingFinal) :=
  (Umpire3.Temporal.Monitors.callbackReference_monitor_equivalent
    Umpire3.Temporal.Product.CallbackReference.matchingFinal).2 (by decide)

example : Umpire3.Temporal.Monitors.callbackResponseConsistency.Holds
    (Umpire3.Temporal.Monitors.callbackResponseObservations
      Umpire3.Temporal.Product.CallbackResponse.consistentFinal) :=
  (Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent
    Umpire3.Temporal.Product.CallbackResponse.consistentFinal).2 (by decide)

end Umpire3.Tests.TemporalCallback
