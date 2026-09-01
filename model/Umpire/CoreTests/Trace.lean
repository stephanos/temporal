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

def initialValue : ModelValue := {
  definitionId := id "trace.state.initial"
  value := "initial"
}

def actionValue : ModelValue := {
  definitionId := id "trace.action.advance"
  value := "advance"
}

def outcomeValue : ModelValue := {
  definitionId := id "trace.outcome.advanced"
  value := "advanced"
}

def resultingValue : ModelValue := {
  definitionId := id "trace.state.advanced"
  value := "advanced"
}

def firstObservationValue : ModelValue := {
  definitionId := id "trace.observation.first"
  value := "first"
}

def secondObservationValue : ModelValue := {
  definitionId := id "trace.observation.second"
  value := "second"
}

def emptyTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := initialValue
  steps := []
}

def oneStepTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := initialValue
  steps := [{
    selectedAction := actionValue
    modelOutcome := outcomeValue
    resultingState := resultingValue
    observations := [firstObservationValue, secondObservationValue]
  }]
}

example : emptyTrace.coordinates = [.initialState] ∧
    emptyTrace.valueAt? .initialState = some initialValue := by
  decide

example : oneStepTrace.coordinates = [
    .initialState,
    .selectedAction 1,
    .modelOutcome 1,
    .resultingState 1,
    .observation 1 1,
    .observation 1 2
  ] := by
  decide

example : oneStepTrace.coordinates.map oneStepTrace.valueAt? = [
    some initialValue,
    some actionValue,
    some outcomeValue,
    some resultingValue,
    some firstObservationValue,
    some secondObservationValue
  ] := by
  decide

example : [
    oneStepTrace.valueAt? (.selectedAction 0),
    oneStepTrace.valueAt? (.modelOutcome 0),
    oneStepTrace.valueAt? (.resultingState 0),
    oneStepTrace.valueAt? (.observation 0 1),
    oneStepTrace.valueAt? (.observation 1 0),
    oneStepTrace.valueAt? (.selectedAction 2),
    oneStepTrace.valueAt? (.modelOutcome 2),
    oneStepTrace.valueAt? (.resultingState 2),
    oneStepTrace.valueAt? (.observation 2 1),
    oneStepTrace.valueAt? (.observation 1 3)
  ] = List.replicate 10 none := by
  decide

example : oneStepTrace.coordinates.map ModelCoordinate.definitionKind = [
    .state,
    .action,
    .outcome,
    .state,
    .observation,
    .observation
  ] := by
  decide

def repeatedValueTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := initialValue
  steps := [
    {
      selectedAction := actionValue
      modelOutcome := outcomeValue
      resultingState := resultingValue
      observations := [firstObservationValue]
    },
    {
      selectedAction := actionValue
      modelOutcome := outcomeValue
      resultingState := resultingValue
      observations := [firstObservationValue]
    }
  ]
}

example : repeatedValueTrace.coordinates = [
    .initialState,
    .selectedAction 1,
    .modelOutcome 1,
    .resultingState 1,
    .observation 1 1,
    .selectedAction 2,
    .modelOutcome 2,
    .resultingState 2,
    .observation 2 1
  ] ∧ repeatedValueTrace.coordinates.map repeatedValueTrace.valueAt? = [
    some initialValue,
    some actionValue,
    some outcomeValue,
    some resultingValue,
    some firstObservationValue,
    some actionValue,
    some outcomeValue,
    some resultingValue,
    some firstObservationValue
  ] := by
  decide

end Umpire.CoreTests
