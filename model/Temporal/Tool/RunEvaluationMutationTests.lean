import Temporal.Tool.RunEvaluation

/-!
Independent caller-closure mutation oracles for each semantic altitude. The fixtures below are
typed literal Evidence and Model Trace values; expected statuses and identities are not projected
from Run Evaluation output.
-/

namespace Temporal.Tool.RunEvaluationMutationTests

open Umpire

private def id (value : String) : DefinitionId := DefinitionId.of value

private def evidenceField
    (field : DefinitionId)
    (value : EvidenceValue)
    (digestPolicy : Option DefinitionId := none)
    (reportedDigestToken : Option String := none) : EvidenceFieldValue := {
  field
  value
  digestPolicy
  reportedDigestToken
}

private def record
    (recordId source kind : DefinitionId)
    (ordinal : Nat)
    (parents : List DefinitionId)
    (fields : List EvidenceFieldValue) : SyntheticEvidenceRecord := {
  id := recordId
  profile := Temporal.System.Nexus.Observation.Profile.id
  profileVersion := 1
  kind
  sequence := ordinal + 1
  origin := some { source, ordinal }
  causalParents := parents
  fields
}

private def cleanupSource := id "umpire.evidence.source.cleanup"
private def controlSource := id "umpire.evidence.source.control-receipt"
private def historySource := id "umpire.evidence.source.history"
private def participantSource := id "umpire.evidence.source.participant-output"

private def cleanupRecordId := id "umpire.runtime.fact.cleanup.fixture"
private def controlRecordId := id "umpire.runtime.fact.control.fixture"
private def participantRecordId := id "umpire.runtime.fact.participant.fixture"

private def historyRecordId (ordinal : Nat) : DefinitionId :=
  id ("umpire.runtime.fact.history." ++ toString (ordinal + 1))

private def historyEventTypes : List String := [
  "temporal.history.WorkflowExecutionStarted",
  "temporal.history.NexusOperationScheduled",
  "temporal.history.NexusOperationStarted",
  "temporal.history.NexusOperationCancelRequested",
  "temporal.history.NexusOperationCancelRequestCompleted",
  "temporal.history.WorkflowExecutionCanceled"
]

private def cleanupRecord : SyntheticEvidenceRecord :=
  record cleanupRecordId cleanupSource Temporal.System.Nexus.Observation.Profile.cleanupKind 0 [] [
    evidenceField Temporal.System.Nexus.Observation.Profile.openHandleCountField (.natural 0),
    evidenceField Temporal.System.Nexus.Observation.Profile.statusField (.text "complete")
  ]

private def controlRecord : SyntheticEvidenceRecord :=
  record controlRecordId controlSource
      Temporal.System.Nexus.Observation.Profile.controlReceiptKind 0 [] [
    evidenceField Temporal.System.Nexus.Observation.Profile.actionField
      (.text "workflow.action.force-close"),
    evidenceField Temporal.System.Nexus.Observation.Profile.attemptField (.natural 1),
    evidenceField Temporal.System.Nexus.Observation.Profile.occurrenceField
      (.text "workflow-nexus.occurrence.force-close"),
    evidenceField Temporal.System.Nexus.Observation.Profile.statusField (.text "accepted")
  ]

private def historyRecord (ordinal : Nat) (eventType : String) : SyntheticEvidenceRecord :=
  record (historyRecordId ordinal) historySource
      Temporal.System.Nexus.Observation.Profile.historyKind ordinal
      (if ordinal == 0 then [] else [historyRecordId (ordinal - 1)]) [
    evidenceField Temporal.System.Nexus.Observation.Profile.eventIdField (.natural (ordinal + 1)),
    evidenceField Temporal.System.Nexus.Observation.Profile.eventTypeField (.text eventType),
    evidenceField Temporal.System.Nexus.Observation.Profile.operationCorrelationField
      (.text "temporal.operation.caller-closure.fixture"),
    evidenceField Temporal.System.Nexus.Observation.Profile.runCorrelationField
      (.text "temporal.run.caller-closure.fixture"),
    evidenceField Temporal.System.Nexus.Observation.Profile.workflowCorrelationField
      (.text "temporal.workflow.caller-closure.fixture")
  ]

private def historyRecords : List SyntheticEvidenceRecord :=
  (List.zip (List.range historyEventTypes.length) historyEventTypes).map fun pair =>
    historyRecord pair.1 pair.2

private def endpointDigest : String :=
  (behaviorFingerprintOf "endpoint").render

private def participantRecord : SyntheticEvidenceRecord :=
  record participantRecordId participantSource
      Temporal.System.Nexus.Observation.Profile.participantKind 0 [] [
    evidenceField Temporal.System.Nexus.Observation.Profile.cancellationCountField (.natural 1),
    evidenceField Temporal.System.Nexus.Observation.Profile.endpointIdentityField
      (.text endpointDigest)
      (some Temporal.System.Nexus.Observation.Profile.endpointDigestPolicyId)
      (some endpointDigest)
  ]

