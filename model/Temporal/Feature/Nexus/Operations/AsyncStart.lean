import Temporal.Feature.Nexus.Operations.Search
import Umpire.Scenario.Elab

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

def authoredProperty : Property := {
  id := (Internal.family).id "property" "async-start"
  source
  requires := [lifecycleCapabilityId]
  clauses := Internal.operationStepClauses "async-start"
    startAction startedState startedOutcome startedObservation
  documentation := "Starting a scheduled Nexus operation produces the target-owned started result."
}

def propertyResult : Except PropertyError CheckedProperty :=
  authoredProperty.check (PropertyCheckContext.ofTarget target)

private theorem propertyResult_isSome : propertyResult.toOption.isSome = true := by
  native_decide

def property : CheckedProperty :=
  authoredProperty.checked (PropertyCheckContext.ofTarget target) propertyResult_isSome

def authoredScenario : Scenario :=
  Scenario.exactly
    (family := Internal.family)
    (key := "async-start")
    (source := source)
    (requires := [lifecycleCapabilityId])
    (roles := [operationRole])
    (setup := [SetupConstraint.roleEquals setupConstraintId operationRoleId scheduledState])
    (occurrences := [{ key := "start", action := startActionId }])
    (documentation := "Select exactly one start action and leave its result to the Nexus model.")

def behaviorResult : Except ScenarioError CheckedScenario :=
  authoredScenario.check (.ofTarget target)

private theorem behaviorResult_isSome : behaviorResult.toOption.isSome = true := by
  native_decide

def behavior : CheckedScenario :=
  authoredScenario.checked (.ofTarget target) behaviorResult_isSome

def intendedTrace : Scenario.Trace :=
  Scenario.Trace.singleStep scheduledSetup scheduledState startAction startedResult

/-- This target-inconsistent trace shows that Property, not Behavior, checks the model result. -/
def wrongOutcomeTrace : Scenario.Trace :=
  Scenario.Trace.singleStep scheduledSetup scheduledState startAction succeededResult

def wrongActionTrace : Scenario.Trace :=
  Scenario.Trace.singleStep scheduledSetup scheduledState reportSuccessAction succeededResult

def querySpec : QuerySpec := Internal.querySpec "async-start" property behavior

def queryDeclaration : QueryDeclaration := querySpec.declaration

def queryResult : Except QueryError (CheckedQuery LawStatement) := querySpec.check target

private theorem queryResult_isSome : queryResult.toOption.isSome = true := by
  native_decide

def query : CheckedQuery LawStatement :=
  querySpec.checked target queryResult_isSome

theorem query_target : query.target = target := by
  rfl

def incrementalKernelResult :
    Except FiniteSearchAdmissionError (SearchView query.target) :=
  SearchView.ofCheckedQuery target.id query

private theorem incrementalKernelResult_isSome :
    incrementalKernelResult.toOption.isSome = true := by
  have completenessIsSome : query.completeness.isSome = true := by rfl
  let evidence := query.completeness.get completenessIsSome
  apply lifecycleIncrementalKernelResult_isSome query query_target evidence
  · exact (Option.some_get completenessIsSome).symm
  · rfl

def incrementalKernel : SearchView query.target :=
  incrementalKernelResult.toOption.get incrementalKernelResult_isSome

def run : Except KnownGapError PlanResult :=
  plan query incrementalKernel

def repeatedRun : Except KnownGapError PlanResult :=
  plan query incrementalKernel

end AsyncStart

end Temporal.Feature.Nexus.Operations
