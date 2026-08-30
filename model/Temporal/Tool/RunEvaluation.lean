import Temporal.Tool.RunEvaluation.Protocol
import Temporal.System.Nexus.ImplementationLink
import Umpire.Observation.Check

/-!
The private evaluator is the sole semantic composition boundary for the local runner. Runtime
JSON is admitted into one closed synthetic bundle before the unchanged Umpire altitude chain runs.
-/

namespace Temporal.Tool.RunEvaluation

open Umpire
open Temporal.Tool.RunEvaluation.Protocol

private def id (value : String) : DefinitionId := DefinitionId.of value

private def value (json : Lean.Json) (field : String) : Except String Lean.Json :=
  match json.getObjVal? field with
  | .ok result => pure result
  | .error _ => throw field

private def textValue (json : Lean.Json) (field : String) : Except String String := do
  match (← value json field).getStr? with
  | .ok result => pure result
  | .error _ => throw field

private def natValue (json : Lean.Json) (field : String) : Except String Nat := do
  match (← value json field).getNat? with
  | .ok result => pure result
  | .error _ => throw field

private def arrayValue (json : Lean.Json) (field : String) : Except String (List Lean.Json) := do
  match (← value json field).getArr? with
  | .ok result => pure result.toList
  | .error _ => throw field

private def exactObject
    (json : Lean.Json)
    (fields : List String)
    (name : String) : Except String Unit := do
  match json.getObj? with
  | .error _ => throw name
  | .ok object =>
      if object.toList.length != fields.length ||
          !(fields.all fun field => (json.getObjVal? field).isOk) then
        throw name

private def idValue (json : Lean.Json) (field : String) : Except String DefinitionId := do
  let result := id (← textValue json field)
  if !result.isNamespaced then throw field
  pure result

private def stringArray (json : Lean.Json) (field : String) : Except String (List String) := do
  (← arrayValue json field).mapM fun item =>
    match item.getStr? with
    | .ok result => pure result
    | .error _ => throw field

private def observationStatusName : ObservationStatus → String
  | .accepted => "accepted"
  | .unknown => "unknown"
  | .conflict => "conflict"
  | .unsupported => "unsupported"

private def observationFailureName : ObservationFailureKind → String
  | .emptyEvidence => "empty-evidence"
  | .evidenceBoundExhausted => "evidence-bound-exhausted"
  | .missingInitialState => "missing-initial-state"
  | .missingClosure => "missing-closure"
  | .sequenceGap => "sequence-gap"
  | .missingCausalParent => "missing-causal-parent"
  | .normalizationFailure => "normalization-failure"
  | .unresolvedBinding => "unresolved-binding"
  | .incomparableOrdering => "incomparable-ordering"
  | .profileMismatch => "profile-mismatch"
  | .profileVersionMismatch => "profile-version-mismatch"
  | .kindMismatch => "kind-mismatch"
  | .fieldMismatch => "field-mismatch"
  | .duplicateEvidenceIdentity => "duplicate-evidence-identity"
  | .contradictoryFact => "contradictory-fact"
  | .contradictoryBinding => "contradictory-binding"
  | .contradictoryOrder => "contradictory-order"
  | .misdirectedFaultReceipt => "misdirected-fault-receipt"
  | .compatibleAlternatives => "compatible-alternatives"
  | .zeroUsableInterpretations => "zero-usable-interpretations"
  | .absentModelCoordinate => "absent-model-coordinate"
  | .duplicateModelCoordinate => "duplicate-model-coordinate"
  | .extraModelCoordinate => "extra-model-coordinate"
  | .inconsistentEvidenceLink => "inconsistent-evidence-link"
  | .unconsumedReference => "unconsumed-reference"
  | .missingClosureSupport => "missing-closure-support"
  | .missingOrderSupport => "missing-order-support"
  | .rawValueLeakage => "raw-value-leakage"
  | .redactedValueLeakage => "redacted-value-leakage"
  | .rejectedValueLeakage => "rejected-value-leakage"
  | .rejectedFieldPresent => "rejected-field-present"
  | .digestPolicyMismatch => "digest-policy-mismatch"
  | .digestCollision => "digest-collision"
  | .disallowedRawMaterial => "disallowed-raw-material"

private def semanticStatusName : SemanticVerdictStatus → String
  | .satisfied => "satisfied"
  | .violated => "violated"
  | .unknown => "unknown"
  | .conflict => "conflict"
  | .unsupported => "unsupported"

private def strictStatusName : StrictQueryStatus → String
  | .satisfied => "satisfied"
  | .violated => "violated"
  | .incomplete => "incomplete"

private def semanticFailureName : SemanticVerdictFailureKind → String
  | .observationEvaluationFailure kind => "observation-evaluation-failure:" ++ observationFailureName kind
  | .semanticTraceUnavailable => "semantic-trace-unavailable"
  | .queryPropertyMismatch => "query-property-mismatch"
  | .invalidEvidenceBound => "invalid-evidence-bound"
  | .missingCapability => "missing-capability"
  | .missingVocabulary => "missing-vocabulary"
  | .ambiguousVocabulary => "ambiguous-vocabulary"
  | .digestMismatch => "digest-mismatch"
  | .missingLogicalTime => "missing-logical-time"

private def artifactCoordinate : ModelCoordinate → ArtifactModelCoordinate
  | .initialState => { kind := "initial-state", step := none, position := none }
  | .selectedAction step => { kind := "selected-action", step := some step, position := none }
  | .modelOutcome step => { kind := "model-outcome", step := some step, position := none }
  | .resultingState step => { kind := "resulting-state", step := some step, position := none }
  | .observation step position => { kind := "observation", step := some step, position := some position }

