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

def propertySpec : PropertySpec := {
  family := Internal.family
  key := "cancellation"
  source
  requires := [lifecycleCapabilityId]
  clauses := Internal.operationTransitionResultClauses "cancellation"
    cancelAction canceledState canceledOutcome canceledObservation
  documentation := "Canceling a started Nexus operation produces the target-owned canceled result."
}

def propertyDeclaration : PropertyDeclaration := propertySpec.declaration

def propertyResult : Except PropertyError CheckedProperty :=
  propertySpec.check (PropertyCheckContext.ofTarget target)

private theorem propertyResult_isSome : propertyResult.toOption.isSome = true := by
  native_decide

def property : CheckedProperty :=
  propertySpec.checked (PropertyCheckContext.ofTarget target) propertyResult_isSome

def behaviorSpec : ExactSequenceSpec := {
  family := Internal.family
  key := "cancellation"
  source
  requires := [lifecycleCapabilityId]
  roles := [operationRole]
  setup := [SetupConstraint.roleEquals setupConstraintId operationRoleId startedState]
  occurrences := [{ key := "cancel", action := cancelActionId }]
  documentation := "Select exactly one cancel action and leave its result to the Nexus model."
}

def behaviorDeclaration : BehaviorDeclaration := behaviorSpec.declaration

def behaviorResult : Except BehaviorError CheckedBehavior :=
  behaviorSpec.check (.ofTarget target)

private theorem behaviorResult_isSome : behaviorResult.toOption.isSome = true := by
  native_decide

def behavior : CheckedBehavior :=
  behaviorSpec.checked (.ofTarget target) behaviorResult_isSome

def intendedTrace : BehaviorTrace :=
  BehaviorTrace.singleStep startedSetup startedState cancelAction canceledResult

/-- This target-inconsistent trace shows that Property, not Behavior, checks the model result. -/
def wrongOutcomeTrace : BehaviorTrace :=
  BehaviorTrace.singleStep startedSetup startedState cancelAction succeededResult

def wrongActionTrace : BehaviorTrace :=
  BehaviorTrace.singleStep startedSetup startedState reportSuccessAction succeededResult

def querySpec : QuerySpec := Internal.querySpec "cancellation" property behavior

def queryDeclaration : QueryDeclaration := querySpec.declaration

def queryResult : Except QueryError (CheckedQuery LawStatement) := querySpec.check target

private theorem queryResult_isSome : queryResult.toOption.isSome = true := by
  native_decide

def query : CheckedQuery LawStatement :=
  querySpec.checked target queryResult_isSome

theorem query_target : query.target = target := by
  rfl

def incrementalKernelResult :
    Except FinitePlannerAdmissionError (IncrementalPlannerKernel query.target) :=
  IncrementalPlannerKernel.ofCheckedQuery target.id query

private theorem incrementalKernelResult_isSome :
    incrementalKernelResult.toOption.isSome = true := by
  have completenessIsSome :
      (CheckedQueryTarget.ofTarget target).completeness.isSome = true := by
    rfl
  let evidence := (CheckedQueryTarget.ofTarget target).completeness.get completenessIsSome
  have targetBneSelf : (target.id != target.id) = false := by
    cases target.id with
    | mk value =>
      change (value != value) = false
      exact bne_self_eq_false value
  apply IncrementalPlannerKernel.ofCheckedQuery_isSome target.id query evidence
  · simpa [query_target] using targetBneSelf
  · rw [show query.completeness = (CheckedQueryTarget.ofTarget target).completeness from rfl]
    exact (Option.some_get completenessIsSome).symm
  · change ∃ domain, finiteMachine.kernel.behaviorDomain = .complete domain
    exact Internal.lifecycleBehaviorDomainComplete
  · change actionDomain.mergeSort (fun left right =>
      decide (modelValueOrderKey left ≤ modelValueOrderKey right)) = actionDomain
    exact Internal.lifecycleActionDomainCanonical
  · intro setup
    change (initialState? setup).toList.mergeSort (fun left right =>
      decide (modelValueOrderKey left ≤ modelValueOrderKey right)) = (initialState? setup).toList
    exact Internal.lifecycleInitialStatesCanonical setup
  · intro state action
    change (stepResult? state action).toList.mergeSort (fun left right =>
      decide (transitionResultOrderKey left ≤ transitionResultOrderKey right)) =
      (stepResult? state action).toList
    exact Internal.lifecycleStepResultsCanonical state action

def incrementalKernel : IncrementalPlannerKernel query.target :=
  incrementalKernelResult.toOption.get incrementalKernelResult_isSome

def run : PlannerRun :=
  plan query incrementalKernel

def repeatedRun : PlannerRun :=
  plan query incrementalKernel

end Cancellation

end Temporal.Feature.Nexus.Operations
