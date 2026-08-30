package artifact_test

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

type crossLanguageGolden struct {
	name        string
	format      string
	leanFixture string
	goFixture   string
	roundTrip   func([]byte) ([]byte, error)
}

func TestCrossLanguageGoldensExactCanonicalFixtures(t *testing.T) {
	goldens := crossLanguageGoldens()
	require.Equal(t, []string{
		artifactv2.ExperimentFormat,
		artifactv2.RuntimeConfigurationFormat,
		artifactv2.ExperimentRunFormat,
		artifactv2.RawEvidenceFormat,
		artifactv2.EvidenceFormat,
		artifactv2.ResultFormat,
	}, goldenFormats(goldens))

	for _, golden := range goldens {
		t.Run(golden.name, func(t *testing.T) {
			authoritative := readExperimentV2Fixture(t, golden.leanFixture)
			mirrored := readExperimentV2Fixture(t, golden.goFixture)
			// This is deliberately byte-exact; semantic JSON equality would hide wire drift.
			//nolint:testifylint
			require.Equal(t, authoritative, mirrored)
			requireCanonicalJSONLine(t, authoritative)
			require.True(t, bytes.HasPrefix(authoritative,
				[]byte("{\n  \"formatVersion\": \""+golden.format+"\",\n")))

			reencoded, err := golden.roundTrip(authoritative)
			require.NoError(t, err)
			require.Equal(t, authoritative, reencoded)
		})
	}
}

func TestCrossLanguageGoldensRejectAlternateWhitespace(t *testing.T) {
	for _, golden := range crossLanguageGoldens() {
		t.Run(golden.name, func(t *testing.T) {
			canonical := readExperimentV2Fixture(t, golden.leanFixture)
			var compact bytes.Buffer
			require.NoError(t, json.Compact(&compact, canonical))
			variants := map[string][]byte{
				"compact":           compact.Bytes(),
				"alternate space":   bytes.Replace(canonical, []byte(": "), []byte(":  "), 1),
				"four-space indent": bytes.ReplaceAll(canonical, []byte("\n  "), []byte("\n    ")),
				"missing LF":        bytes.TrimSuffix(canonical, []byte{'\n'}),
				"extra LF":          append(bytes.Clone(canonical), '\n'),
			}
			for name, encoded := range variants {
				t.Run(name, func(t *testing.T) {
					_, err := golden.roundTrip(encoded)
					requireExperimentV2ErrorCode(t, err, artifact.ErrorNoncanonical)
				})
			}
		})
	}
}

