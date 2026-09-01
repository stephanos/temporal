import Umpire.Observation
import Umpire.Examples.Switch
import Umpire.Shared.Test

/-! Shared profile, mapping, and target vocabulary for Observation compilation tests. -/

namespace Umpire.ObservationTests

open Umpire

def id (value : String) : DefinitionId := Shared.Test.definitionId value

def source : SourceLocation := Shared.Test.sourceLocation "Umpire/Observation/Tests/Fixtures.lean"

def metadata (value : String) (kind : DefinitionKind) : DefinitionMetadata :=
  Shared.Test.definitionMetadata value kind source (value ++ "/v1")

def profileId : DefinitionId := id "test.evidence.profile"
def eventKind : DefinitionId := id "test.evidence.kind.event"
def nameField : DefinitionId := id "test.evidence.field.name"
def secretField : DefinitionId := id "test.evidence.field.secret"
def hashedField : DefinitionId := id "test.evidence.field.hashed"
def rejectedField : DefinitionId := id "test.evidence.field.rejected"

def operationState : DefinitionId := id "test.state.operation"
def contributionObservation : DefinitionId := id "test.observation.contribution"
def digestObservation : DefinitionId := id "test.observation.digest"
def unauthorizedObservation : DefinitionId := id "test.observation.unauthorized"

def completedState : DefinitionId := id "test.state.completed"
def startAction : DefinitionId := id "test.action.start"
def successOutcome : DefinitionId := id "test.outcome.success"
def roleField : DefinitionId := id "test.evidence.field.role"

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

def digestPolicyId : DefinitionId := id "test.digest.synthetic"

def digestPolicy : DigestPolicyDeclaration := {
  id := digestPolicyId
  name := "synthetic.digest"
  version := 1
}

