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
      (PropertyPattern.exact .selectedAction cancelActionId cancelAction.value)
      (PropertyPattern.exact .resultingState operationStateId canceledState.value),
    .transitionContract
      (Internal.id "temporal.nexus.basic-lifecycle.property.cancellation.outcome")
      (PropertyPattern.exact .selectedAction cancelActionId cancelAction.value)
      (PropertyPattern.exact .modelOutcome transitionOutcomeId canceledOutcome.value),
    .inputOutput
      (Internal.id "temporal.nexus.basic-lifecycle.property.cancellation.observation")
      (PropertyPattern.exact .selectedAction cancelActionId cancelAction.value)
      (PropertyPattern.exact .observation lifecycleObservationId canceledObservation.value)
  ]
  documentation := "Canceling a started Nexus operation produces the target-owned canceled result."
}

def propertyResult : Except PropertyError CheckedProperty :=
  checkProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)

private theorem propertyResult_isSome : propertyResult.toOption.isSome = true := by
  native_decide

def property : CheckedProperty :=
  checkedProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)
    propertyResult_isSome

def behaviorDeclaration : BehaviorDeclaration :=
  BehaviorDeclaration.exactlyOneAction behaviorId source
    { id := occurrenceId, action := cancelActionId }
    (requires := [lifecycleCapabilityId])
    (roles := [operationRole])
    (setup := [SetupConstraint.roleEquals setupConstraintId operationRoleId startedState])
    (documentation := "Select exactly one cancel action and leave its result to the Nexus model.")

def behaviorResult : Except BehaviorError CheckedBehavior :=
  Internal.checkBehaviorDeclaration behaviorDeclaration

private theorem behaviorResult_isSome : behaviorResult.toOption.isSome = true := by
  native_decide

def behavior : CheckedBehavior :=
  checkedBehavior (.ofTarget target) behaviorDeclaration behaviorResult_isSome

def intendedTrace : BehaviorTrace :=
  BehaviorTrace.singleStep startedSetup startedState cancelAction canceledResult

/-- This target-inconsistent trace shows that Property, not Behavior, checks the model result. -/
def wrongOutcomeTrace : BehaviorTrace :=
  BehaviorTrace.singleStep startedSetup startedState cancelAction succeededResult

def wrongActionTrace : BehaviorTrace :=
  BehaviorTrace.singleStep startedSetup startedState reportSuccessAction succeededResult

def queryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext (Internal.queryDeclaration queryId property behavior)

private theorem queryResult_isSome : queryResult.toOption.isSome = true := by
  native_decide

def query : CheckedQuery LawStatement :=
  checkedQuery target (Internal.queryDeclaration queryId property behavior)
    queryResult_isSome

theorem query_target : query.target = target := by
  rfl

def run : PlannerRun :=
  plan query incrementalKernel

def repeatedRun : PlannerRun :=
  plan query incrementalKernel

end Cancellation

end Temporal.Feature.Nexus.Operations