func TestCrossLanguageGoldensExactFieldSequencesAndChecksums(t *testing.T) {
	documents := loadCrossLanguageGoldenDocuments(t)
	for _, test := range []struct {
		name   string
		path   string
		fields []string
	}{
		{
			name: "ExperimentSpec", path: "tools/umpire/artifact/testdata/switch-experiment-v2.json",
			fields: []string{"formatVersion", "queryBehaviorFingerprint", "plan", "properties",
				"observationRequirementDefinitionIds", "provenance", "artifactChecksum"},
		},
		{
			name: "RuntimeConfiguration", path: "tools/umpire/artifact/testdata/runtime-configuration-v2.json",
			fields: []string{"formatVersion", "configurationDefinitionId", "behaviorFingerprint", "experiment",
				"authorityProfile", "phaseLimits", "observation", "participantBindings", "knownGaps", "provenance",
				"provenanceChecksum", "artifactChecksum"},
		},
		{
			name: "ExperimentRun", path: "tools/umpire/artifact/testdata/experiment-run-v2.json",
			fields: []string{"formatVersion", "runIdentity", "behaviorFingerprint", "experiment",
				"runtimeConfiguration", "attempt", "operationalStatus", "phaseOutcomes", "controlAttempts",
				"sourceClosures", "cleanup", "limits", "knownGaps", "provenance", "provenanceChecksum",
				"artifactChecksum"},
		},
		{
			name: "RawEvidence", path: "tools/umpire/artifact/testdata/raw-evidence-v2.json",
			fields: []string{"formatVersion", "runIdentity", "behaviorFingerprint", "experiment",
				"runtimeConfiguration", "run", "captureStatus", "sources", "facts", "knownGaps", "provenance",
				"provenanceChecksum", "artifactChecksum"},
		},
		{
			name: "Evidence", path: "tools/umpire/artifact/testdata/evidence-v2.json",
			fields: []string{"formatVersion", "runIdentity", "behaviorFingerprint", "experiment",
				"runtimeConfiguration", "run", "rawEvidence", "observationProgram", "mapping",
				"observationEvaluationStatus", "evidenceBackedModelTrace", "evidenceLinks", "dispositions",
				"diagnostics", "knownGaps", "provenance", "provenanceChecksum", "artifactChecksum"},
		},
		{
			name: "Result", path: "tools/umpire/artifact/testdata/result-v2.json",
			fields: []string{"formatVersion", "runIdentity", "behaviorFingerprint", "experiment",
				"runtimeConfiguration", "run", "rawEvidence", "evidence", "operationalStatus",
				"observationEvaluationStatus", "implementationLink", "implementationLinkStatus",
				"propertyVerdicts", "querySummary", "semanticStatus", "limits", "knownGaps", "cleanupStatus",
				"evaluationOutcomeChecksum", "provenance", "provenanceChecksum", "artifactChecksum"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.fields,
				goldenTopLevelFieldSequence(t, readExperimentV2Fixture(t, test.path)))
		})
	}

	plan := documents.experiment.Plan
	plan.ArtifactChecksum = ""
	requireIndependentGoldenChecksum(t, "umpire.drive-plan/v2",
		documents.experiment.Plan.ArtifactChecksum, plan)
	experiment := documents.experiment
	experiment.ArtifactChecksum = ""
	requireIndependentGoldenChecksum(t, "umpire.experiment-spec/v2",
		documents.experiment.ArtifactChecksum, experiment)
	requireIndependentGoldenChecksum(t, "umpire.provenance/v2",
		documents.runtimeConfiguration.Experiment.ProvenanceChecksum, documents.experiment.Provenance)

	runtimeConfiguration := documents.runtimeConfiguration
	runtimeConfiguration.ArtifactChecksum = ""
	requireIndependentGoldenChecksum(t, "umpire.runtime-configuration/v2",
		documents.runtimeConfiguration.ArtifactChecksum, runtimeConfiguration)
	experimentRun := documents.experimentRun
	experimentRun.ArtifactChecksum = ""
	requireIndependentGoldenChecksum(t, "umpire.experiment-run/v2",
		documents.experimentRun.ArtifactChecksum, experimentRun)
	rawEvidence := documents.rawEvidence
	rawEvidence.ArtifactChecksum = ""
	requireIndependentGoldenChecksum(t, "umpire.raw-evidence/v2",
		documents.rawEvidence.ArtifactChecksum, rawEvidence)
	evidence := documents.evidence
	evidence.ArtifactChecksum = ""
	requireIndependentGoldenChecksum(t, "umpire.evidence/v2", documents.evidence.ArtifactChecksum, evidence)
	result := documents.result
	result.ArtifactChecksum = ""
	requireIndependentGoldenChecksum(t, "umpire.result/v2", documents.result.ArtifactChecksum, result)

	for _, test := range []struct {
		name       string
		provenance artifactv2.Provenance
		checksum   string
	}{
		{name: "RuntimeConfiguration", provenance: documents.runtimeConfiguration.Provenance,
			checksum: documents.runtimeConfiguration.ProvenanceChecksum},
		{name: "ExperimentRun", provenance: documents.experimentRun.Provenance,
			checksum: documents.experimentRun.ProvenanceChecksum},
		{name: "RawEvidence", provenance: documents.rawEvidence.Provenance,
			checksum: documents.rawEvidence.ProvenanceChecksum},
		{name: "Evidence", provenance: documents.evidence.Provenance,
			checksum: documents.evidence.ProvenanceChecksum},
		{name: "Result", provenance: documents.result.Provenance,
			checksum: documents.result.ProvenanceChecksum},
	} {
		t.Run(test.name+" provenance", func(t *testing.T) {
			requireIndependentGoldenChecksum(t, "umpire.provenance/v2", test.checksum, test.provenance)
		})
	}

	require.Equal(t, []string{
		"sha256:d915da489735c26fcb295cbbd5e246f6758f612eb7141d448ab84716b02766d0",
		"sha256:6b81f3a1bc1b67f699b5f2dd7bd030e08c4bcf52c656274d4b25abb374bb87df",
		"sha256:41e30ef6849aec9841e5af3a478e7ca4062f5229142318572b8afd9f36ec7f07",
		"sha256:2a0e83ab40ee0bb739827351e4fca37e29095333c469b975278f882ed3581e8c",
		"sha256:0aa42f873839132836c028886c9be5ad63e5dc66dbc967182ae139159501c8ab",
		"sha256:f6fbf2847d73f198dd50a9c466e6f1834f67042db0df0a54965c2bcb6b4f7a41",
	}, []string{
		documents.experiment.QueryBehaviorFingerprint,
		documents.runtimeConfiguration.BehaviorFingerprint,
		documents.experimentRun.BehaviorFingerprint,
		documents.rawEvidence.BehaviorFingerprint,
		documents.evidence.BehaviorFingerprint,
		documents.result.BehaviorFingerprint,
	})

	view := struct {
		Plan                     artifactv2.DrivePlan                `json:"plan"`
		EvidenceBackedModelTrace artifactv2.EvidenceBackedModelTrace `json:"evidenceBackedModelTrace"`
		EvidenceLinks            []artifactv2.EvidenceLink           `json:"evidenceLinks"`
		ObservationProgram       artifactv2.DefinitionReference      `json:"observationProgram"`
		Mapping                  artifactv2.DefinitionReference      `json:"mapping"`
		ImplementationLink       artifactv2.ImplementationLinkRecord `json:"implementationLink"`
		QuerySummary             artifactv2.QuerySummary             `json:"querySummary"`
		Properties               []artifactv2.Property               `json:"properties"`
		PropertyVerdicts         []artifactv2.PropertyVerdict        `json:"propertyVerdicts"`
		Limits                   []artifactv2.StagedLimit            `json:"limits"`
	}{
		Plan:                     documents.experiment.Plan,
		EvidenceBackedModelTrace: *documents.evidence.EvidenceBackedModelTrace,
		EvidenceLinks:            documents.evidence.EvidenceLinks,
		ObservationProgram:       documents.evidence.ObservationProgram,
		Mapping:                  documents.evidence.Mapping,
		ImplementationLink:       documents.result.ImplementationLink,
		QuerySummary:             documents.result.QuerySummary,
		Properties:               documents.experiment.Properties,
		PropertyVerdicts:         documents.result.PropertyVerdicts,
		Limits:                   documents.result.Limits,
	}
	require.NotNil(t, documents.result.EvaluationOutcomeChecksum)
	requireIndependentGoldenChecksum(t, "umpire.evaluation-outcome/v2",
		*documents.result.EvaluationOutcomeChecksum, view)
}

