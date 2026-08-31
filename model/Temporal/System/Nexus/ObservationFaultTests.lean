import Temporal.System.Nexus.Observation

/-!
Checked schema and identity mutations for the caller-closure duplicate-delivery Observation.
-/

namespace Temporal.System.Nexus.ObservationFaultTests

open Umpire
open Temporal.System.Nexus.Observation.DuplicateDelivery

example : checkedPlan.hasCanonicalIdentity = true ∧
    checkedPlan.id = Mapping.id ∧
    checkedPlan.profile.id = Profile.id ∧
    mappingBehaviorFingerprint = checkedPlan.behaviorFingerprint ∧
    profileVersion = 1 ∧ mappingVersion = 1 ∧ programVersion = 1 ∧
    profileBehaviorFingerprint != behaviorFingerprintOf "" ∧
    programBehaviorFingerprint != behaviorFingerprintOf "" := by
  native_decide

example : semanticCountRule.value = .portable (.text "2") ∧
    mechanicalCallbackCount = 1 ∧ syntheticContributionCount = 1 ∧
    semanticCancellationCount = 2 := by
  native_decide

private def errorKind? :
    Except DuplicateDeliveryCheckError CheckedObservationPlan →
      Option DuplicateDeliveryCheckErrorKind
  | .ok _ => none
  | .error checkError => some checkError.kind

private def profileWithKinds
    (kinds : List EvidenceKindDeclaration) : EvidenceProfileDeclaration := {
  Profile.declaration with kinds
}

private def participantKind : EvidenceKindDeclaration :=
  (Profile.declaration.kinds.find? fun kind => kind.id == Profile.participantKind).get
    (by native_decide)

private def replaceParticipantKind
    (replacement : EvidenceKindDeclaration) : EvidenceProfileDeclaration :=
  profileWithKinds <| Profile.declaration.kinds.map fun kind =>
    if kind.id == Profile.participantKind then replacement else kind

private def missingSyntheticCountProfile : EvidenceProfileDeclaration :=
  replaceParticipantKind {
    participantKind with
    fields := participantKind.fields.filter fun field =>
      field.id != Profile.syntheticContributionCountField
  }

private def duplicateSyntheticCountProfile : EvidenceProfileDeclaration :=
  replaceParticipantKind {
    participantKind with
    fields := participantKind.fields ++ [{
      id := Profile.syntheticContributionCountField
      valueType := .natural
    }]
  }

private def wrongSyntheticCountKindProfile : EvidenceProfileDeclaration :=
  replaceParticipantKind {
    participantKind with
    fields := participantKind.fields.map fun field =>
      if field.id == Profile.syntheticContributionCountField then
        { field with valueType := .text }
      else field
  }

private def staleFaultProfile : EvidenceProfileDeclaration :=
  profileWithKinds <| Profile.declaration.kinds.map fun kind =>
    if kind.id == Profile.controlReceiptKind then {
      kind with fields := kind.fields.map fun field =>
        if field.id == Profile.faultDefinitionField then
          { field with id := DefinitionId.of "umpire.evidence.field.stale-fault-definition-id" }
        else field
    } else kind

private def withoutSyntheticDisposition : ObservationMappingDeclaration := {
  mappingDeclaration with
  dispositions := mappingDeclaration.dispositions.filter fun disposition =>
    disposition.field.field != Profile.syntheticContributionCountField
}

private def impossibleCountDeclaration : ObservationMappingDeclaration := {
  mappingDeclaration with
  rules := mappingDeclaration.rules.map fun rule =>
    if rule.id == Mapping.cancellationCountRuleId then {
      semanticCountRule with
      condition := some (.portable (.equals
        (.field {
          kind := Profile.participantKind
          field := Profile.cancellationCountField
        })
        (.natural 2)))
    } else rule
}

private def unauthorizedRuleDeclaration : ObservationMappingDeclaration := {
  mappingDeclaration with
  rules := mappingDeclaration.rules ++ [{
    semanticCountRule with
    id := DefinitionId.of "temporal.system.nexus.caller-closure.duplicate-delivery.rule.extra"
  }]
}

