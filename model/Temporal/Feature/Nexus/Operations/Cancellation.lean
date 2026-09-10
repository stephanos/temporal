import Temporal.Feature.Nexus.Operations.Planning
import Umpire.Behavior.Authoring

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
  clauses := Internal.operationStepClauses "cancellation"
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
  have completenessIsSome : query.completeness.isSome = true := by rfl
  let evidence := query.completeness.get completenessIsSome
  apply lifecycleIncrementalKernelResult_isSome query query_target evidence
  · exact (Option.some_get completenessIsSome).symm
  · rfl

def incrementalKernel : IncrementalPlannerKernel query.target :=
  incrementalKernelResult.toOption.get incrementalKernelResult_isSome

def run : Except KnownGapError PlannerRun :=
  plan query incrementalKernel

def repeatedRun : Except KnownGapError PlannerRun :=
  plan query incrementalKernel

end Cancellation

end Temporal.Feature.Nexus.Operations
