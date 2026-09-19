import Temporal.Feature.Nexus.Observation

/-! Independent synthetic evidence checks for the ordinary Nexus lifecycle profile. -/

namespace Temporal.Feature.Nexus.ObservationTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Observation

private def id (value : String) : DefinitionId := DefinitionId.of value

private def textField (fieldId : DefinitionId) (value : String) : EvidenceFieldValue := {
  field := fieldId
  value := .text value
}

def initialEvidenceId : DefinitionId := id "temporal.nexus.synthetic.record.initial"
def startEvidenceId : DefinitionId := id "temporal.nexus.synthetic.record.start"

def initialEvidence : SyntheticEvidenceRecord := {
  id := initialEvidenceId
  profile := Profile.id
  profileVersion := 1
  kind := Profile.lifecycleKind
  sequence := 1
  fields := [
    textField Profile.stateField "scheduled",
    textField Profile.actionField "",
    textField Profile.outcomeField "",
    textField Profile.observationField ""
  ]
}

def startEvidence : SyntheticEvidenceRecord := {
  id := startEvidenceId
  profile := Profile.id
  profileVersion := 1
  kind := Profile.lifecycleKind
  sequence := 2
  causalParents := [initialEvidenceId]
  fields := [
    textField Profile.stateField "started",
    textField Profile.actionField "start",
    textField Profile.outcomeField "started",
    textField Profile.observationField "started"
  ]
}

def completeEvidence : SyntheticEvidence := {
  profile := Profile.id
  profileVersion := 1
  records := [startEvidence, initialEvidence]
  closures := [{ kind := Profile.lifecycleKind, lastSequence := 2 }]
}

def expectedTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := scheduledState
  steps := [{
    selectedAction := startAction
    outcome := startedOutcome
    state := startedState
    facts := [startedObservation]
  }]
}

private def acceptedOf (result : ObservationResult) : Option EvidenceBackedTrace :=
  match result with
  | .accepted trace => some trace
  | _ => none

def completeObservation : OfflineObservation :=
  evaluateSyntheticEvidence completeEvidence

private structure EvidenceSupportShape where
  coordinate : ModelCoordinate
  mappingId : DefinitionId
  mappingVersion : Nat
  mappingDigest : String
  profileId : DefinitionId
  profileVersion : Nat
  ruleId : DefinitionId
  evidenceIdentities : List DefinitionId
  bindingIds : List DefinitionId
  orderingSupport : List EvidenceOrderingFact
  closureSupport : List EvidenceClosureFact
  appliedDispositions : List AppliedFieldDisposition
  appliedBound : EvidenceBound
  meaningDigest : String
  deriving BEq, DecidableEq, Repr

private def evidenceSupportShape (evidenceSupport : EvidenceSupport) : EvidenceSupportShape := {
  coordinate := evidenceSupport.coordinate
  mappingId := evidenceSupport.mappingId
  mappingVersion := evidenceSupport.mappingVersion
  mappingDigest := evidenceSupport.mappingDigest
  profileId := evidenceSupport.profileId
  profileVersion := evidenceSupport.profileVersion
  ruleId := evidenceSupport.ruleId
  evidenceIdentities := evidenceSupport.evidenceIdentities
  bindingIds := evidenceSupport.bindingIds
  orderingSupport := evidenceSupport.orderingSupport
  closureSupport := evidenceSupport.closureSupport
  appliedDispositions := evidenceSupport.appliedDispositions
  appliedBound := evidenceSupport.appliedBound
  meaningDigest := evidenceSupport.meaningDigest
}

private def rawProfileDeclaration : EvidenceProfileDeclaration := {
  id := Profile.id
  source := Temporal.Feature.Nexus.Observation.source
  kinds := [{
    id := Profile.lifecycleKind
    fields := [
      { id := Profile.stateField, valueType := .text },
      { id := Profile.actionField, valueType := .text },
      { id := Profile.outcomeField, valueType := .text },
      { id := Profile.observationField, valueType := .text },
      { id := Profile.rejectedField, valueType := .text }
    ]
  }]
}

