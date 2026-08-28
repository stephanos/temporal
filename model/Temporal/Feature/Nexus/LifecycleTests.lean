import Temporal.Feature.Nexus.Lifecycle

namespace Temporal.Feature.Nexus.LifecycleTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle

#check (Temporal.Feature.Nexus.Lifecycle.OperationState : Type)
#check (Temporal.Feature.Nexus.Lifecycle.OperationEvent : Type)
#check (Temporal.Feature.Nexus.Lifecycle.step : OperationState → OperationEvent → Option OperationState)
#check (Temporal.Feature.Nexus.Lifecycle.source : SourceLocation)
#check (Temporal.Feature.Nexus.Lifecycle.definitions : List DefinitionMetadata)
#check (Temporal.Feature.Nexus.Lifecycle.target : QueryTarget LawStatement)

example : source = {
    path := "Temporal/Feature/Nexus/Lifecycle.lean"
    line := 1
    column := 1
    provenance := "lean-model"
  } := by
  native_decide

example : definitions = [
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

example : step .scheduled .start = some .started ∧
    step .started .cancel = some .canceled ∧
    step .started .succeed = some .succeeded := by
  exact ⟨rfl, rfl, rfl⟩

example : step .scheduled .cancel = none ∧
    step .scheduled .succeed = none ∧
    step .started .start = none := by
  exact ⟨rfl, rfl, rfl⟩

example : step .canceled .start = none ∧
    step .canceled .cancel = none ∧
    step .canceled .succeed = none ∧
    step .succeeded .start = none ∧
    step .succeeded .cancel = none ∧
    step .succeeded .succeed = none := by
  exact ⟨rfl, rfl, rfl, rfl, rfl, rfl⟩

example : (checkTarget targetAuthoring).isOk = true ∧
    target.requiredCapabilities = [lifecycleCapabilityId] ∧
    target.providers.map CapabilityProvider.id = [lifecycleProviderId] ∧
    target.connectors = [] := by
  native_decide

example : (checkTarget targetAuthoring).toOption.map (fun checked =>
    (checked.id, checked.source, canonicalCheckedTargetJson checked, checked.behaviorFingerprint)) =
    some (targetId, source, canonicalCheckedTargetJson target, target.behaviorFingerprint) := by
  native_decide

example : targetId.value = "temporal.nexus.basic-lifecycle.target" ∧
    kernelId.value = "temporal.nexus.basic-lifecycle.kernel" ∧
    operationRoleId.value = "temporal.nexus.basic-lifecycle.role.operation" ∧
    startActionId.value = "temporal.nexus.basic-lifecycle.action.start" ∧
    cancelActionId.value = "temporal.nexus.basic-lifecycle.action.cancel" ∧
    reportSuccessActionId.value = "temporal.nexus.basic-lifecycle.action.succeed" := by
  native_decide

example : target.kernel.initialStates scheduledSetup = [scheduledState] ∧
    target.kernel.initialStates startedSetup = [startedState] ∧
    target.kernel.steps scheduledState startAction = [startedResult] ∧
    target.kernel.steps startedState cancelAction = [canceledResult] ∧
    target.kernel.steps startedState reportSuccessAction = [succeededResult] := by
  native_decide

example : target.kernel.initialStates [] = [] ∧
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

example : match target.planning with
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

example : compositionErrorKind (composeTarget missingProviderDeclaration) =
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

example : compositionErrorKind (composeTarget conflictingProviderDeclaration) =
    some .conflictingProviders := by
  native_decide

example : compatibilityTargetAuthors = ["nexus-lifecycle"] := by
  rfl

end Temporal.Feature.Nexus.LifecycleTests