private def artifactEvidenceLimit (bound : EvidenceBound) : ArtifactLimit :=
  { value := bound.value, unit := bound.unit.name }

private def artifactLimit (limit : Limit) : ArtifactLimit :=
  { value := limit.value, unit := limit.unit.name }

private def artifactField (field : EvidenceFieldReference) : ArtifactFieldReference := {
  kindDefinitionId := field.kind
  fieldDefinitionId := field.field
}

private def artifactDisposition
    (disposition : AppliedFieldDisposition) : ArtifactAppliedFieldDisposition :=
  match disposition.evidence with
  | .retained normalizedValue =>
      { field := artifactField disposition.field, kind := "retained",
        normalizedValue := some normalizedValue, digestPolicyDefinitionId := none,
        digestToken := none }
  | .redactedContribution =>
      { field := artifactField disposition.field, kind := "redacted-contribution",
        normalizedValue := none, digestPolicyDefinitionId := none, digestToken := none }
  | .digestToken policy token =>
      { field := artifactField disposition.field, kind := "digest-token",
        normalizedValue := none, digestPolicyDefinitionId := some policy,
        digestToken := some token }
  | .raw normalizedValue =>
      { field := artifactField disposition.field, kind := "raw",
        normalizedValue := some normalizedValue, digestPolicyDefinitionId := none,
        digestToken := none }
  | .rejectedMaterial normalizedValue =>
      { field := artifactField disposition.field, kind := "rejected-material",
        normalizedValue := some normalizedValue, digestPolicyDefinitionId := none,
        digestToken := none }

private def artifactEvidenceLink (link : EvidenceLink) : ArtifactEvidenceLink := {
  coordinate := artifactCoordinate link.coordinate
  mappingDefinitionId := link.mappingId
  mappingVersion := link.mappingVersion
  mappingBehaviorFingerprint :=
    (BehaviorFingerprint.parse? link.mappingDigest).getD (behaviorFingerprintOf link.mappingDigest)
  profileDefinitionId := link.profileId
  profileVersion := link.profileVersion
  evidenceDefinitionIds := link.evidenceIdentities
  ruleDefinitionId := link.ruleId
  bindingDefinitionIds := link.bindingIds
  orderingSupport := link.orderingSupport.map fun fact => {
    factDefinitionId := fact.recordId
    kindDefinitionId := fact.kind
    ordinal := fact.sequence
    causalFactDefinitionIds := fact.causalParents
  }
  closureSupport := link.closureSupport.map fun closure => {
    kindDefinitionId := closure.kind
    lastOrdinal := closure.lastSequence
  }
  appliedDispositions := link.appliedDispositions.map artifactDisposition
  appliedLimit := artifactEvidenceLimit link.appliedBound
  meaningBehaviorFingerprint :=
    (BehaviorFingerprint.parse? link.meaningDigest).getD (behaviorFingerprintOf link.meaningDigest)
}

private def artifactObservationDiagnostic
    (diagnostic : ObservationDiagnostic) : ArtifactObservationDiagnostic := {
  kind := observationFailureName diagnostic.kind
  observationPlanDefinitionId := diagnostic.planId
  relatedDefinitionIds := diagnostic.relatedDefinitionIds
  appliedLimit := diagnostic.limit.map artifactEvidenceLimit
  observedCount := diagnostic.observedCount
  alternatives := diagnostic.alternatives
  missingDiscriminatorDefinitionId := diagnostic.missingDiscriminator
}

private def artifactTrace (trace : EvidenceBackedTrace) : ArtifactEvidenceBackedModelTrace := {
  traceId := trace.traceId
  observationPlan := {
    definitionId := trace.checkedPlan.id
    behaviorFingerprint := trace.checkedPlan.behaviorFingerprint
  }
  mappingDefinitionId := trace.mappingId
  mappingVersion := trace.mappingVersion
  mappingBehaviorFingerprint :=
    (BehaviorFingerprint.parse? trace.mappingDigest).getD (behaviorFingerprintOf trace.mappingDigest)
  source := trace.source
  profileDefinitionId := trace.profileId
  profileVersion := trace.profileVersion
  sourceClosed := trace.sourceClosed
  vocabulary := trace.vocabulary.map fun meaning => {
    definitionId := meaning.definitionId
    kind := meaning.kind
    canonicalBehavior := meaning.canonicalBehavior
  }
  appliedLimit := artifactEvidenceLimit trace.appliedBound
  evidenceDefinitionIds := trace.evidenceIdentities
  trace := {
    traceId := trace.traceId
    initialState := trace.trace.initialState
    steps := trace.trace.steps.mapIdx fun position step => {
      position := position + 1
      selectedAction := step.selectedAction
      modelOutcome := step.modelOutcome
      resultingState := step.resultingState
      observations := step.observations
    }
  }
}

private def artifactDeclaredDisposition
    (declaration : FieldDispositionDeclaration) : ArtifactFieldDispositionRecord := {
  field := artifactField declaration.field
  disposition := declaration.disposition.name
  digestPolicyDefinitionId := match declaration.disposition with
    | .hash policy => policy
    | _ => none
}

private def artifactSemanticDiagnostic
    (diagnostic : SemanticVerdictDiagnostic) : ArtifactSemanticVerdictDiagnostic := {
  kind := semanticFailureName diagnostic.kind
  relatedDefinitionIds := diagnostic.relatedDefinitionIds
  observationDiagnostic := diagnostic.observationEvaluation.map artifactObservationDiagnostic
}

