package artifact_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestRawEvidenceV2CanonicalFixtureRoundTrip(t *testing.T) {
	authoritative := readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/RawEvidenceV2.json")
	encoded := readExperimentV2Fixture(t,
		"tools/umpire/artifact/testdata/raw-evidence-v2.json")
	// The Go testdata remains a byte-exact mirror of its Lean-owned fixture.
	//nolint:testifylint
	require.Equal(t, authoritative, encoded)

	document, err := artifact.DecodeRawEvidenceV2(encoded)
	require.NoError(t, err)
	reencoded, err := artifact.EncodeRawEvidenceV2(document)
	require.NoError(t, err)
	require.Equal(t, encoded, reencoded)
	require.True(t, bytes.HasSuffix(reencoded, []byte{'\n'}))
	require.False(t, bytes.HasSuffix(reencoded, []byte("\n\n")))

	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	require.NoError(t,
		artifact.ValidateRawEvidenceV2Closure(document, experiment, runtimeConfiguration, run))
}

func TestRawEvidenceV2ChecksumsUseExactPrettyPreimages(t *testing.T) {
	document := rawEvidenceV2Document(t)
	provenancePreimage, err := artifact.CanonicalPretty(document.Provenance)
	require.NoError(t, err)
	requireCanonicalJSONLine(t, provenancePreimage)
	require.Equal(t, document.ProvenanceChecksum,
		independentExperimentV2Checksum("umpire.provenance/v2", provenancePreimage))

	document.ArtifactChecksum = ""
	artifactPreimage, err := artifact.CanonicalPretty(document)
	require.NoError(t, err)
	requireCanonicalJSONLine(t, artifactPreimage)
	require.Equal(t, rawEvidenceV2Document(t).ArtifactChecksum,
		independentExperimentV2Checksum("umpire.raw-evidence/v2", artifactPreimage))
}

func TestRawEvidenceV2AcceptsClosedStatusAndValueGrammar(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.RawEvidence)
	}{
		{
			name: "partial capture",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Sources[0].Status = "partial"
				document.CaptureStatus = "partial"
			},
		},
		{
			name: "failed capture",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Sources[0].Status = "failed"
				document.CaptureStatus = "failed"
			},
		},
		{
			name: "plain null",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].Fields[0].Value = nil
			},
		},
		{
			name: "plain negative integer",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].Fields[0].Value = json.Number("-1")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := artifact.DecodeRawEvidenceV2(resealedRawEvidenceV2Mutation(t, test.mutate))
			require.NoError(t, err)
		})
	}
}

func TestRawEvidenceV2RejectsClosedGrammarMutations(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.RawEvidence)
		code   artifact.ErrorCode
	}{
		{
			name: "capture status",
			mutate: func(document *artifactv2.RawEvidence) {
				document.CaptureStatus = "accepted"
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "source order",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Sources[0], document.Sources[1] = document.Sources[1], document.Sources[0]
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "source status",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Sources[0].Status = "succeeded"
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "duplicate source",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Sources[1] = document.Sources[0]
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "source fact count",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Sources[0].FactCount = artifactv2.NaturalFromUint64(2)
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "fact order",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0], document.Facts[1] = document.Facts[1], document.Facts[0]
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "duplicate fact",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[1].FactDefinitionID = document.Facts[0].FactDefinitionID
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "ordinal gap",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].Ordinal = artifactv2.NaturalFromUint64(1)
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "unknown source",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].SourceDefinitionID = "umpire.evidence.source.absent"
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "forward causal reference",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].CausalFactDefinitionIDs = []string{document.Facts[1].FactDefinitionID}
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "dangling causal reference",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[1].CausalFactDefinitionIDs = []string{"switch.evidence.absent"}
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "causal cycle",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[1].CausalFactDefinitionIDs = []string{document.Facts[1].FactDefinitionID}
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "duplicate causal reference",
			mutate: func(document *artifactv2.RawEvidence) {
				parent := document.Facts[5].CausalFactDefinitionIDs[0]
				document.Facts[5].CausalFactDefinitionIDs = []string{parent, parent}
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "causal reference order",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[5].CausalFactDefinitionIDs = []string{
					"switch.evidence.participant.1",
					"switch.evidence.history.1",
				}
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "field order",
			mutate: func(document *artifactv2.RawEvidence) {
				fields := document.Facts[1].Fields
				fields[0], fields[1] = fields[1], fields[0]
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "duplicate field",
			mutate: func(document *artifactv2.RawEvidence) {
				field := document.Facts[1].Fields[0]
				document.Facts[1].Fields = append(document.Facts[1].Fields, field)
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "plain object",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].Fields[0].Value = map[string]any{"accepted": true}
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "plain non-integer number",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].Fields[0].Value = json.Number("1.5")
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "redacted material",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].Fields[0].Disposition = "redacted"
				document.Facts[0].Fields[0].Value = "secret"
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "rejected material",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].Fields[0].Disposition = "rejected"
				document.Facts[0].Fields[0].Value = false
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "sha256 malformed",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].Fields[0].Disposition = "sha256"
				document.Facts[0].Fields[0].Value = "not-a-checksum"
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "unknown disposition",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Facts[0].Fields[0].Disposition = "retained"
			},
			code: artifact.ErrorMalformedValue,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := artifact.DecodeRawEvidenceV2(resealedRawEvidenceV2Mutation(t, test.mutate))
			requireRawEvidenceV2ErrorCode(t, err, test.code)
		})
	}
}

