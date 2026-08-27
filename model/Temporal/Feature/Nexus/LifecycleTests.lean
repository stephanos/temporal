import Temporal.Feature.Nexus.Lifecycle

namespace Temporal.Feature.Nexus.LifecycleTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle

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
    (checked.id, checked.source, canonicalCheckedTargetJson checked, checked.semanticDigest)) =
    some (targetId, source, canonicalCheckedTargetJson target, target.semanticDigest) := by
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
        capability.actions = [cancelAction, startAction, reportSuccessAction] ∧
        capability.roleDomainDigest = "temporal-nexus-basic-lifecycle-role-domain/v1" ∧
        capability.actionDomainDigest = "temporal-nexus-basic-lifecycle-action-domain/v2" := by
  simp [target, checkedTarget, targetAuthoring, finitePlanning, actionDomain]

def missingProviderDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with providers := []
}

/-- Checked composition remains public so callers can inspect its typed declaration error. -/
private def compositionErrorKind (result : Except DeclarationError α) :
    Option DeclarationErrorKind :=
  match result with
  | .error failure => some failure.kind
  | .ok _ => none

example : compositionErrorKind (composeTarget missingProviderDeclaration) =
    some .missingProvider := by
  native_decide

def conflictingProviderId : DeclarationId := DeclarationId.of
  "temporal.nexus.basic-lifecycle.provider.conflicting"

def conflictingProvider : CapabilityProvider LawStatement := {
  id := conflictingProviderId
  source
  contract := lifecycleProvider.contract
  meanings := [{
    declaration := operationStateId
    kind := .state
    semanticDigest := "temporal-nexus-basic-lifecycle-state/conflicting"
  }]
  lawWitnesses := lifecycleProvider.lawWitnesses
}

def conflictingProviderMetadata : DeclarationMetadata := {
  id := conflictingProviderId
  kind := .provider
  source
  contractDigest := "temporal-nexus-basic-lifecycle-provider/conflicting"
}

def conflictingProviderDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with
  declarations := conflictingProviderMetadata :: targetDeclaration.declarations
  providers := [lifecycleProvider, conflictingProvider]
}

example : compositionErrorKind (composeTarget conflictingProviderDeclaration) =
    some .conflictingProviders := by
  native_decide

example : compatibilityTargetAuthors = ["nexus-lifecycle"] := by
  rfl

end Temporal.Feature.Nexus.LifecycleTests
