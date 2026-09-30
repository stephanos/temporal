import Umpire.Core

/-! Exact Model Trace representation checks. -/

namespace Umpire.CoreTests

open Umpire

def id (value : String) : DefinitionId := DefinitionId.of value

def exactTrace : ModelTrace Bool Bool Bool ModelValue := {
  initialState := false
  steps := [{
    selectedAction := true
    outcome := true
    state := true
    facts := [{
      definitionId := id "switch.observation.enabled"
      value := "enabled"
    }]
  }]
}

example : exactTrace.initialState = false ∧
    exactTrace.steps.map ModelTraceStep.selectedAction = [true] ∧
    exactTrace.steps.map ModelTraceStep.outcome = [true] ∧
    exactTrace.steps.map ModelTraceStep.state = [true] ∧
    exactTrace.steps.flatMap ModelTraceStep.facts = [{
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
    outcome := outcomeValue
    state := resultingValue
    facts := [firstObservationValue, secondObservationValue]
  }]
}

example : emptyTrace.coordinates = [.initialState] ∧
    emptyTrace.valueAt? .initialState = some initialValue := by
  decide

example : oneStepTrace.coordinates = [
    .initialState,
    .selectedAction 1,
    .outcome 1,
    .state 1,
    .fact 1 1,
    .fact 1 2
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
    oneStepTrace.valueAt? (.outcome 0),
    oneStepTrace.valueAt? (.state 0),
    oneStepTrace.valueAt? (.fact 0 1),
    oneStepTrace.valueAt? (.fact 1 0),
    oneStepTrace.valueAt? (.selectedAction 2),
    oneStepTrace.valueAt? (.outcome 2),
    oneStepTrace.valueAt? (.state 2),
    oneStepTrace.valueAt? (.fact 2 1),
    oneStepTrace.valueAt? (.fact 1 3)
  ] = List.replicate 10 none := by
  decide

example : oneStepTrace.coordinates.map ModelCoordinate.definitionKind = [
    .state,
    .action,
    .outcome,
    .state,
    .fact,
    .fact
  ] := by
  decide

def repeatedValueTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := initialValue
  steps := [
    {
      selectedAction := actionValue
      outcome := outcomeValue
      state := resultingValue
      facts := [firstObservationValue]
    },
    {
      selectedAction := actionValue
      outcome := outcomeValue
      state := resultingValue
      facts := [firstObservationValue]
    }
  ]
}

example : repeatedValueTrace.coordinates = [
    .initialState,
    .selectedAction 1,
    .outcome 1,
    .state 1,
    .fact 1 1,
    .selectedAction 2,
    .outcome 2,
    .state 2,
    .fact 2 1
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
