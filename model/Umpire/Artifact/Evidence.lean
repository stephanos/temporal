import Umpire.Artifact.Runtime

namespace Umpire

/-! Exact inert v2 transport for bounded raw facts captured during one Run. -/

private def quoteEvidence (value : String) : String := Lean.Json.compress (.str value)

private def evidenceArray (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def definitionIdLeEvidence (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def canonicalEvidenceIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort definitionIdLeEvidence |>.eraseDups

/-- The closed JSON scalar grammar retained by one raw field. -/
inductive RawFieldValue where
  | null
  | boolean (value : Bool)
  | integer (value : Int)
  | text (value : String)
  deriving BEq, DecidableEq, Repr

private def RawFieldValue.canonicalJson : RawFieldValue → String
  | .null => "null"
  | .boolean false => "false"
  | .boolean true => "true"
  | .integer value => toString value
  | .text value => quoteEvidence value

private def RawFieldValue.payloadBytes : RawFieldValue → Nat
  | .null => 0
  | .boolean _ => 1
  | .integer value => (toString value).toUTF8.size
  | .text value => value.toUTF8.size

/-- The four closed non-interpretive field dispositions. -/
inductive RawFieldDisposition where
  | plain
  | redacted
  | sha256
  | rejected
  deriving BEq, DecidableEq, Ord, Repr

def RawFieldDisposition.name : RawFieldDisposition → String
  | .plain => "plain"
  | .redacted => "redacted"
  | .sha256 => "sha256"
  | .rejected => "rejected"

/-- One typed raw field that does not carry accepted model meaning. -/
structure RawEvidenceField where
  fieldDefinitionId : DefinitionId
  disposition : RawFieldDisposition
  value : RawFieldValue
  deriving BEq, DecidableEq, Repr

/-- One source-local raw fact and its already-observed causal parents. -/
structure RawEvidenceFact where
  factDefinitionId : DefinitionId
  sourceDefinitionId : DefinitionId
  ordinal : Nat
  kindDefinitionId : DefinitionId
  causalFactDefinitionIds : List DefinitionId
  fields : List RawEvidenceField
  deriving BEq, DecidableEq, Repr

/-- The closed summary for one bounded raw source. -/
structure RawEvidenceSource where
  sourceDefinitionId : DefinitionId
  status : SourceClosureStatus
  factCount : Nat
  byteCount : Nat
  deriving BEq, DecidableEq, Repr

/-- One inert bounded capture; it contains no Observation Evaluation or Property result. -/
structure RawEvidence where
  formatVersion : String
  runIdentity : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  experiment : ArtifactBinding
  runtimeConfiguration : ArtifactBinding
  run : ArtifactBinding
  captureStatus : SourceClosureStatus
  sources : List RawEvidenceSource
  facts : List RawEvidenceFact
  knownGaps : KnownGapSet
  provenance : ArtifactProvenance
  provenanceChecksum : ArtifactChecksum
  artifactChecksum : ArtifactChecksum
  deriving BEq, DecidableEq, Repr

private def rawEvidenceSourceJson (source : RawEvidenceSource) : String :=
  "{\"sourceDefinitionId\":" ++ quoteEvidence source.sourceDefinitionId.value ++
    ",\"status\":" ++ quoteEvidence source.status.name ++
    ",\"factCount\":" ++ toString source.factCount ++
    ",\"byteCount\":" ++ toString source.byteCount ++ "}"

private def rawEvidenceFieldJson (field : RawEvidenceField) : String :=
  "{\"fieldDefinitionId\":" ++ quoteEvidence field.fieldDefinitionId.value ++
    ",\"disposition\":" ++ quoteEvidence field.disposition.name ++
    ",\"value\":" ++ field.value.canonicalJson ++ "}"

private def rawEvidenceFactJson (fact : RawEvidenceFact) : String :=
  "{\"factDefinitionId\":" ++ quoteEvidence fact.factDefinitionId.value ++
    ",\"sourceDefinitionId\":" ++ quoteEvidence fact.sourceDefinitionId.value ++
    ",\"ordinal\":" ++ toString fact.ordinal ++
    ",\"kindDefinitionId\":" ++ quoteEvidence fact.kindDefinitionId.value ++
    ",\"causalFactDefinitionIds\":" ++ evidenceArray
      (fact.causalFactDefinitionIds.map (quoteEvidence ∘ DefinitionId.value)) ++
    ",\"fields\":" ++ evidenceArray (fact.fields.map rawEvidenceFieldJson) ++ "}"

private def rawEvidenceContentJson (evidence : RawEvidence) : String :=
  "{\"formatVersion\":" ++ quoteEvidence evidence.formatVersion ++
    ",\"runIdentity\":" ++ quoteEvidence evidence.runIdentity.value ++
    ",\"behaviorFingerprint\":" ++ quoteEvidence evidence.behaviorFingerprint.render ++
    ",\"experiment\":" ++ evidence.experiment.canonicalJson ++
    ",\"runtimeConfiguration\":" ++ evidence.runtimeConfiguration.canonicalJson ++
    ",\"run\":" ++ evidence.run.canonicalJson ++
    ",\"captureStatus\":" ++ quoteEvidence evidence.captureStatus.name ++
    ",\"sources\":" ++ evidenceArray (evidence.sources.map rawEvidenceSourceJson) ++
    ",\"facts\":" ++ evidenceArray (evidence.facts.map rawEvidenceFactJson) ++
    ",\"knownGaps\":" ++ evidenceArray (evidence.knownGaps.toList.map canonicalKnownGapJson) ++
    ",\"provenance\":" ++ evidence.provenance.canonicalJson ++
    ",\"provenanceChecksum\":" ++ quoteEvidence evidence.provenanceChecksum.render ++ "}"

def RawEvidence.expectedArtifactChecksum (evidence : RawEvidence) : ArtifactChecksum :=
  rawEvidenceChecksumOf (Json.prettyBytes (rawEvidenceContentJson evidence))

def RawEvidence.seal (evidence : RawEvidence) : RawEvidence :=
  let withProvenance := { evidence with provenanceChecksum := evidence.provenance.expectedChecksum }
  { withProvenance with artifactChecksum := withProvenance.expectedArtifactChecksum }

def RawEvidence.hasValidChecksums (evidence : RawEvidence) : Bool :=
  evidence.provenanceChecksum == evidence.provenance.expectedChecksum &&
    evidence.artifactChecksum == evidence.expectedArtifactChecksum

def canonicalRawEvidenceJson (evidence : RawEvidence) : String :=
  let content := rawEvidenceContentJson evidence
  Json.pretty ((content.dropEnd 1).toString ++
    ",\"artifactChecksum\":" ++ quoteEvidence evidence.artifactChecksum.render ++ "}")

def canonicalRawEvidenceBytes (evidence : RawEvidence) : String :=
  canonicalRawEvidenceJson evidence ++ "\n"

private def rawEvidenceSourceLe (left right : RawEvidenceSource) : Bool :=
  decide (left.sourceDefinitionId.value ≤ right.sourceDefinitionId.value)

private def rawEvidenceFactLe (left right : RawEvidenceFact) : Bool :=
  decide (left.sourceDefinitionId.value < right.sourceDefinitionId.value) ||
    (left.sourceDefinitionId == right.sourceDefinitionId && left.ordinal < right.ordinal) ||
    (left.sourceDefinitionId == right.sourceDefinitionId && left.ordinal == right.ordinal &&
      decide (left.factDefinitionId.value ≤ right.factDefinitionId.value))

private def rawEvidenceFieldLe (left right : RawEvidenceField) : Bool :=
  decide (left.fieldDefinitionId.value ≤ right.fieldDefinitionId.value)

private def rawEvidenceFieldValid (field : RawEvidenceField) : Bool :=
  field.fieldDefinitionId.isNamespaced &&
    match field.disposition, field.value with
    | .plain, _ => true
    | .redacted, .null => true
    | .sha256, .text value => (ArtifactChecksum.parse? value).isSome
    | .rejected, .null => true
    | _, _ => false

private def rawEvidenceFieldsValid (fields : List RawEvidenceField) : Bool :=
  fields.length ≤ 128 && fields == fields.mergeSort rawEvidenceFieldLe &&
    (fields.map RawEvidenceField.fieldDefinitionId).eraseDups.length == fields.length &&
    fields.all rawEvidenceFieldValid

private def rawEvidenceFactsCausalValid : List RawEvidenceFact → List DefinitionId → Bool
  | [], _ => true
  | fact :: rest, seen =>
      fact.factDefinitionId.isNamespaced && fact.sourceDefinitionId.isNamespaced &&
        fact.kindDefinitionId.isNamespaced &&
        fact.causalFactDefinitionIds == canonicalEvidenceIds fact.causalFactDefinitionIds &&
        fact.causalFactDefinitionIds.all seen.contains && rawEvidenceFieldsValid fact.fields &&
        rawEvidenceFactsCausalValid rest (seen ++ [fact.factDefinitionId])

private def rawEvidenceSourceCountsValid
    (sources : List RawEvidenceSource)
    (facts : List RawEvidenceFact) : Bool :=
  sources.all fun source =>
    let sourceFacts := facts.filter fun fact => fact.sourceDefinitionId == source.sourceDefinitionId
    source.factCount == sourceFacts.length &&
      sourceFacts.map RawEvidenceFact.ordinal == List.range source.factCount

private def rawEvidencePayloadValid (facts : List RawEvidenceFact) : Bool :=
  facts.all (fun fact => (fact.fields.map (RawFieldValue.payloadBytes ∘ RawEvidenceField.value)).sum ≤
    1048576) &&
    (facts.map fun fact =>
      (fact.fields.map (RawFieldValue.payloadBytes ∘ RawEvidenceField.value)).sum).sum ≤ 16777216

private def expectedCaptureStatus (sources : List RawEvidenceSource) : SourceClosureStatus :=
  if sources.any fun source => source.status == .failed then .failed
  else if sources.any fun source => source.status == .partiallyClosed then .partiallyClosed
  else .closed

/-- Check exact bounded raw structure without interpreting any field as model meaning. -/
def RawEvidence.isValidTransport (evidence : RawEvidence) : Bool :=
  evidence.formatVersion == "umpire-raw-evidence/v2" && evidence.runIdentity.isNamespaced &&
    evidence.experiment.formatVersion == "umpire-experiment/v2" &&
    evidence.runtimeConfiguration.formatVersion == "umpire-runtime-configuration/v2" &&
    evidence.run.formatVersion == "umpire-experiment-run/v2" &&
    evidence.sources != [] && evidence.sources.length ≤ 64 && evidence.facts.length ≤ 4096 &&
    evidence.sources == evidence.sources.mergeSort rawEvidenceSourceLe &&
    (evidence.sources.map RawEvidenceSource.sourceDefinitionId).eraseDups.length ==
      evidence.sources.length &&
    evidence.sources.all (fun source => source.sourceDefinitionId.isNamespaced) &&
    evidence.facts == evidence.facts.mergeSort rawEvidenceFactLe &&
    (evidence.facts.map RawEvidenceFact.factDefinitionId).eraseDups.length == evidence.facts.length &&
    evidence.facts.all (fun fact => evidence.sources.any fun source =>
      source.sourceDefinitionId == fact.sourceDefinitionId) &&
    rawEvidenceFactsCausalValid evidence.facts [] &&
    rawEvidenceSourceCountsValid evidence.sources evidence.facts &&
    rawEvidencePayloadValid evidence.facts && evidence.captureStatus == expectedCaptureStatus evidence.sources &&
    evidence.provenance.isValidTransport &&
    evidence.hasValidChecksums

private def controlReceiptSourceId : DefinitionId :=
  DefinitionId.of "umpire.evidence.source.control-receipt"

private def controlReceiptKindId : DefinitionId :=
  DefinitionId.of "umpire.evidence.kind.control-receipt"

private def controlReceiptActionFieldId : DefinitionId :=
  DefinitionId.of "umpire.evidence.field.action-definition-id"

private def controlReceiptAttemptFieldId : DefinitionId :=
  DefinitionId.of "umpire.evidence.field.attempt"

private def controlReceiptOccurrenceFieldId : DefinitionId :=
  DefinitionId.of "umpire.evidence.field.occurrence-definition-id"

private def controlReceiptStatusFieldId : DefinitionId :=
  DefinitionId.of "umpire.evidence.field.status"

private def plainFieldEquals
    (fact : RawEvidenceFact)
    (fieldId : DefinitionId)
    (value : RawFieldValue) : Bool :=
  let matchedFields := fact.fields.filter fun field => field.fieldDefinitionId == fieldId
  matchedFields.length == 1 && matchedFields.all fun field =>
    field.disposition == .plain && field.value == value

private def controlReceiptMatches (fact : RawEvidenceFact) (attempt : ControlAttempt) : Bool :=
  fact.sourceDefinitionId == controlReceiptSourceId && fact.kindDefinitionId == controlReceiptKindId &&
    fact.fields.length == 4 &&
    plainFieldEquals fact controlReceiptOccurrenceFieldId (.text attempt.occurrenceDefinitionId.value) &&
    plainFieldEquals fact controlReceiptActionFieldId (.text attempt.actionDefinitionId.value) &&
    plainFieldEquals fact controlReceiptAttemptFieldId (.integer (Int.ofNat attempt.attempt)) &&
    plainFieldEquals fact controlReceiptStatusFieldId (.text attempt.status.name)

private def controlReceiptsClose (evidence : RawEvidence) (run : ExperimentRun) : Bool :=
  let attempted := run.controlAttempts.filter fun attempt => attempt.status != .notAttempted
  let referenced := attempted.filterMap ControlAttempt.receiptFactDefinitionId
  referenced.eraseDups.length == referenced.length && attempted.all fun attempt =>
    match attempt.receiptFactDefinitionId with
    | none => false
    | some receiptId =>
        let matchedFacts := evidence.facts.filter fun fact => fact.factDefinitionId == receiptId
        matchedFacts.length == 1 && matchedFacts.all fun fact => controlReceiptMatches fact attempt
  && evidence.facts.all fun fact =>
    if fact.sourceDefinitionId == controlReceiptSourceId || fact.kindDefinitionId == controlReceiptKindId then
      fact.sourceDefinitionId == controlReceiptSourceId && fact.kindDefinitionId == controlReceiptKindId &&
        referenced.contains fact.factDefinitionId
    else true

private def RawEvidenceSource.asRunClosure (source : RawEvidenceSource) : SourceClosure := {
  sourceDefinitionId := source.sourceDefinitionId
  status := source.status
  recordCount := source.factCount
  byteCount := source.byteCount
}

/-- Close RawEvidence over exact inputs, source summaries, and attempted control receipts. -/
def RawEvidence.closes
    (evidence : RawEvidence)
    (experiment : ExperimentSpec)
    (configuration : RuntimeConfiguration)
    (run : ExperimentRun) : Bool :=
  run.closes experiment configuration && evidence.experiment == experiment.artifactBinding &&
    evidence.runtimeConfiguration == configuration.artifactBinding &&
    evidence.run == run.artifactBinding && evidence.runIdentity == run.runIdentity &&
    evidence.sources.map RawEvidenceSource.asRunClosure == run.sourceClosures &&
    controlReceiptsClose evidence run

end Umpire
