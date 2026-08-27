import Temporal.Feature.Nexus.Observation

/-! Independent synthetic evidence checks for the ordinary Nexus lifecycle profile. -/

namespace Temporal.Feature.Nexus.ObservationTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Observation

private def id (value : String) : DeclarationId := DeclarationId.of value

private def textField (fieldId : DeclarationId) (value : String) : EvidenceFieldValue := {
  field := fieldId
  value := .text value
}

def initialEvidenceId : DeclarationId := id "temporal.nexus.synthetic.record.initial"
def startEvidenceId : DeclarationId := id "temporal.nexus.synthetic.record.start"

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

def expectedTrace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
  initialState := scheduledState
  steps := [{
    selectedAction := startAction
    modelOutcome := startedOutcome
    resultingState := startedState
    observations := [startedObservation]
  }]
}

private def qualifiedOf (result : QualificationResult) : Option QualifiedTrace :=
  match result with
  | .qualified trace => some trace
  | _ => none

def completeObservation : OfflineObservation :=
  evaluateSyntheticEvidence completeEvidence

private structure DerivationShape where
  coordinate : SemanticCoordinate
  mappingId : DeclarationId
  mappingVersion : Nat
  mappingDigest : String
  profileId : DeclarationId
  profileVersion : Nat
  ruleId : DeclarationId
  evidenceIdentities : List DeclarationId
  bindingIds : List DeclarationId
  orderingSupport : List EvidenceOrderingFact
  closureSupport : List EvidenceClosureFact
  appliedDispositions : List AppliedFieldDisposition
  appliedBound : TypedBound
  meaningDigest : String
  deriving BEq, DecidableEq, Repr

private def derivationShape (derivation : SemanticDerivation) : DerivationShape := {
  coordinate := derivation.coordinate
  mappingId := derivation.mappingId
  mappingVersion := derivation.mappingVersion
  mappingDigest := derivation.mappingDigest
  profileId := derivation.profileId
  profileVersion := derivation.profileVersion
  ruleId := derivation.ruleId
  evidenceIdentities := derivation.evidenceIdentities
  bindingIds := derivation.bindingIds
  orderingSupport := derivation.orderingSupport
  closureSupport := derivation.closureSupport
  appliedDispositions := derivation.appliedDispositions
  appliedBound := derivation.appliedBound
  meaningDigest := derivation.meaningDigest
}

/-- The checked mapping admits exactly the target-owned BasicLifecycle vocabulary. -/
example : checkedPlanResult.isOk = true ∧ checkedPlan.meanings = [
    { declaration := cancelActionId, kind := .action,
      semanticDigest := "temporal-nexus-basic-lifecycle-cancel/v1" },
    { declaration := startActionId, kind := .action,
      semanticDigest := "temporal-nexus-basic-lifecycle-start/v1" },
    { declaration := reportSuccessActionId, kind := .action,
      semanticDigest := "temporal-nexus-basic-lifecycle-report-success/v1" },
    { declaration := lifecycleObservationId, kind := .observation,
      semanticDigest := "temporal-nexus-basic-lifecycle-observation/v2" },
    { declaration := transitionOutcomeId, kind := .outcome,
      semanticDigest := "temporal-nexus-basic-lifecycle-outcome/v2" },
    { declaration := operationStateId, kind := .state,
      semanticDigest := "temporal-nexus-basic-lifecycle-state/v2" }
  ] := by
  native_decide

/-- Closed synthetic evidence qualifies to the independently authored lifecycle trace. -/
example : (qualifiedOf completeObservation.qualification).map QualifiedTrace.trace =
    some expectedTrace := by
  native_decide

/-- Every semantic slot has one independently expected evidence and rule derivation. -/
example : (qualifiedOf completeObservation.qualification).map (fun trace =>
    trace.derivations.map derivationShape) = some [{
    coordinate := .initialState
    mappingId := Mapping.id
    mappingVersion := 1
    mappingDigest := checkedPlan.semanticDigest
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
    mappingDigest := checkedPlan.semanticDigest
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
    mappingDigest := checkedPlan.semanticDigest
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
    mappingDigest := checkedPlan.semanticDigest
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
    mappingDigest := checkedPlan.semanticDigest
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
    QualificationStatus × Option QualificationFailureKind ×
      List SemanticVerdictStatus × StrictQueryStatus :=
  (observation.qualification.status,
    observation.qualification.diagnostic?.map QualificationDiagnostic.kind,
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

/-- Every representative non-success fixture retains its exact qualification and verdict status. -/
example : [
    outcomeShape (evaluateSyntheticEvidence incompleteEvidence),
    outcomeShape (evaluateSyntheticEvidence ambiguousEvidence),
    outcomeShape (evaluateSyntheticEvidence conflictingEvidence),
    outcomeShape (evaluateSyntheticEvidence unsupportedEvidence),
    outcomeShape (evaluateSyntheticEvidence rejectedFieldEvidence)
  ] = [
    (.unknown, some .missingClosure, [.unknown], .incomplete),
    (.unknown, some .compatibleAlternatives, [.unknown], .incomplete),
    (.conflict, some .duplicateEvidenceIdentity, [.conflict], .incomplete),
    (.unsupported, some .profileMismatch, [.unsupported], .incomplete),
    (.unsupported, some .rejectedFieldPresent, [.unsupported], .incomplete)
  ] := by
  native_decide

end Temporal.Feature.Nexus.ObservationTests