def field (fieldId : DefinitionId) : ObservationExpression :=
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
  definitions := [
    metadata operationState.value .state,
    metadata contributionObservation.value .observation,
    metadata digestObservation.value .observation,
    metadata unauthorizedObservation.value .observation
  ]
  meanings := [
    { definitionId := operationState, kind := .state,
      canonicalBehavior := operationState.value ++ "/meaning-v1" },
    { definitionId := contributionObservation, kind := .observation,
      canonicalBehavior := contributionObservation.value ++ "/meaning-v1" },
    { definitionId := digestObservation, kind := .observation,
      canonicalBehavior := digestObservation.value ++ "/meaning-v1" }
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
    (declaration : ObservationMappingDeclaration) : Option BehaviorFingerprint :=
  (checkObservation checkContext declaration).toOption.map CheckedObservationPlan.behaviorFingerprint

/-! Independently authored evaluation fixture; it does not derive its expected trace from rules. -/

def stepCondition : ObservationExpressionAuthoring :=
  .portable (.equals (field roleField) (.text "step"))

def evaluationDeclaration : ObservationMappingDeclaration := {
  baseDeclaration with
  id := id "test.mapping.observation-evaluation"
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

def evaluationContext : ObservationCheckContext := {
  context with
  definitions := context.definitions ++ [
    metadata completedState.value .state,
    metadata startAction.value .action,
    metadata successOutcome.value .outcome
  ]
  meanings := context.meanings ++ [
    { definitionId := completedState, kind := .state,
      canonicalBehavior := completedState.value ++ "/meaning-v1" },
    { definitionId := startAction, kind := .action,
      canonicalBehavior := startAction.value ++ "/meaning-v1" },
    { definitionId := successOutcome, kind := .outcome,
      canonicalBehavior := successOutcome.value ++ "/meaning-v1" }
  ]
  profiles := [{ evidenceProfile with kinds := [{
    id := eventKind
    fields := evidenceProfile.kinds.flatMap EvidenceKindDeclaration.fields ++ [
      { id := roleField, valueType := .text }
    ]
  }] }]
}

def evaluateFixture (bundle : EvidenceBundle) : ObservationResult :=
  match checkObservation evaluationContext evaluationDeclaration with
  | .ok plan => evaluateEvidence plan bundle
  | .error _ => .unknown {
      kind := .zeroUsableInterpretations
      planId := evaluationDeclaration.id
    }

def initialEvidenceId : DefinitionId := id "test.evidence.record.initial"
def stepEvidenceId : DefinitionId := id "test.evidence.record.step-1"
def secondStepEvidenceId : DefinitionId := id "test.evidence.record.step-2"

def textField
    (fieldId : DefinitionId)
    (value : String)
    (digestPolicy : Option DefinitionId := none) : EvidenceFieldValue := {
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

def expectedTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := { definitionId := operationState, value := "ready" }
  steps := [{
    selectedAction := { definitionId := startAction, value := "start" }
    modelOutcome := { definitionId := successOutcome, value := "ok" }
    resultingState := { definitionId := completedState, value := "done" }
    observations := [
      { definitionId := contributionObservation, value := "contributed" },
      { definitionId := digestObservation,
        value := "synthetic.digest/v1:3006720707513255331" }
    ]
  }]
}

def resultKindOf (result : ObservationResult) : Option ObservationFailureKind :=
  result.diagnostic?.map ObservationDiagnostic.kind

def resultStatusOf (result : ObservationResult) : ObservationStatus := result.status

def acceptedOf (result : ObservationResult) : Option EvidenceBackedTrace :=
  match result with
  | .accepted trace => some trace
  | _ => none

def uncheckedTraceOf (trace : EvidenceBackedTrace) : UncheckedEvidenceBackedTrace := {
  traceId := trace.traceId
  checkedPlan := trace.checkedPlan
  mappingId := trace.mappingId
  mappingVersion := trace.mappingVersion
  mappingDigest := trace.mappingDigest
  source := trace.source
  profileId := trace.profileId
  profileVersion := trace.profileVersion
  sourceClosed := trace.sourceClosed
  vocabulary := trace.vocabulary
  dispositions := trace.dispositions
  appliedBound := trace.appliedBound
  evidenceIdentities := trace.evidenceIdentities
  recordSupport := trace.recordSupport
  trace := trace.trace
  evidenceLinks := trace.evidenceLinks
}

def diagnosticKindOf
    (result : Except ObservationDiagnostic α) : Option ObservationFailureKind :=
  match result with
  | .ok _ => none
  | .error diagnostic => some diagnostic.kind

def admissionStatusAndKind
    (result : Except ObservationDiagnostic EvidenceBackedTrace) :
    ObservationStatus × Option ObservationFailureKind :=
  match result with
  | .ok _ => (.accepted, none)
  | .error diagnostic => (diagnostic.status, some diagnostic.kind)

def observationResultOfAdmission
    (result : Except ObservationDiagnostic EvidenceBackedTrace) : ObservationResult :=
  match result with
  | .ok trace => .accepted trace
  | .error diagnostic =>
      match diagnostic.status with
      | .unknown => .unknown diagnostic
      | .conflict => .conflict diagnostic
      | .unsupported => .unsupported diagnostic
      | .accepted => .unknown diagnostic

/-! Checked Property and Query inputs for semantic-verdict tests. -/

def verdictCapability : DefinitionId := id "test.capability.observation-verdict"

def verdictPropertyContext : PropertyCheckContext := {
  definitions := evaluationContext.definitions ++ [
    metadata verdictCapability.value .capability
  ]
  providers := [{
    id := verdictCapability
    version := 1
    canonicalBehavior := "test-observation-verdict/v1"
  }]
  meanings := evaluationContext.meanings.map fun meaning => (verdictCapability, meaning)
}

def verdictPattern
    (field : PropertyTraceField)
    (reference : DefinitionId)
    (constraint : ValueConstraint := .present) : PropertyPattern := {
  field
  reference
  constraint
}

def satisfiedPropertyDeclaration : PropertyDeclaration := {
  id := id "test.property.observation.satisfied"
  source
  requires := [verdictCapability]
  clauses := [
    .stateInvariant (id "test.property.observation.satisfied.initial")
      (verdictPattern .state operationState (.equals "ready"))
  ]
}

def violatedPropertyDeclaration : PropertyDeclaration := {
  satisfiedPropertyDeclaration with
  id := id "test.property.observation.violated"
  clauses := [
    .stateInvariant (id "test.property.observation.violated.initial")
      (verdictPattern .state operationState (.equals "not-ready"))
  ]
}

def repeatedPropertyDeclaration : PropertyDeclaration := {
  satisfiedPropertyDeclaration with
  id := id "test.property.observation.repeated"
  clauses := [
    .inputOutput (id "test.property.observation.repeated.step")
      (verdictPattern .selectedAction startAction)
      (verdictPattern .modelOutcome successOutcome)
  ]
}

def logicalTimePropertyDeclaration : PropertyDeclaration := {
  satisfiedPropertyDeclaration with
  id := id "test.property.observation.logical-time"
  logicalTimeSource := some contributionObservation
  clauses := [
    .ordered (id "test.property.observation.logical-time.order")
      (verdictPattern .observation contributionObservation)
      (verdictPattern .observation digestObservation)
      .logicalTime
  ]
}

def satisfiedProperty : CheckedProperty :=
  (checkProperty verdictPropertyContext (.portable satisfiedPropertyDeclaration))
    |>.toOption.get (by native_decide)

def violatedProperty : CheckedProperty :=
  (checkProperty verdictPropertyContext (.portable violatedPropertyDeclaration))
    |>.toOption.get (by native_decide)

def repeatedProperty : CheckedProperty :=
  (checkProperty verdictPropertyContext (.portable repeatedPropertyDeclaration))
    |>.toOption.get (by native_decide)

def logicalTimeProperty : CheckedProperty :=
  (checkProperty verdictPropertyContext (.portable logicalTimePropertyDeclaration))
    |>.toOption.get (by native_decide)

def checkedQueryTemplate : CheckedQuery Umpire.Examples.Switch.LawStatement :=
  Umpire.Examples.Switch.exploratoryQuery

def verdictQuery
    (properties : List CheckedProperty) : CheckedQuery Umpire.Examples.Switch.LawStatement := {
  checkedQueryTemplate with form := .select properties
}

def evaluationDiagnostic (kind : ObservationFailureKind) : ObservationDiagnostic := {
  kind
  planId := evaluationDeclaration.id
}

end Umpire.ObservationTests
