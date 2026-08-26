import Umpire.Target.Tests.Fixtures

/-! Declaration, capability, provider, connector, and law validation checks. -/

namespace Umpire.TargetTests

open Umpire

def emptyIdentityTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := metadata "" .action :: testDeclarations
}

example : (errorOf (composeTarget emptyIdentityTarget)) = some {
    kind := .emptyIdentity
    declarationId := id "umpire.declaration.anonymous"
    sourcePath := "Umpire/TargetTests.lean"
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

end Umpire.TargetTests