func TestRawEvidenceV2RejectsSemanticAndCanonicalMutations(t *testing.T) {
	document := rawEvidenceV2Document(t)
	encoded, err := artifactv2.CanonicalRawEvidenceBytes(document)
	require.NoError(t, err)

	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, encoded))
	unknownSemantic := replaceExperimentV2Once(t, encoded,
		"  \"runIdentity\":", "  \"acceptedModelFact\": true,\n  \"runIdentity\":")
	malformedType := replaceExperimentV2Once(t, encoded,
		"  \"captureStatus\": \"closed\",", "  \"captureStatus\": true,")
	wrongArtifactChecksum := replaceExperimentV2Once(t, encoded, document.ArtifactChecksum,
		"sha256:"+strings.Repeat("0", 64))
	wrongProvenanceChecksum := replaceExperimentV2Once(t, encoded, document.ProvenanceChecksum,
		"sha256:"+strings.Repeat("f", 64))

	for _, test := range []struct {
		name    string
		encoded []byte
		code    artifact.ErrorCode
	}{
		{name: "compact", encoded: compact.Bytes(), code: artifact.ErrorNoncanonical},
		{
			name: "alternate whitespace",
			encoded: replaceExperimentV2Once(t, encoded,
				"{\n  \"formatVersion\":", "{\n    \"formatVersion\":"),
			code: artifact.ErrorNoncanonical,
		},
		{name: "accepted Model Fact field", encoded: unknownSemantic, code: artifact.ErrorUnknownField},
		{name: "malformed field type", encoded: malformedType, code: artifact.ErrorMalformedValue},
		{name: "Artifact Checksum", encoded: wrongArtifactChecksum, code: artifact.ErrorArtifactChecksum},
		{
			name:    "provenance checksum",
			encoded: wrongProvenanceChecksum,
			code:    artifact.ErrorProvenanceChecksum,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := artifact.DecodeRawEvidenceV2(test.encoded)
			requireRawEvidenceV2ErrorCode(t, err, test.code)
		})
	}

}

