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

def completeEvidence : EvidenceBundle := {
  profile := Profile.id
  profileVersion := 1
  records := [startEvidence, initialEvidence]
  closures := [{ kind := Profile.lifecycleKind, lastSequence := 2 }]
}

def expectedTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := scheduledState
  steps := [{
    selectedAction := startAction
    modelOutcome := startedOutcome
    resultingState := startedState
    observations := [startedObservation]
  }]
}

private def acceptedOf (result : ObservationResult) : Option EvidenceBackedTrace :=
  match result with
  | .accepted trace => some trace
  | _ => none

def completeObservation : OfflineObservation :=
  evaluateSyntheticEvidence completeEvidence

private structure EvidenceLinkShape where
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

private def evidenceLinkShape (evidenceLink : EvidenceLink) : EvidenceLinkShape := {
  coordinate := evidenceLink.coordinate
  mappingId := evidenceLink.mappingId
  mappingVersion := evidenceLink.mappingVersion
  mappingDigest := evidenceLink.mappingDigest
  profileId := evidenceLink.profileId
  profileVersion := evidenceLink.profileVersion
  ruleId := evidenceLink.ruleId
  evidenceIdentities := evidenceLink.evidenceIdentities
  bindingIds := evidenceLink.bindingIds
  orderingSupport := evidenceLink.orderingSupport
  closureSupport := evidenceLink.closureSupport
  appliedDispositions := evidenceLink.appliedDispositions
  appliedBound := evidenceLink.appliedBound
  meaningDigest := evidenceLink.meaningDigest
}

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
      "sha256:608e4db6c3a29d0f953640621ee34d34e16b0090309e85804e21f0cb21be30a2" := by
  native_decide

/-- The checked mapping admits exactly the target-owned BasicLifecycle vocabulary. -/
example : checkedPlanResult.isOk = true ∧ checkedPlan.meanings = [
    { definitionId := cancelActionId, kind := .action,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-cancel/v1" },
    { definitionId := startActionId, kind := .action,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-start/v1" },
    { definitionId := reportSuccessActionId, kind := .action,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-report-success/v1" },
    { definitionId := lifecycleObservationId, kind := .observation,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-observation/v2" },
    { definitionId := transitionOutcomeId, kind := .outcome,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-outcome/v2" },
    { definitionId := operationStateId, kind := .state,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-state/v2" }
  ] := by
  native_decide

/-- Closed synthetic Evidence is accepted as the independently authored lifecycle trace. -/
example : (acceptedOf completeObservation.evaluation).map EvidenceBackedTrace.trace =
    some expectedTrace := by
  native_decide

/-- Every Model Trace slot has one independently expected Evidence record and rule Evidence Link. -/
example : (acceptedOf completeObservation.evaluation).map (fun trace =>
    trace.evidenceLinks.map evidenceLinkShape) = some [{
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
    coordinate := .modelOutcome 1
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
    coordinate := .resultingState 1
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
    coordinate := .observation 1 1
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
      [.selectedAction 1, .observation 1 1],
      [.selectedAction 1, .modelOutcome 1],
      [.selectedAction 1, .resultingState 1]
    ]], .satisfied) := by
  native_decide

private def outcomeShape (observation : OfflineObservation) :
    ObservationStatus × Option ObservationFailureKind ×
      List SemanticVerdictStatus × StrictQueryStatus :=
  (observation.evaluation.status,
    observation.evaluation.diagnostic?.map ObservationDiagnostic.kind,
    observation.verdicts.map SemanticPropertyVerdict.status,
    observation.summary.status)

def incompleteEvidence : EvidenceBundle := { completeEvidence with closures := [] }

def ambiguousEvidence : EvidenceBundle := {
  completeEvidence with
  compatibleAlternatives := [
    { id := id "temporal.nexus.synthetic.interpretation.b",
      evidenceIdentities := [startEvidenceId] },
    { id := id "temporal.nexus.synthetic.interpretation.a",
      evidenceIdentities := [initialEvidenceId] }
  ]
  missingDiscriminator := some (id "temporal.nexus.synthetic.field.discriminator")
}

def conflictingEvidence : EvidenceBundle := {
  completeEvidence with
  records := [initialEvidence, { startEvidence with id := initialEvidenceId }]
}

def unsupportedEvidence : EvidenceBundle := {
  completeEvidence with profile := id "temporal.nexus.synthetic.profile.other"
}

def rejectedFieldEvidence : EvidenceBundle := {
  completeEvidence with
  records := [initialEvidence, { startEvidence with fields := startEvidence.fields ++ [
    textField Profile.rejectedField "must-not-cross-the-boundary"
  ] }]
}

def emptyStateEvidence : EvidenceBundle := {
  completeEvidence with
  records := [{ initialEvidence with fields := [
    textField Profile.stateField "",
    textField Profile.actionField "",
    textField Profile.outcomeField "",
    textField Profile.observationField ""
  ] }]
  closures := [{ kind := Profile.lifecycleKind, lastSequence := 1 }]
}

def unknownOutcomeEvidence : EvidenceBundle := {
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

end Temporal.Feature.Nexus.ObservationTests
