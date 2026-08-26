import Temporal.Feature.Nexus.Examples.BasicLifecycle

namespace Temporal.Feature.Nexus.Examples.BasicOperations

open Umpire
open Temporal.Feature.Nexus.Examples.BasicLifecycle

private def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Temporal/Feature/Nexus/Examples/BasicOperations.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def operationRole : ResourceRole := { id := operationRoleId, valueKind := .state }

private def operationIs
    (constraintId : DeclarationId)
    (state : SemanticValue) : SetupConstraint := {
  id := constraintId
  relation := .equal
  left := .role operationRoleId
  right := .value state
}

private def checkBehaviorDeclaration
    (declaration : BehaviorDeclaration) : Except BehaviorError CheckedBehavior :=
  checkBehavior { declarations := target.declarations } declaration

private def queryDeclaration
    (queryId : DeclarationId)
    (property : CheckedProperty)
    (behavior : CheckedBehavior) : QueryDeclaration := {
  id := queryId
  source
  target := target.id
  form := .witness property
  behavior
  bounds
  policy
}

namespace AsyncStart

def propertyId := id "temporal.nexus.basic-lifecycle.property.async-start"
def behaviorId := id "temporal.nexus.basic-lifecycle.behavior.async-start"
def queryId := id "temporal.nexus.basic-lifecycle.query.async-start"
def setupConstraintId := id "temporal.nexus.basic-lifecycle.setup.scheduled"
def occurrenceId := id "temporal.nexus.basic-lifecycle.occurrence.start"

def propertyDeclaration : PropertyDeclaration := {
  id := propertyId
  source
  requires := [lifecycleCapabilityId]
  clauses := [
    .transitionContract (id "temporal.nexus.basic-lifecycle.property.async-start.state")
      { field := .selectedAction, reference := startActionId,
        constraint := .equals startAction.value }
      { field := .resultingState, reference := operationStateId,
        constraint := .equals startedState.value },
    .transitionContract (id "temporal.nexus.basic-lifecycle.property.async-start.outcome")
      { field := .selectedAction, reference := startActionId,
        constraint := .equals startAction.value }
      { field := .modelOutcome, reference := transitionOutcomeId,
        constraint := .equals startedOutcome.value },
    .inputOutput (id "temporal.nexus.basic-lifecycle.property.async-start.observation")
      { field := .selectedAction, reference := startActionId,
        constraint := .equals startAction.value }
      { field := .observation, reference := lifecycleObservationId,
        constraint := .equals startedObservation.value }
  ]
  documentation := "Starting a scheduled Nexus operation produces the target-owned started result."
}

def propertyResult : Except PropertyError CheckedProperty :=
  checkProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)

private theorem propertyResult_isSome : propertyResult.toOption.isSome = true := by
  native_decide

def property : CheckedProperty :=
  propertyResult.toOption.get propertyResult_isSome

def behaviorDeclaration : BehaviorDeclaration := {
  id := behaviorId
  source
  requires := [lifecycleCapabilityId]
  roles := [operationRole]
  setup := [operationIs setupConstraintId scheduledState]
  allowedActions := [startActionId]
  requiredOccurrences := [{ id := occurrenceId, action := startActionId }]
  occurrenceBounds := [OccurrenceBound.exactly startActionId 1]
  actionsExactly := some [startActionId]
  documentation := "Select exactly one start action and leave its result to the Nexus model."
}

def behaviorResult : Except BehaviorError CheckedBehavior :=
  checkBehaviorDeclaration behaviorDeclaration

private theorem behaviorResult_isSome : behaviorResult.toOption.isSome = true := by
  native_decide

def behavior : CheckedBehavior :=
  behaviorResult.toOption.get behaviorResult_isSome

def intendedTrace : BehaviorTrace := {
  setup := scheduledSetup
  trace := {
    initialState := scheduledState
    steps := [{
      selectedAction := startAction
      modelOutcome := startedOutcome
      resultingState := startedState
      observations := [startedObservation]
    }]
  }
}

/-- This target-inconsistent trace shows that Property, not Behavior, checks the model result. -/
def wrongOutcomeTrace : BehaviorTrace := {
  setup := scheduledSetup
  trace := {
    initialState := scheduledState
    steps := [{
      selectedAction := startAction
      modelOutcome := succeededOutcome
      resultingState := succeededState
      observations := [succeededObservation]
    }]
  }
}

def wrongActionTrace : BehaviorTrace := {
  setup := scheduledSetup
  trace := {
    initialState := scheduledState
    steps := [{
      selectedAction := reportSuccessAction
      modelOutcome := succeededOutcome
      resultingState := succeededState
      observations := [succeededObservation]
    }]
  }
}

def queryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext (queryDeclaration queryId property behavior)