private def artifactClause (clause : SemanticClauseVerdict) : ArtifactSemanticClauseVerdict := {
  propertyDefinitionId := clause.propertyId
  clauseDefinitionId := clause.clauseId
  status := semanticStatusName clause.status
  coordinates := clause.coordinates.map artifactCoordinate
  queryLimits := clause.queryLimits
  propertyLimit := clause.propertyLimit.map artifactLimit
  evidenceLimit := artifactEvidenceLimit clause.evidenceBound
  provenanceDefinitionIds := clause.provenance
  evidenceLinks := clause.evidenceLinks.map artifactEvidenceLink
}

private def artifactProperty
    (verdict : SemanticPropertyVerdict) : ArtifactPropertyVerdict := {
  queryDefinitionId := verdict.queryId
  propertyDefinitionId := verdict.propertyId
  propertyBehaviorFingerprint :=
    (BehaviorFingerprint.parse? verdict.propertyDigest).getD
      (behaviorFingerprintOf verdict.propertyDigest)
  traceId := verdict.traceId
  status := semanticStatusName verdict.status
  queryLimits := verdict.queryLimits
  evidenceLimit := verdict.evidenceBound.map artifactEvidenceLimit
  provenanceDefinitionIds := verdict.provenance
  clauses := verdict.clauses.map artifactClause
  diagnostic := verdict.diagnostic.map artifactSemanticDiagnostic
}

private def artifactSummary (summary : StrictQuerySummary) : ArtifactQuerySummary := {
  queryDefinitionId := summary.queryId
  status := strictStatusName summary.status
  queryLimits := summary.queryLimits
  requiredPropertyDefinitionIds := summary.requiredProperties
  propertyVerdicts := summary.verdicts.map artifactProperty
  missingPropertyDefinitionIds := summary.missingProperties
  duplicatePropertyDefinitionIds := summary.duplicateProperties
  unexpectedPropertyDefinitionIds := summary.unexpectedProperties
  divergentPropertyDefinitionIds := summary.divergentProperties
  wrongQueryResultDefinitionIds := summary.wrongQueryResults
  traceIds := summary.traceIds
}

private def artifactTarget
    (reference : ImplementationTargetReference) : ArtifactImplementationTargetReference := {
  definitionId := reference.id
  kind := reference.kind
  behaviorFingerprint := reference.behaviorFingerprint
}

private def artifactImplementationDiagnostic
    (diagnostic : ImplementationLinkDiagnostic) : ArtifactImplementationLinkDiagnostic := {
  kind := diagnostic.kind.name
  coordinate := diagnostic.coordinate.map artifactCoordinate
  relatedDefinitionIds := diagnostic.relatedDefinitionIds
  sourceSetupBehaviorFingerprint := diagnostic.sourceSetupBehaviorFingerprint
  appliedLimit := diagnostic.appliedLimit.map artifactLimit
  observedCount := diagnostic.observedCount
  knownGapCode := diagnostic.knownGapCode
  knownGapReason := diagnostic.knownGapReason
  unsupportedVocabularyKind := diagnostic.unsupportedVocabularyKind
  evidenceLinkBehaviorFingerprint := diagnostic.evidenceLinkBehaviorFingerprint
  identity := diagnostic.identity
}

private def artifactImplementationLink
    (evaluation : Umpire.RunEvaluation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked) :
    ArtifactImplementationLinkRecord := {
  definitionId :=
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.declaration.id
  behaviorFingerprint :=
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.behaviorFingerprint
  sourceTarget := artifactTarget (.ofTarget
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.sourceTarget)
  destinationTarget := artifactTarget (.ofTarget
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.destinationTarget)
  diagnostic := evaluation.implementationLink.bind ImplementationLinkResult.diagnostic? |>.map
    artifactImplementationDiagnostic
}

private def gapKindRank : KnownGapKind → String
  | .capabilityContract => "0"
  | .input => "1"
  | .interpretation => "2"
  | .claim => "3"

private def gapKey (gap : KnownGap) : String :=
  String.intercalate "\u001f" [gapKindRank gap.kind, gap.code.value,
    gap.subject.map DefinitionId.value |>.getD "", gap.detail.getD ""]

private def canonicalGaps (gaps : List KnownGap) : List KnownGap :=
  gaps.mergeSort (fun left right => decide (gapKey left ≤ gapKey right)) |>.eraseDups

private def parseGap (json : Lean.Json) : Except String KnownGap := do
  exactObject json ["kind", "code", "subject", "detail"] "knownGaps"
  let kind ← match KnownGapKind.parse? (← textValue json "kind") with
    | some value => pure value
    | none => throw "knownGaps"
  let subjectValue ← value json "subject"
  let subject ← if subjectValue.isNull then pure none else
    match subjectValue.getStr? with
    | .ok value =>
        let subject := id value
        if subject.isNamespaced then pure (some subject) else throw "knownGaps"
    | .error _ => throw "knownGaps"
  let detailValue ← value json "detail"
  let detail ← if detailValue.isNull then pure none else
    match detailValue.getStr? with
    | .ok value => pure (some value)
    | .error _ => throw "knownGaps"
  pure { kind, code := ← idValue json "code", subject, detail }

private def parseGaps (json : Lean.Json) : Except String (List KnownGap) := do
  let values ← match json.getArr? with
    | .ok values => pure values.toList
    | .error _ => throw "knownGaps"
  let gaps ← values.mapM parseGap
  match validateKnownGaps gaps with
  | .ok _ => pure gaps
  | .error _ => throw "knownGaps"

private structure RawField where
  definitionId : DefinitionId
  disposition : String
  value : EvidenceValue
  deriving BEq

private structure RawFact where
  definitionId : DefinitionId
  sourceDefinitionId : DefinitionId
  ordinal : Nat
  kindDefinitionId : DefinitionId
  causalParents : List DefinitionId
  fields : List RawField
  deriving BEq

