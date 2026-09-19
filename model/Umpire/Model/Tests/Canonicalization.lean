import Umpire.Model.Tests.Fixtures

/-! Canonical ordering, digest sensitivity, documentation, and serializer checks. -/

namespace Umpire.ModelTests

open Umpire

def reorderedTestTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := testTarget.definitions.reverse
  requiredCapabilities := testTarget.requiredCapabilities.reverse
  providers := testTarget.providers.reverse
  connectors := testTarget.connectors.reverse
}

example : ((checkModel (DraftModel.make reorderedTestTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.canonicalMetadata =
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.canonicalMetadata := by
  native_decide

example : ((checkModel (DraftModel.make reorderedTestTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint =
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint := by
  native_decide

def reorderedConflictTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  conflictingTarget with
  definitions := conflictingTarget.definitions.reverse
  providers := conflictingTarget.providers.reverse
}

example : (errorOf ((checkModel (DraftModel.make reorderedConflictTarget) |>.mapError LocatedError.error))).map canonicalDefinitionErrorJson =
    (errorOf ((checkModel (DraftModel.make conflictingTarget) |>.mapError LocatedError.error))).map canonicalDefinitionErrorJson := by
  native_decide

def changedIdentityTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  id := id "test.target.composed-v2"
  definitions := metadata "test.target.composed-v2" .target ::
    testDefinitions.filter (fun declaration => declaration.id != testTarget.id)
}

example : ((checkModel (DraftModel.make changedIdentityTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint ≠
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint := by
  native_decide

def changedContractProvider : Provider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with behaviorVersion := "test-primary-capability/v2" }
}

def changedContractTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [changedContractProvider, secondaryProvider]
}

example : ((checkModel (DraftModel.make changedContractTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint ≠
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint := by
  native_decide

def changedConnector : Connector TestLawStatement := {
  ownershipConnector with behaviorVersion := "test-shared-connector/v2"
}

def changedConnectorTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with connectors := [changedConnector]
}

example : ((checkModel (DraftModel.make changedConnectorTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint ≠
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint := by
  native_decide

def changedKernel : Machine Unit Bool Bool Bool Bool := {
  testKernel with
  steps := fun state action => [{
    outcome := action
    state := action
    facts := [!state]
  }]
  authoritativeStep := fun state action result => result = {
    outcome := action
    state := action
    facts := [!state]
  }
  stepSound := by simp
  stepComplete := by simp
  vocabulary := .complete {
    setups := [()]
    states := [false, true]
    actions := [false, true]
    outcomes := [false, true]
    observations := [false, true]
    encodeSetup := fun _ => "unit"
    encodeState := toString
    encodeAction := toString
    encodeOutcome := toString
    encodeObservation := toString
    setupSound := by simp [testKernel]
    setupComplete := by intro setup _; cases setup; simp
    stateSound := by simp [testKernel]
    stateComplete := by intro state _; cases state <;> simp
    actionSound := by simp [testKernel]
    actionComplete := by intro action _; cases action <;> simp
    outcomeSound := by simp [testKernel]
    outcomeComplete := by intro outcome _; cases outcome <;> simp
    observationSound := by simp [testKernel]
    observationComplete := by intro observation _; cases observation <;> simp
    setupCoverage := by intro setup state member; cases setup; simp
    initialStateCoverage := by intro setup state member; cases state <;> simp
    transitionSourceCoverage := by intro state action result member; cases state <;> simp
    actionCoverage := by intro state action result member; cases action <;> simp
    resultingStateCoverage := by
      intro state action result member
      cases result.state <;> simp
    outcomeCoverage := by intro state action result member; cases result.outcome <;> simp
    observationCoverage := by
      intro state action result value member observationMember
      cases value <;> simp
  }
}

def changedKernelTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with machine := .checked changedKernel
}

example : ((checkModel (DraftModel.make changedKernelTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint ≠
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint := by
  native_decide

def changedLaw : Law := {
  providerLaw with body := "provider-sound/v2"
}

def changedLawProvider : Provider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with requiredLaws := [changedLaw] }
  lawProofs := [witness changedLaw (by exact .inl rfl)]
}

def changedLawSecondaryProvider : Provider TestLawStatement := {
  secondaryProvider with
  contract := { secondaryProvider.contract with requiredLaws := [changedLaw] }
  lawProofs := [witness changedLaw (by exact .inl rfl)]
}

def changedLawTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := testDefinitions.map fun declaration =>
    if declaration.id == providerLaw.id then
      { declaration with behaviorVersion := changedLaw.body }
    else
      declaration
  providers := [changedLawProvider, changedLawSecondaryProvider]
}

example : ((checkModel (DraftModel.make changedLawTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint ≠
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint := by
  native_decide

def lawlessPrimaryProvider : Provider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with requiredLaws := [] }
  lawProofs := []
}

def lawlessSecondaryProvider : Provider TestLawStatement := {
  secondaryProvider with
  contract := { secondaryProvider.contract with requiredLaws := [] }
  lawProofs := []
}

def lawlessTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [lawlessPrimaryProvider, lawlessSecondaryProvider]
}

example : ((checkModel (DraftModel.make lawlessTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint ≠
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint := by
  native_decide

def documentedTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := testDefinitions.map fun declaration =>
    if declaration.id == testTarget.id then
      { declaration with documentation := "Non-semantic explanatory text." }
    else
      declaration
}

example : ((checkModel (DraftModel.make documentedTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint =
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.behaviorFingerprint := by
  native_decide

example : ((checkModel (DraftModel.make documentedTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.canonicalMetadata ≠
    ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map CheckedModel.canonicalMetadata := by
  native_decide

example : (checkModel (authoringOf testTarget)).toOption.map CheckedModel.behaviorFingerprint =
    (checkModel (authoringOf testTarget |>.withoutPlanning)).toOption.map
      CheckedModel.behaviorFingerprint := by
  native_decide

example : canonicalProviderJson primaryProvider =
    canonicalProviderJson primaryProvider := by
  rfl

example : canonicalConnectorJson ownershipConnector =
    canonicalConnectorJson ownershipConnector := by
  rfl

end Umpire.ModelTests