private def closure
    (kind source : DefinitionId)
    (recordCount : Nat) : EvidenceClosureFact := {
  kind
  lastSequence := recordCount
  source := some source
  recordCount := some recordCount
  byteCount := some 0
}

private def completeBundle : EvidenceBundle := {
  profile := Temporal.System.Nexus.Observation.Profile.id
  profileVersion := 1
  records := [cleanupRecord, controlRecord] ++ historyRecords ++ [participantRecord]
  closures := [
    closure Temporal.System.Nexus.Observation.Profile.cleanupKind cleanupSource 1,
    closure Temporal.System.Nexus.Observation.Profile.controlReceiptKind controlSource 1,
    closure Temporal.System.Nexus.Observation.Profile.historyKind historySource 6,
    closure Temporal.System.Nexus.Observation.Profile.participantKind participantSource 1
  ]
  sourceClosed := true
  closedFieldKinds := [
    Temporal.System.Nexus.Observation.Profile.cleanupKind,
    Temporal.System.Nexus.Observation.Profile.controlReceiptKind,
    Temporal.System.Nexus.Observation.Profile.historyKind,
    Temporal.System.Nexus.Observation.Profile.participantKind
  ]
}

private def systemTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := {
    definitionId := id "temporal.system.nexus.caller-closure.state"
    value := "temporal.history.WorkflowExecutionStarted"
  }
  steps := [{
    selectedAction := {
      definitionId := id "temporal.system.nexus.caller-closure.action"
      value := "force-close"
    }
    modelOutcome := {
      definitionId := id "temporal.system.nexus.caller-closure.outcome"
      value := "upgrade"
    }
    resultingState := {
      definitionId := id "temporal.system.nexus.caller-closure.state"
      value := "temporal.history.WorkflowExecutionCanceled"
    }
    observations := [
      { definitionId := id "temporal.system.nexus.caller-closure.observation.delivery",
        value := "true" },
      { definitionId := id "temporal.system.nexus.caller-closure.observation.cancellation-count",
        value := "1" },
      { definitionId := id "temporal.system.nexus.caller-closure.observation.ownership",
        value := "true" }
    ]
  }]
}

private def featureTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := {
    definitionId := id "workflow-nexus.state.config"
    value := "{ op := NexusAutoClose.OpState.started,\n  policy := NexusAutoClose.Policy.requestCancel,\n  cancels := [NexusAutoClose.Initiator.user],\n  callerOpen := true,\n  slack := false }"
  }
  steps := [{
    selectedAction := {
      definitionId := id "workflow.action.force-close"
      value := "force-close"
    }
    modelOutcome := {
      definitionId := id "nexus.outcome.cancellation-upgraded"
      value := "upgrade"
    }
    resultingState := {
      definitionId := id "workflow-nexus.state.config"
      value := "{ op := NexusAutoClose.OpState.started,\n  policy := NexusAutoClose.Policy.requestCancel,\n  cancels := [NexusAutoClose.Initiator.system],\n  callerOpen := false,\n  slack := false }"
    }
    observations := [
      { definitionId := id "nexus.observation.cancellation-delivered", value := "true" },
      { definitionId := id "nexus.observation.pending-cancellation-count", value := "1" },
      { definitionId := id "workflow-nexus.relation.owns-operation", value := "true" }
    ]
  }]
}

private def observationOf (bundle : EvidenceBundle) : ObservationResult :=
  evaluateEvidence Temporal.System.Nexus.Observation.checkedPlan bundle

private def acceptedTrace? (result : ObservationResult) : Option EvidenceBackedTrace :=
  match result with
  | .accepted trace => some trace
  | _ => none

private def completeObservation := observationOf completeBundle

/-! The raw typed bundle reaches exactly the literal System trace and no Feature value. -/
example : completeObservation.status = .accepted ∧
    (acceptedTrace? completeObservation).map EvidenceBackedTrace.trace = some systemTrace := by
  native_decide

private def permutedObservation := observationOf { completeBundle with
  records := completeBundle.records.reverse }

/-! Record array order is non-semantic; both Observation outputs have the same trace and links. -/
example : acceptedTrace? permutedObservation = acceptedTrace? completeObservation := by
  native_decide

private def missingOrderBundle : EvidenceBundle := {
  completeBundle with
  records := completeBundle.records.map fun evidence =>
    if evidence.id == historyRecordId 5 then
      { evidence with origin := none, causalParents := [] }
    else
      evidence
}

private def partialBundle : EvidenceBundle := { completeBundle with sourceClosed := false }

