import Umpire.Evidence
import Umpire.Examples.Switch
import Umpire.Shared.Test

/-! Shared profile, mapping, and target vocabulary for Observation compilation tests. -/

namespace Umpire.EvidenceTests

open Umpire

def id (value : String) : DefinitionId := Shared.Test.definitionId value

def source : SourceLocation := Shared.Test.sourceLocation "Umpire/Observation/Tests/Fixtures.lean"

def metadata (value : String) (kind : DefinitionKind) : DefinitionMetadata :=
  Shared.Test.definitionMetadata value kind source (value ++ "/v1")

def profileId : DefinitionId := id "test.evidence.profile"
def eventKind : DefinitionId := id "test.evidence.kind.event"
def nameFieldSpec : ObservationFieldSpec := {
  kind := eventKind
  field := id "test.evidence.field.name"
  valueType := .text
}
def secretFieldSpec : ObservationFieldSpec := {
  kind := eventKind
  field := id "test.evidence.field.secret"
  valueType := .text
}
def hashedFieldSpec : ObservationFieldSpec := {
  kind := eventKind
  field := id "test.evidence.field.hashed"
  valueType := .text
}
def rejectedFieldSpec : ObservationFieldSpec := {
  kind := eventKind
  field := id "test.evidence.field.rejected"
  valueType := .text
}
def nameField : DefinitionId := nameFieldSpec.field
def secretField : DefinitionId := secretFieldSpec.field
def hashedField : DefinitionId := hashedFieldSpec.field
def rejectedField : DefinitionId := rejectedFieldSpec.field

def operationState : DefinitionId := id "test.state.operation"
def contributionObservation : DefinitionId := id "test.observation.contribution"
def digestObservation : DefinitionId := id "test.observation.digest"
def unauthorizedObservation : DefinitionId := id "test.observation.unauthorized"

def completedState : DefinitionId := id "test.state.completed"
def startAction : DefinitionId := id "test.action.start"
def successOutcome : DefinitionId := id "test.outcome.success"
def roleFieldSpec : ObservationFieldSpec := {
  kind := eventKind
  field := id "test.evidence.field.role"
  valueType := .text
}
def roleField : DefinitionId := roleFieldSpec.field

def evidenceProfileSpec : ObservationProfileSpec := {
  id := profileId
  source
  kinds := [{
    id := eventKind
    fields := [nameFieldSpec, secretFieldSpec, hashedFieldSpec, rejectedFieldSpec]
  }]
}

def evidenceProfile : EvidenceProfileDeclaration :=
  evidenceProfileSpec.declaration

def digestPolicyId : DefinitionId := id "test.digest.synthetic"

def digestPolicy : DigestPolicyDeclaration := {
  id := digestPolicyId
  name := "synthetic.digest"
  version := 1
}

def field (fieldSpec : ObservationFieldSpec) : ObservationExpression :=
  fieldSpec.expression

def normalizedName : ObservationBinding := {
  id := id "test.binding.normalized-name"
  valueType := .text
  expression := .portable (.normalize { name := "text.trim", version := 1 }
    nameFieldSpec.expression)
}

def initialRule : ObservationRule := {
  id := id "test.rule.initial-state"
  output := operationState
  outputKind := .state
  value := .portable (.binding normalizedName.id)
  condition := some (.portable (.and
    (.present nameFieldSpec.expression)
    (.equals (.boolean true) (.boolean true))))
}

def contributionRule : ObservationRule := {
  id := id "test.rule.contribution"
  output := contributionObservation
  outputKind := .fact
  value := .portable (.contributionMarker secretFieldSpec.expression)
}

def digestRule : ObservationRule := {
  id := id "test.rule.digest"
  output := digestObservation
  outputKind := .fact
  value := .portable (.digestToken digestPolicyId hashedFieldSpec.expression)
}

def baseSpec : Evidence.ReadingSpec := {
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
    (nameFieldSpec, .retain),
    (secretFieldSpec, .redact),
    (hashedFieldSpec, .hash (some digestPolicyId)),
    (rejectedFieldSpec, .reject)
  ]
  evidenceBound := { value := 10, unit := .evidenceRecords }
}

def baseDeclaration : Evidence.Reading :=
  baseSpec.declaration

def context : Evidence.ReadingContext := {
  definitions := [
    metadata operationState.value .state,
    metadata contributionObservation.value .fact,
    metadata digestObservation.value .fact,
    metadata unauthorizedObservation.value .fact
  ]
  meanings := [
    { definitionId := operationState, kind := .state,
      behaviorVersion := operationState.value ++ "/meaning-v1" },
    { definitionId := contributionObservation, kind := .fact,
      behaviorVersion := contributionObservation.value ++ "/meaning-v1" },
    { definitionId := digestObservation, kind := .fact,
      behaviorVersion := digestObservation.value ++ "/meaning-v1" }
  ]
  profiles := [evidenceProfile]
}

def errorKindOf
    (result : Except Evidence.ReadingError Evidence.CheckedReading) : Option Evidence.ReadingErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

def planIdentityOf
    (checkContext : Evidence.ReadingContext)
    (declaration : Evidence.Reading) : Option BehaviorFingerprint :=
  (Evidence.checkReading checkContext declaration).toOption.map Evidence.CheckedReading.behaviorFingerprint

/-! Independently authored evaluation fixture; it does not derive its expected trace from rules. -/

def stepCondition : ObservationExpressionAuthoring :=
  .portable (.equals roleFieldSpec.expression (.text "step"))

