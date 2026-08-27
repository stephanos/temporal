import Umpire.Core

/-! Exact Model Trace representation checks. -/

namespace Umpire.CoreTests

open Umpire

def id (value : String) : DefinitionId := DefinitionId.of value

def exactTrace : ModelTrace Bool Bool Bool ModelValue := {
  initialState := false
  steps := [{
    selectedAction := true
    modelOutcome := true
    resultingState := true
    observations := [{
      definitionId := id "switch.observation.enabled"
      value := "enabled"
    }]
  }]
}

example : exactTrace.initialState = false ∧
    exactTrace.steps.map ModelTraceStep.selectedAction = [true] ∧
    exactTrace.steps.map ModelTraceStep.modelOutcome = [true] ∧
    exactTrace.steps.map ModelTraceStep.resultingState = [true] ∧
    exactTrace.steps.flatMap ModelTraceStep.observations = [{
      definitionId := id "switch.observation.enabled"
      value := "enabled"
    }] := by
  native_decide

end Umpire.CoreTests