func TestCrossLanguageGoldensNestedProjectionsAndReceiptLink(t *testing.T) {
	documents := loadCrossLanguageGoldenDocuments(t)
	require.NoError(t, artifact.ValidateRuntimeConfigurationV2Closure(
		documents.runtimeConfiguration, documents.experiment))
	require.NoError(t, artifact.ValidateExperimentRunV2Closure(
		documents.experimentRun, documents.experiment, documents.runtimeConfiguration))
	require.NoError(t, artifact.ValidateRawEvidenceV2Closure(
		documents.rawEvidence, documents.experiment, documents.runtimeConfiguration, documents.experimentRun))
	require.NoError(t, artifact.ValidateEvidenceV2Closure(documents.evidence, documents.experiment,
		documents.runtimeConfiguration, documents.experimentRun, documents.rawEvidence))
	require.NoError(t, artifact.ValidateResultV2Closure(documents.result, documents.experiment,
		documents.runtimeConfiguration, documents.experimentRun, documents.rawEvidence, documents.evidence))

	expectedEvidence := leanFixtureEvidenceV2Document(t, documents.experiment,
		documents.runtimeConfiguration, documents.experimentRun, documents.rawEvidence)
	expectedResult := resolvedResultV2Document(t, documents.experiment,
		documents.runtimeConfiguration, documents.experimentRun, documents.rawEvidence, expectedEvidence)

	require.Equal(t, expectedEvidence, documents.evidence)
	require.Equal(t, expectedResult, documents.result)
	trace := requireGoldenEvidenceTrace(t, documents.evidence)
	expectedTrace := requireGoldenEvidenceTrace(t, expectedEvidence)
	projections := []struct {
		name     string
		expected any
		actual   any
	}{
		{name: "Evidence/ArtifactBindings", expected: []artifactv2.ArtifactBinding{
			expectedEvidence.Experiment, expectedEvidence.RuntimeConfiguration, expectedEvidence.Run,
			expectedEvidence.RawEvidence}, actual: []artifactv2.ArtifactBinding{
			documents.evidence.Experiment, documents.evidence.RuntimeConfiguration, documents.evidence.Run,
			documents.evidence.RawEvidence}},
		{name: "Evidence/DefinitionReferences", expected: []artifactv2.DefinitionReference{
			expectedEvidence.ObservationProgram, expectedEvidence.Mapping, expectedTrace.ObservationPlan},
			actual: []artifactv2.DefinitionReference{
				documents.evidence.ObservationProgram, documents.evidence.Mapping, trace.ObservationPlan}},
		{name: "Evidence/EvidenceBackedModelTrace", expected: expectedTrace, actual: trace},
		{name: "Evidence/SourceLocation", expected: expectedTrace.Source, actual: trace.Source},
		{name: "Evidence/MeaningProvision", expected: expectedTrace.Vocabulary, actual: trace.Vocabulary},
		{name: "Evidence/Limit", expected: expectedTrace.AppliedLimit, actual: trace.AppliedLimit},
		{name: "Evidence/ModelTrace", expected: expectedTrace.Trace, actual: trace.Trace},
		{name: "Evidence/ModelValue", expected: expectedTrace.Trace.InitialState, actual: trace.Trace.InitialState},
		{name: "Evidence/ModelTraceStep", expected: expectedTrace.Trace.Steps, actual: trace.Trace.Steps},
		{name: "Evidence/EvidenceLink", expected: expectedEvidence.EvidenceLinks,
			actual: documents.evidence.EvidenceLinks},
		{name: "Evidence/ModelCoordinate", expected: expectedEvidence.EvidenceLinks[0].Coordinate,
			actual: documents.evidence.EvidenceLinks[0].Coordinate},
		{name: "Evidence/EvidenceOrderingFact", expected: expectedEvidence.EvidenceLinks[0].OrderingSupport,
			actual: documents.evidence.EvidenceLinks[0].OrderingSupport},
		{name: "Evidence/EvidenceClosureFact", expected: expectedEvidence.EvidenceLinks[0].ClosureSupport,
			actual: documents.evidence.EvidenceLinks[0].ClosureSupport},
		{name: "Evidence/AppliedFieldDisposition", expected: expectedEvidence.EvidenceLinks[0].AppliedDispositions,
			actual: documents.evidence.EvidenceLinks[0].AppliedDispositions},
		{name: "Evidence/FieldReference", expected: expectedEvidence.Dispositions[0].Field,
			actual: documents.evidence.Dispositions[0].Field},
		{name: "Evidence/FieldDispositionRecord", expected: expectedEvidence.Dispositions,
			actual: documents.evidence.Dispositions},
		{name: "Evidence/ObservationDiagnostic", expected: expectedEvidence.Diagnostics,
			actual: documents.evidence.Diagnostics},
		{name: "Evidence/KnownGap", expected: expectedEvidence.KnownGaps, actual: documents.evidence.KnownGaps},
		{name: "Evidence/Provenance", expected: expectedEvidence.Provenance, actual: documents.evidence.Provenance},
		{name: "Result/ArtifactBindings", expected: []artifactv2.ArtifactBinding{
			expectedResult.Experiment, expectedResult.RuntimeConfiguration, expectedResult.Run,
			expectedResult.RawEvidence, expectedResult.Evidence}, actual: []artifactv2.ArtifactBinding{
			documents.result.Experiment, documents.result.RuntimeConfiguration, documents.result.Run,
			documents.result.RawEvidence, documents.result.Evidence}},
		{name: "Result/ImplementationLinkRecord", expected: expectedResult.ImplementationLink,
			actual: documents.result.ImplementationLink},
		{name: "Result/ImplementationTargetReference", expected: []artifactv2.ImplementationTargetReference{
			expectedResult.ImplementationLink.SourceTarget, expectedResult.ImplementationLink.DestinationTarget},
			actual: []artifactv2.ImplementationTargetReference{
				documents.result.ImplementationLink.SourceTarget, documents.result.ImplementationLink.DestinationTarget}},
		{name: "Result/ImplementationLinkDiagnostic", expected: expectedResult.ImplementationLink.Diagnostic,
			actual: documents.result.ImplementationLink.Diagnostic},
		{name: "Result/PropertyVerdict", expected: expectedResult.PropertyVerdicts,
			actual: documents.result.PropertyVerdicts},
		{name: "Result/SemanticClauseVerdict", expected: expectedResult.PropertyVerdicts[0].Clauses,
			actual: documents.result.PropertyVerdicts[0].Clauses},
		{name: "Result/SemanticVerdictDiagnostic", expected: expectedResult.PropertyVerdicts[0].Diagnostic,
			actual: documents.result.PropertyVerdicts[0].Diagnostic},
		{name: "Result/QuerySummary", expected: expectedResult.QuerySummary, actual: documents.result.QuerySummary},
		{name: "Result/StagedLimit", expected: expectedResult.Limits, actual: documents.result.Limits},
		{name: "Result/KnownGap", expected: expectedResult.KnownGaps, actual: documents.result.KnownGaps},
		{name: "Result/Provenance", expected: expectedResult.Provenance, actual: documents.result.Provenance},
	}
	for _, projection := range projections {
		t.Run(projection.name, func(t *testing.T) {
			require.Equal(t, projection.expected, projection.actual)
		})
	}

	require.Len(t, documents.experimentRun.ControlAttempts, 1)
	attempt := documents.experimentRun.ControlAttempts[0]
	require.NotNil(t, attempt.ReceiptFactDefinitionID)
	receipt := requireGoldenReceiptFact(t, documents.rawEvidence, *attempt.ReceiptFactDefinitionID)
	require.Equal(t, artifactv2.ControlReceiptSourceDefinitionID, receipt.SourceDefinitionID)
	require.Equal(t, artifactv2.ControlReceiptKindDefinitionID, receipt.KindDefinitionID)
	require.Equal(t, map[string]any{
		artifactv2.ControlReceiptActionFieldDefinitionID:     attempt.ActionDefinitionID,
		artifactv2.ControlReceiptAttemptFieldDefinitionID:    json.Number(attempt.Attempt.String()),
		artifactv2.ControlReceiptOccurrenceFieldDefinitionID: attempt.OccurrenceDefinitionID,
		artifactv2.ControlReceiptStatusFieldDefinitionID:     attempt.Status,
	}, goldenReceiptFields(receipt))
}

