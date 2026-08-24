import Temporal.Monitors
import Temporal.Families.CallbackReference.Refinement

namespace Umpire3.Tests.Families.CallbackReference

set_option maxRecDepth 100000

def matchingReferenceActions : List Umpire3.Temporal.Feature.CallbackReference.Action := [
  .observeAttachment .primary .primary .event .workflowStarted .first false,
  .observeOperationStart .primary .primary .primary .event .workflowStarted .second false,
]

def wrongReferenceActions : List Umpire3.Temporal.Feature.CallbackReference.Action := [
  .observeAttachment .primary .primary .event .workflowStarted .first false,
  .observeOperationStart .primary .primary .secondary .request .optionsUpdated .second false,
]

example : Umpire3.Temporal.Feature.CallbackReference.executable.follow
    [Umpire3.Temporal.Feature.CallbackReference.initial] matchingReferenceActions =
    [Umpire3.Temporal.Feature.CallbackReference.matchingFinal] := by decide

example : Umpire3.Temporal.Feature.CallbackReference.ReferenceQualified
    Umpire3.Temporal.Feature.CallbackReference.matchingFinal := by decide

example : Umpire3.Temporal.Feature.CallbackReference.executable.follow
    [Umpire3.Temporal.Feature.CallbackReference.initial] wrongReferenceActions = [] := by decide

example : Umpire3.Temporal.Feature.CallbackReference.weakenedExecutable.follow
    [Umpire3.Temporal.Feature.CallbackReference.initial] wrongReferenceActions =
    [Umpire3.Temporal.Feature.CallbackReference.wrongReferenceFinal] := by decide

example : ¬Umpire3.Temporal.Feature.CallbackReference.ReferenceConsistency
    Umpire3.Temporal.Feature.CallbackReference.wrongReferenceFinal := by decide

example :
    (Umpire3.Temporal.Feature.CallbackReference.bounded.explore
      { maxDepth := 2, maxResults := 5000 }).all
      (fun execution => Umpire3.Temporal.Feature.CallbackReference.referenceConsistencyB execution.2) = true := by decide

def referenceSystemActions :
    List Umpire3.Temporal.System.CallbackReference.Action := [
  .observeAttachment,
  .observeMatchingOperation,
]

def wrongReferenceSystemActions :
    List Umpire3.Temporal.System.CallbackReference.Action := [
  .observeAttachment,
  .observeWrongOperation,
]

example : Umpire3.Temporal.System.CallbackReference.executable.follow ()
    [Umpire3.Temporal.System.CallbackReference.initial] referenceSystemActions =
    [.matchingOperationObserved] := by decide

example : Umpire3.Temporal.System.CallbackReference.executable.follow ()
    [Umpire3.Temporal.System.CallbackReference.initial]
    wrongReferenceSystemActions = [] := by decide

example : Umpire3.Temporal.System.CallbackReference.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.CallbackReference.initial]
    wrongReferenceSystemActions = [.wrongOperationObserved] := by decide

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.CallbackReference.System.mutatedBehavior
      Umpire3.Temporal.Refinement.CallbackReference.Feature.behavior
      Umpire3.Temporal.Refinement.CallbackReference.Projects
      Umpire3.Temporal.Refinement.CallbackReference.actionMap :=
  Umpire3.Temporal.Refinement.CallbackReference.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.callbackReferenceConsistency.Holds
    (Umpire3.Temporal.Monitors.callbackReferenceObservations
      Umpire3.Temporal.Feature.CallbackReference.matchingFinal) :=
  (Umpire3.Temporal.Monitors.callbackReference_monitor_equivalent
    Umpire3.Temporal.Feature.CallbackReference.matchingFinal).2 (by decide)

end Umpire3.Tests.Families.CallbackReference