private structure RawSource where
  definitionId : DefinitionId
  status : String
  factCount : Nat
  deriving BEq

private def parseField (json : Lean.Json) : Except String RawField := do
  exactObject json ["fieldDefinitionId", "disposition", "value"] "facts.fields"
  let disposition ← textValue json "disposition"
  if disposition != "plain" && disposition != "sha256" then throw "facts.fields.disposition"
  let raw ← value json "value"
  let fieldValue ← match raw.getStr? with
    | .ok value => pure (.text value)
    | .error _ => match raw.getNat? with
      | .ok value => pure (.natural value)
      | .error _ => throw "facts.fields.value"
  pure {
    definitionId := ← idValue json "fieldDefinitionId"
    disposition
    value := fieldValue
  }

private def parseFact (json : Lean.Json) : Except String RawFact := do
  exactObject json ["factDefinitionId", "sourceDefinitionId", "ordinal", "kindDefinitionId",
    "causalFactDefinitionIds", "fields"] "facts"
  let parents ← (← stringArray json "causalFactDefinitionIds").mapM fun value =>
    let parent := id value
    if parent.isNamespaced then pure parent else throw "facts.causalFactDefinitionIds"
  pure {
    definitionId := ← idValue json "factDefinitionId"
    sourceDefinitionId := ← idValue json "sourceDefinitionId"
    ordinal := ← natValue json "ordinal"
    kindDefinitionId := ← idValue json "kindDefinitionId"
    causalParents := parents
    fields := ← (← arrayValue json "fields").mapM parseField
  }

private def parseSource (json : Lean.Json) : Except String RawSource := do
  exactObject json ["sourceDefinitionId", "status", "factCount", "byteCount"] "sources"
  let status ← textValue json "status"
  if status != "closed" && status != "partial" && status != "failed" then
    throw "sources.status"
  pure {
    definitionId := ← idValue json "sourceDefinitionId"
    status
    factCount := ← natValue json "factCount"
  }

private def field? (fact : RawFact) (fieldId : DefinitionId) : Option RawField :=
  fact.fields.find? fun field => field.definitionId == fieldId

private def fieldText? (fact : RawFact) (fieldId : DefinitionId) : Option String := do
  let field ← field? fact fieldId
  match field.value with
  | .text value => some value
  | .natural value => some (toString value)
  | .boolean value => some (if value then "true" else "false")

private def fieldNat? (fact : RawFact) (fieldId : DefinitionId) : Option Nat := do
  let field ← field? fact fieldId
  match field.value with
  | .natural value => some value
  | .text value => value.toNat?
  | .boolean _ => none

private inductive AdapterResult where
  | admitted (bundle : EvidenceBundle)
  | overBound (observed : Nat)
  | unknown
  | conflict
  | unsupported

private def failureBundle : AdapterResult → EvidenceBundle
  | .admitted bundle => bundle
  | .overBound observed => {
      profile := Temporal.System.Nexus.Observation.Profile.id
      profileVersion := 1
      records := (List.range observed).map fun sequence => {
        id := id ("umpire.evidence.limit.record-" ++ toString sequence)
        profile := Temporal.System.Nexus.Observation.Profile.id
        profileVersion := 1
        kind := Temporal.System.Nexus.Observation.Profile.historyKind
        sequence := sequence + 1
        fields := []
      }
      closures := []
    }
  | .unknown => {
      profile := Temporal.System.Nexus.Observation.Profile.id
      profileVersion := 1
      records := []
      closures := []
    }
  | .conflict =>
      let record : SyntheticEvidenceRecord := {
        id := id "umpire.evidence.conflict.duplicate"
        profile := Temporal.System.Nexus.Observation.Profile.id
        profileVersion := 1
        kind := Temporal.System.Nexus.Observation.Profile.historyKind
        sequence := 1
        fields := []
      }
      {
        profile := Temporal.System.Nexus.Observation.Profile.id
        profileVersion := 1
        records := [record, record]
        closures := [
          { kind := Temporal.System.Nexus.Observation.Profile.cleanupKind, lastSequence := 0 },
          { kind := Temporal.System.Nexus.Observation.Profile.controlReceiptKind,
            lastSequence := 0 },
          { kind := Temporal.System.Nexus.Observation.Profile.historyKind, lastSequence := 1 },
          { kind := Temporal.System.Nexus.Observation.Profile.participantKind, lastSequence := 0 }
        ]
      }
  | .unsupported => {
      profile := id "umpire.evidence.profile.unsupported"
      profileVersion := 1
      records := [{
        id := id "umpire.evidence.unsupported.record"
        profile := id "umpire.evidence.profile.unsupported"
        profileVersion := 1
        kind := id "umpire.evidence.kind.unsupported"
        sequence := 1
        fields := []
      }]
      closures := []
    }

private def sourceCleanup := id "umpire.evidence.source.cleanup"
private def sourceControl := id "umpire.evidence.source.control-receipt"
private def sourceHistory := id "umpire.evidence.source.history"
private def sourceParticipant := id "umpire.evidence.source.participant-output"

private def expectedSources : List (DefinitionId × Nat) := [
  (sourceCleanup, 1), (sourceControl, 1), (sourceHistory, 6), (sourceParticipant, 1)
]

private def expectedHistoryEvents : List String := [
  "temporal.history.WorkflowExecutionStarted",
  "temporal.history.NexusOperationScheduled",
  "temporal.history.NexusOperationStarted",
  "temporal.history.NexusOperationCancelRequested",
  "temporal.history.NexusOperationCancelRequestCompleted",
  "temporal.history.WorkflowExecutionCanceled"
]