func TestCrossLanguageGoldensRejectIdentityAndClosureMutations(t *testing.T) {
	t.Run("Definition ID", func(t *testing.T) {
		documents := loadCrossLanguageGoldenDocuments(t)
		documents.evidence.ObservationProgram.DefinitionID = documents.evidence.Mapping.DefinitionID
		documents.evidence = sealGoldenEvidence(t, documents.evidence)
		requireGoldenClosureError(t, artifact.ValidateEvidenceV2Closure(documents.evidence, documents.experiment,
			documents.runtimeConfiguration, documents.experimentRun, documents.rawEvidence))
	})

	t.Run("Behavior Fingerprint in binding receives Artifact Checksum", func(t *testing.T) {
		documents := loadCrossLanguageGoldenDocuments(t)
		documents.runtimeConfiguration.Experiment.BehaviorFingerprint = documents.experiment.ArtifactChecksum
		documents.runtimeConfiguration = sealGoldenRuntimeConfiguration(t, documents.runtimeConfiguration)
		requireGoldenClosureError(t, artifact.ValidateRuntimeConfigurationV2Closure(
			documents.runtimeConfiguration, documents.experiment))
	})

	t.Run("Artifact Checksum in binding receives Behavior Fingerprint", func(t *testing.T) {
		documents := loadCrossLanguageGoldenDocuments(t)
		documents.runtimeConfiguration.Experiment.ArtifactChecksum = documents.experiment.QueryBehaviorFingerprint
		documents.runtimeConfiguration = sealGoldenRuntimeConfiguration(t, documents.runtimeConfiguration)
		requireGoldenClosureError(t, artifact.ValidateRuntimeConfigurationV2Closure(
			documents.runtimeConfiguration, documents.experiment))
	})

	t.Run("Limit", func(t *testing.T) {
		documents := loadCrossLanguageGoldenDocuments(t)
		documents.experimentRun.Limits[0].MaxBytes = artifactv2.NaturalFromUint64(1048577)
		documents.experimentRun = sealGoldenExperimentRun(t, documents.experimentRun)
		requireGoldenClosureError(t, artifact.ValidateExperimentRunV2Closure(documents.experimentRun,
			documents.experiment, documents.runtimeConfiguration))
	})

	t.Run("stale Artifact binding", func(t *testing.T) {
		documents := loadCrossLanguageGoldenDocuments(t)
		staleEvidence := documents.evidence
		staleEvidence.BehaviorFingerprint = documents.result.BehaviorFingerprint
		staleEvidence = sealGoldenEvidence(t, staleEvidence)
		documents.result = sealGoldenResultWithOutcome(t, documents.result, staleEvidence, documents.experiment)
		documents.result.Evidence = artifactv2.EvidenceArtifactBinding(documents.evidence)
		documents.result = sealGoldenResultWithOutcome(t, documents.result, staleEvidence, documents.experiment)
		requireGoldenClosureError(t, artifact.ValidateResultV2Closure(documents.result, documents.experiment,
			documents.runtimeConfiguration, documents.experimentRun, documents.rawEvidence, staleEvidence))
	})

	t.Run("stale Property fingerprint", func(t *testing.T) {
		documents := loadCrossLanguageGoldenDocuments(t)
		stale := documents.rawEvidence.BehaviorFingerprint
		documents.result.PropertyVerdicts[0].PropertyBehaviorFingerprint = stale
		documents.result.QuerySummary.PropertyVerdicts[0].PropertyBehaviorFingerprint = stale
		documents.result = sealGoldenResultWithOutcome(t, documents.result, documents.evidence, documents.experiment)
		requireGoldenClosureError(t, artifact.ValidateResultV2Closure(documents.result, documents.experiment,
			documents.runtimeConfiguration, documents.experimentRun, documents.rawEvidence, documents.evidence))
	})

	t.Run("receipt-fact relationship", func(t *testing.T) {
		documents := loadCrossLanguageGoldenDocuments(t)
		wrongReceipt := "switch.evidence.history.1"
		documents.experimentRun.ControlAttempts[0].ReceiptFactDefinitionID = &wrongReceipt
		documents.experimentRun = sealGoldenExperimentRun(t, documents.experimentRun)
		documents.rawEvidence.Run = artifactv2.ExperimentRunArtifactBinding(documents.experimentRun)
		documents.rawEvidence = sealGoldenRawEvidence(t, documents.rawEvidence)
		requireGoldenClosureError(t, artifact.ValidateRawEvidenceV2Closure(documents.rawEvidence,
			documents.experiment, documents.runtimeConfiguration, documents.experimentRun))
	})

	t.Run("canonical content fields retain old checksum", func(t *testing.T) {
		mutations := []struct {
			name   string
			mutate func(*artifactv2.Experiment, crossLanguageGoldenDocuments)
		}{
			{name: "Definition ID", mutate: func(document *artifactv2.Experiment, _ crossLanguageGoldenDocuments) {
				document.Plan.QueryDefinitionID = "switch.query.exact-actioo"
			}},
			{name: "Behavior Fingerprint", mutate: func(
				document *artifactv2.Experiment,
				documents crossLanguageGoldenDocuments,
			) {
				document.Plan.QueryBehaviorFingerprint = documents.runtimeConfiguration.BehaviorFingerprint
			}},
			{name: "Limit", mutate: func(document *artifactv2.Experiment, _ crossLanguageGoldenDocuments) {
				document.Plan.ExpandedLimits.Search.Value = artifactv2.NaturalFromUint64(9)
			}},
			{name: "Known Gap", mutate: func(document *artifactv2.Experiment, _ crossLanguageGoldenDocuments) {
				document.Plan.KnownGaps[0].Code = "umpire.known-gap.execution-evidencf"
			}},
			{name: "Artifact Checksum", mutate: func(
				document *artifactv2.Experiment,
				documents crossLanguageGoldenDocuments,
			) {
				document.ArtifactChecksum = documents.rawEvidence.ArtifactChecksum
			}},
		}
		for _, mutation := range mutations {
			t.Run(mutation.name, func(t *testing.T) {
				documents := loadCrossLanguageGoldenDocuments(t)
				document := documents.experiment
				mutation.mutate(&document, documents)
				encoded, err := artifactv2.CanonicalExperimentBytes(document)
				require.NoError(t, err)
				_, err = artifact.DecodeExperimentV2(encoded)
				requireExperimentV2ErrorCode(t, err, artifact.ErrorArtifactChecksum)
			})
		}
	})
}