private def rawRule
    (ruleId output : DefinitionId)
    (outputKind : DefinitionKind)
    (fieldId : DefinitionId)
    (condition : ObservationExpression) : ObservationRule := {
  id := ruleId
  output
  outputKind
  value := .portable (.field { kind := Profile.lifecycleKind, field := fieldId })
  condition := some (.portable condition)
}

private def rawEqualsText (fieldId : DefinitionId) (value : String) : ObservationExpression :=
  .equals (.field { kind := Profile.lifecycleKind, field := fieldId }) (.text value)

private def rawEqualsAny (fieldId : DefinitionId) : List String → ObservationExpression
  | [] => .boolean false
  | value :: values =>
      values.foldl (fun condition candidate =>
        .or condition (rawEqualsText fieldId candidate)) (rawEqualsText fieldId value)

private def rawMappingDeclaration : Evidence.Reading := {
  id := Mapping.id
  source := Temporal.Feature.Nexus.Observation.source
  profile := Profile.id
  rules := [
    rawRule Mapping.stateRuleId operationStateId .state Profile.stateField
      (rawEqualsAny Profile.stateField [
        scheduledState.value, startedState.value, canceledState.value, succeededState.value
      ]),
    rawRule Mapping.startRuleId startActionId .action Profile.actionField
      (rawEqualsText Profile.actionField startAction.value),
    rawRule Mapping.cancelRuleId cancelActionId .action Profile.actionField
      (rawEqualsText Profile.actionField cancelAction.value),
    rawRule Mapping.succeedRuleId reportSuccessActionId .action Profile.actionField
      (rawEqualsText Profile.actionField reportSuccessAction.value),
    rawRule Mapping.outcomeRuleId transitionOutcomeId .outcome Profile.outcomeField
      (rawEqualsAny Profile.outcomeField [
        startedOutcome.value, canceledOutcome.value, succeededOutcome.value
      ]),
    rawRule Mapping.observationRuleId lifecycleObservationId .fact Profile.observationField
      (rawEqualsAny Profile.observationField [
        startedObservation.value, canceledObservation.value, succeededObservation.value
      ])
  ]
  ordering := [
    { before := Mapping.startRuleId, after := Mapping.cancelRuleId },
    { before := Mapping.cancelRuleId, after := Mapping.succeedRuleId },
    { before := Mapping.succeedRuleId, after := Mapping.outcomeRuleId },
    { before := Mapping.outcomeRuleId, after := Mapping.stateRuleId },
    { before := Mapping.stateRuleId, after := Mapping.observationRuleId }
  ]
  closures := [{ kind := Profile.lifecycleKind }]
  dispositions := [
    { field := { kind := Profile.lifecycleKind, field := Profile.stateField },
      disposition := .retain },
    { field := { kind := Profile.lifecycleKind, field := Profile.actionField },
      disposition := .retain },
    { field := { kind := Profile.lifecycleKind, field := Profile.outcomeField },
      disposition := .retain },
    { field := { kind := Profile.lifecycleKind, field := Profile.observationField },
      disposition := .retain },
    { field := { kind := Profile.lifecycleKind, field := Profile.rejectedField },
      disposition := .reject }
  ]
  evidenceBound := { value := 2, unit := .evidenceRecords }
  documentation := "Synthetic scheduled-to-terminal evidence for the ordinary Nexus lifecycle."
}

private def rawCheckedPlanResult : Except Evidence.ReadingError Evidence.CheckedReading :=
  Evidence.checkReading
    (Evidence.ReadingContext.ofTarget target [rawProfileDeclaration]) rawMappingDeclaration

/-- Typed construction reproduces the established raw profile, mapping, and checked plan exactly. -/
example : Profile.spec.declaration = rawProfileDeclaration := by
  native_decide

