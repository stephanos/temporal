import Umpire.Observation

/-! Shared profile, mapping, and target vocabulary for Observation compilation tests. -/

namespace Umpire.ObservationTests

open Umpire

def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Umpire/Observation/Tests/Fixtures.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def metadata (value : String) (kind : DeclarationKind) : DeclarationMetadata := {
  id := id value
  kind
  source
  contractDigest := value ++ "/v1"
}

def profileId : DeclarationId := id "test.evidence.profile"
def eventKind : DeclarationId := id "test.evidence.kind.event"
def nameField : DeclarationId := id "test.evidence.field.name"
def secretField : DeclarationId := id "test.evidence.field.secret"
def hashedField : DeclarationId := id "test.evidence.field.hashed"
def rejectedField : DeclarationId := id "test.evidence.field.rejected"

def operationState : DeclarationId := id "test.state.operation"
def contributionObservation : DeclarationId := id "test.observation.contribution"
def digestObservation : DeclarationId := id "test.observation.digest"
def unauthorizedObservation : DeclarationId := id "test.observation.unauthorized"

def evidenceProfile : EvidenceProfileDeclaration := {
  id := profileId
  source
  kinds := [{
    id := eventKind
    fields := [
      { id := nameField, valueType := .text },
      { id := secretField, valueType := .text },
      { id := hashedField, valueType := .text },
      { id := rejectedField, valueType := .text }
    ]
  }]
}

def digestPolicyId : DeclarationId := id "test.digest.synthetic"

def digestPolicy : DigestPolicyDeclaration := {
  id := digestPolicyId
  name := "synthetic.digest"
  version := 1
}

def field (fieldId : DeclarationId) : ObservationExpression :=
  .field { kind := eventKind, field := fieldId }

def normalizedName : ObservationBinding := {
  id := id "test.binding.normalized-name"
  valueType := .text
  expression := .portable (.normalize { name := "text.trim", version := 1 } (field nameField))
}

def initialRule : ObservationRule := {
  id := id "test.rule.initial-state"
  output := operationState
  outputKind := .state
  value := .portable (.binding normalizedName.id)
  condition := some (.portable (.and
    (.present (field nameField))
    (.equals (.boolean true) (.boolean true))))
}

def contributionRule : ObservationRule := {
  id := id "test.rule.contribution"
  output := contributionObservation
  outputKind := .observation
  value := .portable (.contributionMarker (field secretField))
}

def digestRule : ObservationRule := {
  id := id "test.rule.digest"
  output := digestObservation
  outputKind := .observation
  value := .portable (.digestToken digestPolicyId (field hashedField))
}

def baseDeclaration : ObservationMappingDeclaration := {
  id := id "test.mapping.lifecycle"
  source
  profile := profileId
  digestPolicies := [digestPolicy]
  bindings := [normalizedName]
  rules := [initialRule, contributionRule, digestRule]
  ordering := [
    { before := initialRule.id, after := contributionRule.id },
    { before := contributionRule.id, after := digestRule.id }
  ]
  closures := [{ kind := eventKind }]
  dispositions := [
    { field := { kind := eventKind, field := nameField }, disposition := .retain },
    { field := { kind := eventKind, field := secretField }, disposition := .redact },
    { field := { kind := eventKind, field := hashedField },
      disposition := .hash (some digestPolicyId) },
    { field := { kind := eventKind, field := rejectedField }, disposition := .reject }
  ]
  evidenceBound := { value := 10, unit := .evidenceRecords }
}

def context : ObservationCheckContext := {
  declarations := [
    metadata operationState.value .state,
    metadata contributionObservation.value .observation,
    metadata digestObservation.value .observation,
    metadata unauthorizedObservation.value .observation
  ]
  meanings := [
    { declaration := operationState, kind := .state,
      semanticDigest := operationState.value ++ "/meaning-v1" },
    { declaration := contributionObservation, kind := .observation,
      semanticDigest := contributionObservation.value ++ "/meaning-v1" },
    { declaration := digestObservation, kind := .observation,
      semanticDigest := digestObservation.value ++ "/meaning-v1" }
  ]
  profiles := [evidenceProfile]
}

def errorKindOf
    (result : Except ObservationError CheckedObservationPlan) : Option ObservationErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

def planIdentityOf
    (checkContext : ObservationCheckContext)
    (declaration : ObservationMappingDeclaration) : Option String :=
  (checkObservation checkContext declaration).toOption.map CheckedObservationPlan.semanticDigest

end Umpire.ObservationTests
