import Temporal.Feature.Nexus.Operations.Search
import Umpire.Scenario.Elab

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

def authoredProperty : Property := {
  id := (Internal.family).id "property" "successful-completion"
  source
  requires := [lifecycleCapabilityId]
  clauses := Internal.operationStepClauses "successful-completion"
    reportSuccessAction succeededState succeededOutcome succeededObservation
  documentation := "Reporting success for a started Nexus operation produces the target-owned succeeded result."
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
    (key := "successful-completion")
    (source := source)
    (requires := [lifecycleCapabilityId])
    (roles := [operationRole])
    (setup := [SetupConstraint.roleEquals setupConstraintId operationRoleId startedState])
    (occurrences := [{ key := "succeed", action := reportSuccessActionId }])
    (documentation := "Select exactly one success report and leave its result to the Nexus model.")

def behaviorResult : Except ScenarioError CheckedScenario :=
  authoredScenario.check (.ofTarget target)

private theorem behaviorResult_isSome : behaviorResult.toOption.isSome = true := by
  native_decide

def behavior : CheckedScenario :=
  authoredScenario.checked (.ofTarget target) behaviorResult_isSome

def intendedTrace : Scenario.Trace :=
  Scenario.Trace.singleStep startedSetup startedState reportSuccessAction succeededResult

/-- This target-inconsistent trace shows that Property, not Behavior, checks the model result. -/
def wrongOutcomeTrace : Scenario.Trace :=
  Scenario.Trace.singleStep startedSetup startedState reportSuccessAction startedResult

def wrongActionTrace : Scenario.Trace :=
  Scenario.Trace.singleStep startedSetup startedState startAction startedResult

def querySpec : QuerySpec := Internal.querySpec "successful-completion" property behavior

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
  search query incrementalKernel

def repeatedRun : Except KnownGapError PlanResult :=
  search query incrementalKernel

end SuccessfulCompletion

end Temporal.Feature.Nexus.Operations
