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

def definitionErrorOf
    (result : Except DefinitionError α) : Option DefinitionError :=
  match result with
  | .error error => some error
  | .ok _ => none

def uniquenessSource : SourceLocation := {
  path := "Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def uniquenessCapability : DefinitionId :=
  DefinitionId.of "regression.capability.uniqueness"

def pendingCancelCount : DefinitionId :=
  DefinitionId.of "regression.state.pending-cancel-count"

def uniquenessMetadata (id : DefinitionId) (kind : DefinitionKind) : DefinitionMetadata := {
  id
  kind
  source := uniquenessSource
  canonicalBehavior := id.value ++ "/v1"
}

def uniquenessContext : PropertyCheckContext := {
  definitions := [
    uniquenessMetadata uniquenessCapability .capability,
    uniquenessMetadata pendingCancelCount .state
  ]
  providers := [{
    id := uniquenessCapability
    version := 1
    canonicalBehavior := "regression-uniqueness/v1"
  }]
  meanings := [(uniquenessCapability, {
    definitionId := pendingCancelCount
    kind := .state
    canonicalBehavior := "regression-pending-cancel-count/v1"
  })]
}

def uniquenessProperty : PropertyDeclaration := {
  id := DefinitionId.of "regression.property.cancel-is-unique"
  source := uniquenessSource
  requires := [uniquenessCapability]
  clauses := [.stateInvariant
    (DefinitionId.of "regression.property.cancel-is-unique.clause")
    {
      field := .state
      reference := pendingCancelCount
      constraint := .naturalAtMost 1
    }]
}

def modelValue (definitionId : DefinitionId) (value : String) : ModelValue := {
  definitionId
  value
}

def evaluateUniqueness
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Option PropertyEvaluation :=
  (checkProperty uniquenessContext (.portable uniquenessProperty)).toOption.map fun property =>
    evaluateProperty property trace

def nexusUniquenessTrace
    (config : Config) : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := modelValue pendingCancelCount (toString config.cancels.length)
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
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetDefinition.id
  source := targetDefinition.source
  definitions := targetDefinition.definitions
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
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with connectors := []
}

def missingConnectorErrorResult : Option DefinitionError :=
  definitionErrorOf (composeTarget targetWithoutConnector)

private theorem missingConnectorErrorResult_isSome :
    missingConnectorErrorResult.isSome = true := by native_decide

def missingConnectorError : DefinitionError :=
  missingConnectorErrorResult.get missingConnectorErrorResult_isSome

def ownershipConnectorWithoutLaw : CapabilityConnector LawStatement := {
  ownershipConnector with lawWitnesses := []
}

def targetWithoutOwnershipLaw : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with connectors := [ownershipConnectorWithoutLaw]
}

def missingLawErrorResult : Option DefinitionError :=
  definitionErrorOf (composeTarget targetWithoutOwnershipLaw)

private theorem missingLawErrorResult_isSome :
    missingLawErrorResult.isSome = true := by native_decide

def missingLawError : DefinitionError :=
  missingLawErrorResult.get missingLawErrorResult_isSome

def reorderedTargetDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with
  definitions := expertTargetDeclaration.definitions.reverse
  providers := expertTargetDeclaration.providers.reverse
  connectors := expertTargetDeclaration.connectors.reverse
}

example : (checkTarget targetAuthoring).isOk = true := by
  native_decide

example : (checkTarget targetAuthoring).toOption.map (fun checked =>
    (checked.id, checked.source, canonicalCheckedTargetJson checked, checked.behaviorFingerprint)) =
    some (targetId, source, canonicalCheckedTargetJson target, target.behaviorFingerprint) := by
  native_decide

example : (composeTarget reorderedTargetDeclaration).toOption.map CheckedTarget.behaviorFingerprint =
    (composeTarget expertTargetDeclaration).toOption.map CheckedTarget.behaviorFingerprint := by
  native_decide

example : callerClosureProperty.requires =
    [cancellationCapabilityId, ownershipCapabilityId, workflowCapabilityId] := by
  native_decide

example : target.requiredCapabilities =
    [cancellationCapabilityId, ownershipCapabilityId, workflowCapabilityId] := by
  native_decide

