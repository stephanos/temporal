import Umpire.ImplementationLink
import Umpire.Examples.Switch

/-! Total application, source admission, Limit, status, and positional Evidence Link matrices. -/

namespace Umpire.ImplementationLinkApplicationTests

open Umpire

private def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Umpire/ImplementationLink/Tests/Application.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def profileId : DefinitionId := id "test.implementation-link.evidence.profile"
def evidenceKind : DefinitionId := id "test.implementation-link.evidence.kind"
def phaseField : DefinitionId := id "test.implementation-link.evidence.phase"
def stateField : DefinitionId := id "test.implementation-link.evidence.state"
def actionField : DefinitionId := id "test.implementation-link.evidence.action"
def outcomeField : DefinitionId := id "test.implementation-link.evidence.outcome"
def observationField : DefinitionId := id "test.implementation-link.evidence.observation"

def evidenceProfile : EvidenceProfileDeclaration := {
  id := profileId
  source
  kinds := [{
    id := evidenceKind
    fields := [
      { id := phaseField, valueType := .text },
      { id := stateField, valueType := .text },
      { id := actionField, valueType := .text },
      { id := outcomeField, valueType := .text },
      { id := observationField, valueType := .text }
    ]
  }]
}

private def field (fieldId : DefinitionId) : ObservationExpression :=
  .field { kind := evidenceKind, field := fieldId }

private def stepCondition : ObservationExpressionAuthoring :=
  .portable (.equals (field phaseField) (.text "step"))

def stateRuleId : DefinitionId := id "test.implementation-link.rule.state"
def actionRuleId : DefinitionId := id "test.implementation-link.rule.action"
def outcomeRuleId : DefinitionId := id "test.implementation-link.rule.outcome"
def observationRuleId : DefinitionId := id "test.implementation-link.rule.observation"

def observationDeclaration : ObservationMappingDeclaration := {
  id := id "test.implementation-link.observation"
  source
  profile := profileId
  rules := [
    {
      id := stateRuleId
      output := Umpire.Examples.Switch.powerStateId
      outputKind := .state
      value := .portable (field stateField)
    },
    {
      id := actionRuleId
      output := Umpire.Examples.Switch.flipActionId
      outputKind := .action
      value := .portable (field actionField)
      condition := some stepCondition
    },
    {
      id := outcomeRuleId
      output := Umpire.Examples.Switch.deferredOutcomeId
      outputKind := .outcome
      value := .portable (field outcomeField)
      condition := some stepCondition
    },
    {
      id := observationRuleId
      output := Umpire.Examples.Switch.powerObservationId
      outputKind := .observation
      value := .portable (field observationField)
      condition := some stepCondition
    }
  ]
  ordering := [
    { before := actionRuleId, after := outcomeRuleId },
    { before := outcomeRuleId, after := stateRuleId },
    { before := stateRuleId, after := observationRuleId }
  ]
  closures := [{ kind := evidenceKind }]
  dispositions := [
    { field := { kind := evidenceKind, field := phaseField }, disposition := .retain },
    { field := { kind := evidenceKind, field := stateField }, disposition := .retain },
    { field := { kind := evidenceKind, field := actionField }, disposition := .retain },
    { field := { kind := evidenceKind, field := outcomeField }, disposition := .retain },
    { field := { kind := evidenceKind, field := observationField }, disposition := .retain }
  ]
  evidenceBound := { value := 3, unit := .evidenceRecords }
}

def observationPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation (ObservationCheckContext.ofTarget Umpire.Examples.Switch.target [evidenceProfile])
    observationDeclaration

private theorem observationPlanResult_isSome : observationPlanResult.toOption.isSome = true := by
  native_decide

def observationPlan : CheckedObservationPlan :=
  observationPlanResult.toOption.get observationPlanResult_isSome

private def textField (fieldId : DefinitionId) (value : String) : EvidenceFieldValue := {
  field := fieldId
  value := .text value
}

def initialRecordId : DefinitionId := id "test.implementation-link.evidence.initial"
def firstStepRecordId : DefinitionId := id "test.implementation-link.evidence.step-1"
def secondStepRecordId : DefinitionId := id "test.implementation-link.evidence.step-2"

def initialRecord (state : String := "off") : SyntheticEvidenceRecord := {
  id := initialRecordId
  profile := profileId
  profileVersion := 1
  kind := evidenceKind
  sequence := 1
  fields := [textField phaseField "initial", textField stateField state]
}

