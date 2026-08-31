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
  | .knownGap => "known-gap"
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

private def artifactClosureLe
    (left right : ArtifactEvidenceClosureFact) : Bool :=
  decide (left.kindDefinitionId.value < right.kindDefinitionId.value)

private def artifactOrderingLe
    (left right : ArtifactEvidenceOrderingFact) : Bool :=
  decide (left.factDefinitionId.value ≤ right.factDefinitionId.value)

private def artifactEvidenceLink
    (orderingFact : EvidenceOrderingFact → ArtifactEvidenceOrderingFact)
    (closureSupport : List ArtifactEvidenceClosureFact)
    (link : EvidenceLink) : ArtifactEvidenceLink :=
  let orderingSupport := link.orderingSupport.map orderingFact |>.mergeSort artifactOrderingLe
  {
  coordinate := artifactCoordinate link.coordinate
  mappingDefinitionId := link.mappingId
  mappingVersion := link.mappingVersion
  mappingBehaviorFingerprint :=
    (BehaviorFingerprint.parse? link.mappingDigest).getD (behaviorFingerprintOf link.mappingDigest)
  profileDefinitionId := link.profileId
  profileVersion := link.profileVersion
  evidenceDefinitionIds := link.evidenceIdentities.mergeSort fun left right =>
    decide (left.value ≤ right.value)
  ruleDefinitionId := link.ruleId
  bindingDefinitionIds := link.bindingIds
  orderingSupport
  closureSupport
  appliedDispositions := link.appliedDispositions.map artifactDisposition
  appliedLimit := artifactEvidenceLimit link.appliedBound
  meaningBehaviorFingerprint :=
    (BehaviorFingerprint.parse? link.meaningDigest).getD (behaviorFingerprintOf link.meaningDigest)
}

private def artifactObservationDiagnostic
    (diagnostic : ObservationDiagnostic) : ArtifactObservationDiagnostic := {
  kind := observationFailureName diagnostic.kind
  observationPlanDefinitionId := diagnostic.planId
  relatedDefinitionIds := (diagnostic.relatedDefinitionIds.mergeSort fun left right =>
    decide (left.value ≤ right.value)).eraseDups
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
  evidenceDefinitionIds := trace.evidenceIdentities.mergeSort fun left right =>
    decide (left.value ≤ right.value)
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

private def artifactSemanticDiagnostic
    (diagnostic : SemanticVerdictDiagnostic) : ArtifactSemanticVerdictDiagnostic := {
  kind := semanticFailureName diagnostic.kind
  relatedDefinitionIds := diagnostic.relatedDefinitionIds
  observationDiagnostic := diagnostic.observationEvaluation.map artifactObservationDiagnostic
}

private def artifactClause
    (orderingFact : EvidenceOrderingFact → ArtifactEvidenceOrderingFact)
    (closureSupport : List ArtifactEvidenceClosureFact)
    (clause : SemanticClauseVerdict) : ArtifactSemanticClauseVerdict := {
  propertyDefinitionId := clause.propertyId
  clauseDefinitionId := clause.clauseId
  status := semanticStatusName clause.status
  coordinates := clause.coordinates.map artifactCoordinate
  queryLimits := clause.queryLimits
  propertyLimit := clause.propertyLimit.map artifactLimit
  evidenceLimit := artifactEvidenceLimit clause.evidenceBound
  provenanceDefinitionIds := clause.provenance
  evidenceLinks := clause.evidenceLinks.map (artifactEvidenceLink orderingFact closureSupport)
}

private def artifactProperty
    (orderingFact : EvidenceOrderingFact → ArtifactEvidenceOrderingFact)
    (closureSupport : List ArtifactEvidenceClosureFact)
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
  clauses := verdict.clauses.map (artifactClause orderingFact closureSupport)
  diagnostic := verdict.diagnostic.map artifactSemanticDiagnostic
}

private def artifactSummary
    (orderingFact : EvidenceOrderingFact → ArtifactEvidenceOrderingFact)
    (closureSupport : List ArtifactEvidenceClosureFact)
    (summary : StrictQuerySummary) : ArtifactQuerySummary := {
  queryDefinitionId := summary.queryId
  status := strictStatusName summary.status
  queryLimits := summary.queryLimits
  requiredPropertyDefinitionIds := summary.requiredProperties
  propertyVerdicts := summary.verdicts.map (artifactProperty orderingFact closureSupport)
  missingPropertyDefinitionIds := summary.missingProperties
  duplicatePropertyDefinitionIds := summary.duplicateProperties
  unexpectedPropertyDefinitionIds := summary.unexpectedProperties
  divergentPropertyDefinitionIds := summary.divergentProperties
  wrongQueryResultDefinitionIds := summary.wrongQueryResults
  traceIds := summary.traceIds
}