example : (callerClosureProperty.access.meanings.filter fun meaning =>
    meaning.definitionId == ownershipRelationId).map MeaningProvision.canonicalBehavior =
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
      evidence.roleDomainFingerprint, evidence.actionDomainFingerprint)) =
    (CheckedQueryTarget.ofTarget target).completeness.map (fun evidence =>
      (evidence.roleAssignments, evidence.actions,
        evidence.roleDomainFingerprint, evidence.actionDomainFingerprint)) ∧
    exactActionQuery.completeness.map (fun evidence =>
      (evidence.roleAssignments, evidence.actions)) = some ([clashSetup], [forceCloseAction]) := by
  native_decide

example : exactActionQueryResult.toOption.map canonicalQueryJson =
    some (canonicalQueryJson exactActionQuery) := by
  native_decide

example : target.kernel.authoritativeInitial clashSetup clashState := by
  exact ⟨rfl, rfl, wClash_reachable .upgrade⟩

example : target.kernel.authoritativeStep clashState forceCloseAction forceCloseResult := by
  exact target_force_close_is_authoritative

example : missingConnectorError.kind = .conflictingProviders ∧
    missingConnectorError.definitionId = ownershipClaimId ∧
    missingConnectorError.sourcePath = source.path ∧
    missingConnectorError.relatedDefinitionIds =
      [cancellationOwnershipClaimProviderId, workflowOwnershipClaimProviderId] := by
  native_decide

example : missingLawError.kind = .missingLaw ∧
    missingLawError.definitionId = ownershipConnectorId ∧
    missingLawError.sourcePath = source.path ∧
    missingLawError.relatedDefinitionIds = [ownershipLawId] := by
  native_decide

def alternateOwnershipConnectorId : DefinitionId :=
  DefinitionId.of "workflow-nexus.connector.ownership.alternate"

def alternateOwnershipConnectorMetadata : DefinitionMetadata := {
  id := alternateOwnershipConnectorId
  kind := .connector
  source
  canonicalBehavior := "workflow-nexus-ownership-connector-alternate/v1"
}

def alternateOwnershipConnector : CapabilityConnector LawStatement := {
  ownershipConnector with
  id := alternateOwnershipConnectorId
  canonicalBehavior := "workflow-nexus-ownership-connector-alternate/v1"
}

def ambiguousConnectorDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with
  definitions := alternateOwnershipConnectorMetadata :: expertTargetDeclaration.definitions
  connectors := [ownershipConnector, alternateOwnershipConnector]
}

example : (definitionErrorOf (composeTarget ambiguousConnectorDeclaration)).map
    DefinitionError.kind = some .ambiguousConnector := by
  native_decide

example : exploratoryQuery.form = .select [callerClosureProperty] ∧
    exploratoryQuery.quantifier = .exploratory ∧
    exploratoryQuery.claim = .limitedSelection := by
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
    "verified-within-limits",
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

example : lifecycleLaw.body = "workflow-caller-closure-law/v1" ∧
    cancellationLaw.body = "nexus-cancellation-honored-law/v1" ∧
    ownershipLaw.body = "workflow-nexus-ownership-law/v1" ∧
    compiledArtifact.plan.kernelBehaviorFingerprint = target.behaviorFingerprint := by
  native_decide

example : compiledArtifact.formatVersion = "umpire-experiment/v1" ∧
    compiledArtifact.plan.formatVersion = "umpire-drive-plan/v1" ∧
    compiledArtifact.plan.queryDefinitionId = exactActionQueryId ∧
    compiledArtifact.plan.behaviorDefinitionId = exactActionBehaviorId ∧
    compiledArtifact.plan.targetDefinitionId = targetId ∧
    compiledArtifact.plan.kernelDefinitionId = kernelId ∧
    compiledArtifact.properties.map PortableProperty.definitionId =
      [callerClosurePropertyId] := by
  native_decide

example : compatibilityTargetAuthors = ["nexus-experimental-caller-closure"] := by
  rfl

end Temporal.Feature.Nexus.Experimental.CallerClosureTests