private def hasExactFields (json : Lean.Json) (fields : List String) : Bool :=
  match exactObject json fields "projection" with
  | .ok _ => true
  | .error _ => false

private def jsonText? (json : Lean.Json) (field : String) : Option String :=
  (json.getObjVal? field).toOption.bind fun value => value.getStr?.toOption

private def jsonNat? (json : Lean.Json) (field : String) : Option Nat :=
  (json.getObjVal? field).toOption.bind fun value => value.getNat?.toOption

private def jsonNull (json : Lean.Json) (field : String) : Bool :=
  (json.getObjVal? field).toOption.any Lean.Json.isNull

private def validPhaseOutcome (expectedPhase : String) (json : Lean.Json) : Bool :=
  hasExactFields json ["phase", "status", "startedAtUnixMillis", "finishedAtUnixMillis", "code"] &&
    jsonText? json "phase" == some expectedPhase &&
    match jsonText? json "status" with
    | some "not-started" => jsonNull json "startedAtUnixMillis" &&
        jsonNull json "finishedAtUnixMillis" && jsonNull json "code"
    | some "succeeded" => (jsonNat? json "startedAtUnixMillis").isSome &&
        (jsonNat? json "finishedAtUnixMillis").isSome && jsonNull json "code"
    | some status =>
        ["failed", "timed-out", "canceled"].contains status &&
          (jsonNat? json "startedAtUnixMillis").isSome &&
          (jsonNat? json "finishedAtUnixMillis").isSome &&
          (jsonText? json "code").any (id · |>.isNamespaced)
    | none => false

private def validPhaseOutcomes (json : Lean.Json) : Bool :=
  match json.getArr? with
  | .error _ => false
  | .ok values =>
      let phases := ["preparation", "realization", "observation", "isolation", "cleanup"]
      values.size == phases.length &&
        (List.zip phases values.toList).all fun pair => validPhaseOutcome pair.1 pair.2

private def validControlAttempts (json : Lean.Json) : Bool :=
  match json.getArr? with
  | .error _ => false
  | .ok values => match values.toList with
    | [attempt] =>
        hasExactFields attempt ["occurrenceDefinitionId", "actionDefinitionId", "attempt",
          "receiptFactDefinitionId", "status", "code"] &&
          (jsonText? attempt "occurrenceDefinitionId").any (id · |>.isNamespaced) &&
          jsonText? attempt "actionDefinitionId" == some "workflow.action.force-close" &&
          jsonNat? attempt "attempt" == some 1 && jsonText? attempt "status" == some "accepted" &&
          (jsonText? attempt "receiptFactDefinitionId").any (id · |>.isNamespaced) &&
          jsonNull attempt "code"
    | _ => false

private def validSourceClosures (json : Lean.Json) : Bool :=
  match json.getArr? with
  | .error _ => false
  | .ok values =>
      values.size == expectedSources.length &&
        (List.zip values.toList expectedSources).all fun pair =>
          hasExactFields pair.1 ["sourceDefinitionId", "status", "recordCount", "byteCount"] &&
            jsonText? pair.1 "sourceDefinitionId" == some pair.2.1.value &&
            jsonText? pair.1 "status" == some "closed" &&
            jsonNat? pair.1 "recordCount" == some pair.2.2 &&
            (jsonNat? pair.1 "byteCount").isSome

private def validRuntimeProjections (request : Request) : Bool :=
  validPhaseOutcomes request.phaseOutcomes && validControlAttempts request.controlAttempts &&
    validSourceClosures request.sourceClosures

private def duplicateIds (facts : List RawFact) : Bool :=
  (facts.map RawFact.definitionId |>.eraseDups |>.length) != facts.length

private def exactSourceFacts
    (sources : List RawSource)
    (facts : List RawFact) : Bool :=
  sources.length == expectedSources.length &&
    (List.zip sources expectedSources).all fun pair =>
      pair.1.definitionId == pair.2.1 && pair.1.status == "closed" &&
        pair.1.factCount == pair.2.2 &&
        (facts.filter fun fact => fact.sourceDefinitionId == pair.1.definitionId).length == pair.2.2

private def exactSourceOrdinals (facts : List RawFact) : Bool :=
  expectedSources.all fun expected =>
    let localFacts := facts.filter fun fact => fact.sourceDefinitionId == expected.1
    localFacts.map RawFact.ordinal == List.range expected.2

private def exactHistoryChain (history : List RawFact) : Bool :=
  history.length == expectedHistoryEvents.length &&
    (List.zip history expectedHistoryEvents).all fun pair =>
      pair.1.kindDefinitionId == Temporal.System.Nexus.Observation.Profile.historyKind &&
        fieldText? pair.1 Temporal.System.Nexus.Observation.Profile.eventTypeField == some pair.2 &&
        fieldNat? pair.1 Temporal.System.Nexus.Observation.Profile.eventIdField ==
          some (pair.1.ordinal + 1) &&
        (if pair.1.ordinal == 0 then pair.1.causalParents.isEmpty
          else pair.1.causalParents ==
            (history[pair.1.ordinal - 1]?.map fun fact => [fact.definitionId]).getD [])

private def commonCorrelation?
    (history : List RawFact)
    (fieldId : DefinitionId) : Option String := do
  let first ← history.head?
  let value ← fieldText? first fieldId
  if value.isEmpty || !(history.all fun fact => fieldText? fact fieldId == some value) then none
  else some value

private def exactField
    (fact : RawFact)
    (fieldId : DefinitionId)
    (disposition : String)
    (expected : EvidenceValue) : Bool :=
  match fact.fields.filter fun field => field.definitionId == fieldId with
  | [field] => field.disposition == disposition && field.value == expected
  | _ => false

