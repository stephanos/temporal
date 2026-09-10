import Umpire.Model.Tests.Fixtures

/-! Definition, capability, provider, connector, and law validation checks. -/

namespace Umpire.ModelTests

open Umpire

def emptyDefinitionIdTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := metadata "" .action :: testDefinitions
}

example : (errorOf ((checkModel (DraftModel.make emptyDefinitionIdTarget) |>.mapError LocatedError.error))) = some {
    kind := .emptyDefinitionId
    definitionId := id "umpire.definition.anonymous"
    sourcePath := "Umpire/ModelTests.lean"
    offendingValue := "<empty>"
    relatedDefinitionIds := [id ""]
  } := by
  native_decide

def duplicateDefinitionIdTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := metadata "test.target.composed" .target :: testDefinitions
}

example : (errorOf ((checkModel (DraftModel.make duplicateDefinitionIdTarget) |>.mapError LocatedError.error))).map DefinitionError.kind =
    some .duplicateDefinitionId := by
  native_decide

def unknownDefinitionIdTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  requiredCapabilities := [id "test.capability.missing"]
}

example : (errorOf ((checkModel (DraftModel.make unknownDefinitionIdTarget) |>.mapError LocatedError.error))) = some {
    kind := .unknownDefinitionId
    definitionId := testTarget.id
    sourcePath := "Test/CompositeSemantic.lean"
    offendingValue := "test.capability.missing"
    relatedDefinitionIds := [id "test.capability.missing"]
  } := by
  native_decide

def wrongKindTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  requiredCapabilities := [id "test.action.request"]
}

example : (errorOf ((checkModel (DraftModel.make wrongKindTarget) |>.mapError LocatedError.error))).map DefinitionError.kind = some .wrongKind := by
  native_decide

def missingLawProvider : Provider TestLawStatement := {
  primaryProvider with lawProofs := []
}

def missingLawTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [missingLawProvider, secondaryProvider]
}

example : (errorOf ((checkModel (DraftModel.make missingLawTarget) |>.mapError LocatedError.error))).map DefinitionError.kind = some .missingLaw := by
  native_decide

def staleWitnessProvider : Provider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with requiredLaws := [] }
}

def staleWitnessTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [staleWitnessProvider, secondaryProvider]
}

example : (errorOf ((checkModel (DraftModel.make staleWitnessTarget) |>.mapError LocatedError.error))).map DefinitionError.kind =
    some .unexpectedLaw := by
  native_decide

def missingProviderTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [primaryProvider]
}

example : (errorOf ((checkModel (DraftModel.make missingProviderTarget) |>.mapError LocatedError.error))).map DefinitionError.kind =
    some .missingProvider := by
  native_decide

example : (errorOf ((checkModel (DraftModel.make conflictingTarget) |>.mapError LocatedError.error))).map DefinitionError.kind =
    some .conflictingProviders := by
  native_decide

def secondOwnershipConnector : Connector TestLawStatement := {
  ownershipConnector with
  id := id "test.connector.alternate-shared"
  source := source "Test/AlternateCompositeSemantic.lean"
}

def ambiguousConnectorTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := metadata "test.connector.alternate-shared" .connector :: testDefinitions
  connectors := [secondOwnershipConnector, ownershipConnector]
}

example : (errorOf ((checkModel (DraftModel.make ambiguousConnectorTarget) |>.mapError LocatedError.error))).map DefinitionError.kind =
    some .ambiguousConnector := by
  native_decide

def inactiveProviderConnector : Connector TestLawStatement := {
  ownershipConnector with
  id := id "test.connector.inactive-provider"
  reconciliations := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    providers := [primaryProvider.id, id "test.provider.inactive"]
    behaviorVersion := "inactive-provider/reconciled-v1"
  }]
}

def inactiveProviderTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := [
    metadata "test.connector.inactive-provider" .connector,
    metadata "test.provider.inactive" .provider
  ] ++ testDefinitions
  connectors := [inactiveProviderConnector]
}

example : (errorOf ((checkModel (DraftModel.make inactiveProviderTarget) |>.mapError LocatedError.error))).map DefinitionError.kind =
    some .missingProvider := by
  native_decide

example : [id ".", id ".action", id "action.", id "test..action"].all
    (fun definitionId => !definitionId.isNamespaced) = true := by
  native_decide

def compatibleSecondaryProvider : Provider TestLawStatement := {
  secondaryProvider with
  meanings := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    behaviorVersion := "test-primary-shared/v1"
  }]
}

def compatibleTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  providers := [primaryProvider, compatibleSecondaryProvider]
  connectors := []
}

example : ((checkModel (DraftModel.make compatibleTarget) |>.mapError LocatedError.error)).isOk = true := by
  native_decide

def collidingStateEncodingKernel : Machine Unit Bool Bool Bool Bool := {
  testKernel with
  vocabulary := match testKernel.vocabulary with
    | .complete domain => .complete { domain with encodeState := fun _ => "state" }
    | .missing => .missing
    | .incomplete missing => .incomplete missing
}

def collidingStateEncodingTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with machine := .checked collidingStateEncodingKernel
}

/-- A complete finite domain still fails closed when its canonical encoder collapses values. -/
example : (errorOf ((checkModel (DraftModel.make collidingStateEncodingTarget) |>.mapError LocatedError.error))) = some {
    kind := .incompleteVocabulary
    definitionId := testTarget.id
    sourcePath := "Umpire/ModelTests.lean"
    offendingValue := "state-encoding"
    relatedDefinitionIds := [testKernel.metadata.id]
  } := by
  native_decide

def changedStateEncodingKernel : Machine Unit Bool Bool Bool Bool := {
  testKernel with
  vocabulary := match testKernel.vocabulary with
    | .complete domain => .complete {
        domain with encodeState := fun state => "state:" ++ toString state
      }
    | .missing => .missing
    | .incomplete missing => .incomplete missing
}

def checkedTestTarget : CheckedModel TestLawStatement Unit Bool Bool Bool Bool :=
  model (authoringOf testTarget)

/-- Equivalent relations cannot rebind a checked Target when its domain projection changes. -/
example :
    changedStateEncodingKernel.metadata = testKernel.metadata ∧
    changedStateEncodingKernel.setupDomain = testKernel.setupDomain ∧
    changedStateEncodingKernel.stateDomain = testKernel.stateDomain ∧
    changedStateEncodingKernel.actionDomain = testKernel.actionDomain ∧
    changedStateEncodingKernel.outcomeDomain = testKernel.outcomeDomain ∧
    changedStateEncodingKernel.observationDomain = testKernel.observationDomain ∧
    changedStateEncodingKernel.authoritativeInitial = testKernel.authoritativeInitial ∧
    changedStateEncodingKernel.authoritativeStep = testKernel.authoritativeStep := by
  simp [changedStateEncodingKernel]

example : changedStateEncodingKernel.behaviorTable? !=
    some checkedTestTarget.behaviorTable := by
  native_decide

end Umpire.ModelTests
