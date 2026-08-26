import Umpire.Target.Tests.Fixtures

/-! Canonical ordering, digest sensitivity, documentation, and serializer checks. -/

namespace Umpire.TargetTests

open Umpire

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

example : canonicalCapabilityProviderJson primaryProvider =
    canonicalCapabilityProviderJson primaryProvider := by
  rfl

example : canonicalCapabilityConnectorJson ownershipConnector =
    canonicalCapabilityConnectorJson ownershipConnector := by
  rfl

end Umpire.TargetTests