private def admittedBundle
    (request : Request)
    (sources : List RawSource)
    (facts : List RawFact) : AdapterResult :=
  if duplicateIds facts then
    .conflict
  else if request.captureStatus != "closed" ||
      !exactSourceFacts sources facts || !exactSourceOrdinals facts then
    .unknown
  else
    let cleanup := facts.filter fun fact => fact.sourceDefinitionId == sourceCleanup
    let control := facts.filter fun fact => fact.sourceDefinitionId == sourceControl
    let history := facts.filter fun fact => fact.sourceDefinitionId == sourceHistory
    let participant := facts.filter fun fact => fact.sourceDefinitionId == sourceParticipant
    if !exactHistoryChain history then
      .conflict
    else match cleanup, control, participant,
        commonCorrelation? history Temporal.System.Nexus.Observation.Profile.operationCorrelationField,
        commonCorrelation? history Temporal.System.Nexus.Observation.Profile.runCorrelationField,
        commonCorrelation? history Temporal.System.Nexus.Observation.Profile.workflowCorrelationField with
      | [cleanup], [control], [participant], some operationCorrelation,
          some runCorrelation, some workflowCorrelation =>
          if runCorrelation != request.runIdentity.value ||
              cleanup.kindDefinitionId != Temporal.System.Nexus.Observation.Profile.cleanupKind ||
              control.kindDefinitionId != Temporal.System.Nexus.Observation.Profile.controlReceiptKind ||
              participant.kindDefinitionId != Temporal.System.Nexus.Observation.Profile.participantKind ||
              !exactField cleanup Temporal.System.Nexus.Observation.Profile.openHandleCountField
                "plain" (.natural 0) ||
              !exactField cleanup Temporal.System.Nexus.Observation.Profile.statusField
                "plain" (.text "complete") ||
              !exactField control Temporal.System.Nexus.Observation.Profile.actionField
                "plain" (.text "workflow.action.force-close") ||
              !exactField control Temporal.System.Nexus.Observation.Profile.attemptField
                "plain" (.natural 1) ||
              !exactField control Temporal.System.Nexus.Observation.Profile.statusField
                "plain" (.text "accepted") ||
              !exactField participant
                Temporal.System.Nexus.Observation.Profile.cancellationCountField
                "plain" (.natural 1) then
            .conflict
          else match participant.fields.filter fun field =>
              field.definitionId == Temporal.System.Nexus.Observation.Profile.endpointIdentityField with
            | [{ disposition := "sha256", value := .text endpoint, .. }] =>
                if !(endpoint.startsWith "sha256:") || endpoint.length != 71 then .unsupported
                else match history.head?, history.getLast? with
                | some initialFact, some terminalFact =>
                  let initialId := initialFact.definitionId
                  let terminalId := terminalFact.definitionId
                  let initial : SyntheticEvidenceRecord := {
                    id := initialId
                    profile := Temporal.System.Nexus.Observation.Profile.id
                    profileVersion := 1
                    kind := Temporal.System.Nexus.Observation.Profile.historyKind
                    sequence := 1
                    fields := [
                      { field := Temporal.System.Nexus.Observation.Profile.actionField,
                        value := .text "not-selected" },
                      { field := Temporal.System.Nexus.Observation.Profile.attemptField,
                        value := .natural 0 },
                      { field := Temporal.System.Nexus.Observation.Profile.statusField,
                        value := .text "not-attempted" },
                      { field := Temporal.System.Nexus.Observation.Profile.eventIdField,
                        value := .natural 1 },
                      { field := Temporal.System.Nexus.Observation.Profile.eventTypeField,
                        value := .text "temporal.history.WorkflowExecutionStarted" },
                      { field := Temporal.System.Nexus.Observation.Profile.operationCorrelationField,
                        value := .text operationCorrelation },
                      { field := Temporal.System.Nexus.Observation.Profile.runCorrelationField,
                        value := .text runCorrelation },
                      { field := Temporal.System.Nexus.Observation.Profile.workflowCorrelationField,
                        value := .text workflowCorrelation },
                      { field := Temporal.System.Nexus.Observation.Profile.cancellationCountField,
                        value := .natural 0 }
                    ]
                  }
                  let terminal : SyntheticEvidenceRecord := {
                    id := terminalId
                    profile := Temporal.System.Nexus.Observation.Profile.id
                    profileVersion := 1
                    kind := Temporal.System.Nexus.Observation.Profile.historyKind
                    sequence := 2
                    causalParents := [initialId]
                    fields := [
                      { field := Temporal.System.Nexus.Observation.Profile.actionField,
                        value := .text "workflow.action.force-close" },
                      { field := Temporal.System.Nexus.Observation.Profile.attemptField,
                        value := .natural 1 },
                      { field := Temporal.System.Nexus.Observation.Profile.statusField,
                        value := .text "accepted" },
                      { field := Temporal.System.Nexus.Observation.Profile.eventIdField,
                        value := .natural 6 },
                      { field := Temporal.System.Nexus.Observation.Profile.eventTypeField,
                        value := .text "temporal.history.WorkflowExecutionCanceled" },
                      { field := Temporal.System.Nexus.Observation.Profile.operationCorrelationField,
                        value := .text operationCorrelation },
                      { field := Temporal.System.Nexus.Observation.Profile.runCorrelationField,
                        value := .text runCorrelation },
                      { field := Temporal.System.Nexus.Observation.Profile.workflowCorrelationField,
                        value := .text workflowCorrelation },
                      { field := Temporal.System.Nexus.Observation.Profile.cancellationCountField,
                        value := .natural 1 }
                    ]
                  }
                  .admitted {
                    profile := Temporal.System.Nexus.Observation.Profile.id
                    profileVersion := 1
                    records := [initial, terminal]
                    closures := [
                      { kind := Temporal.System.Nexus.Observation.Profile.cleanupKind,
                        lastSequence := 0 },
                      { kind := Temporal.System.Nexus.Observation.Profile.controlReceiptKind,
                        lastSequence := 0 },
                      { kind := Temporal.System.Nexus.Observation.Profile.historyKind,
                        lastSequence := 2 },
                      { kind := Temporal.System.Nexus.Observation.Profile.participantKind,
                        lastSequence := 0 }
                    ]
                  }
                | _, _ => .unknown
            | _ => .unsupported
      | _, _, _, _, _, _ => .unknown