def stepRecord
    (recordId : DefinitionId)
    (sequence : Nat)
    (parent : DefinitionId)
    (outcome : String := "deferred")
    (state : String := "off")
    (observation : String := "off") : SyntheticEvidenceRecord := {
  id := recordId
  profile := profileId
  profileVersion := 1
  kind := evidenceKind
  sequence
  causalParents := [parent]
  fields := [
    textField phaseField "step",
    textField stateField state,
    textField actionField "flip",
    textField outcomeField outcome,
    textField observationField observation
  ]
}

def repeatedEvidence : EvidenceBundle := {
  profile := profileId
  profileVersion := 1
  records := [
    stepRecord secondStepRecordId 3 firstStepRecordId,
    initialRecord,
    stepRecord firstStepRecordId 2 initialRecordId
  ]
  closures := [{ kind := evidenceKind, lastSequence := 3 }]
}

def impossibleInitialEvidence : EvidenceBundle := {
  repeatedEvidence with
  records := [initialRecord "on"]
  closures := [{ kind := evidenceKind, lastSequence := 1 }]
}

def impossibleStepEvidence : EvidenceBundle := {
  repeatedEvidence with
  records := [initialRecord,
    stepRecord firstStepRecordId 2 initialRecordId "applied" "off"]
  closures := [{ kind := evidenceKind, lastSequence := 2 }]
}

private def acceptedTrace? (bundle : EvidenceBundle) : Option EvidenceBackedTrace :=
  match evaluateEvidence observationPlan bundle with
  | .accepted trace => some trace
  | _ => none

private def uncheckedTraceOf (trace : EvidenceBackedTrace) : UncheckedEvidenceBackedTrace := {
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

def repeatedEvidenceTrace : EvidenceBackedTrace :=
  (acceptedTrace? repeatedEvidence).get (by native_decide)

def impossibleInitialTrace : EvidenceBackedTrace :=
  (acceptedTrace? impossibleInitialEvidence).get (by native_decide)

def impossibleStepTrace : EvidenceBackedTrace :=
  (acceptedTrace? impossibleStepEvidence).get (by native_decide)

def capabilityReference : ImplementationSemanticReference :=
  (implementationSemanticReference? Umpire.Examples.Switch.target
    Umpire.Examples.Switch.switchCapabilityId .capability).get (by native_decide)

def capabilityMapping : ImplementationSemanticMapping := {
  source := capabilityReference
  destination := capabilityReference
}

def linkDeclaration : ImplementationLinkDeclaration
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := id "test.implementation-link.switch-identity"
  source
  sourceTarget := .ofTarget Umpire.Examples.Switch.target
  destinationTarget := .ofTarget Umpire.Examples.Switch.target
  setupMappings := [{
    source := Umpire.Examples.Switch.switchSetup
    destination := Umpire.Examples.Switch.switchSetup
  }]
  stateMappings := [
    { source := Umpire.Examples.Switch.offState, destination := Umpire.Examples.Switch.offState },
    { source := Umpire.Examples.Switch.onState, destination := Umpire.Examples.Switch.onState }
  ]
  actionMappings := [{
    source := Umpire.Examples.Switch.flipAction
    destination := Umpire.Examples.Switch.flipAction
  }]
  outcomeMappings := [
    { source := Umpire.Examples.Switch.appliedOutcome,
      destination := Umpire.Examples.Switch.appliedOutcome },
    { source := Umpire.Examples.Switch.deferredOutcome,
      destination := Umpire.Examples.Switch.deferredOutcome }
  ]
  observationMappings := [
    { source := Umpire.Examples.Switch.powerOffObservation,
      destination := Umpire.Examples.Switch.powerOffObservation },
    { source := Umpire.Examples.Switch.powerOnObservation,
      destination := Umpire.Examples.Switch.powerOnObservation }
  ]
  relationMappings := []
  capabilityMappings := [capabilityMapping]
  applicationLimit := { value := 3, unit := .semanticTransitions }
}

