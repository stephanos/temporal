import Umpire.ImplementationLink
import Umpire.Target.Tests.Fixtures

/-! Independently checked finite Targets and exact forward-simulation fixtures. -/

namespace Umpire.ImplementationLinkTests

open Umpire
open Umpire.TargetTests

def checkedSourceTarget : CheckedTarget TestLawStatement Unit Bool Bool Bool Bool :=
  checkedTarget (authoringOf testTarget)

def checkedDestinationTarget : CheckedTarget TestLawStatement Unit Bool Bool Bool Bool :=
  checkedSourceTarget

def versionedPrimaryProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with contract := { primaryProvider.contract with version := 2 }
}

def versionedCapabilityTargetDeclaration :
    TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [versionedPrimaryProvider, secondaryProvider]
}

def checkedVersionedCapabilityTarget :
    CheckedTarget TestLawStatement Unit Bool Bool Bool Bool :=
  checkedTarget (authoringOf versionedCapabilityTargetDeclaration)

def lawDriftPrimaryProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with requiredLaws := [] }
  lawWitnesses := []
}

def lawDriftCapabilityTargetDeclaration :
    TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [lawDriftPrimaryProvider, secondaryProvider]
}

def checkedLawDriftCapabilityTarget :
    CheckedTarget TestLawStatement Unit Bool Bool Bool Bool :=
  checkedTarget (authoringOf lawDriftCapabilityTargetDeclaration)

def conflictingCapabilityProvider : CapabilityProvider TestLawStatement := {
  secondaryProvider with contract := {
    secondaryProvider.contract with id := primaryProvider.contract.id
  }
}

def conflictingCapabilityTargetDeclaration :
    TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  requiredCapabilities := [primaryProvider.contract.id]
  providers := [primaryProvider, conflictingCapabilityProvider]
}

def checkedConflictingCapabilityTarget :
    CheckedTarget TestLawStatement Unit Bool Bool Bool Bool :=
  checkedTarget (authoringOf conflictingCapabilityTargetDeclaration)

inductive SparseOutcome where
  | off
  | on
  | invented
  deriving BEq, DecidableEq, Repr

def sparseTransition
    (state action : Bool) : TransitionResult Bool SparseOutcome Bool := {
  modelOutcome := if action then .on else .off
  resultingState := action
  observations := [state]
}

def sparseOutcomeKernel : TransitionKernel Unit Bool Bool SparseOutcome Bool := {
  metadata := testKernel.metadata
  setupDomain := fun _ => True
  stateDomain := fun _ => True
  actionDomain := fun _ => True
  outcomeDomain := fun outcome => outcome ≠ .invented
  observationDomain := fun _ => True
  initialStates := fun _ => [false]
  authoritativeInitial := fun _ state => state = false
  initialSound := by simp
  initialComplete := by simp
  steps := fun state action => [sparseTransition state action]
  authoritativeStep := fun state action result => result = sparseTransition state action
  stepSound := by simp
  stepComplete := by simp
  behaviorDomain := .complete {
    setups := [()]
    states := [false, true]
    actions := [false, true]
    outcomes := [.off, .on]
    observations := [false, true]
    encodeSetup := fun _ => "unit"
    encodeState := toString
    encodeAction := toString
    encodeOutcome := fun outcome => match outcome with
      | .off => "off"
      | .on => "on"
      | .invented => "invented"
    encodeObservation := toString
    setupSound := by simp
    setupComplete := by intro setup _; cases setup; simp
    stateSound := by simp
    stateComplete := by intro state _; cases state <;> simp
    actionSound := by simp
    actionComplete := by intro action _; cases action <;> simp
    outcomeSound := by simp
    outcomeComplete := by intro outcome admitted; cases outcome <;> simp_all
    observationSound := by simp
    observationComplete := by intro observation _; cases observation <;> simp
    setupCoverage := by intro setup state member; cases setup; simp
    initialStateCoverage := by intro setup state member; cases state <;> simp
    transitionSourceCoverage := by intro state action result member; cases state <;> simp
    actionCoverage := by intro state action result member; cases action <;> simp
    resultingStateCoverage := by
      intro state action result member
      cases result.resultingState <;> simp
    outcomeCoverage := by
      intro state action result member
      simp [sparseTransition] at member
      subst result
      cases action <;> simp
    observationCoverage := by
      intro state action result value member observationMember
      cases value <;> simp
  }
}

