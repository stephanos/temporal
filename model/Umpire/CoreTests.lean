import Umpire.Core

namespace Umpire.CoreTests

open Umpire

def id (value : String) : DeclarationId := DeclarationId.of value

def source (path : String) : SemanticSource := {
  path
  line := 1
  column := 1
  provenance := "lean-test"
}

def metadata
    (value : String)
    (kind : DeclarationKind)
    (digest : String := "contract-v1") : DeclarationMetadata := {
  id := id value
  kind
  source := source "Umpire/CoreTests.lean"
  contractDigest := digest
}

def providerLaw : LawRequirement := {
  id := id "umpire.law.provider-sound"
  semanticDigest := "provider-sound/v1"
}

def connectorLaw : LawRequirement := {
  id := id "umpire.law.connector-sound"
  semanticDigest := "connector-sound/v1"
}

def TestLawStatement (lawId : DeclarationId) : Prop :=
  lawId = providerLaw.id ∨ lawId = connectorLaw.id

def witness
    (requirement : LawRequirement)
    (proof : TestLawStatement requirement.id) : LawWitness TestLawStatement := {
  requirement
  proof
}

def transition (state action : Bool) : TransitionResult Bool Bool Bool := {
  modelOutcome := action
  resultingState := action
  observations := [state]
}

def testKernel : TransitionKernel Unit Bool Bool Bool Bool := {
  metadata := {
    id := id "test.kernel.transition"
    contractDigest := "test-kernel/v1"
    source := source "Umpire/CoreTests.lean"
  }
  initialStates := fun _ => [false]
  authoritativeInitial := fun _ state => state = false
  initialSound := by simp
  initialComplete := by simp_all
  steps := fun state action => [transition state action]
  authoritativeStep := fun state action result => result = transition state action
  stepSound := by simp
  stepComplete := by simp_all
}

def primaryProvider : CapabilityProvider TestLawStatement := {
  id := id "test.provider.primary"
  source := source "Test/PrimarySemantic.lean"
  contract := {
    id := id "test.capability.primary"
    semanticDigest := "test-primary-capability/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    declaration := id "test.relation.shared"
    kind := .relation
    semanticDigest := "test-primary-shared/v1"
  }]
  lawWitnesses := [witness providerLaw (by exact .inl rfl)]
}

def secondaryProvider : CapabilityProvider TestLawStatement := {
  id := id "test.provider.secondary"
  source := source "Test/SecondarySemantic.lean"
  contract := {
    id := id "test.capability.secondary"
    semanticDigest := "test-secondary-capability/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    declaration := id "test.relation.shared"
    kind := .relation
    semanticDigest := "test-secondary-shared/v1"
  }]
  lawWitnesses := [witness providerLaw (by exact .inl rfl)]
}

def ownershipConnector : CapabilityConnector TestLawStatement := {
  id := id "test.connector.shared"
  source := source "Test/CompositeSemantic.lean"
  semanticDigest := "test-shared-connector/v1"
  reconciliations := [{
    declaration := id "test.relation.shared"
    kind := .relation
    providers := [primaryProvider.id, secondaryProvider.id]
    semanticDigest := "test-shared-connector/reconciled-v1"
  }]
  requiredLaws := [connectorLaw]
  lawWitnesses := [witness connectorLaw (by exact .inr rfl)]
}

def testDeclarations : List DeclarationMetadata := [
  metadata "test.target.composed" .target,
  metadata "test.kernel.transition" .kernel,
  metadata "test.capability.primary" .capability,
  metadata "test.capability.secondary" .capability,
  metadata "test.provider.primary" .provider,
  metadata "test.provider.secondary" .provider,
  metadata "umpire.law.provider-sound" .law providerLaw.semanticDigest,
  metadata "umpire.law.connector-sound" .law connectorLaw.semanticDigest,
  metadata "test.connector.shared" .connector,
  metadata "test.relation.shared" .relation,
  metadata "test.action.request" .action,
  metadata "test.observation.completed" .observation
]

def testTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  id := id "test.target.composed"
  source := source "Test/CompositeSemantic.lean"
  declarations := testDeclarations
  requiredCapabilities := [
    id "test.capability.primary",
    id "test.capability.secondary"
  ]
  providers := [primaryProvider, secondaryProvider]
  connectors := [ownershipConnector]
  resolvedSetups := [()]
  kernel := .checked testKernel
}

example : (composeTarget testTarget).isOk = true := by
  native_decide

def switchKernel : TransitionKernel Unit Bool Bool Bool Bool := {
  testKernel with
  metadata := {
    id := id "switch.kernel.transition"
    contractDigest := "switch-kernel/v1"
    source := source "SwitchSemantic.lean"
  }
}

def switchProvider : CapabilityProvider TestLawStatement := {
  id := id "switch.provider.toggle"
  source := source "SwitchSemantic.lean"
  contract := {
    id := id "switch.capability.toggle"
    semanticDigest := "switch-toggle/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    declaration := id "switch.action.toggle"
    kind := .action
    semanticDigest := "switch-action/v1"
  }]
  lawWitnesses := [witness providerLaw (by exact .inl rfl)]
}

def switchTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  id := id "switch.target.two-state"
  source := source "SwitchSemantic.lean"
  declarations := [
    metadata "switch.target.two-state" .target,
    metadata "switch.kernel.transition" .kernel,
    metadata "switch.capability.toggle" .capability,
    metadata "switch.provider.toggle" .provider,
    metadata "switch.action.toggle" .action,
    metadata "umpire.law.provider-sound" .law providerLaw.semanticDigest
  ]
  requiredCapabilities := [id "switch.capability.toggle"]
  providers := [switchProvider]
  connectors := []
  resolvedSetups := [()]
  kernel := .checked switchKernel
}

-- A second model with unrelated vocabulary composes through the exact same public interface.
example : (composeTarget switchTarget).isOk = true := by
  native_decide

def errorOf {Target : Type}
    (result : Except DeclarationError Target) : Option DeclarationError :=
  match result with
  | .error error => some error
  | .ok _ => none

def emptyIdentityTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := metadata "" .action :: testDeclarations
}

example : (errorOf (composeTarget emptyIdentityTarget)) = some {
    kind := .emptyIdentity
    declarationId := id "umpire.declaration.anonymous"
    sourcePath := "Umpire/CoreTests.lean"
    offendingValue := "<empty>"
    relatedIdentities := [id ""]
  } := by
  native_decide

def duplicateIdentityTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := metadata "test.target.composed" .target :: testDeclarations
}

example : (errorOf (composeTarget duplicateIdentityTarget)).map DeclarationError.kind =
    some .duplicateIdentity := by
  native_decide

def unknownIdentityTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  requiredCapabilities := [id "test.capability.missing"]
}

example : (errorOf (composeTarget unknownIdentityTarget)) = some {
    kind := .unknownIdentity
    declarationId := testTarget.id
    sourcePath := "Test/CompositeSemantic.lean"
    offendingValue := "test.capability.missing"
    relatedIdentities := [id "test.capability.missing"]
  } := by
  native_decide

def wrongKindTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  requiredCapabilities := [id "test.action.request"]
}

example : (errorOf (composeTarget wrongKindTarget)).map DeclarationError.kind = some .wrongKind := by
  native_decide

def missingLawProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with lawWitnesses := []
}

def missingLawTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [missingLawProvider, secondaryProvider]
}

example : (errorOf (composeTarget missingLawTarget)).map DeclarationError.kind = some .missingLaw := by
  native_decide

def staleWitnessProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with requiredLaws := [] }
}

def staleWitnessTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [staleWitnessProvider, secondaryProvider]
}

example : (errorOf (composeTarget staleWitnessTarget)).map DeclarationError.kind =
    some .unexpectedLaw := by
  native_decide

def missingProviderTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [primaryProvider]
}

example : (errorOf (composeTarget missingProviderTarget)).map DeclarationError.kind =
    some .missingProvider := by
  native_decide

def conflictingTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with connectors := []
}

example : (errorOf (composeTarget conflictingTarget)).map DeclarationError.kind =
    some .conflictingProviders := by
  native_decide

def secondOwnershipConnector : CapabilityConnector TestLawStatement := {
  ownershipConnector with
  id := id "test.connector.alternate-shared"
  source := source "Test/AlternateCompositeSemantic.lean"
}

def ambiguousConnectorTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := metadata "test.connector.alternate-shared" .connector :: testDeclarations
  connectors := [secondOwnershipConnector, ownershipConnector]
}

example : (errorOf (composeTarget ambiguousConnectorTarget)).map DeclarationError.kind =
    some .ambiguousConnector := by
  native_decide

def inactiveProviderConnector : CapabilityConnector TestLawStatement := {
  ownershipConnector with
  id := id "test.connector.inactive-provider"
  reconciliations := [{
    declaration := id "test.relation.shared"
    kind := .relation
    providers := [primaryProvider.id, id "test.provider.inactive"]
    semanticDigest := "inactive-provider/reconciled-v1"
  }]
}

def inactiveProviderTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := [
    metadata "test.connector.inactive-provider" .connector,
    metadata "test.provider.inactive" .provider
  ] ++ testDeclarations
  connectors := [inactiveProviderConnector]
}

example : (errorOf (composeTarget inactiveProviderTarget)).map DeclarationError.kind =
    some .missingProvider := by
  native_decide

example : [id ".", id ".action", id "action.", id "test..action"].all
    (fun identity => !identity.isNamespaced) = true := by
  native_decide

def compatibleSecondaryProvider : CapabilityProvider TestLawStatement := {
  secondaryProvider with
  meanings := [{
    declaration := id "test.relation.shared"
    kind := .relation
    semanticDigest := "test-primary-shared/v1"
  }]
}

def compatibleTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  providers := [primaryProvider, compatibleSecondaryProvider]
  connectors := []
}

example : (composeTarget compatibleTarget).isOk = true := by
  native_decide

def incompleteKernelTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  kernel := .incomplete testKernel.metadata [
    id "umpire.kernel-proof.initial-complete",
    id "umpire.kernel-proof.step-sound"
  ]
}

example : (errorOf (composeTarget incompleteKernelTarget)) = some {
    kind := .incompleteKernel
    declarationId := testTarget.id
    sourcePath := "Umpire/CoreTests.lean"
    offendingValue := testKernel.metadata.id.value
    relatedIdentities := [
      id "umpire.kernel-proof.initial-complete",
      id "umpire.kernel-proof.step-sound"
    ]
  } := by
  native_decide

-- An emitted step outside the authoritative relation cannot inhabit a checked kernel proof.
def outsideRelation : TransitionResult Bool Bool Bool := {
  modelOutcome := false
  resultingState := false
  observations := [true]
}

example : ¬testKernel.authoritativeStep false true outsideRelation := by
  simp [testKernel, outsideRelation, transition]

example (result : TransitionResult Bool Bool Bool)
    (member : result ∈ testKernel.steps false true) :
    testKernel.authoritativeStep false true result :=
  testKernel.stepSound false true result member

def reorderedTestTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := testTarget.declarations.reverse
  requiredCapabilities := testTarget.requiredCapabilities.reverse
  providers := testTarget.providers.reverse
  connectors := testTarget.connectors.reverse
}

example : (composeTarget reorderedTestTarget).toOption.map CheckedTarget.canonicalMetadata =
    (composeTarget testTarget).toOption.map CheckedTarget.canonicalMetadata := by
  native_decide

example : (composeTarget reorderedTestTarget).toOption.map CheckedTarget.semanticDigest =
    (composeTarget testTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def reorderedConflictTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  conflictingTarget with
  declarations := conflictingTarget.declarations.reverse
  providers := conflictingTarget.providers.reverse
}

