import Temporal.Feature.Nexus.Experimental.CallerClosure
import Umpire.Property

namespace Temporal.Feature.Nexus.Experimental.CallerClosureTests

open _root_.Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Feature.Nexus.Experimental.AutoClose

/-- The experimental Nexus target-author inventory at the migration boundary. -/
def compatibilityTargetAuthors : List String := ["nexus-experimental-caller-closure"]

private def expectedCompiledArtifactJson : String :=
  include_str "testdata/nexus-caller-closure-experiment-spec.json"

def declarationErrorOf
    (result : Except DeclarationError α) : Option DeclarationError :=
  match result with
  | .error error => some error
  | .ok _ => none

def uniquenessSource : SemanticSource := {
  path := "Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def uniquenessCapability : DeclarationId :=
  DeclarationId.of "regression.capability.uniqueness"

def pendingCancelCount : DeclarationId :=
  DeclarationId.of "regression.state.pending-cancel-count"

def uniquenessMetadata (id : DeclarationId) (kind : DeclarationKind) : DeclarationMetadata := {
  id
  kind
  source := uniquenessSource
  contractDigest := id.value ++ "/v1"
}

def uniquenessContext : PropertyCheckContext := {
  declarations := [
    uniquenessMetadata uniquenessCapability .capability,
    uniquenessMetadata pendingCancelCount .state
  ]
  providers := [{
    id := uniquenessCapability
    version := 1
    semanticDigest := "regression-uniqueness/v1"
  }]
  meanings := [(uniquenessCapability, {
    declaration := pendingCancelCount
    kind := .state
    semanticDigest := "regression-pending-cancel-count/v1"
  })]
}

def uniquenessProperty : PropertyDeclaration := {
  id := DeclarationId.of "regression.property.cancel-is-unique"
  source := uniquenessSource
  requires := [uniquenessCapability]
  clauses := [.stateInvariant
    (DeclarationId.of "regression.property.cancel-is-unique.clause")
    {
      field := .state
      reference := pendingCancelCount
      constraint := .naturalAtMost 1
    }]
}

def semanticValue (identity : DeclarationId) (value : String) : SemanticValue := {
  identity
  value
}

def evaluateUniqueness
    (trace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue) :
    Option PropertyEvaluation :=
  (checkProperty uniquenessContext (.portable uniquenessProperty)).toOption.map fun property =>
    evaluateProperty property trace

def nexusUniquenessTrace
    (config : Config) : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
  initialState := semanticValue pendingCancelCount (toString config.cancels.length)
  steps := []
}

example : AtMostOneEvent (autoClose .upgrade wClash) :=
  upgrade_preserves_uniqueness wClash (wClash_reachable .upgrade)

example :
    (evaluateUniqueness (nexusUniquenessTrace (autoClose .upgrade wClash))).map
        PropertyEvaluation.satisfied = some true := by
  native_decide

example : ¬AtMostOneEvent (autoClose .duplicate wClash) := by
  simp [AtMostOneEvent, autoClose, applyResolution, wClash]

example :
    (evaluateUniqueness (nexusUniquenessTrace (autoClose .duplicate wClash))).map
        PropertyEvaluation.satisfied = some false := by
  native_decide

def expertTargetDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  id := targetDefinition.id
  source := targetDefinition.source
  declarations := targetDefinition.declarations
  requiredCapabilities := targetDefinition.requiredCapabilities
  providers := [
    workflowProvider,
    cancellationProvider,
    workflowOwnershipClaimProvider,
    cancellationOwnershipClaimProvider,
    ownershipProvider
  ]
  connectors := [ownershipConnector]
  resolvedSetups := targetDefinition.resolvedSetups
  kernel := targetDefinition.kernel
}

def targetWithoutConnector : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  expertTargetDeclaration with connectors := []
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
  expertTargetDeclaration with connectors := [ownershipConnectorWithoutLaw]
}

def missingLawErrorResult : Option DeclarationError :=
  declarationErrorOf (composeTarget targetWithoutOwnershipLaw)

private theorem missingLawErrorResult_isSome :
    missingLawErrorResult.isSome = true := by native_decide

def missingLawError : DeclarationError :=
  missingLawErrorResult.get missingLawErrorResult_isSome

def reorderedTargetDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  expertTargetDeclaration with
  declarations := expertTargetDeclaration.declarations.reverse
  providers := expertTargetDeclaration.providers.reverse
  connectors := expertTargetDeclaration.connectors.reverse
}

example : (checkTarget targetAuthoring).isOk = true := by
  native_decide

example : (checkTarget targetAuthoring).toOption.map (fun checked =>
    (checked.id, checked.source, canonicalCheckedTargetJson checked, checked.semanticDigest)) =
    some (targetId, source, canonicalCheckedTargetJson target, target.semanticDigest) := by
  native_decide

example : (composeTarget reorderedTargetDeclaration).toOption.map CheckedTarget.semanticDigest =
    (composeTarget expertTargetDeclaration).toOption.map CheckedTarget.semanticDigest := by
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

example : exactActionQuery.completeness.map (fun evidence =>
    (evidence.roleAssignments, evidence.actions,
      evidence.roleDomainDigest, evidence.actionDomainDigest)) =
    some ([clashSetup], [forceCloseAction],
      "workflow-nexus-role-domain/v1", "workflow-nexus-action-domain/v1") := by
  native_decide

example : exactActionQueryResult.toOption.map canonicalQueryJson =
    some (canonicalQueryJson exactActionQuery) := by
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

def alternateOwnershipConnectorId : DeclarationId :=
  DeclarationId.of "workflow-nexus.connector.ownership.alternate"

def alternateOwnershipConnectorMetadata : DeclarationMetadata := {
  id := alternateOwnershipConnectorId
  kind := .connector
  source
  contractDigest := "workflow-nexus-ownership-connector-alternate/v1"
}

def alternateOwnershipConnector : CapabilityConnector LawStatement := {
  ownershipConnector with
  id := alternateOwnershipConnectorId
  semanticDigest := "workflow-nexus-ownership-connector-alternate/v1"
}

def ambiguousConnectorDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  expertTargetDeclaration with
  declarations := alternateOwnershipConnectorMetadata :: expertTargetDeclaration.declarations
  connectors := [ownershipConnector, alternateOwnershipConnector]
}

example : (declarationErrorOf (composeTarget ambiguousConnectorDeclaration)).map
    DeclarationError.kind = some .ambiguousConnector := by
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

example : canonicalExperimentSpecJson compiledArtifact ++ "\n" = expectedCompiledArtifactJson := by
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

example : compatibilityTargetAuthors = ["nexus-experimental-caller-closure"] := by
  rfl

end Temporal.Feature.Nexus.Experimental.CallerClosureTests
