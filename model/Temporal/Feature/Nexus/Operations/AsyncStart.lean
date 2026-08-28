import Temporal.Feature.Nexus.Operations.Planning

/-!
# Asynchronous Nexus operation start

Read this walkthrough from its checked Property and Behavior through its Query and deterministic
planning run. Continue with `Temporal.Feature.Nexus.Operations.Cancellation` for a started operation.
-/

namespace Temporal.Feature.Nexus.Operations

open Umpire
open Temporal.Feature.Nexus.Lifecycle

namespace AsyncStart

def propertyId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.property.async-start"
def behaviorId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.behavior.async-start"
def queryId : DefinitionId := Internal.id "temporal.nexus.basic-lifecycle.query.async-start"
def setupConstraintId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.setup.scheduled"
def occurrenceId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.occurrence.start"

def propertyDeclaration : PropertyDeclaration := {
  id := propertyId
  source
  requires := [lifecycleCapabilityId]
  clauses := [
    .transitionContract
      (Internal.id "temporal.nexus.basic-lifecycle.property.async-start.state")
      { field := .selectedAction, reference := startActionId,
        constraint := .equals startAction.value }
      { field := .resultingState, reference := operationStateId,
        constraint := .equals startedState.value },
    .transitionContract
      (Internal.id "temporal.nexus.basic-lifecycle.property.async-start.outcome")
      { field := .selectedAction, reference := startActionId,
        constraint := .equals startAction.value }
      { field := .modelOutcome, reference := transitionOutcomeId,
        constraint := .equals startedOutcome.value },
    .inputOutput
      (Internal.id "temporal.nexus.basic-lifecycle.property.async-start.observation")
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
  setup := [Internal.operationIs setupConstraintId scheduledState]
  allowedActions := [startActionId]
  requiredOccurrences := [{ id := occurrenceId, action := startActionId }]
  occurrenceBounds := [OccurrenceBound.exactly startActionId 1]
  actionsExactly := some [startActionId]
  documentation := "Select exactly one start action and leave its result to the Nexus model."
}

def behaviorResult : Except BehaviorError CheckedBehavior :=
  Internal.checkBehaviorDeclaration behaviorDeclaration

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

end AsyncStart

end Temporal.Feature.Nexus.Operations