func TestRawEvidenceV2ClosesBindingsSourcesAndControlReceipts(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	canonical := rawEvidenceV2Document(t)
	require.NoError(t,
		artifact.ValidateRawEvidenceV2Closure(canonical, experiment, runtimeConfiguration, run))

	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.RawEvidence)
	}{
		{
			name: "experiment binding",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Experiment = document.Run
			},
		},
		{
			name: "runtime configuration binding",
			mutate: func(document *artifactv2.RawEvidence) {
				document.RuntimeConfiguration = document.Experiment
			},
		},
		{
			name: "Run binding",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Run = document.RuntimeConfiguration
			},
		},
		{
			name: "Run identity",
			mutate: func(document *artifactv2.RawEvidence) {
				document.RunIdentity = "switch.run.crossed"
			},
		},
		{
			name: "source closure",
			mutate: func(document *artifactv2.RawEvidence) {
				document.Sources[0].ByteCount = artifactv2.NaturalFromUint64(65)
			},
		},
		{
			name: "missing receipt",
			mutate: func(document *artifactv2.RawEvidence) {
				rawEvidenceV2ReceiptFact(t, document).FactDefinitionID = "switch.evidence.control-receipt.absent"
			},
		},
		{
			name: "crossed receipt",
			mutate: func(document *artifactv2.RawEvidence) {
				rawEvidenceV2ReceiptFact(t, document).SourceDefinitionID =
					"umpire.evidence.source.history"
			},
		},
		{
			name: "receipt kind",
			mutate: func(document *artifactv2.RawEvidence) {
				rawEvidenceV2ReceiptFact(t, document).KindDefinitionID = "umpire.evidence.kind.history"
			},
		},
		{
			name: "receipt occurrence",
			mutate: func(document *artifactv2.RawEvidence) {
				rawEvidenceV2ReceiptField(t, document,
					artifactv2.ControlReceiptOccurrenceFieldDefinitionID).Value = "switch.occurrence.crossed"
			},
		},
		{
			name: "receipt action",
			mutate: func(document *artifactv2.RawEvidence) {
				rawEvidenceV2ReceiptField(t, document,
					artifactv2.ControlReceiptActionFieldDefinitionID).Value = "switch.action.crossed"
			},
		},
		{
			name: "receipt attempt",
			mutate: func(document *artifactv2.RawEvidence) {
				rawEvidenceV2ReceiptField(t, document,
					artifactv2.ControlReceiptAttemptFieldDefinitionID).Value = json.Number("2")
			},
		},
		{
			name: "receipt status",
			mutate: func(document *artifactv2.RawEvidence) {
				rawEvidenceV2ReceiptField(t, document,
					artifactv2.ControlReceiptStatusFieldDefinitionID).Value = "failed"
			},
		},
		{
			name: "receipt field missing",
			mutate: func(document *artifactv2.RawEvidence) {
				fact := rawEvidenceV2ReceiptFact(t, document)
				fact.Fields = fact.Fields[1:]
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := rawEvidenceV2Document(t)
			test.mutate(&document)
			err := artifact.ValidateRawEvidenceV2Closure(
				document, experiment, runtimeConfiguration, run)
			requireRawEvidenceV2ErrorCode(t, err, artifact.ErrorClosure)
		})
	}

	t.Run("duplicate receipt fact", func(t *testing.T) {
		document := rawEvidenceV2Document(t)
		document.Facts = append(document.Facts, *rawEvidenceV2ReceiptFact(t, &document))
		err := artifact.ValidateRawEvidenceV2Closure(
			document, experiment, runtimeConfiguration, run)
		requireRawEvidenceV2ErrorCode(t, err, artifact.ErrorClosure)
	})

	t.Run("Run receipt crosses to another fact", func(t *testing.T) {
		crossedRun := run
		crossedRun.ControlAttempts = append([]artifactv2.ControlAttempt(nil), run.ControlAttempts...)
		crossedReceipt := "switch.evidence.history.1"
		crossedRun.ControlAttempts[0].ReceiptFactDefinitionID = &crossedReceipt
		err := artifact.ValidateRawEvidenceV2Closure(
			canonical, experiment, runtimeConfiguration, crossedRun)
		requireRawEvidenceV2ErrorCode(t, err, artifact.ErrorClosure)
	})

	t.Run("receipt has an extra field", func(t *testing.T) {
		document := rawEvidenceV2Document(t)
		fact := rawEvidenceV2ReceiptFact(t, &document)
		fact.Fields = append(fact.Fields, artifactv2.RawEvidenceField{
			FieldDefinitionID: "umpire.evidence.field.unexpected", Disposition: "plain", Value: nil,
		})
		err := artifact.ValidateRawEvidenceV2Closure(
			document, experiment, runtimeConfiguration, run)
		requireRawEvidenceV2ErrorCode(t, err, artifact.ErrorClosure)
	})
}