func crossLanguageGoldens() []crossLanguageGolden {
	return []crossLanguageGolden{
		{
			name:        "ExperimentSpec",
			format:      artifactv2.ExperimentFormat,
			leanFixture: "model/Umpire/Artifact/Tests/Fixtures/SwitchExperimentSpecV2.json",
			goFixture:   "tools/umpire/artifact/testdata/switch-experiment-v2.json",
			roundTrip: func(encoded []byte) ([]byte, error) {
				document, err := artifact.DecodeExperimentV2(encoded)
				if err != nil {
					return nil, err
				}
				return artifact.EncodeExperimentV2(document)
			},
		},
		{
			name:        "RuntimeConfiguration",
			format:      artifactv2.RuntimeConfigurationFormat,
			leanFixture: "model/Umpire/Artifact/Tests/Fixtures/RuntimeConfigurationV2.json",
			goFixture:   "tools/umpire/artifact/testdata/runtime-configuration-v2.json",
			roundTrip: func(encoded []byte) ([]byte, error) {
				document, err := artifact.DecodeRuntimeConfigurationV2(encoded)
				if err != nil {
					return nil, err
				}
				return artifact.EncodeRuntimeConfigurationV2(document)
			},
		},
		{
			name:        "ExperimentRun",
			format:      artifactv2.ExperimentRunFormat,
			leanFixture: "model/Umpire/Artifact/Tests/Fixtures/ExperimentRunV2.json",
			goFixture:   "tools/umpire/artifact/testdata/experiment-run-v2.json",
			roundTrip: func(encoded []byte) ([]byte, error) {
				document, err := artifact.DecodeExperimentRunV2(encoded)
				if err != nil {
					return nil, err
				}
				return artifact.EncodeExperimentRunV2(document)
			},
		},
		{
			name:        "RawEvidence",
			format:      artifactv2.RawEvidenceFormat,
			leanFixture: "model/Umpire/Artifact/Tests/Fixtures/RawEvidenceV2.json",
			goFixture:   "tools/umpire/artifact/testdata/raw-evidence-v2.json",
			roundTrip: func(encoded []byte) ([]byte, error) {
				document, err := artifact.DecodeRawEvidenceV2(encoded)
				if err != nil {
					return nil, err
				}
				return artifact.EncodeRawEvidenceV2(document)
			},
		},
		{
			name:        "Evidence",
			format:      artifactv2.EvidenceFormat,
			leanFixture: "model/Umpire/Artifact/Tests/Fixtures/EvidenceV2.json",
			goFixture:   "tools/umpire/artifact/testdata/evidence-v2.json",
			roundTrip: func(encoded []byte) ([]byte, error) {
				document, err := artifact.DecodeEvidenceV2(encoded)
				if err != nil {
					return nil, err
				}
				return artifact.EncodeEvidenceV2(document)
			},
		},
		{
			name:        "Result",
			format:      artifactv2.ResultFormat,
			leanFixture: "model/Umpire/Artifact/Tests/Fixtures/ResultV2.json",
			goFixture:   "tools/umpire/artifact/testdata/result-v2.json",
			roundTrip: func(encoded []byte) ([]byte, error) {
				document, err := artifact.DecodeResultV2(encoded)
				if err != nil {
					return nil, err
				}
				return artifact.EncodeResultV2(document)
			},
		},
	}
}