private def adapt (request : Request) : AdapterResult :=
  if !validRuntimeProjections request then .unsupported
  else if request.captureStatus != "closed" then .unknown
  else
    match request.sources.getArr?, request.facts.getArr? with
    | .ok sourceValues, .ok factValues =>
        if factValues.size > Temporal.System.Nexus.Observation.checkedPlan.evidenceBound.value then
          .overBound factValues.size
        else match sourceValues.toList.mapM parseSource, factValues.toList.mapM parseFact,
            parseGaps request.runKnownGaps, parseGaps request.rawEvidenceKnownGaps with
          | .ok sources, .ok facts, .ok _, .ok _ => admittedBundle request sources facts
          | .error error, _, _, _ | _, .error error, _, _ =>
              if error.contains "disposition" || error.contains "value" ||
                  error.contains "sources" then .unsupported
              else .unknown
          | _, _, .error _, _ | _, _, _, .error _ => .unsupported
    | _, _ => .unsupported

def evaluateSemantics
    (request : Request) : Umpire.RunEvaluation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked :=
  let adapter := adapt request
  Umpire.checkRunEvaluation Temporal.System.Nexus.Observation.checkedPlan
    (failureBundle adapter)
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
    Temporal.System.Nexus.CallerClosure.setup
    Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQuery
    [Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosureProperty]

private def semanticGaps (observation : ObservationResult) : List KnownGap :=
  observation.diagnostic?.map (fun diagnostic => {
    kind := .interpretation
    code := id ("umpire.observation." ++ observationFailureName diagnostic.kind)
    subject := some diagnostic.planId
  }) |>.toList

