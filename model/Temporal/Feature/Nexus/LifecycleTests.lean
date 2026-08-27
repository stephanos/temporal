import Temporal.Feature.Nexus.Lifecycle

namespace Temporal.Feature.Nexus.LifecycleTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle

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

example : targetResult.isOk = true ∧
    target.requiredCapabilities = [lifecycleCapabilityId] ∧
    target.providers.map CapabilityProvider.id = [lifecycleProviderId] ∧
    target.connectors = [] := by
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

example : completeness.roleAssignments = [scheduledSetup, startedSetup] ∧
    completeness.actions = [cancelAction, startAction, reportSuccessAction] ∧
    incrementalKernel.actionLimit = 3 ∧
    incrementalKernel.actionAt 0 = some cancelAction ∧
    incrementalKernel.actionAt 1 = some startAction ∧
    incrementalKernel.actionAt 2 = some reportSuccessAction ∧
    incrementalKernel.actionAt 3 = none := by
  native_decide

example : incrementalKernel.initialAt scheduledSetup 0 = some scheduledState ∧
    incrementalKernel.initialAt startedSetup 0 = some startedState ∧
    incrementalKernel.stepAt scheduledState startAction 0 = some startedResult ∧
    incrementalKernel.stepAt startedState cancelAction 0 = some canceledResult ∧
    incrementalKernel.stepAt startedState reportSuccessAction 0 = some succeededResult ∧
    incrementalKernel.stepAt startedState startAction 0 = none := by
  native_decide

def missingProviderDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with providers := []
}

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

end Temporal.Feature.Nexus.LifecycleTests
