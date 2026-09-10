import Temporal.Feature.Nexus.Lifecycle.Target

namespace Temporal.Feature.Nexus.LifecycleTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle

theorem sourceRemainsAnchoredToLifecycleFacade : source = {
    path := "Temporal/Feature/Nexus/Lifecycle.lean"
    line := 1
    column := 1
    provenance := "lean-model"
  } := by
  native_decide

theorem definitionsRetainCanonicalMetadata : definitions = [
    { id := targetId, kind := .target, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-target/v2", documentation := "" },
    { id := kernelId, kind := .machine, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-kernel/v2", documentation := "" },
    { id := lifecycleCapabilityId, kind := .capability, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle/v2", documentation := "" },
    { id := lifecycleProviderId, kind := .provider, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-provider/v2", documentation := "" },
    { id := lifecycleLawId, kind := .law, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-authoritative-step/v2",
      documentation := "" },
    { id := operationStateId, kind := .state, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-state/v2", documentation := "" },
    { id := startActionId, kind := .action, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-start/v1", documentation := "" },
    { id := cancelActionId, kind := .action, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-cancel/v1", documentation := "" },
    { id := reportSuccessActionId, kind := .action, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-report-success/v1", documentation := "" },
    { id := transitionOutcomeId, kind := .outcome, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-outcome/v2", documentation := "" },
    { id := lifecycleObservationId, kind := .fact, source, version := 1,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-observation/v2", documentation := "" }
  ] := by
  native_decide

/-- The ordinary Nexus target-author inventory at the migration boundary. -/
def compatibilityTargetAuthors : List String := ["nexus-lifecycle"]

theorem targetAuthoringChecksWithRequiredComposition : (checkTarget targetAuthoring).isOk = true ∧
    target.requiredCapabilities = [lifecycleCapabilityId] ∧
    target.providers.map CapabilityProvider.id = [lifecycleProviderId] ∧
    target.connectors = [] := by
  native_decide

theorem checkedTargetRetainsCanonicalIdentity :
    (checkTarget targetAuthoring).toOption.map (fun checked =>
      (checked.id, checked.source, canonicalCheckedTargetJson checked,
        checked.behaviorFingerprint)) =
    some (targetId, source, canonicalCheckedTargetJson target, target.behaviorFingerprint) := by
  native_decide

private def baselineTargetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [lifecycleCapabilityId]
  resolvedSetups := roleAssignments
  kernel := finiteMachine.machineAvailability
}

private def baselineTargetAuthoring : AuthoredTarget LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make baselineTargetDefinition targetComposition
    (.available machine rfl finitePlanning)

theorem migratedTargetMatchesIndependentBaseline :
    targetDefinition = baselineTargetDefinition ∧
    (checkTarget targetAuthoring).toOption.map (fun checked =>
      (checked.id, checked.source, checked.definitions, checked.requiredCapabilities,
        checked.providers.map CapabilityProvider.id, checked.connectors,
        checked.resolvedSetups, canonicalCheckedTargetJson checked,
        checked.behaviorFingerprint)) =
    (checkTarget baselineTargetAuthoring).toOption.map (fun checked =>
      (checked.id, checked.source, checked.definitions, checked.requiredCapabilities,
        checked.providers.map CapabilityProvider.id, checked.connectors,
        checked.resolvedSetups, canonicalCheckedTargetJson checked,
        checked.behaviorFingerprint)) := by
  exact ⟨rfl, rfl⟩

theorem targetMachineryUsesFiniteMachineCapabilities : machine = finiteMachine.kernel ∧
    finitePlanning = finiteMachine.planning ∧
    targetDefinition.kernel = finiteMachine.machineAvailability := by
  exact ⟨rfl, rfl, rfl⟩

theorem targetBehaviorFingerprintRemainsStable : target.behaviorFingerprint.render =
    "sha256:8a55f0d5c46e705fe3f06ca9a16381104380f55be83b633c2208f433a5eba58c" := by
  native_decide

theorem targetAndActionDefinitionIdsRemainStable :
    targetId.value = "temporal.nexus.basic-lifecycle.target" ∧
    kernelId.value = "temporal.nexus.basic-lifecycle.kernel" ∧
    operationRoleId.value = "temporal.nexus.basic-lifecycle.role.operation" ∧
    startActionId.value = "temporal.nexus.basic-lifecycle.action.start" ∧
    cancelActionId.value = "temporal.nexus.basic-lifecycle.action.cancel" ∧
    reportSuccessActionId.value = "temporal.nexus.basic-lifecycle.action.succeed" := by
  native_decide

theorem targetKernelEnumeratesExposedLifecycleTransitions :
    target.kernel.initialStates scheduledSetup = [scheduledState] ∧
    target.kernel.initialStates startedSetup = [startedState] ∧
    target.kernel.steps scheduledState startAction = [startedResult] ∧
    target.kernel.steps startedState cancelAction = [canceledResult] ∧
    target.kernel.steps startedState reportSuccessAction = [succeededResult] := by
  native_decide

theorem targetKernelRejectsInvalidAndTerminalTransitions : target.kernel.initialStates [] = [] ∧
    target.kernel.steps scheduledState cancelAction = [] ∧
    target.kernel.steps scheduledState reportSuccessAction = [] ∧
    target.kernel.steps startedState startAction = [] ∧
    target.kernel.steps canceledState startAction = [] ∧
    target.kernel.steps canceledState cancelAction = [] ∧
    target.kernel.steps canceledState reportSuccessAction = [] ∧
    target.kernel.steps succeededState startAction = [] ∧
    target.kernel.steps succeededState cancelAction = [] ∧
    target.kernel.steps succeededState reportSuccessAction = [] := by
  native_decide

theorem finitePlanningEnumeratesTheExposedActionDomain : match target.planning with
    | .unavailable => False
    | .available capability =>
        capability.actions = [cancelAction, startAction, reportSuccessAction] := by
  change finiteMachine.actions = [cancelAction, startAction, reportSuccessAction]
  rfl

def expertTargetDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetDefinition.id
  source := targetDefinition.source
  definitions := targetDefinition.definitions
  requiredCapabilities := targetDefinition.requiredCapabilities
  providers := [lifecycleProvider]
  connectors := []
  resolvedSetups := targetDefinition.resolvedSetups
  kernel := targetDefinition.kernel
}

def missingProviderDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with providers := []
}

/-- Checked composition remains public so callers can inspect its typed Definition Error. -/
private def compositionError (result : Except DefinitionError α) : Option DefinitionError :=
  match result with
  | .error failure => some failure
  | .ok _ => none

theorem missingProviderReportsCompositionError :
    compositionError (composeTarget missingProviderDeclaration) = some {
      kind := .missingProvider
      definitionId := targetId
      sourcePath := "Temporal/Feature/Nexus/Lifecycle.lean"
      offendingValue := lifecycleCapabilityId.value
      relatedDefinitionIds := [lifecycleCapabilityId]
    } := by
  native_decide

def conflictingProviderId : DefinitionId := DefinitionId.of
  "temporal.nexus.basic-lifecycle.provider.conflicting"

def conflictingProvider : CapabilityProvider LawStatement := {
  id := conflictingProviderId
  source
  contract := lifecycleProvider.contract
  meanings := [{
    definitionId := operationStateId
    kind := .state
    canonicalBehavior := "temporal-nexus-basic-lifecycle-state/conflicting"
  }]
  lawWitnesses := lifecycleProvider.lawWitnesses
}

def conflictingProviderMetadata : DefinitionMetadata := {
  id := conflictingProviderId
  kind := .provider
  source
  canonicalBehavior := "temporal-nexus-basic-lifecycle-provider/conflicting"
}

def conflictingProviderDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with
  definitions := conflictingProviderMetadata :: expertTargetDeclaration.definitions
  providers := [lifecycleProvider, conflictingProvider]
}

theorem conflictingProvidersReportCompositionError :
    compositionError (composeTarget conflictingProviderDeclaration) = some {
      kind := .conflictingProviders
      definitionId := operationStateId
      sourcePath := "Temporal/Feature/Nexus/Lifecycle.lean"
      offendingValue := operationStateId.value
      relatedDefinitionIds := [lifecycleProviderId, conflictingProviderId]
    } := by
  native_decide

theorem compatibilityTargetAuthorsRetainsMigrationBoundary :
    compatibilityTargetAuthors = ["nexus-lifecycle"] := by
  rfl

#print axioms targetAuthoring

end Temporal.Feature.Nexus.LifecycleTests