def sparseOutcomeTargetDeclaration :
    TargetDeclaration TestLawStatement Unit Bool Bool SparseOutcome Bool := {
  testTarget with kernel := .checked sparseOutcomeKernel
}

def sparseOutcomeTargetAuthoring :
    AuthoredTarget TestLawStatement Unit Bool Bool SparseOutcome Bool :=
  AuthoredTarget.make {
    id := sparseOutcomeTargetDeclaration.id
    source := sparseOutcomeTargetDeclaration.source
    definitions := sparseOutcomeTargetDeclaration.definitions
    requiredCapabilities := sparseOutcomeTargetDeclaration.requiredCapabilities
    resolvedSetups := sparseOutcomeTargetDeclaration.resolvedSetups
    kernel := sparseOutcomeTargetDeclaration.kernel
  } (targetCompositionOf testTarget) .unavailable

def checkedSparseOutcomeTarget :
    CheckedTarget TestLawStatement Unit Bool Bool SparseOutcome Bool :=
  checkedTarget sparseOutcomeTargetAuthoring

def relationReference : ImplementationSemanticReference :=
  (implementationSemanticReference? checkedSourceTarget (id "test.relation.shared")
    .relation).get (by native_decide)

def primaryCapabilityReference : ImplementationSemanticReference :=
  (implementationSemanticReference? checkedSourceTarget (id "test.capability.primary")
    .capability).get (by native_decide)

def secondaryCapabilityReference : ImplementationSemanticReference :=
  (implementationSemanticReference? checkedSourceTarget (id "test.capability.secondary")
    .capability).get (by native_decide)

def relationMapping : ImplementationSemanticMapping := {
  source := relationReference
  destination := relationReference
}

def primaryCapabilityMapping : ImplementationSemanticMapping := {
  source := primaryCapabilityReference
  destination := primaryCapabilityReference
}

def secondaryCapabilityMapping : ImplementationSemanticMapping := {
  source := secondaryCapabilityReference
  destination := secondaryCapabilityReference
}

def baseDeclaration :
    ImplementationLinkDeclaration Unit Bool Bool Bool Bool Unit Bool Bool Bool Bool := {
  id := id "test.implementation-link.identity"
  source := source "Test/ImplementationLink.lean"
  sourceTarget := .ofTarget checkedSourceTarget
  destinationTarget := .ofTarget checkedDestinationTarget
  setupMappings := [{ source := (), destination := () }]
  stateMappings := [
    { source := false, destination := false },
    { source := true, destination := true }
  ]
  actionMappings := [
    { source := false, destination := false },
    { source := true, destination := true }
  ]
  outcomeMappings := [
    { source := false, destination := false },
    { source := true, destination := true }
  ]
  observationMappings := [
    { source := false, destination := false },
    { source := true, destination := true }
  ]
  relationMappings := [relationMapping]
  capabilityMappings := [primaryCapabilityMapping, secondaryCapabilityMapping]
  applicationLimit := { value := 10, unit := .semanticTransitions }
  documentation := "Identity fixture whose documentation is non-semantic."
}

theorem baseCoverage : ImplementationLinkRequiredCoverage baseDeclaration checkedSourceTarget
    (fun value => value) (fun value => value) (fun value => value)
    (fun value => value) (fun value => value) := {
  setup := by intro value _; cases value; simp [baseDeclaration]
  state := by intro value _; cases value <;> simp [baseDeclaration]
  action := by intro value _; cases value <;> simp [baseDeclaration]
  outcome := by intro value _; cases value <;> simp [baseDeclaration]
  observation := by intro value _; cases value <;> simp [baseDeclaration]
  relation := by native_decide
  capability := by native_decide
}