example : Mapping.spec.declaration = rawMappingDeclaration := by
  native_decide

example : checkedPlan = rawCheckedPlanResult.toOption.get (by native_decide) := by
  native_decide

/-- Checked Observation authoring returns the typed checker's complete canonical plan. -/
example : checkedPlan = checkedPlanResult.toOption.get (by native_decide) := by
  native_decide

/-- Field specifications retain the authored profile shape and checked mapping identity. -/
example :
    Profile.declaration.kinds = [{
      id := Profile.lifecycleKind
      fields := [
        Profile.stateFieldSpec.declaration,
        Profile.actionFieldSpec.declaration,
        Profile.outcomeFieldSpec.declaration,
        Profile.observationFieldSpec.declaration,
        Profile.rejectedFieldSpec.declaration
      ]
    }] ∧
    checkedPlan.source = Temporal.Feature.Nexus.Observation.source ∧
    checkedPlan.behaviorFingerprint.render =
      "sha256:efbeb9aac712c8fe820ccdcd7c46f22824e2459fdafed48ee68101cdbfca55e2" := by
  native_decide

/-- The checked mapping admits exactly the target-owned BasicLifecycle vocabulary. -/
example : checkedPlanResult.isOk = true ∧ checkedPlan.meanings = [
    { definitionId := cancelActionId, kind := .action,
      behaviorVersion := "temporal-nexus-basic-lifecycle-cancel/v1" },
    { definitionId := startActionId, kind := .action,
      behaviorVersion := "temporal-nexus-basic-lifecycle-start/v1" },
    { definitionId := reportSuccessActionId, kind := .action,
      behaviorVersion := "temporal-nexus-basic-lifecycle-report-success/v1" },
    { definitionId := lifecycleObservationId, kind := .fact,
      behaviorVersion := "temporal-nexus-basic-lifecycle-observation/v2" },
    { definitionId := transitionOutcomeId, kind := .outcome,
      behaviorVersion := "temporal-nexus-basic-lifecycle-outcome/v2" },
    { definitionId := operationStateId, kind := .state,
      behaviorVersion := "temporal-nexus-basic-lifecycle-state/v2" }
  ] := by
  native_decide

/-- Closed synthetic Evidence is accepted as the independently authored lifecycle trace. -/
example : (acceptedOf completeObservation.evaluation).map EvidenceBackedTrace.trace =
    some expectedTrace := by
  native_decide