func goldenFormats(goldens []crossLanguageGolden) []string {
	formats := make([]string, len(goldens))
	for index, golden := range goldens {
		formats[index] = golden.format
	}
	return formats
}

type crossLanguageGoldenDocuments struct {
	experiment           artifactv2.Experiment
	runtimeConfiguration artifactv2.RuntimeConfiguration
	experimentRun        artifactv2.ExperimentRun
	rawEvidence          artifactv2.RawEvidence
	evidence             artifactv2.Evidence
	result               artifactv2.Result
}

func loadCrossLanguageGoldenDocuments(t *testing.T) crossLanguageGoldenDocuments {
	t.Helper()
	experiment, err := artifact.DecodeExperimentV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/SwitchExperimentSpecV2.json"))
	require.NoError(t, err)
	runtimeConfiguration, err := artifact.DecodeRuntimeConfigurationV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/RuntimeConfigurationV2.json"))
	require.NoError(t, err)
	experimentRun, err := artifact.DecodeExperimentRunV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/ExperimentRunV2.json"))
	require.NoError(t, err)
	rawEvidence, err := artifact.DecodeRawEvidenceV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/RawEvidenceV2.json"))
	require.NoError(t, err)
	evidence, err := artifact.DecodeEvidenceV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/EvidenceV2.json"))
	require.NoError(t, err)
	result, err := artifact.DecodeResultV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/ResultV2.json"))
	require.NoError(t, err)
	return crossLanguageGoldenDocuments{
		experiment:           experiment,
		runtimeConfiguration: runtimeConfiguration,
		experimentRun:        experimentRun,
		rawEvidence:          rawEvidence,
		evidence:             evidence,
		result:               result,
	}
}

