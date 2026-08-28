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
    { id := kernelId, kind := .kernel, source, version := 1,
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
    { id := lifecycleObservationId, kind := .observation, source, version := 1,
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

theorem targetMachineryUsesFiniteMachineCapabilities : transitionKernel = finiteMachine.kernel ∧
    finitePlanning = finiteMachine.planning ∧
    targetDefinition.kernel = finiteMachine.kernelAvailability := by
  exact ⟨rfl, rfl, rfl⟩

theorem targetBehaviorFingerprintRemainsStable : target.behaviorFingerprint.render =
    "sha256:2dffda3904f7425aa7ef89876393dc1648edcca0a944139672b6e35dd1651d93" := by
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
  simp [target, checkedTarget, targetAuthoring, AuthoredTarget.make, targetDefinition,
    finitePlanning, actionDomain]

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
private def compositionErrorKind (result : Except DefinitionError α) :
    Option DefinitionErrorKind :=
  match result with
  | .error failure => some failure.kind
  | .ok _ => none

theorem missingProviderReportsCompositionError :
    compositionErrorKind (composeTarget missingProviderDeclaration) =
      some .missingProvider := by
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
    compositionErrorKind (composeTarget conflictingProviderDeclaration) =
      some .conflictingProviders := by
  native_decide

theorem compatibilityTargetAuthorsRetainsMigrationBoundary :
    compatibilityTargetAuthors = ["nexus-lifecycle"] := by
  rfl

end Temporal.Feature.Nexus.LifecycleTests
