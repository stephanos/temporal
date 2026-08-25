import Temporal
import Temporal.Experiment.Inspect
import Temporal.Experiment.SemanticsTests
import Temporal.Experiment.PropertyTests
import Temporal.Experiment.BehaviorTests
import Temporal.Experiment.QueryTests
import Temporal.Experiment.PlannerTests

namespace Temporal.ExperimentTests

open Temporal.Experiment
open Temporal.Experiment.NexusCallerClosure
open NexusAutoClose

def declarationErrorOf
    (result : Except DeclarationError α) : Option DeclarationError :=
  match result with
  | .error error => some error
  | .ok _ => none

def targetWithoutConnector : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with connectors := []
}

def missingConnectorErrorResult : Option DeclarationError :=
  declarationErrorOf (composeTarget targetWithoutConnector)

private theorem missingConnectorErrorResult_isSome :
    missingConnectorErrorResult.isSome = true := by native_decide

def missingConnectorError : DeclarationError :=
  missingConnectorErrorResult.get missingConnectorErrorResult_isSome

def ownershipConnectorWithoutLaw : CapabilityConnector LawStatement := {
  ownershipConnector with lawWitnesses := []
}

def targetWithoutOwnershipLaw : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with connectors := [ownershipConnectorWithoutLaw]
}

def missingLawErrorResult : Option DeclarationError :=
  declarationErrorOf (composeTarget targetWithoutOwnershipLaw)

private theorem missingLawErrorResult_isSome :
    missingLawErrorResult.isSome = true := by native_decide

def missingLawError : DeclarationError :=
  missingLawErrorResult.get missingLawErrorResult_isSome

def reorderedTargetDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with
  declarations := targetDeclaration.declarations.reverse
  providers := targetDeclaration.providers.reverse
  connectors := targetDeclaration.connectors.reverse
}

example : (composeTarget targetDeclaration).isOk = true := by
  native_decide

example : (composeTarget reorderedTargetDeclaration).toOption.map CheckedTarget.semanticDigest =
    (composeTarget targetDeclaration).toOption.map CheckedTarget.semanticDigest := by
  native_decide

example : callerClosureProperty.requires =
    [cancellationCapabilityId, workflowCapabilityId] := by
  native_decide

example : target.requiredCapabilities =
    [cancellationCapabilityId, workflowCapabilityId] := by
  native_decide

example : target.kernel.initialStates clashSetup = [clashState] := by
  native_decide

example : target.kernel.steps clashState forceCloseAction = [forceCloseResult] := by
  native_decide

example : completeness.roleAssignments = [clashSetup] ∧
    completeness.actions = [forceCloseAction] := by
  native_decide

example : target.kernel.authoritativeInitial clashSetup clashState := by
  exact ⟨rfl, rfl, wClash_reachable .upgrade⟩

example : target.kernel.authoritativeStep clashState forceCloseAction forceCloseResult := by
  exact target_force_close_is_authoritative

example : missingConnectorError.kind = .conflictingProviders ∧
    missingConnectorError.declarationId = ownershipRelationId ∧
    missingConnectorError.sourcePath = source.path ∧
    missingConnectorError.relatedIdentities =
      [cancellationProviderId, workflowProviderId] := by
  native_decide

example : missingLawError.kind = .missingLaw ∧
    missingLawError.declarationId = ownershipConnectorId ∧
    missingLawError.sourcePath = source.path ∧
    missingLawError.relatedIdentities = [ownershipLawId] := by
  native_decide

example : exploratoryQuery.form = .select [callerClosureProperty] ∧
    exploratoryQuery.quantifier = .exploratory ∧
    exploratoryQuery.claim = .boundedSelection := by
  native_decide

example : exactActionBehavior.actionsExactly = some [forceCloseActionId] ∧
    exactActionBehavior.traceExactly = none := by
  native_decide

example : exactActionQuery.form = .witness callerClosureProperty ∧
    exactActionQuery.claim = .satisfyingWitness := by
  native_decide

example : exactTraceBehavior.traceExactly.isSome = true ∧
    exactTraceQuery.form = .witness callerClosureProperty := by
  native_decide

example : [
    verifyRun.result.outcome.name,
    exploratoryRun.result.outcome.name,
    exactActionRun.result.outcome.name,
    exactTraceRun.result.outcome.name
  ] = [
    "verified-within-bounds",
    "found",
    "found",
    "found"
  ] := by
  native_decide

example : exactActionRun.artifact = some compiledArtifact := by
  native_decide

example : compiledArtifact.plan.requestedActions = [forceCloseAction] ∧
    compiledArtifact.plan.modelOutcomes = [upgradedOutcome] ∧
    compiledArtifact.plan.resultingStates = [closedState] := by
  native_decide

def expectedStdout : String := canonicalExperimentSpecJson compiledArtifact ++ "\n"

example : runCli [exactActionQueryId.value] = {
    status := 0
    stdout := expectedStdout
    stderr := ""
  } := by
  native_decide

def repeatedOutput : List String :=
  (List.range 2).map fun _ => (runCli [exactActionQueryId.value]).stdout

example : repeatedOutput = List.replicate 2 expectedStdout := by
  native_decide

example : runCli ["missing-scenario"] = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"unknown-scenario\",\"subject\":\"missing-scenario\"," ++
        "\"context\":\"scenario registry\"}\n"
  } := by
  native_decide

def invalidCompositionScenario : Scenario := {
  id := "workflow-nexus.query.invalid-composition"
  result := .error (.declaration missingConnectorError)
}

example : runInspector [invalidCompositionScenario] [invalidCompositionScenario.id] = {
    status := 1
    stdout := ""
    stderr := canonicalDeclarationErrorJson missingConnectorError ++ "\n"
  } := by
  native_decide

end Temporal.ExperimentTests