private structure SemanticEvaluation where
  observation : ObservationResult
  implementationLinkDefinitionId : DefinitionId
  implementationLinkBehaviorFingerprint : BehaviorFingerprint
  implementationLinkStatus : String
  implementationLinkDiagnostic : Option ImplementationLinkDiagnostic
  querySummary : StrictQuerySummary
  checkedPlan : CheckedObservationPlan
  querySearchLimit : Limit
  experiment : ExperimentSpec

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
    (evaluation : SemanticEvaluation) :
    ArtifactImplementationLinkRecord := {
  definitionId := evaluation.implementationLinkDefinitionId
  behaviorFingerprint := evaluation.implementationLinkBehaviorFingerprint
  sourceTarget := artifactTarget (.ofTarget
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.sourceTarget)
  destinationTarget := artifactTarget (.ofTarget
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.destinationTarget)
  diagnostic := evaluation.implementationLinkDiagnostic.map artifactImplementationDiagnostic
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
  byteCount : Nat
  deriving BEq

private structure RawClosure where
  definitionId : DefinitionId
  status : String
  recordCount : Nat
  byteCount : Nat
  deriving BEq

private structure RawControlAttempt where
  occurrence : DefinitionId
  action : DefinitionId
  attempt : Nat
  receipt : Option DefinitionId
  status : String
  code : Option DefinitionId
  deriving BEq

private def parseField (json : Lean.Json) : Except String RawField := do
  exactObject json ["fieldDefinitionId", "disposition", "value"] "facts.fields"
  let disposition ← textValue json "disposition"
  if !["plain", "sha256", "redacted", "rejected"].contains disposition then
    throw "facts.fields.disposition"
  let raw ← value json "value"
  let fieldValue ←
    if disposition == "redacted" || disposition == "rejected" then
      if raw.isNull then pure (.text "") else throw "facts.fields.value"
    else match raw.getStr? with
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
    byteCount := ← natValue json "byteCount"
  }

private def parseClosure (json : Lean.Json) : Except String RawClosure := do
  exactObject json ["sourceDefinitionId", "status", "recordCount", "byteCount"]
    "sourceClosures"
  let status ← textValue json "status"
  if status != "closed" && status != "partial" && status != "failed" then
    throw "sourceClosures.status"
  pure {
    definitionId := ← idValue json "sourceDefinitionId"
    status
    recordCount := ← natValue json "recordCount"
    byteCount := ← natValue json "byteCount"
  }

private def optionalIdValue
    (json : Lean.Json)
    (field : String) : Except String (Option DefinitionId) := do
  let raw ← value json field
  if raw.isNull then pure none
  else
    let result ← idValue json field
    pure (some result)

private def callerClosureOccurrence? : Option DefinitionId :=
  (Temporal.System.Execution.Nexus.canonicalParticipantProgramDefinition.occurrences.find?
    fun occurrence =>
      occurrence.actionDefinitionId == Temporal.System.Execution.Nexus.actionDefinitionId).map
        fun occurrence => occurrence.definitionId

private def parseControlAttempts (json : Lean.Json) : Except String RawControlAttempt := do
  let values ← match json.getArr? with
    | .ok values => pure values.toList
    | .error _ => throw "controlAttempts"
  let attempt ← match values with
    | [attempt] => pure attempt
    | _ => throw "controlAttempts"
  exactObject attempt ["occurrenceDefinitionId", "actionDefinitionId", "attempt",
    "receiptFactDefinitionId", "status", "code"] "controlAttempts"
  let projection : RawControlAttempt := {
    occurrence := ← idValue attempt "occurrenceDefinitionId"
    action := ← idValue attempt "actionDefinitionId"
    attempt := ← natValue attempt "attempt"
    receipt := ← optionalIdValue attempt "receiptFactDefinitionId"
    status := ← textValue attempt "status"
    code := ← optionalIdValue attempt "code"
  }
  if some projection.occurrence != callerClosureOccurrence? ||
      projection.action != Temporal.System.Execution.Nexus.actionDefinitionId ||
      projection.attempt != 1 then
    throw "controlAttempts"
  match projection.status with
  | "not-attempted" =>
      if projection.receipt.isSome || projection.code.isSome then throw "controlAttempts"
  | "accepted" =>
      if projection.receipt.isNone || projection.code.isSome then throw "controlAttempts"
  | "rejected" | "unsupported" | "failed" | "canceled" =>
      if projection.receipt.isNone || projection.code.isNone then throw "controlAttempts"
  | _ => throw "controlAttempts"
  pure projection

private def sourceCleanup := id "umpire.evidence.source.cleanup"
private def sourceControl := id "umpire.evidence.source.control-receipt"
private def sourceHistory := id "umpire.evidence.source.history"
private def sourceParticipant := id "umpire.evidence.source.participant-output"

private structure SourceSchema where
  source : DefinitionId
  kind : DefinitionId
  rawKinds : List DefinitionId

