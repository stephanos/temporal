import Umpire.Target.Tests.Fixtures

/-! Definition, capability, provider, connector, and law validation checks. -/

namespace Umpire.TargetTests

open Umpire

def emptyDefinitionIdTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := metadata "" .action :: testDefinitions
}

example : (errorOf (composeTarget emptyDefinitionIdTarget)) = some {
    kind := .emptyDefinitionId
    definitionId := id "umpire.definition.anonymous"
    sourcePath := "Umpire/TargetTests.lean"
    offendingValue := "<empty>"
    relatedDefinitionIds := [id ""]
  } := by
  native_decide

def duplicateDefinitionIdTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := metadata "test.target.composed" .target :: testDefinitions
}

example : (errorOf (composeTarget duplicateDefinitionIdTarget)).map DefinitionError.kind =
    some .duplicateDefinitionId := by
  native_decide

def unknownDefinitionIdTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  requiredCapabilities := [id "test.capability.missing"]
}

example : (errorOf (composeTarget unknownDefinitionIdTarget)) = some {
    kind := .unknownDefinitionId
    definitionId := testTarget.id
    sourcePath := "Test/CompositeSemantic.lean"
    offendingValue := "test.capability.missing"
    relatedDefinitionIds := [id "test.capability.missing"]
  } := by
  native_decide

def wrongKindTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  requiredCapabilities := [id "test.action.request"]
}

example : (errorOf (composeTarget wrongKindTarget)).map DefinitionError.kind = some .wrongKind := by
  native_decide

def missingLawProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with lawWitnesses := []
}

def missingLawTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [missingLawProvider, secondaryProvider]
}

example : (errorOf (composeTarget missingLawTarget)).map DefinitionError.kind = some .missingLaw := by
  native_decide

def staleWitnessProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with requiredLaws := [] }
}

def staleWitnessTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [staleWitnessProvider, secondaryProvider]
}

example : (errorOf (composeTarget staleWitnessTarget)).map DefinitionError.kind =
    some .unexpectedLaw := by
  native_decide

def missingProviderTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [primaryProvider]
}

example : (errorOf (composeTarget missingProviderTarget)).map DefinitionError.kind =
    some .missingProvider := by
  native_decide

example : (errorOf (composeTarget conflictingTarget)).map DefinitionError.kind =
    some .conflictingProviders := by
  native_decide

def secondOwnershipConnector : CapabilityConnector TestLawStatement := {
  ownershipConnector with
  id := id "test.connector.alternate-shared"
  source := source "Test/AlternateCompositeSemantic.lean"
}

def ambiguousConnectorTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := metadata "test.connector.alternate-shared" .connector :: testDefinitions
  connectors := [secondOwnershipConnector, ownershipConnector]
}

example : (errorOf (composeTarget ambiguousConnectorTarget)).map DefinitionError.kind =
    some .ambiguousConnector := by
  native_decide

def inactiveProviderConnector : CapabilityConnector TestLawStatement := {
  ownershipConnector with
  id := id "test.connector.inactive-provider"
  reconciliations := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    providers := [primaryProvider.id, id "test.provider.inactive"]
    canonicalBehavior := "inactive-provider/reconciled-v1"
  }]
}

def inactiveProviderTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := [
    metadata "test.connector.inactive-provider" .connector,
    metadata "test.provider.inactive" .provider
  ] ++ testDefinitions
  connectors := [inactiveProviderConnector]
}

example : (errorOf (composeTarget inactiveProviderTarget)).map DefinitionError.kind =
    some .missingProvider := by
  native_decide

example : [id ".", id ".action", id "action.", id "test..action"].all
    (fun definitionId => !definitionId.isNamespaced) = true := by
  native_decide

def compatibleSecondaryProvider : CapabilityProvider TestLawStatement := {
  secondaryProvider with
  meanings := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    canonicalBehavior := "test-primary-shared/v1"
  }]
}

def compatibleTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  providers := [primaryProvider, compatibleSecondaryProvider]
  connectors := []
}

example : (composeTarget compatibleTarget).isOk = true := by
  native_decide

def collidingStateEncodingKernel : TransitionKernel Unit Bool Bool Bool Bool := {
  testKernel with
  behaviorDomain := match testKernel.behaviorDomain with
    | .complete domain => .complete { domain with encodeState := fun _ => "state" }
    | .missing => .missing
    | .incomplete missing => .incomplete missing
}

def collidingStateEncodingTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with kernel := .checked collidingStateEncodingKernel
}

/-- A complete finite domain still fails closed when its canonical encoder collapses values. -/
example : (errorOf (composeTarget collidingStateEncodingTarget)) = some {
    kind := .incompleteBehaviorDomain
    definitionId := testTarget.id
    sourcePath := "Umpire/TargetTests.lean"
    offendingValue := "state-encoding"
    relatedDefinitionIds := [testKernel.metadata.id]
  } := by
  native_decide

end Umpire.TargetTests
