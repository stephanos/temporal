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
      (PropertyPattern.exact .selectedAction reportSuccessActionId reportSuccessAction.value)
      (PropertyPattern.exact .resultingState operationStateId succeededState.value),
    .transitionContract
      (Internal.id "temporal.nexus.basic-lifecycle.property.successful-completion.outcome")
      (PropertyPattern.exact .selectedAction reportSuccessActionId reportSuccessAction.value)
      (PropertyPattern.exact .modelOutcome transitionOutcomeId succeededOutcome.value),
    .inputOutput
      (Internal.id "temporal.nexus.basic-lifecycle.property.successful-completion.observation")
      (PropertyPattern.exact .selectedAction reportSuccessActionId reportSuccessAction.value)
      (PropertyPattern.exact .observation lifecycleObservationId succeededObservation.value)
  ]
  documentation := "Reporting success for a started Nexus operation produces the target-owned succeeded result."
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
    { id := occurrenceId, action := reportSuccessActionId }
    (requires := [lifecycleCapabilityId])
    (roles := [operationRole])
    (setup := [SetupConstraint.roleEquals setupConstraintId operationRoleId startedState])
    (documentation := "Select exactly one success report and leave its result to the Nexus model.")

def behaviorResult : Except BehaviorError CheckedBehavior :=
  Internal.checkBehaviorDeclaration behaviorDeclaration

private theorem behaviorResult_isSome : behaviorResult.toOption.isSome = true := by
  native_decide

def behavior : CheckedBehavior :=
  checkedBehavior (.ofTarget target) behaviorDeclaration behaviorResult_isSome

def intendedTrace : BehaviorTrace :=
  BehaviorTrace.singleStep startedSetup startedState reportSuccessAction succeededResult

/-- This target-inconsistent trace shows that Property, not Behavior, checks the model result. -/
def wrongOutcomeTrace : BehaviorTrace :=
  BehaviorTrace.singleStep startedSetup startedState reportSuccessAction startedResult

def wrongActionTrace : BehaviorTrace :=
  BehaviorTrace.singleStep startedSetup startedState startAction startedResult

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

end SuccessfulCompletion

end Temporal.Feature.Nexus.Operations
