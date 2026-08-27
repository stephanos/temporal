import Umpire.Target

/-! Shared semantic-composition vocabulary used by the Target concern tests. -/

namespace Umpire.TargetTests

open Umpire

def id (value : String) : DefinitionId := DefinitionId.of value

def source (path : String) : SourceLocation := {
  path
  line := 1
  column := 1
  provenance := "lean-test"
}

def metadata
    (value : String)
    (kind : DefinitionKind)
    (digest : String := "contract-v1") : DefinitionMetadata := {
  id := id value
  kind
  source := source "Umpire/TargetTests.lean"
  canonicalBehavior := digest
}

def providerLaw : LawDefinition := {
  id := id "umpire.law.provider-sound"
  body := "provider-sound/v1"
}

def connectorLaw : LawDefinition := {
  id := id "umpire.law.connector-sound"
  body := "connector-sound/v1"
}

def TestLawStatement (law : LawDefinition) : Prop :=
  law.id = providerLaw.id ∨ law.id = connectorLaw.id

def witness
    (definition : LawDefinition)
    (proof : TestLawStatement definition) : LawWitness TestLawStatement := {
  definition
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
    source := source "Umpire/TargetTests.lean"
  }
  setupDomain := fun _ => True
  stateDomain := fun _ => True
  actionDomain := fun _ => True
  outcomeDomain := fun _ => True
  observationDomain := fun _ => True
  initialStates := fun _ => [false]
  authoritativeInitial := fun _ state => state = false
  initialSound := by simp
  initialComplete := by simp
  steps := fun state action => [transition state action]
  authoritativeStep := fun state action result => result = transition state action
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
    setupSound := by simp
    setupComplete := by intro setup _; cases setup; simp
    stateSound := by simp
    stateComplete := by intro state _; cases state <;> simp
    actionSound := by simp
    actionComplete := by intro action _; cases action <;> simp
    outcomeSound := by simp
    outcomeComplete := by intro outcome _; cases outcome <;> simp
    observationSound := by simp
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

def primaryProvider : CapabilityProvider TestLawStatement := {
  id := id "test.provider.primary"
  source := source "Test/PrimarySemantic.lean"
  contract := {
    id := id "test.capability.primary"
    canonicalBehavior := "test-primary-capability/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    canonicalBehavior := "test-primary-shared/v1"
  }]
  lawWitnesses := [witness providerLaw (by exact .inl rfl)]
}

def secondaryProvider : CapabilityProvider TestLawStatement := {
  id := id "test.provider.secondary"
  source := source "Test/SecondarySemantic.lean"
  contract := {
    id := id "test.capability.secondary"
    canonicalBehavior := "test-secondary-capability/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    canonicalBehavior := "test-secondary-shared/v1"
  }]
  lawWitnesses := [witness providerLaw (by exact .inl rfl)]
}

def ownershipConnector : CapabilityConnector TestLawStatement := {
  id := id "test.connector.shared"
  source := source "Test/CompositeSemantic.lean"
  canonicalBehavior := "test-shared-connector/v1"
  reconciliations := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    providers := [primaryProvider.id, secondaryProvider.id]
    canonicalBehavior := "test-shared-connector/reconciled-v1"
  }]
  requiredLaws := [connectorLaw]
  lawWitnesses := [witness connectorLaw (by exact .inr rfl)]
}

def testDefinitions : List DefinitionMetadata := [
  metadata "test.target.composed" .target,
  metadata "test.kernel.transition" .kernel,
  metadata "test.capability.primary" .capability,
  metadata "test.capability.secondary" .capability,
  metadata "test.provider.primary" .provider,
  metadata "test.provider.secondary" .provider,
  metadata "umpire.law.provider-sound" .law providerLaw.body,
  metadata "umpire.law.connector-sound" .law connectorLaw.body,
  metadata "test.connector.shared" .connector,
  metadata "test.relation.shared" .relation,
  metadata "test.action.request" .action,
  metadata "test.observation.completed" .observation
]

def testTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  id := id "test.target.composed"
  source := source "Test/CompositeSemantic.lean"
  definitions := testDefinitions
  requiredCapabilities := [
    id "test.capability.primary",
    id "test.capability.secondary"
  ]
  providers := [primaryProvider, secondaryProvider]
  connectors := [ownershipConnector]
  resolvedSetups := [()]
  kernel := .checked testKernel
}

def targetDefinitionOf
    (target : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool) :
    TargetDefinition Unit Bool Bool Bool Bool := {
  id := target.id
  source := target.source
  definitions := target.definitions
  requiredCapabilities := target.requiredCapabilities
  resolvedSetups := target.resolvedSetups
  kernel := target.kernel
}

def targetCompositionOf
    (target : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool) :
    TargetComposition TestLawStatement :=
  let providers := target.providers.foldl (fun result provider => result.provide provider)
    TargetComposition.empty
  target.connectors.foldl (fun result connector => result.connect connector) providers

def authoringOf
    (target : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool)
    (planning : AuthoredPlanningCapability target.kernel := .unavailable)
    (occurrences : List AuthoringOccurrence := []) :
    AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  AuthoredTarget.make (targetDefinitionOf target) (targetCompositionOf target) planning occurrences

def errorOf {Target : Type}
    (result : Except DefinitionError Target) : Option DefinitionError :=
  match result with
  | .error error => some error
  | .ok _ => none

def conflictingTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with connectors := []
}

end Umpire.TargetTests