func TestRawEvidenceV2EvidenceCeilings(t *testing.T) {
	for _, test := range []struct {
		name      string
		atLimit   func() artifactv2.RawEvidence
		overLimit func() artifactv2.RawEvidence
		code      artifact.ErrorCode
	}{
		{
			name: "sources",
			atLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2SourcesLimitDocument(t, artifact.MaximumEvidenceSources)
			},
			overLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2SourcesLimitDocument(t, artifact.MaximumEvidenceSources+1)
			},
			code: artifact.ErrorCollectionLimit,
		},
		{
			name: "facts",
			atLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2FactsLimitDocument(t, artifact.MaximumEvidenceFacts)
			},
			overLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2FactsLimitDocument(t, artifact.MaximumEvidenceFacts+1)
			},
			code: artifact.ErrorCollectionLimit,
		},
		{
			name: "fields per fact",
			atLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2FieldsLimitDocument(t, artifact.MaximumFieldsPerEvidenceFact)
			},
			overLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2FieldsLimitDocument(t, artifact.MaximumFieldsPerEvidenceFact+1)
			},
			code: artifact.ErrorCollectionLimit,
		},
		{
			name: "decoded payload per fact",
			atLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2PayloadLimitDocument(t, artifact.MaximumEvidenceFactPayloadBytes, false)
			},
			overLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2PayloadLimitDocument(t, artifact.MaximumEvidenceFactPayloadBytes+1, false)
			},
			code: artifact.ErrorPayloadLimit,
		},
		{
			name: "decoded aggregate payload",
			atLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2PayloadLimitDocument(t, artifact.MaximumRawEvidencePayloadBytes, true)
			},
			overLimit: func() artifactv2.RawEvidence {
				return rawEvidenceV2PayloadLimitDocument(t, artifact.MaximumRawEvidencePayloadBytes+1, true)
			},
			code: artifact.ErrorPayloadLimit,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			atLimit := sealedRawEvidenceV2Bytes(t, test.atLimit())
			_, err := artifact.DecodeRawEvidenceV2(atLimit)
			require.NoError(t, err)

			overLimit := sealedRawEvidenceV2Bytes(t, test.overLimit())
			_, err = artifact.DecodeRawEvidenceV2(overLimit)
			requireRawEvidenceV2ErrorCode(t, err, test.code)
		})
	}
}

func rawEvidenceV2Document(t *testing.T) artifactv2.RawEvidence {
	t.Helper()
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)

	document := artifactv2.RawEvidence{
		FormatVersion:        artifactv2.RawEvidenceFormat,
		RunIdentity:          run.RunIdentity,
		BehaviorFingerprint:  "sha256:2a0e83ab40ee0bb739827351e4fca37e29095333c469b975278f882ed3581e8c",
		Experiment:           experimentBinding,
		RuntimeConfiguration: artifactv2.RuntimeConfigurationArtifactBinding(runtimeConfiguration),
		Run:                  artifactv2.ExperimentRunArtifactBinding(run),
		CaptureStatus:        "closed",
		Sources: []artifactv2.RawEvidenceSource{
			{SourceDefinitionID: "umpire.evidence.source.cleanup", Status: "closed",
				FactCount: artifactv2.NaturalFromUint64(1), ByteCount: artifactv2.NaturalFromUint64(64)},
			{SourceDefinitionID: "umpire.evidence.source.control-receipt", Status: "closed",
				FactCount: artifactv2.NaturalFromUint64(1), ByteCount: artifactv2.NaturalFromUint64(128)},
			{SourceDefinitionID: "umpire.evidence.source.history", Status: "closed",
				FactCount: artifactv2.NaturalFromUint64(2), ByteCount: artifactv2.NaturalFromUint64(512)},
			{SourceDefinitionID: "umpire.evidence.source.participant-output", Status: "closed",
				FactCount: artifactv2.NaturalFromUint64(2), ByteCount: artifactv2.NaturalFromUint64(256)},
		},
		Facts: []artifactv2.RawEvidenceFact{
			{
				FactDefinitionID: "switch.evidence.cleanup.1", SourceDefinitionID: "umpire.evidence.source.cleanup",
				Ordinal: artifactv2.NaturalFromUint64(0), KindDefinitionID: "umpire.evidence.kind.cleanup",
				CausalFactDefinitionIDs: []string{},
				Fields: []artifactv2.RawEvidenceField{{
					FieldDefinitionID: "umpire.evidence.field.status", Disposition: "plain", Value: "complete",
				}},
			},
			{
				FactDefinitionID:   "switch.evidence.control-receipt.1",
				SourceDefinitionID: artifactv2.ControlReceiptSourceDefinitionID,
				Ordinal:            artifactv2.NaturalFromUint64(0), KindDefinitionID: artifactv2.ControlReceiptKindDefinitionID,
				CausalFactDefinitionIDs: []string{},
				Fields: []artifactv2.RawEvidenceField{
					{FieldDefinitionID: artifactv2.ControlReceiptActionFieldDefinitionID,
						Disposition: "plain", Value: "switch.action.flip"},
					{FieldDefinitionID: artifactv2.ControlReceiptAttemptFieldDefinitionID,
						Disposition: "plain", Value: json.Number("1")},
					{FieldDefinitionID: artifactv2.ControlReceiptOccurrenceFieldDefinitionID,
						Disposition: "plain", Value: "switch.occurrence.flip"},
					{FieldDefinitionID: artifactv2.ControlReceiptStatusFieldDefinitionID,
						Disposition: "plain", Value: "accepted"},
				},
			},
			{
				FactDefinitionID: "switch.evidence.history.1", SourceDefinitionID: "umpire.evidence.source.history",
				Ordinal: artifactv2.NaturalFromUint64(0), KindDefinitionID: "umpire.evidence.kind.history",
				CausalFactDefinitionIDs: []string{},
				Fields: []artifactv2.RawEvidenceField{{
					FieldDefinitionID: "umpire.evidence.field.event", Disposition: "plain", Value: "flip-requested",
				}},
			},
			{
				FactDefinitionID: "switch.evidence.history.2", SourceDefinitionID: "umpire.evidence.source.history",
				Ordinal: artifactv2.NaturalFromUint64(1), KindDefinitionID: "umpire.evidence.kind.history",
				CausalFactDefinitionIDs: []string{"switch.evidence.history.1"},
				Fields: []artifactv2.RawEvidenceField{{
					FieldDefinitionID: "umpire.evidence.field.event", Disposition: "plain", Value: "flip-completed",
				}},
			},
			{
				FactDefinitionID:   "switch.evidence.participant.1",
				SourceDefinitionID: "umpire.evidence.source.participant-output",
				Ordinal:            artifactv2.NaturalFromUint64(0), KindDefinitionID: "umpire.evidence.kind.participant-output",
				CausalFactDefinitionIDs: []string{},
				Fields: []artifactv2.RawEvidenceField{{
					FieldDefinitionID: "umpire.evidence.field.state", Disposition: "plain", Value: false,
				}},
			},
			{
				FactDefinitionID:   "switch.evidence.participant.2",
				SourceDefinitionID: "umpire.evidence.source.participant-output",
				Ordinal:            artifactv2.NaturalFromUint64(1), KindDefinitionID: "umpire.evidence.kind.participant-output",
				CausalFactDefinitionIDs: []string{"switch.evidence.participant.1"},
				Fields: []artifactv2.RawEvidenceField{
					{FieldDefinitionID: "umpire.evidence.field.digest", Disposition: "sha256",
						Value: "sha256:463fc89536c47a3158c1d27030df5d1b9a5665bc256a7079632485eb3b0e3f86"},
					{FieldDefinitionID: "umpire.evidence.field.rejected", Disposition: "rejected", Value: nil},
					{FieldDefinitionID: "umpire.evidence.field.secret", Disposition: "redacted", Value: nil},
					{FieldDefinitionID: "umpire.evidence.field.state", Disposition: "plain", Value: true},
				},
			},
		},
		KnownGaps: []artifactv2.KnownGap{},
		Provenance: artifactv2.Provenance{
			SourceDefinitionIDs: []string{"switch.raw-evidence.1"},
			SourceLocations: []artifactv2.SourceLocation{{
				Path: "Umpire/Artifact/Tests/Evidence.lean", Line: artifactv2.NaturalFromUint64(1),
				Column: artifactv2.NaturalFromUint64(1), Provenance: "lean-model",
			}},
		},
	}
	document, err = artifactv2.SealRawEvidence(document)
	require.NoError(t, err)
	return document
}