def mutationKinds : List (Option DuplicateDeliveryCheckErrorKind) := [
    errorKind? <| checkDuplicateDeliveryObservation missingSyntheticCountProfile
      mappingDeclaration,
    errorKind? <| checkDuplicateDeliveryObservation duplicateSyntheticCountProfile
      mappingDeclaration,
    errorKind? <| checkDuplicateDeliveryObservation wrongSyntheticCountKindProfile
      mappingDeclaration,
    errorKind? <| checkDuplicateDeliveryObservation staleFaultProfile mappingDeclaration,
    errorKind? <| checkDuplicateDeliveryObservation Profile.declaration
      withoutSyntheticDisposition,
    errorKind? <| checkDuplicateDeliveryObservation Profile.declaration
      impossibleCountDeclaration,
    errorKind? <| checkDuplicateDeliveryObservation Profile.declaration
      unauthorizedRuleDeclaration
  ]

/-- Every schema, identity, disposition, count-relation, and output mutation fails compilation. -/
example : mutationKinds = [
    some .observation,
    some .observation,
    some .observation,
    some .observation,
    some .observation,
    some .contractDrift,
    some .observation
  ] := by
  native_decide

private def reorderedProfile : EvidenceProfileDeclaration := {
  Profile.declaration with
  kinds := Profile.declaration.kinds.reverse |>.map fun kind => {
    kind with fields := kind.fields.reverse
  }
}

private def reorderedDeclaration : ObservationMappingDeclaration := {
  mappingDeclaration with
  rules := mappingDeclaration.rules.reverse
  ordering := mappingDeclaration.ordering.reverse
  closures := mappingDeclaration.closures.reverse
  dispositions := mappingDeclaration.dispositions.reverse
}

def reorderedPlan :=
  (checkDuplicateDeliveryObservation reorderedProfile reorderedDeclaration).toOption.get
    (by native_decide)

example : reorderedPlan.behaviorFingerprint = checkedPlan.behaviorFingerprint := by
  native_decide

private def evidenceField
    (field : DefinitionId)
    (value : EvidenceValue) : EvidenceFieldValue := { field, value }

private def record
    (recordId evidenceSource kind : DefinitionId)
    (ordinal : Nat)
    (parents : List DefinitionId)
    (fields : List EvidenceFieldValue) : SyntheticEvidenceRecord := {
  id := recordId
  profile := Profile.id
  profileVersion := 1
  kind
  sequence := ordinal + 1
  origin := some { source := evidenceSource, ordinal }
  causalParents := parents
  fields
}

private def cleanupSource := DefinitionId.of "umpire.evidence.source.cleanup"
private def controlSource := DefinitionId.of "umpire.evidence.source.control-receipt"
private def historySource := DefinitionId.of "umpire.evidence.source.history"
private def participantSource := DefinitionId.of "umpire.evidence.source.participant-output"

private def cleanupRecordId := DefinitionId.of "umpire.runtime.fact.cleanup.fault-fixture"
private def controlRecordId := DefinitionId.of "umpire.runtime.fact.control.fault-fixture"
private def participantRecordId := DefinitionId.of "umpire.runtime.fact.participant.fault-fixture"

private def historyRecordId (ordinal : Nat) : DefinitionId :=
  DefinitionId.of ("umpire.runtime.fact.history.fault-" ++ toString (ordinal + 1))

private def operationCorrelation := "temporal.operation.caller-closure.fault-fixture"
private def runCorrelation := "temporal.run.caller-closure.fault-fixture"
private def workflowCorrelation := "temporal.workflow.caller-closure.fault-fixture"

private def cleanupRecord : SyntheticEvidenceRecord :=
  record cleanupRecordId cleanupSource Profile.cleanupKind 0 [] [
    evidenceField Temporal.System.Nexus.Observation.Profile.openHandleCountField (.natural 0),
    evidenceField Profile.statusField (.text "complete")
  ]

