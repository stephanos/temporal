import Umpire.Artifact.Evidence
import Umpire.Artifact.Tests.Runtime

/-! RawEvidence exact v2 bytes, checksums, bounded grammar, causality, and Run closure. -/

namespace Umpire.Artifact.Tests.Evidence

open Umpire
open Umpire.Examples.Switch
open Umpire.Artifact.Tests.Runtime

#check (RawEvidence.knownGaps : RawEvidence → KnownGapSet)

private def id (value : String) : DefinitionId := DefinitionId.of value

private def fingerprint (value : String) : BehaviorFingerprint :=
  (BehaviorFingerprint.parse? value).getD (behaviorFingerprintOf "invalid")

private def checksum (value : String) : ArtifactChecksum :=
  (ArtifactChecksum.parse? value).getD (drivePlanChecksumOf "invalid")

private def emptyChecksum : ArtifactChecksum :=
  checksum "sha256:0000000000000000000000000000000000000000000000000000000000000000"

private def interpretationKnownGaps : KnownGapSet :=
  (KnownGapSet.ofUnordered [{
    kind := .interpretation
    code := id "switch.gap.interpretation"
  }]).toOption.getD KnownGapSet.empty

private def field
    (definitionId : String)
    (disposition : RawFieldDisposition)
    (value : RawFieldValue) : RawEvidenceField := {
  fieldDefinitionId := id definitionId
  disposition
  value
}

private def fact
    (definitionId source : String)
    (ordinal : Nat)
    (kind : String)
    (causes : List String)
    (fields : List RawEvidenceField) : RawEvidenceFact := {
  factDefinitionId := id definitionId
  sourceDefinitionId := id source
  ordinal
  kindDefinitionId := id kind
  causalFactDefinitionIds := causes.map id
  fields
}

private def rawEvidenceDraft : RawEvidence := {
  formatVersion := "umpire-raw-evidence/v2"
  runIdentity := experimentRun.runIdentity
  behaviorFingerprint :=
    fingerprint "sha256:2a0e83ab40ee0bb739827351e4fca37e29095333c469b975278f882ed3581e8c"
  experiment := compiledArtifact.artifactBinding
  runtimeConfiguration := runtimeConfiguration.artifactBinding
  run := experimentRun.artifactBinding
  captureStatus := .closed
  sources := [
    { sourceDefinitionId := id "umpire.evidence.source.cleanup", status := .closed,
      factCount := 1, byteCount := 64 },
    { sourceDefinitionId := id "umpire.evidence.source.control-receipt", status := .closed,
      factCount := 1, byteCount := 128 },
    { sourceDefinitionId := id "umpire.evidence.source.history", status := .closed,
      factCount := 2, byteCount := 512 },
    { sourceDefinitionId := id "umpire.evidence.source.participant-output", status := .closed,
      factCount := 2, byteCount := 256 }
  ]
  facts := [
    fact "switch.evidence.cleanup.1" "umpire.evidence.source.cleanup" 0
      "umpire.evidence.kind.cleanup" [] [
        field "umpire.evidence.field.status" .plain (.text "complete")
      ],
    fact "switch.evidence.control-receipt.1" "umpire.evidence.source.control-receipt" 0
      "umpire.evidence.kind.control-receipt" [] [
        field "umpire.evidence.field.action-definition-id" .plain (.text "switch.action.flip"),
        field "umpire.evidence.field.attempt" .plain (.integer 1),
        field "umpire.evidence.field.occurrence-definition-id" .plain
          (.text "switch.occurrence.flip"),
        field "umpire.evidence.field.status" .plain (.text "accepted")
      ],
    fact "switch.evidence.history.1" "umpire.evidence.source.history" 0
      "umpire.evidence.kind.history" [] [
        field "umpire.evidence.field.event" .plain (.text "flip-requested")
      ],
    fact "switch.evidence.history.2" "umpire.evidence.source.history" 1
      "umpire.evidence.kind.history" ["switch.evidence.history.1"] [
        field "umpire.evidence.field.event" .plain (.text "flip-completed")
      ],
    fact "switch.evidence.participant.1" "umpire.evidence.source.participant-output" 0
      "umpire.evidence.kind.participant-output" [] [
        field "umpire.evidence.field.state" .plain (.boolean false)
      ],
    fact "switch.evidence.participant.2" "umpire.evidence.source.participant-output" 1
      "umpire.evidence.kind.participant-output" ["switch.evidence.participant.1"] [
        field "umpire.evidence.field.digest" .sha256
          (.text "sha256:463fc89536c47a3158c1d27030df5d1b9a5665bc256a7079632485eb3b0e3f86"),
        field "umpire.evidence.field.rejected" .rejected .null,
        field "umpire.evidence.field.secret" .redacted .null,
        field "umpire.evidence.field.state" .plain (.boolean true)
      ]
  ]
  knownGaps := KnownGapSet.empty
  provenance := {
    sourceDefinitionIds := [id "switch.raw-evidence.1"]
    sourceLocations := [{
      path := "Umpire/Artifact/Tests/Evidence.lean"
      line := 1
      column := 1
      provenance := "lean-model"
    }]
  }
  provenanceChecksum := emptyChecksum
  artifactChecksum := emptyChecksum
}

def rawEvidence : RawEvidence := rawEvidenceDraft.seal

/-! Lean owns the authoritative RawEvidence fixture bytes. -/
example : canonicalRawEvidenceBytes rawEvidence =
    include_str "Fixtures/RawEvidenceV2.json" := by
  native_decide