theorem linkCoverage : ImplementationLinkRequiredCoverage linkDeclaration
    Umpire.Examples.Switch.target (fun value => value) (fun value => value)
    (fun value => value) (fun value => value) (fun value => value) := {
  setup := by
    intro value admitted
    change value = Umpire.Examples.Switch.switchSetup at admitted
    subst value
    simp [linkDeclaration]
  state := by
    intro value admitted
    change value = Umpire.Examples.Switch.offState ∨
      value = Umpire.Examples.Switch.onState at admitted
    rcases admitted with rfl | rfl <;> simp [linkDeclaration]
  action := by
    intro value admitted
    change value = Umpire.Examples.Switch.flipAction at admitted
    subst value
    simp [linkDeclaration]
  outcome := by
    intro value admitted
    change value = Umpire.Examples.Switch.appliedOutcome ∨
      value = Umpire.Examples.Switch.deferredOutcome at admitted
    rcases admitted with rfl | rfl <;> simp [linkDeclaration]
  observation := by
    intro value admitted
    change value = Umpire.Examples.Switch.powerOffObservation ∨
      value = Umpire.Examples.Switch.powerOnObservation at admitted
    rcases admitted with rfl | rfl <;> simp [linkDeclaration]
  relation := by native_decide
  capability := by native_decide
}

def linkWitness : ImplementationLinkWitness linkDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target := {
  index := implementationLinkWitnessIndex linkDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target
  mapSetup := fun value => value
  mapState := fun value => value
  mapAction := fun value => value
  mapOutcome := fun value => value
  mapObservation := fun value => value
  initialForward := by intro _ _ admitted; exact admitted
  stepForward := by intro _ _ _ admitted; simpa using admitted
  requiredCoverage := linkCoverage
}

def checkedLinkResult := checkImplementationLink linkDeclaration Umpire.Examples.Switch.target
  Umpire.Examples.Switch.target linkWitness

private theorem checkedLinkResult_isSome : checkedLinkResult.toOption.isSome = true := by
  native_decide

def checkedLink := checkedLinkResult.toOption.get checkedLinkResult_isSome

def observedPowerOffObservation : ModelValue := {
  Umpire.Examples.Switch.powerOffObservation with value := "observed-off"
}

def observedEvidence : EvidenceBundle := {
  repeatedEvidence with
  records := [
    stepRecord secondStepRecordId 3 firstStepRecordId
      (observation := observedPowerOffObservation.value),
    initialRecord,
    stepRecord firstStepRecordId 2 initialRecordId
      (observation := observedPowerOffObservation.value)
  ]
}

def observedTranslationDeclaration : ObservedTraceTranslationDeclaration := {
  id := id "test.implementation-link.switch-observed-translation"
  source
  observationMappings := linkDeclaration.observationMappings ++ [{
    source := observedPowerOffObservation
    destination := observedPowerOffObservation
  }]
}

def checkedObservedTranslationResult :=
  checkObservedTraceTranslation checkedLink observedTranslationDeclaration

private theorem checkedObservedTranslationResult_isSome :
    checkedObservedTranslationResult.toOption.isSome = true := by
  native_decide

def checkedObservedTranslation :=
  checkedObservedTranslationResult.toOption.get checkedObservedTranslationResult_isSome

def observedTrace : EvidenceBackedTrace :=
  (acceptedTrace? observedEvidence).get (by native_decide)

def strictObservedApplication := applyImplementationLink checkedLink
  Umpire.Examples.Switch.switchSetup observedTrace

def checkedObservedApplication := applyObservedTraceTranslation checkedObservedTranslation
  Umpire.Examples.Switch.switchSetup observedTrace

/-- Observed translation accepts an explicitly checked value without claiming Target authority. -/
example :
    strictObservedApplication.status = .invalid ∧
    strictObservedApplication.diagnostic?.map ImplementationLinkDiagnostic.kind =
      some .nonAuthoritativeSourceStep ∧
    checkedObservedApplication.status = .applied ∧
    checkedObservedApplication.translated?.map (fun translation =>
      (translation.trace == observedTrace.trace,
        translation.evidenceLinks.length,
        translation.hasAuthorityClaim)) = some (true, 9, false) := by
  native_decide

private def observedErrorKind?
    (result : Except ObservedTraceTranslationError
      (CheckedObservedTraceTranslation checkedLink)) :
    Option ObservedTraceTranslationErrorKind :=
  match result with
  | .ok _ => none
  | .error translationError => some translationError.kind

def missingObservedMappingDeclaration : ObservedTraceTranslationDeclaration := {
  observedTranslationDeclaration with
  observationMappings := observedTranslationDeclaration.observationMappings.tail
}

