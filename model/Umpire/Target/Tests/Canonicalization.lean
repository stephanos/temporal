import Umpire.Target.Tests.Fixtures

/-! Canonical ordering, digest sensitivity, documentation, and serializer checks. -/

namespace Umpire.TargetTests

open Umpire

def reorderedTestTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := testTarget.definitions.reverse
  requiredCapabilities := testTarget.requiredCapabilities.reverse
  providers := testTarget.providers.reverse
  connectors := testTarget.connectors.reverse
}

example : (composeTarget reorderedTestTarget).toOption.map CheckedTarget.canonicalMetadata =
    (composeTarget testTarget).toOption.map CheckedTarget.canonicalMetadata := by
  native_decide

example : (composeTarget reorderedTestTarget).toOption.map CheckedTarget.behaviorFingerprint =
    (composeTarget testTarget).toOption.map CheckedTarget.behaviorFingerprint := by
  native_decide

def reorderedConflictTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  conflictingTarget with
  definitions := conflictingTarget.definitions.reverse
  providers := conflictingTarget.providers.reverse
}

example : (errorOf (composeTarget reorderedConflictTarget)).map canonicalDefinitionErrorJson =
    (errorOf (composeTarget conflictingTarget)).map canonicalDefinitionErrorJson := by
  native_decide

def changedIdentityTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  id := id "test.target.composed-v2"
  definitions := metadata "test.target.composed-v2" .target ::
    testDefinitions.filter (fun declaration => declaration.id != testTarget.id)
}

example : (composeTarget changedIdentityTarget).toOption.map CheckedTarget.behaviorFingerprint ≠
    (composeTarget testTarget).toOption.map CheckedTarget.behaviorFingerprint := by
  native_decide

def changedContractProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with canonicalBehavior := "test-primary-capability/v2" }
}

def changedContractTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [changedContractProvider, secondaryProvider]
}

example : (composeTarget changedContractTarget).toOption.map CheckedTarget.behaviorFingerprint ≠
    (composeTarget testTarget).toOption.map CheckedTarget.behaviorFingerprint := by
  native_decide

def changedConnector : CapabilityConnector TestLawStatement := {
  ownershipConnector with canonicalBehavior := "test-shared-connector/v2"
}

def changedConnectorTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with connectors := [changedConnector]
}

example : (composeTarget changedConnectorTarget).toOption.map CheckedTarget.behaviorFingerprint ≠
    (composeTarget testTarget).toOption.map CheckedTarget.behaviorFingerprint := by
  native_decide

def changedKernel : TransitionKernel Unit Bool Bool Bool Bool := {
  testKernel with
  steps := fun state action => [{
    modelOutcome := action
    resultingState := action
    observations := [!state]
  }]
  authoritativeStep := fun state action result => result = {
    modelOutcome := action
    resultingState := action
    observations := [!state]
  }
  stepSound := by simp
  stepComplete := by simp
  behaviorDomain := .complete {
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
      cases result.resultingState <;> simp
    outcomeCoverage := by intro state action result member; cases result.modelOutcome <;> simp
    observationCoverage := by
      intro state action result value member observationMember
      cases value <;> simp
  }
}

def changedKernelTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with kernel := .checked changedKernel
}

example : (composeTarget changedKernelTarget).toOption.map CheckedTarget.behaviorFingerprint ≠
    (composeTarget testTarget).toOption.map CheckedTarget.behaviorFingerprint := by
  native_decide

def changedLaw : LawDefinition := {
  providerLaw with body := "provider-sound/v2"
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
  definitions := testDefinitions.map fun declaration =>
    if declaration.id == providerLaw.id then
      { declaration with canonicalBehavior := changedLaw.body }
    else
      declaration
  providers := [changedLawProvider, changedLawSecondaryProvider]
}

example : (composeTarget changedLawTarget).toOption.map CheckedTarget.behaviorFingerprint ≠
    (composeTarget testTarget).toOption.map CheckedTarget.behaviorFingerprint := by
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

example : (composeTarget lawlessTarget).toOption.map CheckedTarget.behaviorFingerprint ≠
    (composeTarget testTarget).toOption.map CheckedTarget.behaviorFingerprint := by
  native_decide

def documentedTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := testDefinitions.map fun declaration =>
    if declaration.id == testTarget.id then
      { declaration with documentation := "Non-semantic explanatory text." }
    else
      declaration
}

example : (composeTarget documentedTarget).toOption.map CheckedTarget.behaviorFingerprint =
    (composeTarget testTarget).toOption.map CheckedTarget.behaviorFingerprint := by
  native_decide

example : (composeTarget documentedTarget).toOption.map CheckedTarget.canonicalMetadata ≠
    (composeTarget testTarget).toOption.map CheckedTarget.canonicalMetadata := by
  native_decide

example : (checkTarget (authoringOf testTarget)).toOption.map CheckedTarget.behaviorFingerprint =
    (checkTarget (authoringOf testTarget |>.withoutPlanning)).toOption.map
      CheckedTarget.behaviorFingerprint := by
  native_decide

example : canonicalCapabilityProviderJson primaryProvider =
    canonicalCapabilityProviderJson primaryProvider := by
  rfl

example : canonicalCapabilityConnectorJson ownershipConnector =
    canonicalCapabilityConnectorJson ownershipConnector := by
  rfl

end Umpire.TargetTests