def baseWitness : ImplementationLinkWitness baseDeclaration checkedSourceTarget
    checkedDestinationTarget := {
  index := implementationLinkWitnessIndex baseDeclaration checkedSourceTarget checkedDestinationTarget
  mapSetup := fun value => value
  mapState := fun value => value
  mapAction := fun value => value
  mapOutcome := fun value => value
  mapObservation := fun value => value
  initialForward := by intro setup state admitted; exact admitted
  stepForward := by
    intro state action result admitted
    simpa [checkedDestinationTarget] using admitted
  requiredCoverage := baseCoverage
}

def alternateProofWitness : ImplementationLinkWitness baseDeclaration checkedSourceTarget
    checkedDestinationTarget := {
  baseWitness with
  initialForward := by
    intro setup state admitted
    simpa [baseWitness, checkedDestinationTarget] using admitted
  stepForward := by
    intro state action result admitted
    simpa [baseWitness, checkedDestinationTarget] using admitted
}

def reorderedDeclaration :
    ImplementationLinkDeclaration Unit Bool Bool Bool Bool Unit Bool Bool Bool Bool := {
  baseDeclaration with
  stateMappings := baseDeclaration.stateMappings.reverse
  actionMappings := baseDeclaration.actionMappings.reverse
  outcomeMappings := baseDeclaration.outcomeMappings.reverse
  observationMappings := baseDeclaration.observationMappings.reverse
  relationMappings := baseDeclaration.relationMappings.reverse
  capabilityMappings := baseDeclaration.capabilityMappings.reverse
  documentation := "Different non-semantic documentation."
}

theorem reorderedCoverage : ImplementationLinkRequiredCoverage reorderedDeclaration checkedSourceTarget
    (fun value => value) (fun value => value) (fun value => value)
    (fun value => value) (fun value => value) := {
  setup := by intro value _; cases value; simp [reorderedDeclaration, baseDeclaration]
  state := by intro value _; cases value <;> simp [reorderedDeclaration, baseDeclaration]
  action := by intro value _; cases value <;> simp [reorderedDeclaration, baseDeclaration]
  outcome := by intro value _; cases value <;> simp [reorderedDeclaration, baseDeclaration]
  observation := by intro value _; cases value <;> simp [reorderedDeclaration, baseDeclaration]
  relation := by native_decide
  capability := by native_decide
}

def reorderedWitness : ImplementationLinkWitness reorderedDeclaration checkedSourceTarget
    checkedDestinationTarget := {
  index := implementationLinkWitnessIndex reorderedDeclaration checkedSourceTarget
    checkedDestinationTarget
  mapSetup := fun value => value
  mapState := fun value => value
  mapAction := fun value => value
  mapOutcome := fun value => value
  mapObservation := fun value => value
  initialForward := by intro setup state admitted; exact admitted
  stepForward := by
    intro state action result admitted
    simpa [checkedDestinationTarget] using admitted
  requiredCoverage := reorderedCoverage
}

def errorKindOf
    (result : Except ImplementationLinkError Checked) : Option ImplementationLinkErrorKind :=
  match result with
  | .ok _ => none
  | .error linkError => some linkError.kind

def checkedIdentityOf
    (declaration : ImplementationLinkDeclaration Unit Bool Bool Bool Bool Unit Bool Bool Bool Bool)
    (witness : ImplementationLinkWitnessAuthoring declaration checkedSourceTarget
      checkedDestinationTarget) : Option BehaviorFingerprint :=
  (checkImplementationLink declaration checkedSourceTarget checkedDestinationTarget witness).toOption.map
    CheckedImplementationLink.behaviorFingerprint

def incompleteWitness
    (declaration : ImplementationLinkDeclaration Unit Bool Bool Bool Bool Unit Bool Bool Bool Bool)
    (missing : List ImplementationLinkObligation := []) :
    ImplementationLinkWitnessAuthoring declaration checkedSourceTarget checkedDestinationTarget :=
  .incomplete (implementationLinkWitnessIndex declaration checkedSourceTarget checkedDestinationTarget)
    missing

end Umpire.ImplementationLinkTests