def duplicateObservedMappingDeclaration : ObservedTraceTranslationDeclaration := {
  observedTranslationDeclaration with
  observationMappings := observedTranslationDeclaration.observationMappings ++
    [observedPowerOffObservation, observedPowerOffObservation].map fun value => {
      source := value
      destination := value
    }
}

def ambiguousObservedMappingDeclaration : ObservedTraceTranslationDeclaration := {
  observedTranslationDeclaration with
  observationMappings := observedTranslationDeclaration.observationMappings ++ [{
    source := observedPowerOffObservation
    destination := Umpire.Examples.Switch.powerOffObservation
  }]
}

def incompatibleObservedMappingDeclaration : ObservedTraceTranslationDeclaration := {
  observedTranslationDeclaration with
  observationMappings := observedTranslationDeclaration.observationMappings ++ [{
    source := { observedPowerOffObservation with
      definitionId := id "test.implementation-link.observation.unexpected" }
    destination := observedPowerOffObservation
  }]
}

/-- Observed mapping compilation fails closed before any trace translation. -/
example : [
    observedErrorKind? <| checkObservedTraceTranslation checkedLink
      { observedTranslationDeclaration with version := 0 },
    observedErrorKind? <| checkObservedTraceTranslation checkedLink
      missingObservedMappingDeclaration,
    observedErrorKind? <| checkObservedTraceTranslation checkedLink
      duplicateObservedMappingDeclaration,
    observedErrorKind? <| checkObservedTraceTranslation checkedLink
      ambiguousObservedMappingDeclaration,
    observedErrorKind? <| checkObservedTraceTranslation checkedLink
      incompatibleObservedMappingDeclaration
  ] = [
    some .invalidVersion,
    some .incompleteSupportPartition,
    some .duplicateMapping,
    some .ambiguousMapping,
    some .incompatibleMapping
  ] := by
  native_decide

def reorderedObservedTranslation :=
  (checkObservedTraceTranslation checkedLink {
    observedTranslationDeclaration with
    observationMappings := observedTranslationDeclaration.observationMappings.reverse
  }).toOption.get (by native_decide)

def changedObservedTranslation :=
  (checkObservedTraceTranslation checkedLink {
    observedTranslationDeclaration with
    observationMappings := observedTranslationDeclaration.observationMappings.map fun mapping =>
      if mapping.source == observedPowerOffObservation then {
        mapping with destination := { mapping.destination with value := "observed-off-changed" }
      } else mapping
  }).toOption.get (by native_decide)

/-- Source order is non-semantic while every observed value participates in identity. -/
example :
    checkedObservedTranslation.hasCanonicalIdentity ∧
    reorderedObservedTranslation.behaviorFingerprint =
      checkedObservedTranslation.behaviorFingerprint ∧
    changedObservedTranslation.behaviorFingerprint !=
      checkedObservedTranslation.behaviorFingerprint := by
  native_decide

def limitedDeclaration := {
  linkDeclaration with applicationLimit := { value := 1, unit := .semanticTransitions }
}

theorem limitedCoverage : ImplementationLinkRequiredCoverage limitedDeclaration
    Umpire.Examples.Switch.target (fun value => value) (fun value => value)
    (fun value => value) (fun value => value) (fun value => value) := {
  setup := by simpa [limitedDeclaration] using linkCoverage.setup
  state := by simpa [limitedDeclaration] using linkCoverage.state
  action := by simpa [limitedDeclaration] using linkCoverage.action
  outcome := by simpa [limitedDeclaration] using linkCoverage.outcome
  observation := by simpa [limitedDeclaration] using linkCoverage.observation
  relation := by simpa [limitedDeclaration] using linkCoverage.relation
  capability := by simpa [limitedDeclaration] using linkCoverage.capability
}

def limitedWitness : ImplementationLinkWitness limitedDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target := {
  index := implementationLinkWitnessIndex limitedDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target
  mapSetup := fun value => value
  mapState := fun value => value
  mapAction := fun value => value
  mapOutcome := fun value => value
  mapObservation := fun value => value
  initialForward := by intro _ _ admitted; exact admitted
  stepForward := by intro _ _ _ admitted; simpa using admitted
  requiredCoverage := limitedCoverage
}

def checkedLimitedLink :=
  (checkImplementationLink limitedDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target limitedWitness).toOption.get (by native_decide)

def gapCode : DefinitionId := id "test.implementation-link.known-gap.deferred"