example : (errorOf (composeTarget reorderedConflictTarget)).map canonicalDeclarationErrorJson =
    (errorOf (composeTarget conflictingTarget)).map canonicalDeclarationErrorJson := by
  native_decide

def changedIdentityTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  id := id "test.target.composed-v2"
  declarations := metadata "test.target.composed-v2" .target ::
    testDeclarations.filter (fun declaration => declaration.id != testTarget.id)
}

example : (composeTarget changedIdentityTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget testTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def changedContractProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with semanticDigest := "test-primary-capability/v2" }
}

def changedContractTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [changedContractProvider, secondaryProvider]
}

example : (composeTarget changedContractTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget testTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def changedConnector : CapabilityConnector TestLawStatement := {
  ownershipConnector with semanticDigest := "test-shared-connector/v2"
}

def changedConnectorTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with connectors := [changedConnector]
}

example : (composeTarget changedConnectorTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget testTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def changedKernel : TransitionKernel Unit Bool Bool Bool Bool := {
  testKernel with
  metadata := { testKernel.metadata with contractDigest := "test-kernel/v2" }
}

def changedKernelTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with kernel := .checked changedKernel
}

example : (composeTarget changedKernelTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget testTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def changedLaw : LawRequirement := {
  providerLaw with semanticDigest := "provider-sound/v2"
}

def changedLawProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with requiredLaws := [changedLaw] }
  lawWitnesses := [witness changedLaw (by exact .inl rfl)]
}

def changedLawSecondaryProvider : CapabilityProvider TestLawStatement := {
  secondaryProvider with
  contract := { secondaryProvider.contract with requiredLaws := [changedLaw] }
  lawWitnesses := [witness changedLaw (by exact .inl rfl)]
}

def changedLawTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := testDeclarations.map fun declaration =>
    if declaration.id == providerLaw.id then
      { declaration with contractDigest := changedLaw.semanticDigest }
    else
      declaration
  providers := [changedLawProvider, changedLawSecondaryProvider]
}

example : (composeTarget changedLawTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget testTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def lawlessPrimaryProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with requiredLaws := [] }
  lawWitnesses := []
}

def lawlessSecondaryProvider : CapabilityProvider TestLawStatement := {
  secondaryProvider with
  contract := { secondaryProvider.contract with requiredLaws := [] }
  lawWitnesses := []
}

def lawlessTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [lawlessPrimaryProvider, lawlessSecondaryProvider]
}

example : (composeTarget lawlessTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget testTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def documentedTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := testDeclarations.map fun declaration =>
    if declaration.id == testTarget.id then
      { declaration with documentation := "Non-semantic explanatory text." }
    else
      declaration
}

example : (composeTarget documentedTarget).toOption.map CheckedTarget.semanticDigest =
    (composeTarget testTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

example : (composeTarget documentedTarget).toOption.map CheckedTarget.canonicalMetadata ≠
    (composeTarget testTarget).toOption.map CheckedTarget.canonicalMetadata := by
  native_decide

def exactTrace : SemanticTrace Bool Bool Bool SemanticValue := {
  initialState := false
  steps := [{
    selectedAction := true
    modelOutcome := true
    resultingState := true
    observations := [{
      identity := id "switch.observation.enabled"
      value := "enabled"
    }]
  }]
}

example : exactTrace.initialState = false ∧
    exactTrace.steps.map SemanticTraceStep.selectedAction = [true] ∧
    exactTrace.steps.map SemanticTraceStep.modelOutcome = [true] ∧
    exactTrace.steps.map SemanticTraceStep.resultingState = [true] ∧
    exactTrace.steps.flatMap SemanticTraceStep.observations = [{
      identity := id "switch.observation.enabled"
      value := "enabled"
    }] := by
  native_decide

example : canonicalCapabilityProviderJson primaryProvider =
    canonicalCapabilityProviderJson primaryProvider := by
  rfl

example : canonicalCapabilityConnectorJson ownershipConnector =
    canonicalCapabilityConnectorJson ownershipConnector := by
  rfl

end Umpire.CoreTests
