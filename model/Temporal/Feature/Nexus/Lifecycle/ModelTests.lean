import Temporal.Feature.Nexus.Lifecycle.Model

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
      behaviorVersion := "temporal-nexus-basic-lifecycle-target/v2", documentation := "" },
    { id := kernelId, kind := .machine, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle-kernel/v2", documentation := "" },
    { id := lifecycleCapabilityId, kind := .capability, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle/v2", documentation := "" },
    { id := lifecycleProviderId, kind := .provider, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle-provider/v2", documentation := "" },
    { id := lifecycleLawId, kind := .law, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle-authoritative-step/v2",
      documentation := "" },
    { id := operationStateId, kind := .state, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle-state/v2", documentation := "" },
    { id := startActionId, kind := .action, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle-start/v1", documentation := "" },
    { id := cancelActionId, kind := .action, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle-cancel/v1", documentation := "" },
    { id := reportSuccessActionId, kind := .action, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle-report-success/v1", documentation := "" },
    { id := transitionOutcomeId, kind := .outcome, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle-outcome/v2", documentation := "" },
    { id := lifecycleObservationId, kind := .fact, source, version := 1,
      behaviorVersion := "temporal-nexus-basic-lifecycle-observation/v2", documentation := "" }
  ] := by
  native_decide

/-- The ordinary Nexus target-author inventory at the migration boundary. -/
def compatibilityTargetAuthors : List String := ["nexus-lifecycle"]

theorem targetAuthoringChecksWithRequiredComposition : (checkModel targetAuthoring).isOk = true ∧
    target.requiredCapabilities = [lifecycleCapabilityId] ∧
    target.providers.map Provider.id = [lifecycleProviderId] ∧
    target.connectors = [] := by
  native_decide

theorem checkedTargetRetainsCanonicalIdentity :
    (checkModel targetAuthoring).toOption.map (fun checked =>
      (checked.id, checked.source, canonicalCheckedModelJson checked,
        checked.behaviorFingerprint)) =
    some (targetId, source, canonicalCheckedModelJson target, target.behaviorFingerprint) := by
  native_decide

private def baselineModelSpec : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [lifecycleCapabilityId]
  resolvedSetups := roleAssignments
  machine := finiteMachine.machineAvailability
}

private def baselineTargetAuthoring : DraftModel LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make baselineModelSpec modelProviders
    (.available machine rfl finitePlanning)

theorem migratedTargetMatchesIndependentBaseline :
    modelSpec = baselineModelSpec ∧
    (checkModel targetAuthoring).toOption.map (fun checked =>
      (checked.id, checked.source, checked.definitions, checked.requiredCapabilities,
        checked.providers.map Provider.id, checked.connectors,
        checked.resolvedSetups, canonicalCheckedModelJson checked,
        checked.behaviorFingerprint)) =
    (checkModel baselineTargetAuthoring).toOption.map (fun checked =>
      (checked.id, checked.source, checked.definitions, checked.requiredCapabilities,
        checked.providers.map Provider.id, checked.connectors,
        checked.resolvedSetups, canonicalCheckedModelJson checked,
        checked.behaviorFingerprint)) := by
  exact ⟨rfl, rfl⟩

theorem targetMachineryUsesFiniteMachineCapabilities : machine = finiteMachine.kernel ∧
    finitePlanning = finiteMachine.planning ∧
    modelSpec.machine = finiteMachine.machineAvailability := by
  exact ⟨rfl, rfl, rfl⟩

theorem targetBehaviorFingerprintRemainsStable : target.behaviorFingerprint.render =
    "sha256:bf81a3382115f48aa4f04d2668b9c587a47b95f50dbad00abb10b2d2ad806dc6" := by
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
    target.machine.initialStates scheduledSetup = [scheduledState] ∧
    target.machine.initialStates startedSetup = [startedState] ∧
    target.machine.steps scheduledState startAction = [startedResult] ∧
    target.machine.steps startedState cancelAction = [canceledResult] ∧
    target.machine.steps startedState reportSuccessAction = [succeededResult] := by
  native_decide

theorem targetKernelRejectsInvalidAndTerminalTransitions : target.machine.initialStates [] = [] ∧
    target.machine.steps scheduledState cancelAction = [] ∧
    target.machine.steps scheduledState reportSuccessAction = [] ∧
    target.machine.steps startedState startAction = [] ∧
    target.machine.steps canceledState startAction = [] ∧
    target.machine.steps canceledState cancelAction = [] ∧
    target.machine.steps canceledState reportSuccessAction = [] ∧
    target.machine.steps succeededState startAction = [] ∧
    target.machine.steps succeededState cancelAction = [] ∧
    target.machine.steps succeededState reportSuccessAction = [] := by
  native_decide

theorem finitePlanningEnumeratesTheExposedActionDomain : match target.planning with
    | .unavailable => False
    | .available capability =>
        capability.actions = [cancelAction, startAction, reportSuccessAction] := by
  change finiteMachine.actions = [cancelAction, startAction, reportSuccessAction]
  rfl

def expertModelSpec : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := modelSpec.id
  source := modelSpec.source
  definitions := modelSpec.definitions
  requiredCapabilities := modelSpec.requiredCapabilities
  providers := [lifecycleProvider]
  connectors := []
  resolvedSetups := modelSpec.resolvedSetups
  machine := modelSpec.machine
}

def missingProviderDeclaration : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertModelSpec with providers := []
}

/-- Checked composition remains public so callers can inspect its typed Definition Error. -/
private def compositionError (result : Except DefinitionError α) : Option DefinitionError :=
  match result with
  | .error failure => some failure
  | .ok _ => none

theorem missingProviderReportsCompositionError :
    compositionError ((checkModel (DraftModel.make missingProviderDeclaration) |>.mapError LocatedError.error)) = some {
      kind := .missingProvider
      definitionId := targetId
      sourcePath := "Temporal/Feature/Nexus/Lifecycle.lean"
      offendingValue := lifecycleCapabilityId.value
      relatedDefinitionIds := [lifecycleCapabilityId]
    } := by
  native_decide

def conflictingProviderId : DefinitionId := DefinitionId.of
  "temporal.nexus.basic-lifecycle.provider.conflicting"

def conflictingProvider : Provider LawStatement := {
  id := conflictingProviderId
  source
  contract := lifecycleProvider.contract
  meanings := [{
    definitionId := operationStateId
    kind := .state
    behaviorVersion := "temporal-nexus-basic-lifecycle-state/conflicting"
  }]
  lawProofs := lifecycleProvider.lawProofs
}

def conflictingProviderMetadata : DefinitionMetadata := {
  id := conflictingProviderId
  kind := .provider
  source
  behaviorVersion := "temporal-nexus-basic-lifecycle-provider/conflicting"
}

def conflictingProviderDeclaration : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertModelSpec with
  definitions := conflictingProviderMetadata :: expertModelSpec.definitions
  providers := [lifecycleProvider, conflictingProvider]
}

theorem conflictingProvidersReportCompositionError :
    compositionError ((checkModel (DraftModel.make conflictingProviderDeclaration) |>.mapError LocatedError.error)) = some {
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