func requireIndependentGoldenChecksum(t *testing.T, domain, expected string, value any) {
	t.Helper()
	preimage, err := artifact.CanonicalPretty(value)
	require.NoError(t, err)
	requireCanonicalJSONLine(t, preimage)
	require.Equal(t, expected, independentExperimentV2Checksum(domain, preimage))
}

func goldenTopLevelFieldSequence(t *testing.T, encoded []byte) []string {
	t.Helper()
	fields := make([]string, 0)
	for _, line := range strings.Split(string(encoded), "\n") {
		if !strings.HasPrefix(line, "  \"") || strings.HasPrefix(line, "    ") {
			continue
		}
		separator := strings.Index(line[2:], "\":")
		require.NotEqual(t, -1, separator)
		name, err := strconv.Unquote(line[2 : 2+separator+1])
		require.NoError(t, err)
		fields = append(fields, name)
	}
	return fields
}

func requireGoldenEvidenceTrace(t *testing.T, document artifactv2.Evidence) artifactv2.EvidenceBackedModelTrace {
	t.Helper()
	require.NotNil(t, document.EvidenceBackedModelTrace)
	return *document.EvidenceBackedModelTrace
}

func requireGoldenReceiptFact(
	t *testing.T,
	document artifactv2.RawEvidence,
	definitionID string,
) artifactv2.RawEvidenceFact {
	t.Helper()
	matches := make([]artifactv2.RawEvidenceFact, 0, 1)
	for _, fact := range document.Facts {
		if fact.FactDefinitionID == definitionID {
			matches = append(matches, fact)
		}
	}
	require.Len(t, matches, 1)
	return matches[0]
}