func rawEvidenceV2ClosureInputs(t *testing.T) (
	artifactv2.Experiment,
	artifactv2.RuntimeConfiguration,
	artifactv2.ExperimentRun,
) {
	t.Helper()
	experiment, err := artifact.DecodeExperimentV2(readExperimentV2Fixture(t,
		"tools/umpire/artifact/testdata/switch-experiment-v2.json"))
	require.NoError(t, err)
	return experiment, runtimeConfigurationV2Fixture(t), experimentRunV2Fixture(t)
}

func rawEvidenceV2ReceiptFact(t *testing.T, document *artifactv2.RawEvidence) *artifactv2.RawEvidenceFact {
	t.Helper()
	for index := range document.Facts {
		if document.Facts[index].FactDefinitionID == "switch.evidence.control-receipt.1" {
			return &document.Facts[index]
		}
	}
	require.FailNow(t, "control receipt fact is missing")
	return nil
}

func rawEvidenceV2ReceiptField(
	t *testing.T,
	document *artifactv2.RawEvidence,
	fieldDefinitionID string,
) *artifactv2.RawEvidenceField {
	t.Helper()
	fact := rawEvidenceV2ReceiptFact(t, document)
	for index := range fact.Fields {
		if fact.Fields[index].FieldDefinitionID == fieldDefinitionID {
			return &fact.Fields[index]
		}
	}
	require.FailNow(t, "control receipt field is missing", fieldDefinitionID)
	return nil
}

