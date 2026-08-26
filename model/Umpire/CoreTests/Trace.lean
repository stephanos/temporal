import Umpire.Core

/-! Exact semantic-trace representation checks. -/

namespace Umpire.CoreTests

open Umpire

def id (value : String) : DeclarationId := DeclarationId.of value

def exactTrace : SemanticTrace Bool Bool Bool SemanticValue := {
  initialState := false
  steps := [{
    selectedAction := true
    modelOutcome := true
    resultingState := true
    observations := [{
      identity := id "switch.observation.enabled"
      value := "enabled"
    }]
  }]
}

example : exactTrace.initialState = false ∧
    exactTrace.steps.map SemanticTraceStep.selectedAction = [true] ∧
    exactTrace.steps.map SemanticTraceStep.modelOutcome = [true] ∧
    exactTrace.steps.map SemanticTraceStep.resultingState = [true] ∧
    exactTrace.steps.flatMap SemanticTraceStep.observations = [{
      identity := id "switch.observation.enabled"
      value := "enabled"
    }] := by
  native_decide

end Umpire.CoreTests