/-! Both checksum layers use the exact deterministic pretty preimages. -/
example : rawEvidence.hasValidChecksums &&
    rawEvidence.provenanceChecksum.render =
      "sha256:58874d22fb498df81f0ad4a5812183031af5827e3f528d963d147cb760ee5bb7" &&
    rawEvidence.artifactChecksum.render =
      "sha256:02980732154cfc8fa80487fc945931fa09046d5ed32c620f890b23876dfec67d" := by
  native_decide

/-! Nonempty checked Known Gaps preserve their exact canonical JSON projection. -/
example :
    let evidence := { rawEvidence with knownGaps := interpretationKnownGaps }.seal
    evidence.isValidTransport &&
      (canonicalRawEvidenceBytes evidence).contains
        "\"knownGaps\": [\n    {\n      \"kind\": \"interpretation\",\n      \"code\": \"switch.gap.interpretation\",\n      \"subject\": null,\n      \"detail\": null\n    }\n  ]" := by
  native_decide

/-! The canonical value closes exact bindings, source summaries, and the attempted receipt. -/
example : rawEvidence.isValidTransport &&
    rawEvidence.closes compiledArtifact runtimeConfiguration experimentRun := by
  native_decide

/-! Partial and failed source summaries determine the matching capture status. -/
example :
    (let partialEvidence := { rawEvidence with
      captureStatus := .partiallyClosed
      sources := rawEvidence.sources.map fun (source : RawEvidenceSource) =>
        if source.sourceDefinitionId == id "umpire.evidence.source.cleanup" then
          { source with status := .partiallyClosed }
        else source };
      partialEvidence.seal.isValidTransport) &&
    (let failedEvidence := { rawEvidence with
      captureStatus := .failed
      sources := rawEvidence.sources.map fun (source : RawEvidenceSource) =>
        if source.sourceDefinitionId == id "umpire.evidence.source.cleanup" then
          { source with status := .failed }
        else source };
      failedEvidence.seal.isValidTransport) := by
  native_decide

/-! Source/fact/field order, ordinal gaps, and non-prior causes reject without normalization. -/
example :
    !({ rawEvidence with sources := rawEvidence.sources.reverse }).seal.isValidTransport &&
    !({ rawEvidence with facts := rawEvidence.facts.reverse }).seal.isValidTransport &&
    !({ rawEvidence with facts := rawEvidence.facts.map fun (current : RawEvidenceFact) =>
      if current.factDefinitionId == id "switch.evidence.cleanup.1" then
        { current with ordinal := 1 }
      else current }).seal.isValidTransport &&
    !({ rawEvidence with facts := rawEvidence.facts.map fun (current : RawEvidenceFact) =>
      if current.factDefinitionId == id "switch.evidence.cleanup.1" then
        { current with causalFactDefinitionIds := [id "switch.evidence.control-receipt.1"] }
      else current }).seal.isValidTransport &&
    !({ rawEvidence with facts := rawEvidence.facts.map fun (current : RawEvidenceFact) =>
      if current.factDefinitionId == id "switch.evidence.control-receipt.1" then
        { current with fields := current.fields.reverse }
      else current }).seal.isValidTransport := by
  native_decide

/-! Redacted, rejected, and digest dispositions cannot retain prohibited raw values. -/
example :
    !({ rawEvidence with facts := rawEvidence.facts.map fun (current : RawEvidenceFact) =>
      if current.factDefinitionId == id "switch.evidence.cleanup.1" then
        { current with fields := [field "umpire.evidence.field.status" .redacted (.text "secret")] }
      else current }).seal.isValidTransport &&
    !({ rawEvidence with facts := rawEvidence.facts.map fun (current : RawEvidenceFact) =>
      if current.factDefinitionId == id "switch.evidence.cleanup.1" then
        { current with fields := [field "umpire.evidence.field.status" .rejected (.boolean false)] }
      else current }).seal.isValidTransport &&
    !({ rawEvidence with facts := rawEvidence.facts.map fun (current : RawEvidenceFact) =>
      if current.factDefinitionId == id "switch.evidence.cleanup.1" then
        { current with fields := [field "umpire.evidence.field.status" .sha256 (.text "malformed")] }
      else current }).seal.isValidTransport := by
  native_decide

/-! Receipt source, kind, occurrence, action, attempt, and status remain exact Run bindings. -/
example :
    !({ rawEvidence with facts := rawEvidence.facts.map fun (current : RawEvidenceFact) =>
      if current.factDefinitionId == id "switch.evidence.control-receipt.1" then
        { current with kindDefinitionId := id "umpire.evidence.kind.history" }
      else current }).closes compiledArtifact runtimeConfiguration experimentRun &&
    !({ rawEvidence with facts := rawEvidence.facts.map fun (current : RawEvidenceFact) =>
      if current.factDefinitionId == id "switch.evidence.control-receipt.1" then
        { current with fields := current.fields.map fun (currentField : RawEvidenceField) =>
          if currentField.fieldDefinitionId == id "umpire.evidence.field.status" then
            { currentField with value := .text "failed" }
          else currentField }
      else current }).closes compiledArtifact runtimeConfiguration experimentRun &&
    !({ rawEvidence with facts := rawEvidence.facts.map fun (current : RawEvidenceFact) =>
      if current.factDefinitionId == id "switch.evidence.control-receipt.1" then
        { current with fields := current.fields ++
          [field "umpire.evidence.field.unexpected" .plain .null] }
      else current }).closes compiledArtifact runtimeConfiguration experimentRun &&
    !({ rawEvidence with facts := rawEvidence.facts ++
      rawEvidence.facts.filter fun (current : RawEvidenceFact) =>
        current.factDefinitionId == id "switch.evidence.control-receipt.1" }).closes
          compiledArtifact runtimeConfiguration experimentRun := by
  native_decide

end Umpire.Artifact.Tests.Evidence