func rawEvidenceV2SourcesLimitDocument(t *testing.T, count int) artifactv2.RawEvidence {
	t.Helper()
	document := rawEvidenceV2Document(t)
	document.Sources = make([]artifactv2.RawEvidenceSource, count)
	document.Facts = []artifactv2.RawEvidenceFact{}
	for index := range document.Sources {
		document.Sources[index] = artifactv2.RawEvidenceSource{
			SourceDefinitionID: fmt.Sprintf("test.evidence.source.%04d", index), Status: "closed",
			FactCount: artifactv2.NaturalFromUint64(0), ByteCount: artifactv2.NaturalFromUint64(0),
		}
	}
	return document
}

func rawEvidenceV2FactsLimitDocument(t *testing.T, count int) artifactv2.RawEvidence {
	t.Helper()
	document := rawEvidenceV2Document(t)
	document.Sources = []artifactv2.RawEvidenceSource{{
		SourceDefinitionID: "test.evidence.source.limit", Status: "closed",
		FactCount: artifactv2.NaturalFromUint64(uint64(count)), ByteCount: artifactv2.NaturalFromUint64(0),
	}}
	document.Facts = make([]artifactv2.RawEvidenceFact, count)
	for index := range document.Facts {
		document.Facts[index] = artifactv2.RawEvidenceFact{
			FactDefinitionID:   fmt.Sprintf("test.evidence.fact.%04d", index),
			SourceDefinitionID: "test.evidence.source.limit", Ordinal: artifactv2.NaturalFromUint64(uint64(index)),
			KindDefinitionID: "test.evidence.kind.limit", CausalFactDefinitionIDs: []string{},
			Fields: []artifactv2.RawEvidenceField{},
		}
	}
	return document
}

func rawEvidenceV2FieldsLimitDocument(t *testing.T, count int) artifactv2.RawEvidence {
	t.Helper()
	document := rawEvidenceV2FactsLimitDocument(t, 1)
	document.Facts[0].Fields = make([]artifactv2.RawEvidenceField, count)
	for index := range document.Facts[0].Fields {
		document.Facts[0].Fields[index] = artifactv2.RawEvidenceField{
			FieldDefinitionID: fmt.Sprintf("test.evidence.field.%04d", index),
			Disposition:       "plain", Value: nil,
		}
	}
	return document
}

func rawEvidenceV2PayloadLimitDocument(t *testing.T, payloadBytes int, aggregate bool) artifactv2.RawEvidence {
	t.Helper()
	if !aggregate {
		document := rawEvidenceV2FactsLimitDocument(t, 1)
		left := payloadBytes / 2
		right := payloadBytes - left
		document.Facts[0].Fields = []artifactv2.RawEvidenceField{
			{FieldDefinitionID: "test.evidence.field.left", Disposition: "plain", Value: strings.Repeat("a", left)},
			{FieldDefinitionID: "test.evidence.field.right", Disposition: "plain", Value: strings.Repeat("b", right)},
		}
		return document
	}

	fullFacts := payloadBytes / artifact.MaximumEvidenceFactPayloadBytes
	remainder := payloadBytes % artifact.MaximumEvidenceFactPayloadBytes
	factCount := fullFacts
	if remainder > 0 {
		factCount++
	}
	document := rawEvidenceV2FactsLimitDocument(t, factCount)
	for index := range document.Facts {
		length := artifact.MaximumEvidenceFactPayloadBytes
		if index == fullFacts && remainder > 0 {
			length = remainder
		}
		document.Facts[index].Fields = []artifactv2.RawEvidenceField{{
			FieldDefinitionID: "test.evidence.field.payload", Disposition: "plain",
			Value: strings.Repeat("p", length),
		}}
	}
	return document
}

func resealedRawEvidenceV2Mutation(t *testing.T, mutate func(*artifactv2.RawEvidence)) []byte {
	t.Helper()
	document := rawEvidenceV2Document(t)
	mutate(&document)
	return sealedRawEvidenceV2Bytes(t, document)
}

func sealedRawEvidenceV2Bytes(t *testing.T, document artifactv2.RawEvidence) []byte {
	t.Helper()
	document, err := artifactv2.SealRawEvidence(document)
	require.NoError(t, err)
	encoded, err := artifactv2.CanonicalRawEvidenceBytes(document)
	require.NoError(t, err)
	return encoded
}

func requireRawEvidenceV2ErrorCode(t *testing.T, err error, expected artifact.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	code, ok := artifact.CodeOf(err)
	require.True(t, ok, err)
	require.Equal(t, expected, code)
}
