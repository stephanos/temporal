import Temporal.Feature.Nexus.Operations.Planning

/-!
# Nexus operation cancellation

Read this walkthrough from its checked Property and Behavior through its Query and deterministic
planning run. Continue with `Temporal.Feature.Nexus.Operations.SuccessfulCompletion` for the other
basic outcome of a started operation.
-/

namespace Temporal.Feature.Nexus.Operations

open Umpire
open Temporal.Feature.Nexus.Lifecycle

namespace Cancellation

def propertyId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.property.cancellation"
def behaviorId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.behavior.cancellation"
def queryId : DefinitionId := Internal.id "temporal.nexus.basic-lifecycle.query.cancellation"
def setupConstraintId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.setup.cancellation-started"
def occurrenceId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.occurrence.cancel"

def propertyDeclaration : PropertyDeclaration := {
  id := propertyId
  source
  requires := [lifecycleCapabilityId]
  clauses := [
    .transitionContract
      (Internal.id "temporal.nexus.basic-lifecycle.property.cancellation.state")
      { field := .selectedAction, reference := cancelActionId,
        constraint := .equals cancelAction.value }
      { field := .resultingState, reference := operationStateId,
        constraint := .equals canceledState.value },
    .transitionContract
      (Internal.id "temporal.nexus.basic-lifecycle.property.cancellation.outcome")
      { field := .selectedAction, reference := cancelActionId,
        constraint := .equals cancelAction.value }
      { field := .modelOutcome, reference := transitionOutcomeId,
        constraint := .equals canceledOutcome.value },
    .inputOutput
      (Internal.id "temporal.nexus.basic-lifecycle.property.cancellation.observation")
      { field := .selectedAction, reference := cancelActionId,
        constraint := .equals cancelAction.value }
      { field := .observation, reference := lifecycleObservationId,
        constraint := .equals canceledObservation.value }
  ]
  documentation := "Canceling a started Nexus operation produces the target-owned canceled result."
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
  setup := [Internal.operationIs setupConstraintId startedState]
  allowedActions := [cancelActionId]
  requiredOccurrences := [{ id := occurrenceId, action := cancelActionId }]
  occurrenceBounds := [OccurrenceBound.exactly cancelActionId 1]
  actionsExactly := some [cancelActionId]
  documentation := "Select exactly one cancel action and leave its result to the Nexus model."
}

def behaviorResult : Except BehaviorError CheckedBehavior :=
  Internal.checkBehaviorDeclaration behaviorDeclaration

private theorem behaviorResult_isSome : behaviorResult.toOption.isSome = true := by
  native_decide

def behavior : CheckedBehavior :=
  behaviorResult.toOption.get behaviorResult_isSome

def intendedTrace : BehaviorTrace := {
  setup := startedSetup
  trace := {
    initialState := startedState
    steps := [{
      selectedAction := cancelAction
      modelOutcome := canceledOutcome
      resultingState := canceledState
      observations := [canceledObservation]
    }]
  }
}

/-- This target-inconsistent trace shows that Property, not Behavior, checks the model result. -/
def wrongOutcomeTrace : BehaviorTrace := {
  setup := startedSetup
  trace := {
    initialState := startedState
    steps := [{
      selectedAction := cancelAction
      modelOutcome := succeededOutcome
      resultingState := succeededState
      observations := [succeededObservation]
    }]
  }
}

def wrongActionTrace : BehaviorTrace := {
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

def queryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext (Internal.queryDeclaration queryId property behavior)

private theorem queryResult_isSome : queryResult.toOption.isSome = true := by
  native_decide

def query : CheckedQuery LawStatement :=
  materializeQuery (queryResult.toOption.get queryResult_isSome)

theorem query_target : query.target = target := by
  rfl

def run : PlannerRun :=
  plan query incrementalKernel

def repeatedRun : PlannerRun :=
  plan query incrementalKernel

end Cancellation

end Temporal.Feature.Nexus.Operations
