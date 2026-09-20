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

-- A state is its own definition under the commands, so the evidence names one rule per position.
def stateOffRuleId : DefinitionId := id "test.implementation-link.rule.state-off"
def stateOnRuleId : DefinitionId := id "test.implementation-link.rule.state-on"
def actionRuleId : DefinitionId := id "test.implementation-link.rule.action"
def outcomeRuleId : DefinitionId := id "test.implementation-link.rule.outcome"
def observationRuleId : DefinitionId := id "test.implementation-link.rule.observation"

def observationDeclaration : Evidence.Reading := {
  id := id "test.implementation-link.observation"
  source
  profile := profileId
  rules := [
    {
      id := stateOffRuleId
      output := Umpire.Examples.Switch.offState.definitionId
      outputKind := .state
      value := .portable (field stateField)
      condition := some (.portable (.equals (field stateField) (.text "off")))
    },
    {
      id := stateOnRuleId
      output := Umpire.Examples.Switch.onState.definitionId
      outputKind := .state
      value := .portable (field stateField)
      condition := some (.portable (.equals (field stateField) (.text "on")))
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
      outputKind := .fact
      value := .portable (field observationField)
      condition := some stepCondition
    }
  ]
  ordering := [
    { before := actionRuleId, after := outcomeRuleId },
    { before := outcomeRuleId, after := stateOffRuleId },
    { before := outcomeRuleId, after := stateOnRuleId },
    { before := stateOffRuleId, after := observationRuleId },
    { before := stateOnRuleId, after := observationRuleId }
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

def observationPlanResult : Except Evidence.ReadingError Evidence.CheckedReading :=
  Evidence.checkReading (Evidence.ReadingContext.ofTarget Umpire.Examples.Switch.target [evidenceProfile])
    observationDeclaration

private theorem observationPlanResult_isSome : observationPlanResult.toOption.isSome = true := by
  native_decide

def observationPlan : Evidence.CheckedReading :=
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

def repeatedEvidence : SyntheticEvidence := {
  profile := profileId
  profileVersion := 1
  records := [
    stepRecord secondStepRecordId 3 firstStepRecordId,
    initialRecord,
    stepRecord firstStepRecordId 2 initialRecordId
  ]
  closures := [{ kind := evidenceKind, lastSequence := 3 }]
}

def impossibleInitialEvidence : SyntheticEvidence := {
  repeatedEvidence with
  records := [initialRecord "on"]
  closures := [{ kind := evidenceKind, lastSequence := 1 }]
}

def impossibleStepEvidence : SyntheticEvidence := {
  repeatedEvidence with
  records := [initialRecord,
    stepRecord firstStepRecordId 2 initialRecordId "applied" "off"]
  closures := [{ kind := evidenceKind, lastSequence := 2 }]
}

private def acceptedTrace? (bundle : SyntheticEvidence) : Option EvidenceBackedTrace :=
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
  evidenceSupports := trace.evidenceSupports
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

def capabilityMapping : ImplementationSemanticMapping :=
  .forward capabilityReference capabilityReference

/-- The switch's two enumerated rows are relation definitions of its target that no provider means,
so no semantic reference reaches them; the link records each as a relation Known Gap. -/
def relationGap (relationId : DefinitionId) : UnmappedSource DefinitionId := {
  source := relationId
  code := id "test.implementation-link.gap.unmeant-relation"
  reason := "an enumerated row is a relation definition without a provided meaning"
}

def linkDeclaration : ImplementationLinkDeclaration
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := id "test.implementation-link.switch-identity"
  source
  sourceTarget := .ofTarget Umpire.Examples.Switch.target
  destinationTarget := .ofTarget Umpire.Examples.Switch.target
  setupMappings := [
    .forward Umpire.Examples.Switch.switchSetup Umpire.Examples.Switch.switchSetup
  ]
  stateMappings := [
    .forward Umpire.Examples.Switch.offState Umpire.Examples.Switch.offState,
    .forward Umpire.Examples.Switch.onState Umpire.Examples.Switch.onState
  ]
  actionMappings := [
    .forward Umpire.Examples.Switch.flipAction Umpire.Examples.Switch.flipAction
  ]
  outcomeMappings := [
    .forward Umpire.Examples.Switch.appliedOutcome Umpire.Examples.Switch.appliedOutcome,
    .forward Umpire.Examples.Switch.deferredOutcome Umpire.Examples.Switch.deferredOutcome
  ]
  observationMappings := [
    .forward Umpire.Examples.Switch.powerOffObservation
      Umpire.Examples.Switch.powerOffObservation,
    .forward Umpire.Examples.Switch.powerOnObservation Umpire.Examples.Switch.powerOnObservation
  ]
  relationMappings := []
  relationKnownGaps := [
    relationGap Umpire.Examples.Switch.offFlipRelationId,
    relationGap Umpire.Examples.Switch.onFlipRelationId
  ]
  capabilityMappings := [capabilityMapping]
  applicationLimit := { value := 3, unit := .steps }
}

theorem linkCoverage : ImplementationLinkRequiredCoverage linkDeclaration
    Umpire.Examples.Switch.target (fun value => value) (fun value => value)
    (fun value => value) (fun value => value) (fun value => value) := {
  -- Each domain lemma names the switch's members, and each member's identity mapping is at a fixed
  -- position of the declaration's list, so membership is exhibited rather than searched for: the
  -- values are the command's own and nothing here evaluates them.
  setup := by
    intro value admitted
    obtain rfl := Umpire.Examples.Switch.target_setupDomain value admitted
    exact .inl (List.Mem.head _)
  state := by
    intro value admitted
    rcases Umpire.Examples.Switch.target_stateDomain value admitted with rfl | rfl
    · exact .inl (List.Mem.head _)
    · exact .inl (List.Mem.tail _ (List.Mem.head _))
  action := by
    intro value admitted
    obtain rfl := Umpire.Examples.Switch.target_actionDomain value admitted
    exact .inl (List.Mem.head _)
  outcome := by
    intro value admitted
    rcases Umpire.Examples.Switch.target_outcomeDomain value admitted with rfl | rfl
    · exact .inl (List.Mem.head _)
    · exact .inl (List.Mem.tail _ (List.Mem.head _))
  observation := by
    intro value admitted
    rcases Umpire.Examples.Switch.target_observationDomain value admitted with rfl | rfl
    · exact .inl (List.Mem.head _)
    · exact .inl (List.Mem.tail _ (List.Mem.head _))
  relation := by native_decide
  capability := by native_decide
}

def linkWitness : ImplementationLinkWitness linkDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target := {
  index := implementationLinkWitnessIndex linkDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target
  stepPreservation := {
    morphism := {
      mapSetup := fun value => value
      mapState := fun value => value
      mapAction := fun value => value
      mapOutcome := fun value => value
      mapObservation := fun value => value
    }
    initialForward := by intro _ _ admitted; exact admitted
    stepForward := by
      intro _ _ result admitted
      cases result
      simpa [ValueTranslation.mapStep, Step.map] using admitted
  }
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

def observedEvidence : SyntheticEvidence := {
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
  observationMappings := linkDeclaration.observationMappings ++
    [.forward observedPowerOffObservation observedPowerOffObservation]
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
        translation.evidenceSupports.length,
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
  linkDeclaration with applicationLimit := { value := 1, unit := .steps }
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
  stepPreservation := linkWitness.stepPreservation
  requiredCoverage := limitedCoverage
}

def checkedLimitedLink :=
  (checkImplementationLink limitedDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target limitedWitness).toOption.get (by native_decide)

def gapCode : DefinitionId := id "test.implementation-link.known-gap.deferred"

def gapDeclaration := {
  linkDeclaration with
  outcomeMappings := [
    .forward Umpire.Examples.Switch.appliedOutcome Umpire.Examples.Switch.appliedOutcome
  ]
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
    rcases Umpire.Examples.Switch.target_outcomeDomain value admitted with rfl | rfl
    · exact .inl (List.Mem.head _)
    · exact .inr ⟨_, List.Mem.head _, rfl⟩
  observation := by simpa [gapDeclaration] using linkCoverage.observation
  relation := by native_decide
  capability := by native_decide
}

def gapWitness : ImplementationLinkWitness gapDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target := {
  index := implementationLinkWitnessIndex gapDeclaration Umpire.Examples.Switch.target
    Umpire.Examples.Switch.target
  stepPreservation := linkWitness.stepPreservation
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
  evidenceSupports := repeatedEvidenceTrace.evidenceSupports.mapIdx fun index evidenceSupport =>
    if index == 0 then { evidenceSupport with coordinate := .selectedAction 0 } else evidenceSupport
}

def absentCoordinateTrace : UncheckedEvidenceBackedTrace := {
  repeatedUncheckedEvidenceTrace with evidenceSupports := repeatedEvidenceTrace.evidenceSupports.tail
}

def duplicateCoordinateTrace : UncheckedEvidenceBackedTrace := {
  repeatedUncheckedEvidenceTrace with
  evidenceSupports := repeatedEvidenceTrace.evidenceSupports.head?.toList ++
    repeatedEvidenceTrace.evidenceSupports
}

def contradictoryCoordinateTrace : UncheckedEvidenceBackedTrace := {
  repeatedUncheckedEvidenceTrace with
  evidenceSupports := (repeatedEvidenceTrace.evidenceSupports.head?.map fun evidenceSupport => {
    evidenceSupport with ruleId := id "test.implementation-link.rule.contradiction"
  }).toList ++ repeatedEvidenceTrace.evidenceSupports
}

def mismatchedEvidenceSupportTrace : UncheckedEvidenceBackedTrace := {
  repeatedUncheckedEvidenceTrace with
  evidenceSupports := repeatedEvidenceTrace.evidenceSupports.mapIdx fun index evidenceSupport =>
    if index == 0 then { evidenceSupport with mappingDigest := "sha256:mismatched" }
    else evidenceSupport
}

def limitApplication := applyImplementationLink checkedLimitedLink
  Umpire.Examples.Switch.switchSetup repeatedEvidenceTrace

def knownGapApplication := applyImplementationLink checkedGapLink
  Umpire.Examples.Switch.switchSetup repeatedEvidenceTrace

/-- Malformed unchecked wrappers fail at Observation admission before Link application. -/
def observationAdmissionFailureMatrix :
    List (ObservationStatus × Option ObservationFailureKind) :=
  [invalidCoordinateTrace, absentCoordinateTrace, duplicateCoordinateTrace,
    contradictoryCoordinateTrace, mismatchedEvidenceSupportTrace].map fun trace =>
      match validateEvidenceBackedTrace trace with
      | .ok _ => (.accepted, none)
      | .error diagnostic => (diagnostic.status, some diagnostic.kind)

example : observationAdmissionFailureMatrix = [
  (.unknown, some .absentModelCoordinate),
  (.unknown, some .absentModelCoordinate),
  (.conflict, some .duplicateModelCoordinate),
  (.conflict, some .duplicateModelCoordinate),
  (.conflict, some .inconsistentEvidenceSupport)
] := by
  native_decide

/-- The positive application returns the complete repeated-value trace with one link per position. -/
example : completeApplication.applied?.map (fun application =>
    (application.trace == repeatedEvidenceTrace.trace,
      application.evidenceSupports.map (fun evidenceSupport =>
      (evidenceSupport.coordinate, evidenceSupport.sourceValue, evidenceSupport.destinationValue,
        evidenceSupport.sourceEvidenceSupport.coordinate)))) = some (
    true,
    [
      (.initialState, Umpire.Examples.Switch.offState, Umpire.Examples.Switch.offState,
        .initialState),
      (.selectedAction 1, Umpire.Examples.Switch.flipAction, Umpire.Examples.Switch.flipAction,
        .selectedAction 1),
      (.outcome 1, Umpire.Examples.Switch.deferredOutcome,
        Umpire.Examples.Switch.deferredOutcome, .outcome 1),
      (.state 1, Umpire.Examples.Switch.offState, Umpire.Examples.Switch.offState,
        .state 1),
      (.fact 1 1, Umpire.Examples.Switch.powerOffObservation,
        Umpire.Examples.Switch.powerOffObservation, .fact 1 1),
      (.selectedAction 2, Umpire.Examples.Switch.flipAction, Umpire.Examples.Switch.flipAction,
        .selectedAction 2),
      (.outcome 2, Umpire.Examples.Switch.deferredOutcome,
        Umpire.Examples.Switch.deferredOutcome, .outcome 2),
      (.state 2, Umpire.Examples.Switch.offState, Umpire.Examples.Switch.offState,
        .state 2),
      (.fact 2 1, Umpire.Examples.Switch.powerOffObservation,
        Umpire.Examples.Switch.powerOffObservation, .fact 2 1)
    ]) := by
  native_decide

/-- Evidence Link identities bind their exact positional source evidence and translated fact. -/
example : completeApplication.applied?.map (fun application =>
    application.evidenceSupports.all fun evidenceSupport =>
      evidenceSupport.identity != behaviorFingerprintOf "" &&
        evidenceSupport.sourceEvidenceSupportBehaviorFingerprint ==
          behaviorFingerprintOf (reprStr evidenceSupport.sourceEvidenceSupport)) = some true := by
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
  .evidenceSupportMismatch,
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
    some { value := 1, unit := .steps },
    some 2) := by
  native_decide

/-- Distinct invalid source setups each retain a canonical diagnostic identity that fingerprints
the source setup. The switch's kernel is derived from its finite table, and a table kernel encodes a
setup outside its catalog as the empty key, so two invalid setups fingerprint alike: the identity is
canonical, not distinct between them.
CONSIDER(umpire): render an out-of-catalog setup structurally so two invalid setups diagnose apart. -/
example : setupMismatchApplication.diagnostic?.bind (fun first =>
    otherSetupMismatchApplication.diagnostic?.map fun second =>
      first.hasCanonicalIdentity && second.hasCanonicalIdentity &&
        first.sourceSetupBehaviorFingerprint.isSome &&
        second.sourceSetupBehaviorFingerprint.isSome) =
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