private def controlRecord : SyntheticEvidenceRecord :=
  record controlRecordId controlSource Profile.controlReceiptKind 0 [] [
    evidenceField Profile.actionField (.text "workflow.action.force-close"),
    evidenceField Profile.attemptField (.natural 1),
    evidenceField Profile.occurrenceField (.text forceCloseOccurrenceId.value),
    evidenceField Profile.statusField (.text "accepted"),
    evidenceField Profile.faultDefinitionField (.text faultDefinitionId.value),
    evidenceField Profile.faultReceiptField (.text faultReceiptId.value),
    evidenceField Profile.capabilityDefinitionField (.text cancellationCapabilityId.value),
    evidenceField Profile.operationCorrelationField (.text operationCorrelation)
  ]

private def historyEventTypes : List String := [
  "temporal.history.WorkflowExecutionStarted",
  "temporal.history.NexusOperationCancelRequested",
  "temporal.history.NexusOperationCancelRequestCompleted",
  "temporal.history.WorkflowExecutionCanceled"
]

private def historyRecord (ordinal : Nat) (eventType : String) : SyntheticEvidenceRecord :=
  record (historyRecordId ordinal) historySource Profile.historyKind ordinal
    (if ordinal == 0 then [] else [historyRecordId (ordinal - 1)]) [
      evidenceField Profile.eventIdField (.natural (ordinal + 1)),
      evidenceField Profile.eventTypeField (.text eventType),
      evidenceField Profile.operationCorrelationField (.text operationCorrelation),
      evidenceField Profile.runCorrelationField (.text runCorrelation),
      evidenceField Profile.workflowCorrelationField (.text workflowCorrelation)
    ]

private def historyRecords : List SyntheticEvidenceRecord :=
  (List.zip (List.range historyEventTypes.length) historyEventTypes).map fun pair =>
    historyRecord pair.1 pair.2

private def participantRecord : SyntheticEvidenceRecord :=
  record participantRecordId participantSource Profile.participantKind 0 [] [
    evidenceField Profile.cancellationCountField (.natural mechanicalCallbackCount),
    evidenceField Profile.syntheticContributionCountField (.natural syntheticContributionCount),
    evidenceField Profile.syntheticMarkerField (.text injectedMarker),
    evidenceField Profile.cancellationRequestedCountField (.natural 1),
    evidenceField Profile.cancellationCompletedCountField (.natural 1),
    evidenceField Profile.faultDefinitionField (.text faultDefinitionId.value),
    evidenceField Profile.faultReceiptField (.text faultReceiptId.value),
    evidenceField Profile.capabilityDefinitionField (.text cancellationCapabilityId.value),
    evidenceField Profile.operationCorrelationField (.text operationCorrelation),
    evidenceField Profile.runCorrelationField (.text runCorrelation),
    evidenceField Profile.workflowCorrelationField (.text workflowCorrelation)
  ]

private def closure
    (kind evidenceSource : DefinitionId)
    (recordCount : Nat) : EvidenceClosureFact := {
  kind
  lastSequence := recordCount
  source := some evidenceSource
  recordCount := some recordCount
  byteCount := some 0
}

def completeBundle : EvidenceBundle := {
  profile := Profile.id
  profileVersion := 1
  records := [cleanupRecord, controlRecord] ++ historyRecords ++ [participantRecord]
  closures := [
    closure Profile.cleanupKind cleanupSource 1,
    closure Profile.controlReceiptKind controlSource 1,
    closure Profile.historyKind historySource historyRecords.length,
    closure Profile.participantKind participantSource 1
  ]
  sourceClosed := true
}

def completeObservation : ObservationResult := evaluateEvidence checkedPlan completeBundle

def acceptedTrace? : ObservationResult → Option EvidenceBackedTrace
  | .accepted trace => some trace
  | _ => none

def completeTrace : EvidenceBackedTrace :=
  (acceptedTrace? completeObservation).get (by native_decide)

example : completeObservation.status = .accepted ∧
    completeTrace.trace.steps.map (fun step => step.observations) = [[
      Temporal.System.Nexus.CallerClosure.deliveryObservation,
      { Temporal.System.Nexus.CallerClosure.cancellationCountObservation with value := "2" },
      Temporal.System.Nexus.CallerClosure.ownershipObservation
    ]] := by
  native_decide

end Temporal.System.Nexus.ObservationFaultTests
