import Umpire.Target

/-! Shared semantic-composition vocabulary used by the Target concern tests. -/

namespace Umpire.TargetTests

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
  source := source "Umpire/TargetTests.lean"
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
    source := source "Umpire/TargetTests.lean"
  }
  initialStates := fun _ => [false]
  authoritativeInitial := fun _ state => state = false
  initialSound := by simp
  initialComplete := by simp
  steps := fun state action => [transition state action]
  authoritativeStep := fun state action result => result = transition state action
  stepSound := by simp
  stepComplete := by simp
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

def errorOf {Target : Type}
    (result : Except DeclarationError Target) : Option DeclarationError :=
  match result with
  | .error error => some error
  | .ok _ => none

def conflictingTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with connectors := []
}

end Umpire.TargetTests