def gapDeclaration := {
  linkDeclaration with
  outcomeMappings := linkDeclaration.outcomeMappings.filter fun mapping =>
    mapping.source != Umpire.Examples.Switch.deferredOutcome
  outcomeKnownGaps := [{
    source := Umpire.Examples.Switch.deferredOutcome
    code := gapCode
    reason := "Deferred outcomes are intentionally outside this application."
  }]
}

theorem gapCoverage : ImplementationLinkRequiredCoverage gapDeclaration
    Umpire.Examples.Switch.target (fun value => value) (fun value => value)
    (fun value => value) (fun value => value) (fun value => value) := {
  setup := by simpa [gapDeclaration] using linkCoverage.setup
  state := by simpa [gapDeclaration] using linkCoverage.state
  action := by simpa [gapDeclaration] using linkCoverage.action
  outcome := by
    intro value admitted
    change value = Umpire.Examples.Switch.appliedOutcome ∨
      value = Umpire.Examples.Switch.deferredOutcome at admitted
    rcases admitted with rfl | rfl
    · left
      change ({
        source := Umpire.Examples.Switch.appliedOutcome
        destination := Umpire.Examples.Switch.appliedOutcome
      } : ImplementationValueMapping ModelValue ModelValue) ∈ [{
        source := Umpire.Examples.Switch.appliedOutcome
        destination := Umpire.Examples.Switch.appliedOutcome
      }]
      exact List.Mem.head _
    · right
      refine ⟨{
        source := Umpire.Examples.Switch.deferredOutcome
        code := gapCode
        reason := "Deferred outcomes are intentionally outside this application."
      }, ?_, rfl⟩
      simp [gapDeclaration]
  observation := by simpa [gapDeclaration] using linkCoverage.observation
  relation := by native_decide
  capability := by native_decide
}

def gapWitness : ImplementationLinkWitness gapDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target := {
  index := implementationLinkWitnessIndex gapDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target
  mapSetup := fun value => value
  mapState := fun value => value
  mapAction := fun value => value
  mapOutcome := fun value => value
  mapObservation := fun value => value
  initialForward := by intro _ _ admitted; exact admitted
  stepForward := by intro _ _ _ admitted; simpa using admitted
  requiredCoverage := gapCoverage
}

def checkedGapLink :=
  (checkImplementationLink gapDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target gapWitness).toOption.get (by native_decide)

def completeApplication := applyImplementationLink checkedLink
  Umpire.Examples.Switch.switchSetup repeatedEvidenceTrace

def setupMismatchApplication := applyImplementationLink checkedLink [] repeatedEvidenceTrace

def otherSetupMismatchApplication := applyImplementationLink checkedLink [{
  role := Umpire.Examples.Switch.switchRoleId
  value := Umpire.Examples.Switch.onState
}] repeatedEvidenceTrace

def impossibleInitialApplication := applyImplementationLink checkedLink
  Umpire.Examples.Switch.switchSetup impossibleInitialTrace

def impossibleStepApplication := applyImplementationLink checkedLink
  Umpire.Examples.Switch.switchSetup impossibleStepTrace

private def repeatedUncheckedEvidenceTrace : UncheckedEvidenceBackedTrace :=
  uncheckedTraceOf repeatedEvidenceTrace

def invalidCoordinateTrace : UncheckedEvidenceBackedTrace := {
  repeatedUncheckedEvidenceTrace with
  evidenceLinks := repeatedEvidenceTrace.evidenceLinks.mapIdx fun index evidenceLink =>
    if index == 0 then { evidenceLink with coordinate := .selectedAction 0 } else evidenceLink
}

def absentCoordinateTrace : UncheckedEvidenceBackedTrace := {
  repeatedUncheckedEvidenceTrace with evidenceLinks := repeatedEvidenceTrace.evidenceLinks.tail
}

def duplicateCoordinateTrace : UncheckedEvidenceBackedTrace := {
  repeatedUncheckedEvidenceTrace with
  evidenceLinks := repeatedEvidenceTrace.evidenceLinks.head?.toList ++
    repeatedEvidenceTrace.evidenceLinks
}

def contradictoryCoordinateTrace : UncheckedEvidenceBackedTrace := {
  repeatedUncheckedEvidenceTrace with
  evidenceLinks := (repeatedEvidenceTrace.evidenceLinks.head?.map fun evidenceLink => {
    evidenceLink with ruleId := id "test.implementation-link.rule.contradiction"
  }).toList ++ repeatedEvidenceTrace.evidenceLinks
}

