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

def completedState : DeclarationId := id "test.state.completed"
def startAction : DeclarationId := id "test.action.start"
def successOutcome : DeclarationId := id "test.outcome.success"
def roleField : DeclarationId := id "test.evidence.field.role"

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

/-! Independently authored qualification fixture; it does not derive its expected trace from rules. -/

def stepCondition : ObservationExpressionAuthoring :=
  .portable (.equals (field roleField) (.text "step"))

def qualificationDeclaration : ObservationMappingDeclaration := {
  baseDeclaration with
  id := id "test.mapping.qualification"
  rules := [
    { initialRule with condition := some (.portable
        (.equals (field roleField) (.text "initial"))) },
    {
      id := id "test.rule.step-action"
      output := startAction
      outputKind := .action
      value := .portable (.text "start")
      condition := some stepCondition
    },
    {
      id := id "test.rule.step-outcome"
      output := successOutcome
      outputKind := .outcome
      value := .portable (.text "ok")
      condition := some stepCondition
    },
    {
      id := id "test.rule.step-state"
      output := completedState
      outputKind := .state
      value := .portable (.text "done")
      condition := some stepCondition
    },
    { contributionRule with condition := some stepCondition },
    { digestRule with condition := some stepCondition }
  ]
  ordering := [
    { before := initialRule.id, after := id "test.rule.step-action" },
    { before := id "test.rule.step-action", after := id "test.rule.step-outcome" },
    { before := id "test.rule.step-outcome", after := id "test.rule.step-state" },
    { before := id "test.rule.step-state", after := contributionRule.id },
    { before := contributionRule.id, after := digestRule.id }
  ]
  dispositions := baseDeclaration.dispositions ++ [
    { field := { kind := eventKind, field := roleField }, disposition := .retain }
  ]
  evidenceBound := { value := 3, unit := .evidenceRecords }
}

def qualificationContext : ObservationCheckContext := {
  context with
  declarations := context.declarations ++ [
    metadata completedState.value .state,
    metadata startAction.value .action,
    metadata successOutcome.value .outcome
  ]
  meanings := context.meanings ++ [
    { declaration := completedState, kind := .state,
      semanticDigest := completedState.value ++ "/meaning-v1" },
    { declaration := startAction, kind := .action,
      semanticDigest := startAction.value ++ "/meaning-v1" },
    { declaration := successOutcome, kind := .outcome,
      semanticDigest := successOutcome.value ++ "/meaning-v1" }
  ]
  profiles := [{ evidenceProfile with kinds := [{
    id := eventKind
    fields := evidenceProfile.kinds.flatMap EvidenceKindDeclaration.fields ++ [
      { id := roleField, valueType := .text }
    ]
  }] }]
}

def qualifyFixture (bundle : EvidenceBundle) : QualificationResult :=
  match checkObservation qualificationContext qualificationDeclaration with
  | .ok plan => qualifyEvidence plan bundle
  | .error _ => .unknown {
      kind := .zeroUsableInterpretations
      planId := qualificationDeclaration.id
    }

def initialEvidenceId : DeclarationId := id "test.evidence.record.initial"
def stepEvidenceId : DeclarationId := id "test.evidence.record.step-1"
def secondStepEvidenceId : DeclarationId := id "test.evidence.record.step-2"

def textField
    (fieldId : DeclarationId)
    (value : String)
    (digestPolicy : Option DeclarationId := none) : EvidenceFieldValue := {
  field := fieldId
  value := .text value
  digestPolicy
}

def initialEvidence : SyntheticEvidenceRecord := {
  id := initialEvidenceId
  profile := profileId
  profileVersion := 1
  kind := eventKind
  sequence := 1
  fields := [
    textField roleField "initial",
    textField nameField "  ready  "
  ]
}

def stepEvidence : SyntheticEvidenceRecord := {
  id := stepEvidenceId
  profile := profileId
  profileVersion := 1
  kind := eventKind
  sequence := 2
  causalParents := [initialEvidenceId]
  fields := [
    textField roleField "step",
    textField secretField "forbidden-secret",
    textField hashedField "forbidden-hash-material" (some digestPolicyId)
  ]
}

def completeEvidence : EvidenceBundle := {
  profile := profileId
  profileVersion := 1
  records := [stepEvidence, initialEvidence]
  closures := [{ kind := eventKind, lastSequence := 2 }]
}

def expectedTrace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
  initialState := { identity := operationState, value := "ready" }
  steps := [{
    selectedAction := { identity := startAction, value := "start" }
    modelOutcome := { identity := successOutcome, value := "ok" }
    resultingState := { identity := completedState, value := "done" }
    observations := [
      { identity := contributionObservation, value := "contributed" },
      { identity := digestObservation,
        value := "synthetic.digest/v1:3006720707513255331" }
    ]
  }]
}

def resultKindOf (result : QualificationResult) : Option QualificationFailureKind :=
  result.diagnostic?.map QualificationDiagnostic.kind

def resultStatusOf (result : QualificationResult) : QualificationStatus := result.status

def qualifiedOf (result : QualificationResult) : Option QualifiedTrace :=
  match result with
  | .qualified trace => some trace
  | _ => none

def diagnosticKindOf
    (result : Except QualificationDiagnostic Unit) : Option QualificationFailureKind :=
  match result with
  | .ok _ => none
  | .error diagnostic => some diagnostic.kind

end Umpire.ObservationTests
