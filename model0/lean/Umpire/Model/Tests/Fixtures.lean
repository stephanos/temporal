import Umpire.Model
import Umpire.Shared.Test

/-! Shared semantic-composition vocabulary used by the Target concern tests. -/

namespace Umpire.ModelTests

open Umpire

def id (value : String) : DefinitionId := Shared.Test.definitionId value

def source (path : String) : SourceLocation := Shared.Test.sourceLocation path

def metadata
    (value : String)
    (kind : DefinitionKind)
    (digest : String := "contract-v1") : DefinitionMetadata :=
  Shared.Test.definitionMetadata value kind (source "Umpire/ModelTests.lean") digest

def providerLaw : Law := {
  id := id "umpire.law.provider-sound"
  body := "provider-sound/v1"
}

def connectorLaw : Law := {
  id := id "umpire.law.connector-sound"
  body := "connector-sound/v1"
}

def TestLawStatement (law : Law) : Prop :=
  law.id = providerLaw.id ∨ law.id = connectorLaw.id

def witness
    (definition : Law)
    (proof : TestLawStatement definition) : LawProof TestLawStatement := {
  definition
  proof
}

def transition (state action : Bool) : Step Bool Bool Bool := {
  outcome := action
  state := action
  facts := [state]
}

def testKernel : Machine Unit Bool Bool Bool Bool := {
  metadata := {
    id := id "test.kernel.transition"
    source := source "Umpire/ModelTests.lean"
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
      cases result.state <;> simp
    outcomeCoverage := by intro state action result member; cases result.outcome <;> simp
    observationCoverage := by
      intro state action result value member observationMember
      cases value <;> simp
  }
}

def primaryProvider : Provider TestLawStatement := {
  id := id "test.provider.primary"
  source := source "Test/PrimarySemantic.lean"
  contract := {
    id := id "test.capability.primary"
    behaviorVersion := "test-primary-capability/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    behaviorVersion := "test-primary-shared/v1"
  }]
  lawProofs := [witness providerLaw (by exact .inl rfl)]
}

def secondaryProvider : Provider TestLawStatement := {
  id := id "test.provider.secondary"
  source := source "Test/SecondarySemantic.lean"
  contract := {
    id := id "test.capability.secondary"
    behaviorVersion := "test-secondary-capability/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    behaviorVersion := "test-secondary-shared/v1"
  }]
  lawProofs := [witness providerLaw (by exact .inl rfl)]
}

def ownershipConnector : Connector TestLawStatement := {
  id := id "test.connector.shared"
  source := source "Test/CompositeSemantic.lean"
  behaviorVersion := "test-shared-connector/v1"
  reconciliations := [{
    definitionId := id "test.relation.shared"
    kind := .relation
    providers := [primaryProvider.id, secondaryProvider.id]
    behaviorVersion := "test-shared-connector/reconciled-v1"
  }]
  requiredLaws := [connectorLaw]
  lawProofs := [witness connectorLaw (by exact .inr rfl)]
}

def testDefinitions : List DefinitionMetadata := [
  metadata "test.target.composed" .target,
  metadata "test.kernel.transition" .machine,
  metadata "test.capability.primary" .capability,
  metadata "test.capability.secondary" .capability,
  metadata "test.provider.primary" .provider,
  metadata "test.provider.secondary" .provider,
  metadata "umpire.law.provider-sound" .law providerLaw.body,
  metadata "umpire.law.connector-sound" .law connectorLaw.body,
  metadata "test.connector.shared" .connector,
  metadata "test.relation.shared" .relation,
  metadata "test.action.request" .action,
  metadata "test.observation.completed" .fact
]

def testTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
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
  machine := .checked testKernel
}

/-- Collect a spec's providers and connectors back into a `Providers` value. -/
def providersOf
    (target : ModelSpec TestLawStatement Unit Bool Bool Bool Bool) :
    Providers TestLawStatement :=
  let providers := target.providers.foldl (fun result provider => result.provide provider)
    Providers.empty
  target.connectors.foldl (fun result connector => result.connect connector) providers

def authoringOf
    (target : ModelSpec TestLawStatement Unit Bool Bool Bool Bool)
    (planning : AuthoredPlanningCapability target.machine := .unavailable)
    (occurrences : List SourceRef := []) :
    DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  DraftModel.make target .empty planning occurrences

def errorOf {Target : Type}
    (result : Except DefinitionError Target) : Option DefinitionError :=
  match result with
  | .error error => some error
  | .ok _ => none

def conflictingTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with connectors := []
}

end Umpire.ModelTests