def mismatchedEvidenceLinkTrace : UncheckedEvidenceBackedTrace := {
  repeatedUncheckedEvidenceTrace with
  evidenceLinks := repeatedEvidenceTrace.evidenceLinks.mapIdx fun index evidenceLink =>
    if index == 0 then { evidenceLink with mappingDigest := "sha256:mismatched" }
    else evidenceLink
}

def limitApplication := applyImplementationLink checkedLimitedLink
  Umpire.Examples.Switch.switchSetup repeatedEvidenceTrace

def knownGapApplication := applyImplementationLink checkedGapLink
  Umpire.Examples.Switch.switchSetup repeatedEvidenceTrace

/-- Malformed unchecked wrappers fail at Observation admission before Link application. -/
def observationAdmissionFailureMatrix :
    List (ObservationStatus × Option ObservationFailureKind) :=
  [invalidCoordinateTrace, absentCoordinateTrace, duplicateCoordinateTrace,
    contradictoryCoordinateTrace, mismatchedEvidenceLinkTrace].map fun trace =>
      match validateEvidenceBackedTrace trace with
      | .ok _ => (.accepted, none)
      | .error diagnostic => (diagnostic.status, some diagnostic.kind)

example : observationAdmissionFailureMatrix = [
  (.unknown, some .absentModelCoordinate),
  (.unknown, some .absentModelCoordinate),
  (.conflict, some .duplicateModelCoordinate),
  (.conflict, some .duplicateModelCoordinate),
  (.conflict, some .inconsistentEvidenceLink)
] := by
  native_decide

/-- The positive application returns the complete repeated-value trace with one link per position. -/
example : completeApplication.applied?.map (fun application =>
    (application.trace == repeatedEvidenceTrace.trace,
      application.evidenceLinks.map (fun evidenceLink =>
      (evidenceLink.coordinate, evidenceLink.sourceValue, evidenceLink.destinationValue,
        evidenceLink.sourceEvidenceLink.coordinate)))) = some (
    true,
    [
      (.initialState, Umpire.Examples.Switch.offState, Umpire.Examples.Switch.offState,
        .initialState),
      (.selectedAction 1, Umpire.Examples.Switch.flipAction, Umpire.Examples.Switch.flipAction,
        .selectedAction 1),
      (.modelOutcome 1, Umpire.Examples.Switch.deferredOutcome,
        Umpire.Examples.Switch.deferredOutcome, .modelOutcome 1),
      (.resultingState 1, Umpire.Examples.Switch.offState, Umpire.Examples.Switch.offState,
        .resultingState 1),
      (.observation 1 1, Umpire.Examples.Switch.powerOffObservation,
        Umpire.Examples.Switch.powerOffObservation, .observation 1 1),
      (.selectedAction 2, Umpire.Examples.Switch.flipAction, Umpire.Examples.Switch.flipAction,
        .selectedAction 2),
      (.modelOutcome 2, Umpire.Examples.Switch.deferredOutcome,
        Umpire.Examples.Switch.deferredOutcome, .modelOutcome 2),
      (.resultingState 2, Umpire.Examples.Switch.offState, Umpire.Examples.Switch.offState,
        .resultingState 2),
      (.observation 2 1, Umpire.Examples.Switch.powerOffObservation,
        Umpire.Examples.Switch.powerOffObservation, .observation 2 1)
    ]) := by
  native_decide

/-- Evidence Link identities bind their exact positional source evidence and translated fact. -/
example : completeApplication.applied?.map (fun application =>
    application.evidenceLinks.all fun evidenceLink =>
      evidenceLink.identity != behaviorFingerprintOf "" &&
        evidenceLink.sourceEvidenceLinkBehaviorFingerprint ==
          behaviorFingerprintOf (reprStr evidenceLink.sourceEvidenceLink)) = some true := by
  native_decide

def failureMatrix : List (ImplementationLinkStatus × Option ImplementationLinkFailureKind) := [
  (setupMismatchApplication.status,
    setupMismatchApplication.diagnostic?.map ImplementationLinkDiagnostic.kind),
  (impossibleInitialApplication.status,
    impossibleInitialApplication.diagnostic?.map ImplementationLinkDiagnostic.kind),
  (impossibleStepApplication.status,
    impossibleStepApplication.diagnostic?.map ImplementationLinkDiagnostic.kind),
  (limitApplication.status, limitApplication.diagnostic?.map ImplementationLinkDiagnostic.kind),
  (knownGapApplication.status,
    knownGapApplication.diagnostic?.map ImplementationLinkDiagnostic.kind)
]