/-- Every Model Trace slot has one independently expected Evidence record and rule Evidence Link. -/
example : (acceptedOf completeObservation.evaluation).map (fun trace =>
    trace.evidenceSupports.map evidenceSupportShape) = some [{
    coordinate := .initialState
    mappingId := Mapping.id
    mappingVersion := 1
    mappingDigest := checkedPlan.behaviorFingerprint.render
    profileId := Profile.id
    profileVersion := 1
    ruleId := Mapping.stateRuleId
    evidenceIdentities := [initialEvidenceId]
    bindingIds := []
    orderingSupport := [{
      recordId := initialEvidenceId
      kind := Profile.lifecycleKind
      sequence := 1
      causalParents := []
    }]
    closureSupport := [{ kind := Profile.lifecycleKind, lastSequence := 2 }]
    appliedDispositions := [{
      field := { kind := Profile.lifecycleKind, field := Profile.stateField }
      evidence := .retained "scheduled"
    }]
    appliedBound := { value := 2, unit := .evidenceRecords }
    meaningDigest := "temporal-nexus-basic-lifecycle-state/v2"
  }, {
    coordinate := .selectedAction 1
    mappingId := Mapping.id
    mappingVersion := 1
    mappingDigest := checkedPlan.behaviorFingerprint.render
    profileId := Profile.id
    profileVersion := 1
    ruleId := Mapping.startRuleId
    evidenceIdentities := [startEvidenceId]
    bindingIds := []
    orderingSupport := [{
      recordId := startEvidenceId
      kind := Profile.lifecycleKind
      sequence := 2
      causalParents := [initialEvidenceId]
    }]
    closureSupport := [{ kind := Profile.lifecycleKind, lastSequence := 2 }]
    appliedDispositions := [{
      field := { kind := Profile.lifecycleKind, field := Profile.actionField }
      evidence := .retained "start"
    }]
    appliedBound := { value := 2, unit := .evidenceRecords }
    meaningDigest := "temporal-nexus-basic-lifecycle-start/v1"
  }, {
    coordinate := .outcome 1
    mappingId := Mapping.id
    mappingVersion := 1
    mappingDigest := checkedPlan.behaviorFingerprint.render
    profileId := Profile.id
    profileVersion := 1
    ruleId := Mapping.outcomeRuleId
    evidenceIdentities := [startEvidenceId]
    bindingIds := []
    orderingSupport := [{
      recordId := startEvidenceId
      kind := Profile.lifecycleKind
      sequence := 2
      causalParents := [initialEvidenceId]
    }]
    closureSupport := [{ kind := Profile.lifecycleKind, lastSequence := 2 }]
    appliedDispositions := [{
      field := { kind := Profile.lifecycleKind, field := Profile.outcomeField }
      evidence := .retained "started"
    }]
    appliedBound := { value := 2, unit := .evidenceRecords }
    meaningDigest := "temporal-nexus-basic-lifecycle-outcome/v2"
  }, {
    coordinate := .state 1
    mappingId := Mapping.id
    mappingVersion := 1
    mappingDigest := checkedPlan.behaviorFingerprint.render
    profileId := Profile.id
    profileVersion := 1
    ruleId := Mapping.stateRuleId
    evidenceIdentities := [startEvidenceId]
    bindingIds := []
    orderingSupport := [{
      recordId := startEvidenceId
      kind := Profile.lifecycleKind
      sequence := 2
      causalParents := [initialEvidenceId]
    }]
    closureSupport := [{ kind := Profile.lifecycleKind, lastSequence := 2 }]
    appliedDispositions := [{
      field := { kind := Profile.lifecycleKind, field := Profile.stateField }
      evidence := .retained "started"
    }]
    appliedBound := { value := 2, unit := .evidenceRecords }
    meaningDigest := "temporal-nexus-basic-lifecycle-state/v2"
  }, {
    coordinate := .fact 1 1
    mappingId := Mapping.id
    mappingVersion := 1
    mappingDigest := checkedPlan.behaviorFingerprint.render
    profileId := Profile.id
    profileVersion := 1
    ruleId := Mapping.observationRuleId
    evidenceIdentities := [startEvidenceId]
    bindingIds := []
    orderingSupport := [{
      recordId := startEvidenceId
      kind := Profile.lifecycleKind
      sequence := 2
      causalParents := [initialEvidenceId]
    }]
    closureSupport := [{ kind := Profile.lifecycleKind, lastSequence := 2 }]
    appliedDispositions := [{
      field := { kind := Profile.lifecycleKind, field := Profile.observationField }
      evidence := .retained "started"
    }]
    appliedBound := { value := 2, unit := .evidenceRecords }
    meaningDigest := "temporal-nexus-basic-lifecycle-observation/v2"
  }
  ] := by
  native_decide

/-- The unchanged checked Property and Query produce one satisfied strict summary. -/
example :
    (completeObservation.verdicts.map SemanticPropertyVerdict.status,
      completeObservation.verdicts.map (fun verdict =>
        verdict.clauses.map SemanticClauseVerdict.coordinates),
      completeObservation.summary.status) =
    ([.satisfied], [[
      [.selectedAction 1, .fact 1 1],
      [.selectedAction 1, .outcome 1],
      [.selectedAction 1, .state 1]
    ]], .satisfied) := by
  native_decide