private def jsonArray (values : List String) : Lean.Json :=
  (Lean.Json.parse ("[" ++ String.intercalate "," values ++ "]")).toOption.getD (.arr #[])

private def gapsJson (gaps : List KnownGap) : Lean.Json :=
  jsonArray (gaps.map canonicalKnownGapJson)

private def jsonField (json : Lean.Json) (field : String) : Lean.Json :=
  (json.getObjVal? field).toOption.getD .null

private def canonicalJsonValue (text : String) : Lean.Json :=
  (Lean.Json.parse text).toOption.getD .null

private def emptyChecksum : ArtifactChecksum := drivePlanChecksumOf ""

private def provenance : ArtifactProvenance := {
  sourceDefinitionIds := [id "temporal.tool.run-evaluation"]
  sourceLocations := [{
    path := "Temporal/Tool/RunEvaluation.lean"
    line := 1
    column := 1
    provenance := "lean-model"
  }]
}

private def observationProjection
    (evaluation : Umpire.RunEvaluation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked) :
    Option ArtifactEvidenceBackedModelTrace × List ArtifactEvidenceLink ×
      List ArtifactFieldDispositionRecord × List ArtifactObservationDiagnostic :=
  match evaluation.observation with
  | .accepted trace =>
      (some (artifactTrace trace), trace.evidenceLinks.map artifactEvidenceLink,
        trace.dispositions.map artifactDeclaredDisposition, [])
  | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic =>
      (none, [], Temporal.System.Nexus.Observation.checkedPlan.dispositions.map
        artifactDeclaredDisposition, [artifactObservationDiagnostic diagnostic])

private def evidenceArtifact
    (request : Request)
    (evaluation : Umpire.RunEvaluation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked)
    (observationGaps : List KnownGap) : EvidenceArtifact :=
  let projection := observationProjection evaluation
  ({
    formatVersion := "umpire-evidence/v2"
    runIdentity := request.runIdentity
    behaviorFingerprint := behaviorFingerprintOf (reprStr evaluation.observation)
    experiment := request.experiment
    runtimeConfiguration := request.runtimeConfiguration
    run := request.run
    rawEvidence := request.rawEvidence
    observationProgram := {
      definitionId := request.observationProgram.definitionId
      behaviorFingerprint := request.observationProgram.behaviorFingerprint
    }
    mapping := {
      definitionId := request.mapping.definitionId
      behaviorFingerprint := request.mapping.behaviorFingerprint
    }
    observationEvaluationStatus := observationStatusName evaluation.observation.status
    evidenceBackedModelTrace := projection.1
    evidenceLinks := projection.2.1
    dispositions := projection.2.2.1
    diagnostics := projection.2.2.2
    knownGaps := observationGaps
    provenance
    provenanceChecksum := emptyChecksum
    artifactChecksum := emptyChecksum
  } : EvidenceArtifact).seal

private def operationalStatus (request : Request) : String :=
  match request.phaseOutcomes.getArr? with
  | .error _ => "failed"
  | .ok outcomes =>
      if outcomes.toList.all fun outcome =>
          (outcome.getObjVal? "status").toOption.bind
            (fun value => value.getStr?.toOption) == some "succeeded" then "succeeded"
      else "failed"

private def cleanupStatus (request : Request) : String :=
  match request.phaseOutcomes.getArr? with
  | .error _ => "failed"
  | .ok outcomes =>
      let cleanup := outcomes.toList.find? fun outcome =>
        (outcome.getObjVal? "phase").toOption.bind
          (fun value => value.getStr?.toOption) == some "cleanup"
      (cleanup.bind fun outcome =>
        (outcome.getObjVal? "status").toOption.bind (fun value => value.getStr?.toOption)).getD
          "failed"

private def resultArtifact
    (request : Request)
    (evaluation : Umpire.RunEvaluation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked)
    (evidence : EvidenceArtifact)
    (resultGaps : List KnownGap) : ResultArtifact :=
  let properties := evaluation.querySummary.verdicts.map artifactProperty
  let summary := artifactSummary evaluation.querySummary
  let implementationStatus := evaluation.implementationLink.map
    (ImplementationLinkStatus.name ∘ ImplementationLinkResult.status) |>.getD "not-evaluated"
  let limits : List ArtifactStagedLimit := [
    { stage := "observation-evaluation",
      limit := artifactEvidenceLimit Temporal.System.Nexus.Observation.checkedPlan.evidenceBound },
    { stage := "query",
      limit := artifactLimit
        Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQuery.limits.search }
  ]
  let draft : ResultArtifact := {
    formatVersion := "umpire-result/v2"
    runIdentity := request.runIdentity
    behaviorFingerprint := behaviorFingerprintOf
      (reprStr (evaluation.observation.status, implementationStatus, evaluation.querySummary.status))
    experiment := request.experiment
    runtimeConfiguration := request.runtimeConfiguration
    run := request.run
    rawEvidence := request.rawEvidence
    evidence := evidence.artifactBinding
    operationalStatus := operationalStatus request
    observationEvaluationStatus := observationStatusName evaluation.observation.status
    implementationLink := artifactImplementationLink evaluation
    implementationLinkStatus := implementationStatus
    propertyVerdicts := properties
    querySummary := summary
    semanticStatus := strictStatusName evaluation.querySummary.status
    limits
    knownGaps := resultGaps
    cleanupStatus := cleanupStatus request
    evaluationOutcomeChecksum := none
    provenance
    provenanceChecksum := emptyChecksum
    artifactChecksum := emptyChecksum
  }
  let outcome := draft.expectedEvaluationOutcomeChecksum evidence
    Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact
  ({ draft with evaluationOutcomeChecksum := outcome } : ResultArtifact).seal

def evaluateRequest (request : Request) : Response :=
  let evaluation := evaluateSemantics request
  let observationGaps := semanticGaps evaluation.observation
  let runGaps := (parseGaps request.runKnownGaps).toOption.getD []
  let rawGaps := (parseGaps request.rawEvidenceKnownGaps).toOption.getD []
  let resultGaps := canonicalGaps (runGaps ++ rawGaps ++ observationGaps)
  let evidence := evidenceArtifact request evaluation observationGaps
  let result := resultArtifact request evaluation evidence resultGaps
  let evidenceJson := canonicalJsonValue (canonicalEvidenceArtifactJson evidence)
  let resultJson := canonicalJsonValue (canonicalResultArtifactJson result)
  {
    formatVersion := responseFormatVersion
    checkerIdentity
    checkerVersion
    checkerBehaviorFingerprint
    experimentArtifactChecksum := request.experiment.artifactChecksum
    runtimeConfigurationArtifactChecksum := request.runtimeConfiguration.artifactChecksum
    runArtifactChecksum := request.run.artifactChecksum
    rawEvidenceArtifactChecksum := request.rawEvidence.artifactChecksum
    experimentBehaviorFingerprint := request.experiment.behaviorFingerprint
    runtimeConfigurationBehaviorFingerprint := request.runtimeConfiguration.behaviorFingerprint
    runIdentity := request.runIdentity
    observationEvaluationStatus := observationStatusName evaluation.observation.status
    evidenceBackedModelTrace := jsonField evidenceJson "evidenceBackedModelTrace"
    evidenceLinks := jsonField evidenceJson "evidenceLinks"
    dispositions := jsonField evidenceJson "dispositions"
    diagnostics := jsonField evidenceJson "diagnostics"
    observationKnownGaps := gapsJson observationGaps
    propertyVerdicts := jsonField resultJson "propertyVerdicts"
    querySummary := jsonField resultJson "querySummary"
    semanticStatus := strictStatusName evaluation.querySummary.status
    resultKnownGaps := gapsJson resultGaps
    evaluationOutcomeChecksum := result.evaluationOutcomeChecksum
  }

structure CheckerResult where
  status : UInt32
  stdout : ByteArray
  stderr : String
  deriving BEq

private def protocolErrorName : ErrorKind → String
  | .oversized => "oversized"
  | .invalidUtf8 => "invalid-utf8"
  | .malformedJson => "malformed-json"
  | .nonCanonical => "non-canonical"
  | .wrongShape => "wrong-shape"
  | .invalidValue => "invalid-value"
  | .closureDrift => "closure-drift"

private def errorLine (error : Protocol.Error) : String :=
  Lean.Json.compress (Lean.Json.mkObj [
    ("kind", .str (protocolErrorName error.kind)), ("field", .str error.field)
  ]) ++ "\n"

def runBytes (input : ByteArray) : CheckerResult :=
  match decodeRequest input with
  | .error failure => { status := 1, stdout := .empty, stderr := errorLine failure }
  | .ok request =>
      match encodeResponse (evaluateRequest request) with
      | .error failure => { status := 1, stdout := .empty, stderr := errorLine failure }
      | .ok output => { status := 0, stdout := output, stderr := "" }

end Temporal.Tool.RunEvaluation