def evaluationSpec : Evidence.ReadingSpec := {
  baseSpec with
  id := id "test.mapping.observation-evaluation"
  rules := [
    { initialRule with condition := some (.portable
        (.equals roleFieldSpec.expression (.text "initial"))) },
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
  dispositions := baseSpec.dispositions ++ [
    (roleFieldSpec, .retain)
  ]
  evidenceBound := { value := 3, unit := .evidenceRecords }
}

def evaluationDeclaration : Evidence.Reading :=
  evaluationSpec.declaration

def evaluationContext : Evidence.ReadingContext := {
  context with
  definitions := context.definitions ++ [
    metadata completedState.value .state,
    metadata startAction.value .action,
    metadata successOutcome.value .outcome
  ]
  meanings := context.meanings ++ [
    { definitionId := completedState, kind := .state,
      behaviorVersion := completedState.value ++ "/meaning-v1" },
    { definitionId := startAction, kind := .action,
      behaviorVersion := startAction.value ++ "/meaning-v1" },
    { definitionId := successOutcome, kind := .outcome,
      behaviorVersion := successOutcome.value ++ "/meaning-v1" }
  ]
  profiles := [{ evidenceProfile with kinds := [{
    id := eventKind
    fields := evidenceProfile.kinds.flatMap EvidenceKindDeclaration.fields ++ [
      roleFieldSpec.declaration
    ]
  }] }]
}

def evaluateFixture (bundle : SyntheticEvidence) : ObservationResult :=
  match Evidence.checkReading evaluationContext evaluationDeclaration with
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

def completeEvidence : SyntheticEvidence := {
  profile := profileId
  profileVersion := 1
  records := [stepEvidence, initialEvidence]
  closures := [{ kind := eventKind, lastSequence := 2 }]
}

def expectedTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := ModelValue.named operationState "ready"
  steps := [{
    selectedAction := ModelValue.named startAction "start"
    outcome := ModelValue.named successOutcome "ok"
    state := ModelValue.named completedState "done"
    facts := [
      ModelValue.named contributionObservation "contributed",
      ModelValue.named digestObservation "synthetic.digest/v1:3006720707513255331"
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
  evidenceSupports := trace.evidenceSupports
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
    behaviorVersion := "test-observation-verdict/v1"
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

def satisfiedPropertyDeclaration : Property := {
  id := id "test.property.observation.satisfied"
  source
  requires := [verdictCapability]
  clauses := [
    .stateInvariant (id "test.property.observation.satisfied.initial")
      (verdictPattern .state operationState (.equals "ready"))
  ]
}

def violatedPropertyDeclaration : Property := {
  satisfiedPropertyDeclaration with
  id := id "test.property.observation.violated"
  clauses := [
    .stateInvariant (id "test.property.observation.violated.initial")
      (verdictPattern .state operationState (.equals "not-ready"))
  ]
}

def repeatedPropertyDeclaration : Property := {
  satisfiedPropertyDeclaration with
  id := id "test.property.observation.repeated"
  clauses := [
    .inputOutput (id "test.property.observation.repeated.step")
      (verdictPattern .selectedAction startAction)
      (verdictPattern .outcome successOutcome)
  ]
}

def logicalTimePropertyDeclaration : Property := {
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
  (Property.check verdictPropertyContext (satisfiedPropertyDeclaration))
    |>.toOption.get (by native_decide)

def violatedProperty : CheckedProperty :=
  (Property.check verdictPropertyContext (violatedPropertyDeclaration))
    |>.toOption.get (by native_decide)

def repeatedProperty : CheckedProperty :=
  (Property.check verdictPropertyContext (repeatedPropertyDeclaration))
    |>.toOption.get (by native_decide)

def logicalTimeProperty : CheckedProperty :=
  (Property.check verdictPropertyContext (logicalTimePropertyDeclaration))
    |>.toOption.get (by native_decide)

def guardedPropertyDeclaration : Property := {
  satisfiedPropertyDeclaration with
  id := id "test.property.observation.guarded"
  version := 2
  clauses := [.branches {
    id := id "test.property.observation.guarded.group"
    source
    guard := .atom {
      field := .selectedAction
      reference := startAction
      constraint := .equals (.text "start")
    }
    cases := [{
      id := id "test.property.observation.guarded.case"
      source
      guard := .atom {
        field := .priorState
        reference := operationState
        constraint := .equals (.text "ready")
      }
      clauses := [{
        id := id "test.property.observation.guarded.case.state"
        source
        expectation := .atom {
          field := .resultingState
          reference := completedState
          constraint := .equals (.text "done")
        }
      }]
    }]
    complete := true
    exclusive := true
  }]
}

def guardedTemporalPropertyDeclaration : Property := {
  satisfiedPropertyDeclaration with
  id := id "test.property.observation.guarded-temporal"
  version := 2
  clauses := [.eventuallyWithin
      (id := (id "test.property.observation.guarded-temporal.clause"))
      (source := source)
      (guard := some (.atom {
      field := .selectedAction
      reference := startAction
      constraint := .equals (.text "start")
    }))
      (exception := none)
      (trigger := (verdictPattern .selectedAction startAction))
      (response := (verdictPattern .outcome successOutcome))
      (limit := { value := 0, unit := .steps })]
}

def checkedQueryTemplate : CheckedQuery Umpire.Examples.Switch.LawStatement :=
  Umpire.Examples.Switch.exploratoryQuery

def verdictQuery
    (properties : List CheckedProperty) : CheckedQuery Umpire.Examples.Switch.LawStatement := {
  checkedQueryTemplate with form := .pick properties
}

def evaluationDiagnostic (kind : ObservationFailureKind) : ObservationDiagnostic := {
  kind
  planId := evaluationDeclaration.id
}

end Umpire.EvidenceTests