/-- Source admission, Limit, and Known Gap failures stay exact for admitted traces. -/
example : failureMatrix = [
  (.invalid, some .sourceSetupMismatch),
  (.invalid, some .nonAuthoritativeSourceInitial),
  (.invalid, some .nonAuthoritativeSourceStep),
  (.unknown, some .limitReached),
  (.unsupported, some .knownGap)
] := by
  native_decide

def allFailureKinds : List ImplementationLinkFailureKind := [
  .staleSourceTarget,
  .staleDestinationTarget,
  .behaviorFingerprintDrift,
  .sourceSetupMismatch,
  .nonAuthoritativeSourceInitial,
  .nonAuthoritativeSourceStep,
  .invalidCoordinate,
  .absentCoordinate,
  .limitReached,
  .duplicateCoordinate,
  .contradictoryCoordinate,
  .multipleMappings,
  .evidenceLinkMismatch,
  .knownGap,
  .unsupportedVocabulary
]

/-- The failure-to-status assignment is exhaustive and caller-independent. -/
example : allFailureKinds.map ImplementationLinkFailureKind.status = [
  .invalid, .invalid, .invalid, .invalid, .invalid, .invalid, .invalid,
  .unknown, .unknown,
  .conflict, .conflict, .conflict, .conflict,
  .unsupported, .unsupported
] := by
  native_decide

/-- No non-success can expose even a prefix of the destination trace to a Property consumer. -/
example : [
  setupMismatchApplication.applied?.isSome,
  impossibleInitialApplication.applied?.isSome,
  impossibleStepApplication.applied?.isSome,
  limitApplication.applied?.isSome,
  knownGapApplication.applied?.isSome
] = List.replicate 5 false := by
  native_decide

/-- Diagnostic identity is the fingerprint of every canonical provenance field. -/
example : limitApplication.diagnostic?.map (fun diagnostic =>
    (diagnostic.hasCanonicalIdentity,
      diagnostic.identity,
      behaviorFingerprintOf (canonicalImplementationLinkDiagnosticJson diagnostic),
      diagnostic.appliedLimit,
      diagnostic.observedCount)) = some (
    true,
    (limitApplication.diagnostic?.get (by native_decide)).identity,
    (limitApplication.diagnostic?.get (by native_decide)).identity,
    some { value := 1, unit := .semanticTransitions },
    some 2) := by
  native_decide

/-- Distinct invalid source setups retain distinct canonical diagnostic identities. -/
example : setupMismatchApplication.diagnostic?.bind (fun first =>
    otherSetupMismatchApplication.diagnostic?.map fun second =>
      first.hasCanonicalIdentity && second.hasCanonicalIdentity &&
        first.sourceSetupBehaviorFingerprint.isSome &&
        second.sourceSetupBehaviorFingerprint.isSome && first.identity != second.identity) =
    some true := by
  native_decide

private def unsupportedKindDiagnostic (kind : DefinitionKind) : ImplementationLinkDiagnostic :=
  let base := setupMismatchApplication.diagnostic?.get (by native_decide)
  let canonicalFields : ImplementationLinkDiagnostic := {
    base with
    kind := .unsupportedVocabulary
    relatedDefinitionIds := [id "test.implementation-link.unsupported-vocabulary"]
    sourceSetupBehaviorFingerprint := none
    unsupportedVocabularyKind := some kind
    identity := behaviorFingerprintOf ""
  }
  { canonicalFields with
    identity := behaviorFingerprintOf (canonicalImplementationLinkDiagnosticJson canonicalFields) }

/-- Unsupported vocabulary kinds participate in canonical diagnostic identity. -/
example : let lawDiagnostic := unsupportedKindDiagnostic .law
    let providerDiagnostic := unsupportedKindDiagnostic .provider
    lawDiagnostic.hasCanonicalIdentity && providerDiagnostic.hasCanonicalIdentity &&
      lawDiagnostic.identity != providerDiagnostic.identity = true := by
  native_decide

end Umpire.ImplementationLinkApplicationTests
