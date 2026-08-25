import Temporal
import Temporal.Umpire.Inspect
import UmpireTests

namespace Temporal.UmpireTests

open _root_.Umpire
open Temporal.Umpire
open Temporal.Umpire.NexusCallerClosure
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
    [cancellationCapabilityId, ownershipCapabilityId, workflowCapabilityId] := by
  native_decide

example : target.requiredCapabilities =
    [cancellationCapabilityId, ownershipCapabilityId, workflowCapabilityId] := by
  native_decide

example : (callerClosureProperty.access.meanings.filter fun meaning =>
    meaning.declaration == ownershipRelationId).map MeaningProvision.semanticDigest =
      ["workflow-nexus-operation-ownership/v1"] := by
  native_decide

example : target.kernel.initialStates clashSetup = [clashState] := by
  native_decide

example : target.kernel.steps clashState forceCloseAction = [forceCloseResult] := by
  native_decide

example : CallerOwnsOperation wClash closedConfig := by
  exact clashOwnershipProof

example : ownershipObservation.value = "true" := by
  native_decide

example : completeness.roleAssignments = [clashSetup] ∧
    completeness.actions = [forceCloseAction] := by
  native_decide

example : target.kernel.authoritativeInitial clashSetup clashState := by
  exact ⟨rfl, rfl, wClash_reachable .upgrade⟩

example : target.kernel.authoritativeStep clashState forceCloseAction forceCloseResult := by
  exact target_force_close_is_authoritative

example : missingConnectorError.kind = .conflictingProviders ∧
    missingConnectorError.declarationId = ownershipClaimId ∧
    missingConnectorError.sourcePath = source.path ∧
    missingConnectorError.relatedIdentities =
      [cancellationOwnershipClaimProviderId, workflowOwnershipClaimProviderId] := by
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

example : exactActionQueryId.value = "workflow-nexus.query.exact-action-caller-closure" ∧
    exactActionBehaviorId.value = "workflow-nexus.behavior.exact-action" ∧
    targetId.value = "workflow-nexus.target.caller-closure" ∧
    kernelId.value = "workflow-nexus.kernel.caller-closure" ∧
    callerClosurePropertyId.value = "workflow-nexus.property.caller-closure" := by
  native_decide

example : lifecycleLaw.semanticDigest = "workflow-caller-closure-law/v1" ∧
    cancellationLaw.semanticDigest = "nexus-cancellation-honored-law/v1" ∧
    ownershipLaw.semanticDigest = "workflow-nexus-ownership-law/v1" ∧
    compiledArtifact.plan.kernelSemanticDigest =
      "workflow-nexus-caller-closure-kernel/v1" := by
  native_decide

example : compiledArtifact.formatVersion = "umpire-experiment/v1" ∧
    compiledArtifact.plan.formatVersion = "umpire-drive-plan/v1" ∧
    compiledArtifact.plan.queryIdentity = exactActionQueryId ∧
    compiledArtifact.plan.behaviorIdentity = exactActionBehaviorId ∧
    compiledArtifact.plan.targetIdentity = targetId ∧
    compiledArtifact.plan.kernelIdentity = kernelId ∧
    compiledArtifact.properties.map PortableProperty.identity =
      [callerClosurePropertyId] := by
  native_decide

example : (composeTarget _root_.Umpire.Examples.Switch.targetDeclaration).isOk = true := by
  native_decide

example : _root_.Umpire.Examples.Switch.target.kernel.initialStates _root_.Umpire.Examples.Switch.switchSetup =
    [_root_.Umpire.Examples.Switch.offState] ∧
    _root_.Umpire.Examples.Switch.target.kernel.steps _root_.Umpire.Examples.Switch.offState _root_.Umpire.Examples.Switch.flipAction =
      [_root_.Umpire.Examples.Switch.appliedResult, _root_.Umpire.Examples.Switch.deferredResult] := by
  native_decide

example : _root_.Umpire.Examples.Switch.target.requiredCapabilities = [_root_.Umpire.Examples.Switch.switchCapabilityId] ∧
    _root_.Umpire.Examples.Switch.flipProperty.requires = [_root_.Umpire.Examples.Switch.switchCapabilityId] ∧
    _root_.Umpire.Examples.Switch.exploratoryBehavior.requires = [_root_.Umpire.Examples.Switch.switchCapabilityId] ∧
    _root_.Umpire.Examples.Switch.exactActionQuery.targetComposition =
      [_root_.Umpire.Examples.Switch.switchCapabilityId, _root_.Umpire.Examples.Switch.switchProviderId] := by
  native_decide

example : _root_.Umpire.Examples.Switch.exactActionBehavior.admits _root_.Umpire.Examples.Switch.appliedTrace &&
    _root_.Umpire.Examples.Switch.exactActionBehavior.admits _root_.Umpire.Examples.Switch.deferredTrace := by
  native_decide

example : _root_.Umpire.Examples.Switch.exactTraceBehavior.admits _root_.Umpire.Examples.Switch.appliedTrace &&
    !_root_.Umpire.Examples.Switch.exactTraceBehavior.admits _root_.Umpire.Examples.Switch.deferredTrace := by
  native_decide

example : [
    _root_.Umpire.Examples.Switch.exploratoryRun.result.outcome.name,
    _root_.Umpire.Examples.Switch.exactActionRun.result.outcome.name,
    _root_.Umpire.Examples.Switch.exactTraceRun.result.outcome.name
  ] = ["found", "found", "found"] := by
  native_decide

example : _root_.Umpire.Examples.Switch.compiledArtifact.plan.requestedActions =
      [_root_.Umpire.Examples.Switch.flipAction] ∧
    _root_.Umpire.Examples.Switch.compiledArtifact.plan.modelOutcomes = [_root_.Umpire.Examples.Switch.appliedOutcome] ∧
    _root_.Umpire.Examples.Switch.compiledArtifact.plan.resultingStates = [_root_.Umpire.Examples.Switch.onState] ∧
    _root_.Umpire.Examples.Switch.compiledArtifact.properties.map PortableProperty.identity =
      [_root_.Umpire.Examples.Switch.flipPropertyId] := by
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

def expectedSwitchStdout : String :=
  canonicalExperimentSpecJson _root_.Umpire.Examples.Switch.compiledArtifact ++ "\n"

def repeatedSwitchOutput : List String :=
  (List.range 2).map fun _ => (runCli [_root_.Umpire.Examples.Switch.exactActionQueryId.value]).stdout

example : runCli [_root_.Umpire.Examples.Switch.exactActionQueryId.value] = {
    status := 0
    stdout := expectedSwitchStdout
    stderr := ""
  } := by
  native_decide

example : repeatedSwitchOutput = List.replicate 2 expectedSwitchStdout := by
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

end Temporal.UmpireTests
