import Temporal.Feature.Nexus.Operations.Planning

/-!
# Successful Nexus operation completion

Read this walkthrough from its checked Property and Behavior through its Query and deterministic
planning run. Return to `Temporal.Feature.Nexus.Operations` for the complete ordinary operation map.
-/

namespace Temporal.Feature.Nexus.Operations

open Umpire
open Temporal.Feature.Nexus.Lifecycle

namespace SuccessfulCompletion

def propertyId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.property.successful-completion"
def behaviorId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.behavior.successful-completion"
def queryId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.query.successful-completion"
def setupConstraintId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.setup.started"
def occurrenceId : DefinitionId :=
  Internal.id "temporal.nexus.basic-lifecycle.occurrence.succeed"

def propertyDeclaration : PropertyDeclaration := {
  id := propertyId
  source
  requires := [lifecycleCapabilityId]
  clauses := [
    .transitionContract
      (Internal.id "temporal.nexus.basic-lifecycle.property.successful-completion.state")
      { field := .selectedAction, reference := reportSuccessActionId,
        constraint := .equals reportSuccessAction.value }
      { field := .resultingState, reference := operationStateId,
        constraint := .equals succeededState.value },
    .transitionContract
      (Internal.id "temporal.nexus.basic-lifecycle.property.successful-completion.outcome")
      { field := .selectedAction, reference := reportSuccessActionId,
        constraint := .equals reportSuccessAction.value }
      { field := .modelOutcome, reference := transitionOutcomeId,
        constraint := .equals succeededOutcome.value },
    .inputOutput
      (Internal.id "temporal.nexus.basic-lifecycle.property.successful-completion.observation")
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
  setup := [Internal.operationIs setupConstraintId startedState]
  allowedActions := [reportSuccessActionId]
  requiredOccurrences := [{ id := occurrenceId, action := reportSuccessActionId }]
  occurrenceBounds := [OccurrenceBound.exactly reportSuccessActionId 1]
  actionsExactly := some [reportSuccessActionId]
  documentation := "Select exactly one success report and leave its result to the Nexus model."
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

end SuccessfulCompletion

end Temporal.Feature.Nexus.Operations