private def outcomeShape (observation : OfflineObservation) :
    ObservationStatus × Option ObservationFailureKind ×
      List Evidence.PropertyStatus × QueryStatus :=
  (observation.evaluation.status,
    observation.evaluation.diagnostic?.map ObservationDiagnostic.kind,
    observation.verdicts.map SemanticPropertyVerdict.status,
    observation.summary.status)

def incompleteEvidence : SyntheticEvidence := { completeEvidence with closures := [] }

def ambiguousEvidence : SyntheticEvidence := {
  completeEvidence with
  compatibleAlternatives := [
    { id := id "temporal.nexus.synthetic.interpretation.b",
      evidenceIdentities := [startEvidenceId] },
    { id := id "temporal.nexus.synthetic.interpretation.a",
      evidenceIdentities := [initialEvidenceId] }
  ]
  missingDiscriminator := some (id "temporal.nexus.synthetic.field.discriminator")
}

def conflictingEvidence : SyntheticEvidence := {
  completeEvidence with
  records := [initialEvidence, { startEvidence with id := initialEvidenceId }]
}

def unsupportedEvidence : SyntheticEvidence := {
  completeEvidence with profile := id "temporal.nexus.synthetic.profile.other"
}

def rejectedFieldEvidence : SyntheticEvidence := {
  completeEvidence with
  records := [initialEvidence, { startEvidence with fields := startEvidence.fields ++ [
    textField Profile.rejectedField "must-not-cross-the-boundary"
  ] }]
}

def emptyStateEvidence : SyntheticEvidence := {
  completeEvidence with
  records := [{ initialEvidence with fields := [
    textField Profile.stateField "",
    textField Profile.actionField "",
    textField Profile.outcomeField "",
    textField Profile.observationField ""
  ] }]
  closures := [{ kind := Profile.lifecycleKind, lastSequence := 1 }]
}

def unknownOutcomeEvidence : SyntheticEvidence := {
  completeEvidence with
  records := [initialEvidence, { startEvidence with fields := [
    textField Profile.stateField "started",
    textField Profile.actionField "start",
    textField Profile.outcomeField "not-a-basic-lifecycle-outcome",
    textField Profile.observationField "started"
  ] }]
}

/-- Every representative non-success fixture retains its exact Observation Evaluation and verdict status. -/
example : [
    outcomeShape (evaluateSyntheticEvidence incompleteEvidence),
    outcomeShape (evaluateSyntheticEvidence ambiguousEvidence),
    outcomeShape (evaluateSyntheticEvidence conflictingEvidence),
    outcomeShape (evaluateSyntheticEvidence unsupportedEvidence),
    outcomeShape (evaluateSyntheticEvidence rejectedFieldEvidence),
    outcomeShape (evaluateSyntheticEvidence emptyStateEvidence),
    outcomeShape (evaluateSyntheticEvidence unknownOutcomeEvidence)
  ] = [
    (.unknown, some .missingClosure, [.unknown], .incomplete),
    (.unknown, some .compatibleAlternatives, [.unknown], .incomplete),
    (.conflict, some .duplicateEvidenceIdentity, [.conflict], .incomplete),
    (.unsupported, some .profileMismatch, [.unsupported], .incomplete),
    (.unsupported, some .rejectedFieldPresent, [.unsupported], .incomplete),
    (.unknown, some .missingInitialState, [.unknown], .incomplete),
    (.unknown, some .sequenceGap, [.unknown], .incomplete)
  ] := by
  native_decide

#print axioms Profile.spec
#print axioms Profile.declaration
#print axioms Mapping.spec
#print axioms Mapping.declaration
#print axioms rawMappingDeclaration
#print axioms checkedPlanResult
#print axioms rawCheckedPlanResult
#print axioms checkedPlan

end Temporal.Feature.Nexus.ObservationTests