private theorem queryResult_isSome : queryResult.toOption.isSome = true := by
  native_decide

def query : CheckedQuery LawStatement :=
  materializeQuery (queryResult.toOption.get queryResult_isSome)

theorem query_target : query.target = target := by
  rfl

def run : PlannerRun :=
  plan query (kernelFor query query_target)

def repeatedRun : PlannerRun :=
  plan query (kernelFor query query_target)

end AsyncStart

namespace SuccessfulCompletion

def propertyId := id "temporal.nexus.basic-lifecycle.property.successful-completion"
def behaviorId := id "temporal.nexus.basic-lifecycle.behavior.successful-completion"
def queryId := id "temporal.nexus.basic-lifecycle.query.successful-completion"
def setupConstraintId := id "temporal.nexus.basic-lifecycle.setup.started"
def occurrenceId := id "temporal.nexus.basic-lifecycle.occurrence.succeed"

def propertyDeclaration : PropertyDeclaration := {
  id := propertyId
  source
  requires := [lifecycleCapabilityId]
  clauses := [
    .transitionContract
      (id "temporal.nexus.basic-lifecycle.property.successful-completion.state")
      { field := .selectedAction, reference := reportSuccessActionId,
        constraint := .equals reportSuccessAction.value }
      { field := .resultingState, reference := operationStateId,
        constraint := .equals succeededState.value },
    .transitionContract
      (id "temporal.nexus.basic-lifecycle.property.successful-completion.outcome")
      { field := .selectedAction, reference := reportSuccessActionId,
        constraint := .equals reportSuccessAction.value }
      { field := .modelOutcome, reference := transitionOutcomeId,
        constraint := .equals succeededOutcome.value },
    .inputOutput
      (id "temporal.nexus.basic-lifecycle.property.successful-completion.observation")
      { field := .selectedAction, reference := reportSuccessActionId,
        constraint := .equals reportSuccessAction.value }
      { field := .observation, reference := lifecycleObservationId,
        constraint := .equals succeededObservation.value }
  ]
  documentation := "Reporting success for a started Nexus operation produces the target-owned succeeded result."
}

def propertyResult : Except PropertyError CheckedProperty :=
  checkProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)

private theorem propertyResult_isSome : propertyResult.toOption.isSome = true := by
  native_decide

def property : CheckedProperty :=
  propertyResult.toOption.get propertyResult_isSome

def behaviorDeclaration : BehaviorDeclaration := {
  id := behaviorId
  source
  requires := [lifecycleCapabilityId]
  roles := [operationRole]
  setup := [operationIs setupConstraintId startedState]
  allowedActions := [reportSuccessActionId]
  requiredOccurrences := [{ id := occurrenceId, action := reportSuccessActionId }]
  occurrenceBounds := [OccurrenceBound.exactly reportSuccessActionId 1]
  actionsExactly := some [reportSuccessActionId]
  documentation := "Select exactly one success report and leave its result to the Nexus model."
}

def behaviorResult : Except BehaviorError CheckedBehavior :=
  checkBehaviorDeclaration behaviorDeclaration

private theorem behaviorResult_isSome : behaviorResult.toOption.isSome = true := by
  native_decide

def behavior : CheckedBehavior :=
  behaviorResult.toOption.get behaviorResult_isSome

def intendedTrace : BehaviorTrace := {
  setup := startedSetup
  trace := {
    initialState := startedState
    steps := [{
      selectedAction := reportSuccessAction
      modelOutcome := succeededOutcome
      resultingState := succeededState
      observations := [succeededObservation]
    }]
  }
}

/-- This target-inconsistent trace shows that Property, not Behavior, checks the model result. -/
def wrongOutcomeTrace : BehaviorTrace := {
  setup := startedSetup
  trace := {
    initialState := startedState
    steps := [{
      selectedAction := reportSuccessAction
      modelOutcome := startedOutcome
      resultingState := startedState
      observations := [startedObservation]
    }]
  }
}

def wrongActionTrace : BehaviorTrace := {
  setup := startedSetup
  trace := {
    initialState := startedState
    steps := [{
      selectedAction := startAction
      modelOutcome := startedOutcome
      resultingState := startedState
      observations := [startedObservation]
    }]
  }
}

def queryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext (queryDeclaration queryId property behavior)

private theorem queryResult_isSome : queryResult.toOption.isSome = true := by
  native_decide

def query : CheckedQuery LawStatement :=
  materializeQuery (queryResult.toOption.get queryResult_isSome)

theorem query_target : query.target = target := by
  rfl

def run : PlannerRun :=
  plan query (kernelFor query query_target)

def repeatedRun : PlannerRun :=
  plan query (kernelFor query query_target)

end SuccessfulCompletion

end Temporal.Feature.Nexus.Examples.BasicOperations