private def expectedSources : List SourceSchema := [
  { source := sourceCleanup, kind := Temporal.System.Nexus.Observation.Profile.cleanupKind,
    rawKinds := [Temporal.System.Nexus.Observation.Profile.cleanupKind,
      Temporal.System.Nexus.Observation.Profile.participantKind,
      id "umpire.evidence.kind.environment-cleanup"] },
  { source := sourceControl,
    kind := Temporal.System.Nexus.Observation.Profile.controlReceiptKind,
    rawKinds := [Temporal.System.Nexus.Observation.Profile.controlReceiptKind] },
  { source := sourceHistory, kind := Temporal.System.Nexus.Observation.Profile.historyKind,
    rawKinds := [Temporal.System.Nexus.Observation.Profile.historyKind] },
  { source := sourceParticipant,
    kind := Temporal.System.Nexus.Observation.Profile.participantKind,
    rawKinds := [Temporal.System.Nexus.Observation.Profile.participantKind,
      id "umpire.evidence.kind.environment-lifecycle"] }
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

private def validRuntimeProjections (request : Request) : Bool :=
  validPhaseOutcomes request.phaseOutcomes

private def sourceSchema? (source : DefinitionId) : Option SourceSchema :=
  expectedSources.find? fun schema => schema.source == source

private def exactSourceSet (sources : List DefinitionId) : Bool :=
  sources.length == expectedSources.length &&
    expectedSources.all fun expected => sources.contains expected.source

private def validateSourceClosure
    (sources : List RawSource)
    (closures : List RawClosure)
    (facts : List RawFact) : Except String Unit := do
  if !exactSourceSet (sources.map RawSource.definitionId) then
    throw "sources.set"
  if !exactSourceSet (closures.map RawClosure.definitionId) then
    throw "sourceClosures.set"
  for source in sources do
    if (sourceSchema? source.definitionId).isNone then
      throw "sources.sourceDefinitionId"
    let closure ← match closures.find? fun closure => closure.definitionId == source.definitionId with
      | some closure => pure closure
      | none => throw "sourceClosures.sourceDefinitionId"
    if source.status != closure.status || source.factCount != closure.recordCount ||
        source.byteCount != closure.byteCount then
      throw "sourceClosures"
    if source.factCount !=
        (facts.filter fun fact => fact.sourceDefinitionId == source.definitionId).length then
      throw "sources.factCount"

private def exactRawField
    (fact : RawFact)
    (fieldId : DefinitionId)
    (expected : EvidenceValue) : Bool :=
  match fact.fields.filter fun field => field.definitionId == fieldId with
  | [{ disposition := "plain", value, .. }] => value == expected
  | _ => false

private def isDuplicateDeliveryRequest (request : Request) : Bool :=
  request.experiment == expectedDuplicateDeliveryExperimentBinding &&
    request.runtimeConfiguration == expectedDuplicateDeliveryRuntimeConfigurationBinding &&
    request.observationProgram == expectedDuplicateDeliveryObservationProgram &&
    request.mapping == expectedDuplicateDeliveryMapping

private def validateControlAttempt
    (request : Request)
    (attempt : RawControlAttempt)
    (facts : List RawFact) : Except String Unit := do
  let controlFacts := facts.filter fun fact => fact.sourceDefinitionId == sourceControl
  if attempt.status == "not-attempted" then
    if !controlFacts.isEmpty then throw "controlAttempts"
    return
  let receipt ← match attempt.receipt with
    | some receipt => pure receipt
    | none => throw "controlAttempts"
  let fact ← match controlFacts with
    | [fact] => pure fact
    | _ => throw "controlAttempts"
  if fact.definitionId != receipt ||
      fact.kindDefinitionId != Temporal.System.Nexus.Observation.Profile.controlReceiptKind then
    throw "controlAttempts"
  if !isDuplicateDeliveryRequest request &&
      (fact.fields.length != 4 ||
        !exactRawField fact Temporal.System.Nexus.Observation.Profile.actionField
          (.text attempt.action.value) ||
        !exactRawField fact Temporal.System.Nexus.Observation.Profile.attemptField
          (.natural attempt.attempt) ||
        !exactRawField fact Temporal.System.Nexus.Observation.Profile.occurrenceField
          (.text attempt.occurrence.value) ||
        !exactRawField fact Temporal.System.Nexus.Observation.Profile.statusField
          (.text attempt.status)) then
    throw "controlAttempts"

private def evidenceField (field : RawField) : Except String EvidenceFieldValue := do
  if field.disposition == "plain" then
    pure { field := field.definitionId, value := field.value }
  else if field.disposition == "redacted" || field.disposition == "rejected" then
    pure { field := field.definitionId, value := field.value }
  else
    let token ← match field.value with
      | .text token => pure token
      | _ => throw "facts.fields.value"
    if (BehaviorFingerprint.parse? token).isNone then throw "facts.fields.value"
    pure {
      field := field.definitionId
      value := field.value
      digestPolicy := some Temporal.System.Nexus.Observation.Profile.endpointDigestPolicyId
      reportedDigestToken := some token
    }

private def rawFactHasField (fact : RawFact) (field : DefinitionId) : Bool :=
  fact.fields.any fun candidate => candidate.definitionId == field

private def rawFactHasExactFields (fact : RawFact) (fields : List DefinitionId) : Bool :=
  fact.fields.length == fields.length && fields.all fun field =>
    (fact.fields.filter fun candidate => candidate.definitionId == field).length == 1

private def exactRawText? (fact : RawFact) (fieldId : DefinitionId) : Option String :=
  match fact.fields.filter fun field => field.definitionId == fieldId with
  | [{ disposition := "plain", value := .text value, .. }] => some value
  | _ => none

private def participantCancellationFields : List DefinitionId := [
  Temporal.System.Nexus.Observation.Profile.cancellationCountField,
  Temporal.System.Nexus.Observation.Profile.commandKindField,
  Temporal.System.Nexus.Observation.Profile.endpointIdentityField,
  Temporal.System.Nexus.Observation.Profile.namespaceIdentityField,
  Temporal.System.Nexus.Observation.Profile.operationCorrelationField,
  Temporal.System.Nexus.Observation.Profile.runCorrelationField,
  Temporal.System.Nexus.Observation.Profile.statusField,
  Temporal.System.Nexus.Observation.Profile.taskQueueIdentityField,
  Temporal.System.Nexus.Observation.Profile.workflowCorrelationField
]

private def participantCancellationIsBound
    (request : Request)
    (facts : List RawFact)
    (candidate : RawFact) : Bool :=
  let historyFacts := facts.filter fun fact => fact.sourceDefinitionId == sourceHistory
  let realizations := facts.filter fun fact =>
    fact.sourceDefinitionId == sourceParticipant &&
      fact.kindDefinitionId == Temporal.System.Nexus.Observation.Profile.participantKind &&
      exactRawField fact Temporal.System.Nexus.Observation.Profile.commandKindField (.text "realize")
  realizations == [candidate] &&
      rawFactHasExactFields candidate participantCancellationFields &&
      exactRawField candidate Temporal.System.Nexus.Observation.Profile.cancellationCountField
        (.natural 1) &&
      exactRawField candidate Temporal.System.Nexus.Observation.Profile.statusField
        (.text "accepted") &&
      exactRawField candidate Temporal.System.Nexus.Observation.Profile.runCorrelationField
        (.text request.runIdentity.value) &&
      !historyFacts.isEmpty && historyFacts.all fun history =>
        exactRawText? history Temporal.System.Nexus.Observation.Profile.runCorrelationField ==
          some request.runIdentity.value &&
        exactRawText? history Temporal.System.Nexus.Observation.Profile.operationCorrelationField ==
          exactRawText? candidate
            Temporal.System.Nexus.Observation.Profile.operationCorrelationField &&
        exactRawText? history Temporal.System.Nexus.Observation.Profile.workflowCorrelationField ==
          exactRawText? candidate
            Temporal.System.Nexus.Observation.Profile.workflowCorrelationField

private def participantCancellationInterpretation : DefinitionId :=
  id "temporal.system.nexus.caller-closure.interpretation.participant-cancellation"

private def participantCancellationAlternatives
    (request : Request)
    (facts : List RawFact) : List CompatibleInterpretation :=
  let candidates := facts.filter fun fact =>
    rawFactHasField fact Temporal.System.Nexus.Observation.Profile.cancellationCountField
  match candidates with
  | [] => []
  | [candidate] =>
      if participantCancellationIsBound request facts candidate then []
      else [
        { id := participantCancellationInterpretation, evidenceIdentities := [] },
        { id := participantCancellationInterpretation,
          evidenceIdentities := [candidate.definitionId] }
      ]
  | candidates =>
      candidates.map fun candidate => {
        id := participantCancellationInterpretation
        evidenceIdentities := [candidate.definitionId]
      }

private def semanticFactKind (schema : SourceSchema) (fact : RawFact) : DefinitionId :=
  if schema.source == sourceParticipant &&
      !rawFactHasField fact Temporal.System.Nexus.Observation.Profile.cancellationCountField then
    Temporal.System.Nexus.Observation.Profile.cleanupKind
  else
    schema.kind

private def requestFacts (request : Request) : List RawFact :=
  match request.facts.getArr? with
  | .error _ => []
  | .ok values => (values.toList.mapM parseFact).toOption.getD []

private def artifactOrderingFactFor
    (facts : List RawFact)
    (fact : EvidenceOrderingFact) : ArtifactEvidenceOrderingFact :=
  match facts.find? fun raw => raw.definitionId == fact.recordId with
  | some raw => {
      factDefinitionId := raw.definitionId
      kindDefinitionId := raw.kindDefinitionId
      ordinal := raw.ordinal
      causalFactDefinitionIds := raw.causalParents
    }
  | none => {
      factDefinitionId := fact.recordId
      kindDefinitionId := fact.kind
      ordinal := fact.origin.map EvidenceOrigin.ordinal |>.getD fact.sequence
      causalFactDefinitionIds := fact.causalParents
    }

private def artifactRawClosures (facts : List RawFact) : List ArtifactEvidenceClosureFact :=
  let kinds := facts.map RawFact.kindDefinitionId |>.eraseDups
  (kinds.map fun kind => {
    kindDefinitionId := kind
    lastOrdinal := facts.filter (fun fact => fact.kindDefinitionId == kind)
      |>.foldl (fun current fact => Nat.max current fact.ordinal) 0
  }).mergeSort artifactClosureLe

private def artifactEvidenceClosures
    (facts : List RawFact)
    (evidenceIdentities : List DefinitionId) : List ArtifactEvidenceClosureFact :=
  artifactRawClosures <| facts.filter fun fact => evidenceIdentities.contains fact.definitionId

private def artifactDispositionForRaw
    (plan : CheckedObservationPlan)
    (duplicateDelivery : Bool)
    (fact : RawFact)
    (field : RawField) : Option ArtifactFieldDispositionRecord := do
  let schema ← sourceSchema? fact.sourceDefinitionId
  let semanticKind := if duplicateDelivery then schema.kind else semanticFactKind schema fact
  let declaration ← plan.dispositions.find? fun item =>
    item.field == { kind := semanticKind, field := field.definitionId }
  some {
    field := { kindDefinitionId := fact.kindDefinitionId, fieldDefinitionId := field.definitionId }
    disposition := declaration.disposition.name
    digestPolicyDefinitionId := match declaration.disposition with
      | .hash policy => policy
      | _ => none
  }

private def artifactDispositionLe
    (left right : ArtifactFieldDispositionRecord) : Bool :=
  decide (left.field.kindDefinitionId.value < right.field.kindDefinitionId.value) ||
    (left.field.kindDefinitionId == right.field.kindDefinitionId &&
      decide (left.field.fieldDefinitionId.value ≤ right.field.fieldDefinitionId.value))

private def artifactRawDispositions
    (plan : CheckedObservationPlan)
    (duplicateDelivery : Bool)
    (facts : List RawFact) : List ArtifactFieldDispositionRecord :=
  facts.flatMap (fun fact =>
      fact.fields.filterMap (artifactDispositionForRaw plan duplicateDelivery fact))
    |>.mergeSort artifactDispositionLe |>.eraseDups

private def evidenceRecord (fact : RawFact) : Except String SyntheticEvidenceRecord := do
  let schema ← match sourceSchema? fact.sourceDefinitionId with
    | some schema => pure schema
    | none => throw "facts.sourceDefinitionId"
  if !schema.rawKinds.contains fact.kindDefinitionId then throw "facts.kindDefinitionId"
  pure {
    id := fact.definitionId
    profile := Temporal.System.Nexus.Observation.Profile.id
    profileVersion := 1
    kind := semanticFactKind schema fact
    sequence := fact.ordinal + 1
    origin := some { source := fact.sourceDefinitionId, ordinal := fact.ordinal }
    causalParents := fact.causalParents
    fields := ← fact.fields.mapM evidenceField
  }

private def duplicateDeliveryRecord
    (fact : RawFact)
    (kind : DefinitionId)
    (ordinal : Nat)
    (causalParents : List DefinitionId)
    (fields : List RawField)
    (faultTarget : Option DefinitionId := none) : Except String SyntheticEvidenceRecord := do
  pure {
    id := fact.definitionId
    profile := Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.id
    profileVersion := 1
    kind
    sequence := ordinal + 1
    origin := some { source := fact.sourceDefinitionId, ordinal }
    causalParents
    fields := ← fields.mapM evidenceField
    faultTarget
  }

private def duplicateDeliveryHistoryTypes : List String := [
  "temporal.history.WorkflowExecutionStarted",
  "temporal.history.NexusOperationCancelRequested",
  "temporal.history.NexusOperationCancelRequestCompleted",
  "temporal.history.WorkflowExecutionCanceled"
]

private def rawFactOrdinalLe (left right : RawFact) : Bool := left.ordinal ≤ right.ordinal

private def completeRawHistoryChain (facts : List RawFact) : Bool :=
  let history := (facts.filter fun fact => fact.sourceDefinitionId == sourceHistory)
    |>.mergeSort rawFactOrdinalLe
  !history.isEmpty && history.zipIdx.all fun pair =>
    pair.1.ordinal == pair.2 &&
      pair.1.causalParents == if pair.2 == 0 then [] else
        (history[pair.2 - 1]?).map RawFact.definitionId |>.toList

private def duplicateDeliverySyntheticDiscriminatorFields : List DefinitionId := [
  Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.syntheticMarkerField,
  Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.syntheticContributionCountField,
  Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.faultDefinitionField,
  Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.faultReceiptField
]

private def duplicateDeliverySyntheticCandidate (fact : RawFact) : Bool :=
  fact.sourceDefinitionId == sourceParticipant &&
    duplicateDeliverySyntheticDiscriminatorFields.any (rawFactHasField fact)

private def duplicateDeliverySelectedSyntheticFact (fact : RawFact) : Bool :=
  fact.sourceDefinitionId == sourceParticipant &&
    exactRawField fact
      Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.syntheticMarkerField
      (.text Temporal.System.Nexus.Observation.DuplicateDelivery.injectedMarker) &&
    exactRawField fact
      Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.faultDefinitionField
      (.text Temporal.System.Nexus.Observation.DuplicateDelivery.faultDefinitionId.value) &&
    exactRawField fact
      Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.faultReceiptField
      (.text Temporal.System.Nexus.Observation.DuplicateDelivery.faultReceiptId.value)

private def duplicateDeliveryCallbackFields : List DefinitionId := [
  Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.cancellationCountField,
  Temporal.System.Nexus.Observation.Profile.endpointIdentityField
]

private def mergeParticipantFields
    (callback synthetic : RawFact) : List RawField :=
  synthetic.fields ++ callback.fields.filter fun field =>
    duplicateDeliveryCallbackFields.contains field.definitionId

private def duplicateDeliveryRecords
    (facts : List RawFact) : Except String (List SyntheticEvidenceRecord) := do
  let ordinary := facts.filter fun fact =>
    fact.sourceDefinitionId != sourceHistory && fact.sourceDefinitionId != sourceParticipant &&
      (fact.sourceDefinitionId != sourceCleanup ||
        rawFactHasField fact Temporal.System.Nexus.Observation.Profile.openHandleCountField)
  let ordinaryRecords ← ordinary.mapM fun fact => do
    if fact.sourceDefinitionId == sourceCleanup then
      duplicateDeliveryRecord fact
        Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.cleanupKind 0 [] fact.fields
    else
      let record ← evidenceRecord fact
      pure { record with
        profile := Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.id }
  let rawHistory := (facts.filter fun fact => fact.sourceDefinitionId == sourceHistory)
    |>.mergeSort rawFactOrdinalLe
  let projectedHistory := if completeRawHistoryChain facts then
      rawHistory.filter fun fact =>
        (exactRawText? fact Temporal.System.Nexus.Observation.Profile.eventTypeField).any
          duplicateDeliveryHistoryTypes.contains
    else rawHistory
  let historyRecords ← projectedHistory.zipIdx.mapM fun pair =>
    duplicateDeliveryRecord pair.1
      Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.historyKind pair.2
      (if pair.2 == 0 then [] else
        (projectedHistory[pair.2 - 1]?).map RawFact.definitionId |>.toList) pair.1.fields
  let participants := (facts.filter fun fact => fact.sourceDefinitionId == sourceParticipant)
    |>.mergeSort rawFactOrdinalLe
  let callbacks := participants.filter fun fact =>
    rawFactHasField fact
      Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.cancellationCountField
  let syntheticCandidates := participants.filter duplicateDeliverySyntheticCandidate
  let selectedSynthetic := syntheticCandidates.filter duplicateDeliverySelectedSyntheticFact
  let participantRecords ← match callbacks, syntheticCandidates, selectedSynthetic with
    | [callback], [_], [synthetic] =>
        let completed? := projectedHistory.find? fun fact =>
          exactRawText? fact Temporal.System.Nexus.Observation.Profile.eventTypeField ==
            some "temporal.history.NexusOperationCancelRequestCompleted"
        let completedId? := completed?.map RawFact.definitionId
        let structuralParents := if synthetic.causalParents == [callback.definitionId] then
            [callback.definitionId] ++ completedId?.toList
          else synthetic.causalParents
        let proxy ← duplicateDeliveryRecord callback
          Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.cleanupKind 0 [] []
        let combined ← duplicateDeliveryRecord synthetic
          Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.participantKind 1
          structuralParents (mergeParticipantFields callback synthetic) completedId?
        pure [proxy, combined]
    | _, candidates, _ => do
        let proxyRecords ← callbacks.zipIdx.mapM fun pair =>
          duplicateDeliveryRecord pair.1
            Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.cleanupKind
            pair.2 [] []
        let candidateRecords ← candidates.zipIdx.mapM fun pair =>
          duplicateDeliveryRecord pair.1
            Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.participantKind
            (callbacks.length + pair.2) pair.1.causalParents pair.1.fields (faultTarget := none)
        pure (proxyRecords ++ candidateRecords)
  pure (ordinaryRecords ++ historyRecords ++ participantRecords)

private def evidenceClosures
    (closures : List RawClosure)
    (records : List SyntheticEvidenceRecord) : List EvidenceClosureFact :=
  closures.flatMap fun closure =>
    let sourceRecords := records.filter fun record =>
      record.origin.any fun origin => origin.source == closure.definitionId
    let kinds := sourceRecords.map SyntheticEvidenceRecord.kind |>.eraseDups
    kinds.map fun kind =>
      let kindRecords := sourceRecords.filter fun record => record.kind == kind
      {
        kind
        lastSequence := kindRecords.foldl (fun current record =>
          Nat.max current (record.origin.map (fun origin => origin.ordinal + 1) |>.getD 0)) 0
        source := some closure.definitionId
        recordCount := some kindRecords.length
        byteCount := some closure.byteCount
      }

private def evidenceGap (gap : KnownGap) : EvidenceGap := {
  code := gap.code
  relatedDefinitionIds := gap.subject.toList
}

private def adapt (request : Request) : Except String EvidenceBundle := do
  if !validRuntimeProjections request then throw "runtimeProjections"
  if !["closed", "partial", "failed"].contains request.captureStatus then
    throw "captureStatus"
  let sourceValues ← match request.sources.getArr? with
    | .ok values => pure values.toList
    | .error _ => throw "sources"
  let closureValues ← match request.sourceClosures.getArr? with
    | .ok values => pure values.toList
    | .error _ => throw "sourceClosures"
  let factValues ← match request.facts.getArr? with
    | .ok values => pure values.toList
    | .error _ => throw "facts"
  let sources ← sourceValues.mapM parseSource
  let closures ← closureValues.mapM parseClosure
  let facts ← factValues.mapM parseFact
  let controlAttempt ← parseControlAttempts request.controlAttempts
  let runGaps ← parseGaps request.runKnownGaps
  let rawGaps ← parseGaps request.rawEvidenceKnownGaps
  validateSourceClosure sources closures facts
  validateControlAttempt request controlAttempt facts
  let records ← if isDuplicateDeliveryRequest request then
      duplicateDeliveryRecords facts
    else facts.mapM evidenceRecord
  let cancellationAlternatives := if isDuplicateDeliveryRequest request then []
    else participantCancellationAlternatives request facts
  pure {
    profile := if isDuplicateDeliveryRequest request then
        Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.id
      else Temporal.System.Nexus.Observation.Profile.id
    profileVersion := 1
    records
    closures := evidenceClosures closures records
    knownGaps := (runGaps ++ rawGaps).map evidenceGap
    sourceClosed := request.captureStatus == "closed" &&
      sources.all (fun source => source.status == "closed") &&
      closures.all (fun closure => closure.status == "closed")
    closedFieldKinds := expectedSources.filterMap fun schema =>
      if schema.source == sourceCleanup || schema.source == sourceParticipant then none
      else some schema.kind
    compatibleAlternatives := cancellationAlternatives
    missingDiscriminator := if cancellationAlternatives.isEmpty then none
      else some Temporal.System.Nexus.Observation.Profile.commandKindField
  }

private def adapterError (field : String) : Protocol.Error := {
  kind := .invalidValue
  field
}

private def duplicateDeliveryDispositionMismatch?
    (request : Request) : Option DefinitionId :=
  requestFacts request |>.findSome? fun fact => do
    let schema ← sourceSchema? fact.sourceDefinitionId
    fact.fields.findSome? fun field => do
      let declaration ←
        Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan.dispositions.find? fun item =>
          item.field == { kind := schema.kind, field := field.definitionId }
      let compatible := match declaration.disposition with
        | .retain => field.disposition == "plain"
        | .redact => field.disposition == "redacted"
        | .reject => field.disposition == "rejected"
        | .hash _ => field.disposition == "sha256"
      if compatible then none else some field.definitionId

private def duplicateDeliverySchemaMismatch?
    (request : Request) : Option DefinitionId :=
  requestFacts request |>.findSome? fun fact => do
    let schema ← sourceSchema? fact.sourceDefinitionId
    if !schema.rawKinds.contains fact.kindDefinitionId then
      some fact.kindDefinitionId
    else
      fact.fields.findSome? fun field =>
        if Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan.dispositions.any fun item =>
            item.field == { kind := schema.kind, field := field.definitionId } then
          none
        else
          some field.definitionId

private def duplicateDeliveryCorrelationFields : List DefinitionId := [
  Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.operationCorrelationField,
  Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.runCorrelationField,
  Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.workflowCorrelationField
]

private def duplicateDeliveryParticipantCorrelationFailure?
    (request : Request) : Option ObservationResult :=
  let participants := requestFacts request |>.filter fun fact =>
    fact.sourceDefinitionId == sourceParticipant
  let callbacks := participants.filter fun fact =>
    rawFactHasField fact
      Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.cancellationCountField
  let synthetic := participants.filter duplicateDeliverySelectedSyntheticFact
  match callbacks, synthetic with
  | [callback], [synthetic] =>
      match duplicateDeliveryCorrelationFields.find? fun field =>
          (exactRawText? callback field).isNone || (exactRawText? synthetic field).isNone with
      | some field => some <| .unknown {
          kind := .unresolvedBinding
          planId := Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan.id
          relatedDefinitionIds := [callback.definitionId, synthetic.definitionId, field]
        }
      | none =>
          match duplicateDeliveryCorrelationFields.find? fun field =>
              exactRawText? callback field != exactRawText? synthetic field with
          | some field => some <| .conflict {
              kind := .contradictoryBinding
              planId := Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan.id
              relatedDefinitionIds := [callback.definitionId, synthetic.definitionId, field]
            }
          | none =>
              if exactRawText? callback
                  Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.runCorrelationField !=
                  some request.runIdentity.value then
                some <| .conflict {
                  kind := .contradictoryBinding
                  planId := Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan.id
                  relatedDefinitionIds := [callback.definitionId, synthetic.definitionId,
                    Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.runCorrelationField]
                }
              else none
  | _, _ => none

private def duplicateDeliverySyntheticCandidateFailure?
    (request : Request) : Option ObservationResult :=
  let candidates := requestFacts request |>.filter duplicateDeliverySyntheticCandidate
  if candidates.length > 1 then
    some <| .conflict {
      kind := .contradictoryFact
      planId := Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan.id
      relatedDefinitionIds := candidates.map RawFact.definitionId
    }
  else none

private def duplicateDeliveryMissingCausalParent?
    (request : Request) : Option DefinitionId :=
  let facts := requestFacts request
  let participant? := facts.find? fun fact =>
    duplicateDeliverySelectedSyntheticFact fact && fact.causalParents.isEmpty
  let history? := facts.find? fun fact =>
    fact.sourceDefinitionId == sourceHistory && fact.ordinal > 0 && fact.causalParents.isEmpty
  (participant? <|> history?).map RawFact.definitionId

def evaluateSemantics
    (request : Request) : Except Protocol.Error (Umpire.RunEvaluation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked) := do
  let bundle ← (adapt request).mapError adapterError
  pure <| Umpire.checkRunEvaluation Temporal.System.Nexus.Observation.checkedPlan bundle
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
    Temporal.System.Nexus.CallerClosure.setup
    Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQuery
    [Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosureProperty]

private def duplicateDeliveryQuery : CheckedQuery
    Temporal.Feature.Nexus.Experimental.CallerClosure.LawStatement := {
  Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQuery with
  id := expectedDuplicateDeliveryQuery.definitionId
  behaviorFingerprint := expectedDuplicateDeliveryQuery.behaviorFingerprint
}

private def evaluateDuplicateDeliverySemantics
    (request : Request) : Except Protocol.Error (Umpire.ObservedRunEvaluation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
      Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.checkedObservedTranslation) := do
  let bundle ← (adapt request).mapError adapterError
  let observation := match duplicateDeliveryMissingCausalParent? request with
    | some fact => .unknown {
        kind := .missingCausalParent
        planId := Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan.id
        relatedDefinitionIds := [fact]
      }
    | none => match duplicateDeliveryDispositionMismatch? request with
      | some field => .unsupported {
          kind := .fieldMismatch
          planId := Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan.id
          relatedDefinitionIds := [field]
        }
      | none => match duplicateDeliverySchemaMismatch? request with
        | some field => .unsupported {
            kind := .fieldMismatch
            planId := Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan.id
            relatedDefinitionIds := [field]
          }
        | none => match duplicateDeliverySyntheticCandidateFailure? request with
          | some failure => failure
          | none => match duplicateDeliveryParticipantCorrelationFailure? request with
            | some failure => failure
            | none =>
                Temporal.System.Nexus.Observation.DuplicateDelivery.qualifyDuplicateDeliveryObservation bundle
  pure <| Umpire.checkObservedRunEvaluation observation
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
    Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.checkedObservedTranslation
    Temporal.System.Nexus.CallerClosure.setup duplicateDeliveryQuery
    [Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosureProperty]

private def strictSemanticEvaluation
    (evaluation : Umpire.RunEvaluation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked) : SemanticEvaluation := {
  observation := evaluation.observation
  implementationLinkDefinitionId :=
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.declaration.id
  implementationLinkBehaviorFingerprint :=
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.behaviorFingerprint
  implementationLinkStatus := evaluation.implementationLink.map
    (ImplementationLinkStatus.name ∘ ImplementationLinkResult.status) |>.getD "not-evaluated"
  implementationLinkDiagnostic :=
    evaluation.implementationLink.bind ImplementationLinkResult.diagnostic?
  querySummary := evaluation.querySummary
  checkedPlan := Temporal.System.Nexus.Observation.checkedPlan
  querySearchLimit := Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQuery.limits.search
  experiment := Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact
}

private def observedSemanticEvaluation
    (evaluation : Umpire.ObservedRunEvaluation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
      Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.checkedObservedTranslation) :
    SemanticEvaluation := {
  observation := evaluation.observation
  implementationLinkDefinitionId :=
    Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.observedImplementationLinkId
  implementationLinkBehaviorFingerprint :=
    Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.behaviorFingerprint
  implementationLinkStatus := evaluation.implementationLink.map
    (ImplementationLinkStatus.name ∘ ObservedTraceTranslationResult.status) |>.getD "not-evaluated"
  implementationLinkDiagnostic :=
    evaluation.implementationLink.bind ObservedTraceTranslationResult.diagnostic?
  querySummary := evaluation.querySummary
  checkedPlan := Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan
  querySearchLimit := duplicateDeliveryQuery.limits.search
  experiment := expectedDuplicateDeliveryExperiment
}

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
    (request : Request)
    (evaluation : SemanticEvaluation) :
    Option ArtifactEvidenceBackedModelTrace × List ArtifactEvidenceLink ×
      List ArtifactFieldDispositionRecord × List ArtifactObservationDiagnostic :=
  let facts := requestFacts request
  let orderingFact := artifactOrderingFactFor facts
  let dispositions := artifactRawDispositions evaluation.checkedPlan
    (isDuplicateDeliveryRequest request) facts
  match evaluation.observation with
  | .accepted trace =>
      let closures := artifactEvidenceClosures facts trace.evidenceIdentities
      (some (artifactTrace trace),
        trace.evidenceLinks.map (artifactEvidenceLink orderingFact closures), dispositions, [])
  | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic =>
      (none, [], dispositions, [artifactObservationDiagnostic diagnostic])

private def evidenceArtifact
    (request : Request)
    (evaluation : SemanticEvaluation)
    (observationGaps : List KnownGap) : EvidenceArtifact :=
  let projection := observationProjection request evaluation
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
      match cleanup.bind fun outcome =>
          (outcome.getObjVal? "status").toOption.bind (fun value => value.getStr?.toOption) with
      | some "succeeded" => "complete"
      | some "not-started" => "incomplete"
      | _ => "failed"

private def resultArtifact
    (request : Request)
    (evaluation : SemanticEvaluation)
    (evidence : EvidenceArtifact)
    (resultGaps : List KnownGap) : ResultArtifact :=
  let implementationStatus := evaluation.implementationLinkStatus
  let facts := requestFacts request
  let closures := match evaluation.observation with
    | .accepted trace => artifactEvidenceClosures facts trace.evidenceIdentities
    | _ => artifactRawClosures facts
  let evaluatedSummary := artifactSummary (artifactOrderingFactFor facts)
    closures evaluation.querySummary
  let summary :=
    if evaluation.observation.status == .accepted && implementationStatus == "applied" then
      evaluatedSummary
    else
      { evaluatedSummary with
        status := "incomplete"
        propertyVerdicts := []
        missingPropertyDefinitionIds := evaluatedSummary.requiredPropertyDefinitionIds
        traceIds := [] }
  let properties := summary.propertyVerdicts
  let limits : List ArtifactStagedLimit := [
    { stage := "observation-evaluation",
      limit := artifactEvidenceLimit evaluation.checkedPlan.evidenceBound },
    { stage := "query",
      limit := artifactLimit
        evaluation.querySearchLimit }
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
    semanticStatus := summary.status
    limits
    knownGaps := resultGaps
    cleanupStatus := cleanupStatus request
    evaluationOutcomeChecksum := none
    provenance
    provenanceChecksum := emptyChecksum
    artifactChecksum := emptyChecksum
  }
  let outcome := draft.expectedEvaluationOutcomeChecksum evidence
    evaluation.experiment
  ({ draft with evaluationOutcomeChecksum := outcome } : ResultArtifact).seal

def evaluateRequest (request : Request) : Except Protocol.Error Response := do
  let evaluation ← if isDuplicateDeliveryRequest request then
      observedSemanticEvaluation <$> evaluateDuplicateDeliverySemantics request
    else strictSemanticEvaluation <$> evaluateSemantics request
  let observationGaps := semanticGaps evaluation.observation
  let runGaps := (parseGaps request.runKnownGaps).toOption.getD []
  let rawGaps := (parseGaps request.rawEvidenceKnownGaps).toOption.getD []
  let resultGaps := canonicalGaps (runGaps ++ rawGaps ++ observationGaps)
  let evidence := evidenceArtifact request evaluation observationGaps
  let result := resultArtifact request evaluation evidence resultGaps
  if !evidence.isValidTransport then throw (adapterError "evidence")
  if !result.isValidTransport then
    throw (adapterError "result")
  let evidenceJson := canonicalJsonValue (canonicalEvidenceArtifactJson evidence)
  let resultJson := canonicalJsonValue (canonicalResultArtifactJson result)
  pure {
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
    implementationLink := jsonField resultJson "implementationLink"
    implementationLinkStatus := result.implementationLinkStatus
    evidenceBackedModelTrace := jsonField evidenceJson "evidenceBackedModelTrace"
    evidenceLinks := jsonField evidenceJson "evidenceLinks"
    dispositions := jsonField evidenceJson "dispositions"
    diagnostics := jsonField evidenceJson "diagnostics"
    observationKnownGaps := gapsJson observationGaps
    propertyVerdicts := jsonField resultJson "propertyVerdicts"
    querySummary := jsonField resultJson "querySummary"
    semanticStatus := result.semanticStatus
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
      match evaluateRequest request with
      | .error failure => { status := 1, stdout := .empty, stderr := errorLine failure }
      | .ok response => match encodeResponse response with
        | .error failure => { status := 1, stdout := .empty, stderr := errorLine failure }
        | .ok output => { status := 0, stdout := output, stderr := "" }

end Temporal.Tool.RunEvaluation