func goldenReceiptFields(receipt artifactv2.RawEvidenceFact) map[string]any {
	fields := make(map[string]any, len(receipt.Fields))
	for _, field := range receipt.Fields {
		fields[field.FieldDefinitionID] = field.Value
	}
	return fields
}

func sealGoldenRuntimeConfiguration(
	t *testing.T,
	document artifactv2.RuntimeConfiguration,
) artifactv2.RuntimeConfiguration {
	t.Helper()
	document, err := artifactv2.SealRuntimeConfiguration(document)
	require.NoError(t, err)
	_, err = artifact.EncodeRuntimeConfigurationV2(document)
	require.NoError(t, err)
	return document
}

func sealGoldenExperimentRun(t *testing.T, document artifactv2.ExperimentRun) artifactv2.ExperimentRun {
	t.Helper()
	document, err := artifactv2.SealExperimentRun(document)
	require.NoError(t, err)
	_, err = artifact.EncodeExperimentRunV2(document)
	require.NoError(t, err)
	return document
}

func sealGoldenRawEvidence(t *testing.T, document artifactv2.RawEvidence) artifactv2.RawEvidence {
	t.Helper()
	document, err := artifactv2.SealRawEvidence(document)
	require.NoError(t, err)
	_, err = artifact.EncodeRawEvidenceV2(document)
	require.NoError(t, err)
	return document
}

func sealGoldenEvidence(t *testing.T, document artifactv2.Evidence) artifactv2.Evidence {
	t.Helper()
	document, err := artifactv2.SealEvidence(document)
	require.NoError(t, err)
	_, err = artifact.EncodeEvidenceV2(document)
	require.NoError(t, err)
	return document
}

func sealGoldenResultWithOutcome(
	t *testing.T,
	document artifactv2.Result,
	evidence artifactv2.Evidence,
	experiment artifactv2.Experiment,
) artifactv2.Result {
	t.Helper()
	checksum, err := artifactv2.ExpectedEvaluationOutcomeChecksum(document, evidence, experiment)
	require.NoError(t, err)
	document.EvaluationOutcomeChecksum = &checksum
	document, err = artifactv2.SealResult(document)
	require.NoError(t, err)
	_, err = artifact.EncodeResultV2(document)
	require.NoError(t, err)
	return document
}

func requireGoldenClosureError(t *testing.T, err error) {
	t.Helper()
	requireExperimentV2ErrorCode(t, err, artifact.ErrorClosure)
}