private def ambiguousBundle : EvidenceBundle := {
  completeBundle with
  compatibleAlternatives := [
    { id := id "temporal.system.nexus.interpretation.first", evidenceIdentities := [controlRecordId] },
    { id := id "temporal.system.nexus.interpretation.second", evidenceIdentities := [controlRecordId] }
  ]
  missingDiscriminator := some (id "temporal.system.nexus.discriminator.receipt")
}

private def conflictBundle : EvidenceBundle := {
  completeBundle with records := completeBundle.records ++ [cleanupRecord]
}

private def clearEndpointRecord : SyntheticEvidenceRecord := {
  participantRecord with
  fields := [
    evidenceField Temporal.System.Nexus.Observation.Profile.cancellationCountField (.natural 1),
    evidenceField Temporal.System.Nexus.Observation.Profile.endpointIdentityField
      (.text "clear-endpoint.example")
  ]
}

private def unsupportedBundle : EvidenceBundle := {
  completeBundle with
  records := completeBundle.records.map fun evidence =>
    if evidence.id == participantRecordId then clearEndpointRecord else evidence
}

private def diagnosticKind? (result : ObservationResult) : Option ObservationFailureKind :=
  result.diagnostic?.map ObservationDiagnostic.kind

/-! Every Observation mutation is owned by Observation and exposes no accepted partial trace. -/
example : [
    (observationOf missingOrderBundle).status,
    (observationOf partialBundle).status,
    (observationOf ambiguousBundle).status,
    (observationOf conflictBundle).status,
    (observationOf unsupportedBundle).status
  ] = [.unknown, .unknown, .unknown, .conflict, .unsupported] ∧
  [
    diagnosticKind? (observationOf missingOrderBundle),
    diagnosticKind? (observationOf partialBundle),
    diagnosticKind? (observationOf ambiguousBundle),
    diagnosticKind? (observationOf conflictBundle),
    diagnosticKind? (observationOf unsupportedBundle)
  ] = [
    some .incomparableOrdering,
    some .missingClosure,
    some .compatibleAlternatives,
    some .duplicateEvidenceIdentity,
    some .digestPolicyMismatch
  ] := by
  native_decide

private def completeTrace : EvidenceBackedTrace :=
  (acceptedTrace? completeObservation).get (by native_decide)

private def missingLinkTrace : EvidenceBackedTrace := {
  completeTrace with evidenceLinks := completeTrace.evidenceLinks.dropLast
}

private def missingLinkResult := applyImplementationLink
  Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
  Temporal.System.Nexus.CallerClosure.setup missingLinkTrace

private def mismatchedLinkTrace : EvidenceBackedTrace := {
  completeTrace with
  evidenceLinks := completeTrace.evidenceLinks.mapIdx fun index evidenceLink =>
    if index == 0 then
      { evidenceLink with mappingDigest := (behaviorFingerprintOf "mismatched-mapping").render }
    else evidenceLink
}

private def mismatchedLinkResult := applyImplementationLink
  Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
  Temporal.System.Nexus.CallerClosure.setup mismatchedLinkTrace

/-! A valid System Observation cannot bypass an absent or mismatched checked Evidence Link. -/
example : missingLinkResult.status = .unknown ∧ missingLinkResult.applied?.isNone ∧
    missingLinkResult.diagnostic?.map ImplementationLinkDiagnostic.kind =
      some .absentCoordinate ∧
    mismatchedLinkResult.status = .conflict ∧ mismatchedLinkResult.applied?.isNone ∧
    mismatchedLinkResult.diagnostic?.map ImplementationLinkDiagnostic.kind =
      some .evidenceLinkMismatch := by
  native_decide

private def appliedLink := applyImplementationLink
  Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
  Temporal.System.Nexus.CallerClosure.setup completeTrace

/-! Feature trace identity is the literal destination trace and every coordinate retains its link. -/
example : appliedLink.applied?.map (fun application =>
    (application.trace, application.evidenceLinks.map ImplementationLinkEvidenceLink.coordinate)) =
    some (featureTrace, [
      .initialState, .selectedAction 1, .modelOutcome 1, .resultingState 1,
      .observation 1 1, .observation 1 2, .observation 1 3
    ]) := by
  native_decide

private def violatedFeatureTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  featureTrace with
  steps := featureTrace.steps.map fun step => {
    step with
    observations := step.observations.map fun observation =>
      if observation.definitionId == id "nexus.observation.pending-cancellation-count" then
        { observation with value := "2" }
      else
        observation
  }
}

/-! The unchanged Feature Property alone owns the final satisfied-to-violated mutation. -/
example :
    (evaluateProperty Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosureProperty
      featureTrace).satisfied = true ∧
    (evaluateProperty Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosureProperty
      violatedFeatureTrace).satisfied = false := by
  native_decide

end Temporal.Tool.RunEvaluationMutationTests
