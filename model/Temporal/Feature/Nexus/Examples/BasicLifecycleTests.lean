import Temporal.Feature.Nexus.Examples.BasicLifecycle

namespace Temporal.Feature.Nexus.Examples.BasicLifecycleTests

open Umpire
open Temporal.Feature.Nexus.Examples.BasicLifecycle

example : AutoClose.step .scheduled .start = some .started ∧
    AutoClose.step .started .succeed = some .succeeded := by
  exact ⟨rfl, rfl⟩

example : targetResult.isOk = true ∧
    target.requiredCapabilities = [lifecycleCapabilityId] ∧
    target.providers.map CapabilityProvider.id = [lifecycleProviderId] ∧
    target.connectors = [] := by
  native_decide

example : target.kernel.initialStates scheduledSetup = [scheduledState] ∧
    target.kernel.initialStates startedSetup = [startedState] ∧
    target.kernel.steps scheduledState startAction = [startedResult] ∧
    target.kernel.steps startedState reportSuccessAction = [succeededResult] := by
  native_decide

example : target.kernel.steps startedState startAction = [] ∧
    target.kernel.steps succeededState reportSuccessAction = [] := by
  native_decide

example : completeness.roleAssignments = [scheduledSetup, startedSetup] ∧
    completeness.actions = [startAction, reportSuccessAction] ∧
    incrementalKernel.actionLimit = 2 ∧
    incrementalKernel.actionAt 0 = some startAction ∧
    incrementalKernel.actionAt 1 = some reportSuccessAction ∧
    incrementalKernel.actionAt 2 = none := by
  native_decide

example : incrementalKernel.initialAt scheduledSetup 0 = some scheduledState ∧
    incrementalKernel.initialAt startedSetup 0 = some startedState ∧
    incrementalKernel.stepAt scheduledState startAction 0 = some startedResult ∧
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

end Temporal.Feature.Nexus.Examples.BasicLifecycleTests
