package artifact_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestResultV2AcceptedEvidenceAndResolvedResultRoundTrip(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := leanFixtureEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)

	evidenceBytes, err := artifact.EncodeEvidenceV2(evidence)
	require.NoError(t, err)
	decodedEvidence, err := artifact.DecodeEvidenceV2(evidenceBytes)
	require.NoError(t, err)
	require.Equal(t, evidence, decodedEvidence)
	require.NoError(t, artifact.ValidateEvidenceV2Closure(
		decodedEvidence, experiment, runtimeConfiguration, run, rawEvidence,
	))
	require.True(t, bytes.HasSuffix(evidenceBytes, []byte{'\n'}))
	require.False(t, bytes.HasSuffix(evidenceBytes, []byte("\n\n")))

	result := resolvedResultV2Document(t, experiment, runtimeConfiguration, run, rawEvidence, evidence)
	resultBytes, err := artifact.EncodeResultV2(result)
	require.NoError(t, err)
	decodedResult, err := artifact.DecodeResultV2(resultBytes)
	require.NoError(t, err)
	require.Equal(t, result, decodedResult)
	require.NoError(t, artifact.ValidateResultV2Closure(
		decodedResult, experiment, runtimeConfiguration, run, rawEvidence, evidence,
	))
	require.True(t, bytes.HasSuffix(resultBytes, []byte{'\n'}))
	require.False(t, bytes.HasSuffix(resultBytes, []byte("\n\n")))
}

func TestResultV2AdmitsKindMajorMultistepCoordinateOrder(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	secondStep := document.EvidenceBackedModelTrace.Trace.Steps[0]
	secondStep.Position = artifactv2.NaturalFromUint64(2)
	document.EvidenceBackedModelTrace.Trace.Steps = append(
		document.EvidenceBackedModelTrace.Trace.Steps, secondStep,
	)
	stepTwo := artifactv2.NaturalFromUint64(2)
	selectedTwo := document.EvidenceLinks[1]
	selectedTwo.Coordinate.Step = &stepTwo
	outcomeTwo := document.EvidenceLinks[2]
	outcomeTwo.Coordinate.Step = &stepTwo
	resultingTwo := document.EvidenceLinks[3]
	resultingTwo.Coordinate.Step = &stepTwo
	observationTwo := document.EvidenceLinks[4]
	observationTwo.Coordinate.Step = &stepTwo
	document.EvidenceLinks = []artifactv2.EvidenceLink{
		document.EvidenceLinks[0],
		document.EvidenceLinks[1], selectedTwo,
		document.EvidenceLinks[2], outcomeTwo,
		document.EvidenceLinks[3], resultingTwo,
		document.EvidenceLinks[4], observationTwo,
	}
	document = sealedEvidenceV2Document(t, document)

	encoded, err := artifact.EncodeEvidenceV2(document)
	require.NoError(t, err)
	decoded, err := artifact.DecodeEvidenceV2(encoded)
	require.NoError(t, err)
	require.Equal(t, document, decoded)
}

func TestResultV2CanonicalLeanFixtureParity(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := leanFixtureEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	result := resolvedResultV2Document(t, experiment, runtimeConfiguration, run, rawEvidence, evidence)

	for _, test := range []struct {
		name     string
		fixture  string
		expected []byte
		decode   func([]byte) error
	}{
		{
			name:    "Evidence",
			fixture: "model/Umpire/Artifact/Tests/Fixtures/EvidenceV2.json",
			expected: func() []byte {
				encoded, err := artifactv2.CanonicalEvidenceBytes(evidence)
				require.NoError(t, err)
				return encoded
			}(),
			decode: func(encoded []byte) error {
				_, err := artifact.DecodeEvidenceV2(encoded)
				return err
			},
		},
		{
			name:    "Result",
			fixture: "model/Umpire/Artifact/Tests/Fixtures/ResultV2.json",
			expected: func() []byte {
				encoded, err := artifactv2.CanonicalResultBytes(result)
				require.NoError(t, err)
				return encoded
			}(),
			decode: func(encoded []byte) error {
				_, err := artifact.DecodeResultV2(encoded)
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded := readExperimentV2Fixture(t, test.fixture)
			require.Equal(t, test.expected, encoded)
			require.NoError(t, test.decode(encoded))
		})
	}

}
func TestResultV2EvidenceClosedStatusAndNullabilityMatrix(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)

	for _, test := range []struct {
		name           string
		status         string
		diagnosticKind string
	}{
		{name: "unknown", status: "unknown", diagnosticKind: "empty-evidence"},
		{name: "conflict", status: "conflict", diagnosticKind: "duplicate-evidence-identity"},
		{name: "unsupported", status: "unsupported", diagnosticKind: "profile-mismatch"},
	} {
		t.Run("admits "+test.name, func(t *testing.T) {
			document := nonAcceptedEvidenceV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, test.status, test.diagnosticKind,
			)
			encoded, err := artifact.EncodeEvidenceV2(document)
			require.NoError(t, err)
			decoded, err := artifact.DecodeEvidenceV2(encoded)
			require.NoError(t, err)
			require.Equal(t, document, decoded)
			require.NoError(t, artifact.ValidateEvidenceV2Closure(
				decoded, experiment, runtimeConfiguration, run, rawEvidence,
			))
		})
	}

	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.Evidence)
	}{
		{
			name: "accepted without trace",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceBackedModelTrace = nil
			},
		},
		{
			name: "accepted without every link",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceLinks = document.EvidenceLinks[:len(document.EvidenceLinks)-1]
			},
		},
		{
			name: "accepted with diagnostic",
			mutate: func(document *artifactv2.Evidence) {
				document.Diagnostics = []artifactv2.ObservationDiagnostic{
					observationDiagnosticV2(document, "empty-evidence"),
				}
			},
		},
		{
			name: "non-accepted with partial trace",
			mutate: func(document *artifactv2.Evidence) {
				document.ObservationEvaluationStatus = "unknown"
				document.EvidenceLinks = []artifactv2.EvidenceLink{}
				document.Diagnostics = []artifactv2.ObservationDiagnostic{
					observationDiagnosticV2(document, "empty-evidence"),
				}
			},
		},
		{
			name: "non-accepted with links",
			mutate: func(document *artifactv2.Evidence) {
				document.ObservationEvaluationStatus = "unknown"
				document.EvidenceBackedModelTrace = nil
				document.Diagnostics = []artifactv2.ObservationDiagnostic{
					observationDiagnosticV2(document, "empty-evidence"),
				}
			},
		},
		{
			name: "non-accepted without diagnostic",
			mutate: func(document *artifactv2.Evidence) {
				document.ObservationEvaluationStatus = "unknown"
				document.EvidenceBackedModelTrace = nil
				document.EvidenceLinks = []artifactv2.EvidenceLink{}
			},
		},
		{
			name: "diagnostic class disagrees with status",
			mutate: func(document *artifactv2.Evidence) {
				document.ObservationEvaluationStatus = "conflict"
				document.EvidenceBackedModelTrace = nil
				document.EvidenceLinks = []artifactv2.EvidenceLink{}
				document.Diagnostics = []artifactv2.ObservationDiagnostic{
					observationDiagnosticV2(document, "empty-evidence"),
				}
			},
		},
		{
			name: "diagnostic nullable fields disagree with kind",
			mutate: func(document *artifactv2.Evidence) {
				document.ObservationEvaluationStatus = "unknown"
				document.EvidenceBackedModelTrace = nil
				document.EvidenceLinks = []artifactv2.EvidenceLink{}
				diagnostic := observationDiagnosticV2(document, "empty-evidence")
				diagnostic.AppliedLimit = &artifactv2.Limit{
					Value: artifactv2.NaturalFromUint64(1), Unit: "evidence-records",
				}
				document.Diagnostics = []artifactv2.ObservationDiagnostic{diagnostic}
			},
		},
		{
			name: "null diagnostics",
			mutate: func(document *artifactv2.Evidence) {
				document.Diagnostics = nil
			},
		},
		{
			name: "unknown observation status",
			mutate: func(document *artifactv2.Evidence) {
				document.ObservationEvaluationStatus = "partial"
			},
		},
	} {
		t.Run("rejects "+test.name, func(t *testing.T) {
			document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
			test.mutate(&document)
			document = sealedEvidenceV2Document(t, document)
			_, err := artifact.EncodeEvidenceV2(document)
			requireResultV2ErrorCode(t, err, artifact.ErrorMalformedValue)
		})
	}

}

func TestResultV2ExhaustiveClosedDiagnosticClassifications(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)

	for _, test := range []struct {
		kind   string
		status string
	}{
		{kind: "empty-evidence", status: "unknown"},
		{kind: "evidence-bound-exhausted", status: "unknown"},
		{kind: "missing-initial-state", status: "unknown"},
		{kind: "missing-closure", status: "unknown"},
		{kind: "sequence-gap", status: "unknown"},
		{kind: "missing-causal-parent", status: "unknown"},
		{kind: "normalization-failure", status: "unknown"},
		{kind: "unresolved-binding", status: "unknown"},
		{kind: "incomparable-ordering", status: "unknown"},
		{kind: "profile-mismatch", status: "unsupported"},
		{kind: "profile-version-mismatch", status: "unsupported"},
		{kind: "kind-mismatch", status: "unsupported"},
		{kind: "field-mismatch", status: "unsupported"},
		{kind: "duplicate-evidence-identity", status: "conflict"},
		{kind: "contradictory-fact", status: "conflict"},
		{kind: "contradictory-binding", status: "conflict"},
		{kind: "contradictory-order", status: "conflict"},
		{kind: "misdirected-fault-receipt", status: "conflict"},
		{kind: "compatible-alternatives", status: "unknown"},
		{kind: "zero-usable-interpretations", status: "unknown"},
		{kind: "absent-model-coordinate", status: "unknown"},
		{kind: "duplicate-model-coordinate", status: "conflict"},
		{kind: "extra-model-coordinate", status: "conflict"},
		{kind: "inconsistent-evidence-link", status: "conflict"},
		{kind: "unconsumed-reference", status: "unknown"},
		{kind: "missing-closure-support", status: "unknown"},
		{kind: "missing-order-support", status: "unknown"},
		{kind: "raw-value-leakage", status: "unsupported"},
		{kind: "redacted-value-leakage", status: "unsupported"},
		{kind: "rejected-value-leakage", status: "unsupported"},
		{kind: "rejected-field-present", status: "unsupported"},
		{kind: "digest-policy-mismatch", status: "unsupported"},
		{kind: "digest-collision", status: "conflict"},
		{kind: "disallowed-raw-material", status: "unsupported"},
	} {
		t.Run("Observation "+test.kind, func(t *testing.T) {
			document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
			document.ObservationEvaluationStatus = test.status
			document.EvidenceBackedModelTrace = nil
			document.EvidenceLinks = []artifactv2.EvidenceLink{}
			diagnostic := observationDiagnosticV2(&document, test.kind)
			switch test.kind {
			case "evidence-bound-exhausted":
				limit := artifactv2.Limit{
					Value: artifactv2.NaturalFromUint64(1), Unit: "evidence-records",
				}
				diagnostic.AppliedLimit = &limit
				diagnostic.ObservedCount = naturalV2Pointer(2)
			case "compatible-alternatives":
				discriminator := "switch.observation.discriminator.power"
				diagnostic.Alternatives = []string{
					"switch.observation.alternative.off", "switch.observation.alternative.on",
				}
				diagnostic.MissingDiscriminatorDefinitionID = &discriminator
			}
			document.Diagnostics = []artifactv2.ObservationDiagnostic{diagnostic}
			document = sealedEvidenceV2Document(t, document)
			_, err := artifact.EncodeEvidenceV2(document)
			require.NoError(t, err)
		})
	}

	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	for _, test := range []struct {
		kind   string
		status string
	}{
		{kind: "stale-source-target", status: "invalid"},
		{kind: "stale-destination-target", status: "invalid"},
		{kind: "behavior-fingerprint-drift", status: "invalid"},
		{kind: "source-setup-mismatch", status: "invalid"},
		{kind: "non-authoritative-source-initial", status: "invalid"},
		{kind: "non-authoritative-source-step", status: "invalid"},
		{kind: "invalid-coordinate", status: "invalid"},
		{kind: "absent-coordinate", status: "unknown"},
		{kind: "limit-reached", status: "unknown"},
		{kind: "duplicate-coordinate", status: "conflict"},
		{kind: "contradictory-coordinate", status: "conflict"},
		{kind: "multiple-mappings", status: "conflict"},
		{kind: "evidence-link-mismatch", status: "conflict"},
		{kind: "known-gap", status: "unsupported"},
		{kind: "unsupported-vocabulary", status: "unsupported"},
	} {
		t.Run("Implementation Link "+test.kind, func(t *testing.T) {
			document := unresolvedResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
			)
			document.ImplementationLinkStatus = test.status
			diagnostic := implementationLinkDiagnosticV2(test.kind)
			switch test.kind {
			case "limit-reached":
				limit := artifactv2.Limit{
					Value: artifactv2.NaturalFromUint64(1), Unit: "semantic-transitions",
				}
				diagnostic.AppliedLimit = &limit
				diagnostic.ObservedCount = naturalV2Pointer(2)
			case "unsupported-vocabulary":
				kind := "law"
				diagnostic.UnsupportedVocabularyKind = &kind
			}
			document.ImplementationLink.Diagnostic = diagnostic
			sealImplementationLinkDiagnosticV2(t, &document.ImplementationLink)
			document = sealedResultV2Document(t, document)
			_, err := artifact.EncodeResultV2(document)
			require.NoError(t, err)
		})
	}

	for _, test := range []struct {
		kind                  string
		status                string
		observationDiagnostic string
	}{
		{kind: "observation-evaluation-failure", status: "unknown", observationDiagnostic: "empty-evidence"},
		{kind: "observation-evaluation-failure", status: "conflict", observationDiagnostic: "duplicate-evidence-identity"},
		{kind: "observation-evaluation-failure", status: "unsupported", observationDiagnostic: "profile-mismatch"},
		{kind: "query-property-mismatch", status: "unsupported"},
		{kind: "invalid-evidence-bound", status: "unknown"},
		{kind: "missing-capability", status: "unsupported"},
		{kind: "missing-vocabulary", status: "unsupported"},
		{kind: "ambiguous-vocabulary", status: "unsupported"},
		{kind: "digest-mismatch", status: "unsupported"},
		{kind: "missing-logical-time", status: "unknown"},
	} {
		t.Run("Property "+test.status+" "+test.kind+" "+test.observationDiagnostic, func(t *testing.T) {
			document := propertyNonSuccessResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence, test.status, test.kind,
			)
			if test.observationDiagnostic != "" {
				diagnostic := observationDiagnosticV2(&evidence, test.observationDiagnostic)
				document.PropertyVerdicts[0].Diagnostic.ObservationDiagnostic = &diagnostic
				document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
				document = sealedResultV2Document(t, document)
			}
			_, err := artifact.EncodeResultV2(document)
			require.NoError(t, err)
		})
	}
}

func TestResultV2SpecializedDiagnosticStringBounds(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	identityAtLimit := "a." + strings.Repeat("x", artifact.MaximumIdentityBytes-2)
	identityOverLimit := identityAtLimit + "x"
	diagnosticAtLimit := strings.Repeat("x", artifact.MaximumDiagnosticBytes)
	diagnosticOverLimit := diagnosticAtLimit + "x"

	t.Run("Evidence diagnostic alternatives", func(t *testing.T) {
		encode := func(alternative string) error {
			document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
			document.ObservationEvaluationStatus = "unknown"
			document.EvidenceBackedModelTrace = nil
			document.EvidenceLinks = []artifactv2.EvidenceLink{}
			diagnostic := observationDiagnosticV2(&document, "compatible-alternatives")
			diagnostic.Alternatives = []string{alternative}
			discriminator := "switch.observation.discriminator.power"
			diagnostic.MissingDiscriminatorDefinitionID = &discriminator
			document.Diagnostics = []artifactv2.ObservationDiagnostic{diagnostic}
			document = sealedEvidenceV2Document(t, document)
			_, err := artifact.EncodeEvidenceV2(document)
			return err
		}

		require.NoError(t, encode(identityAtLimit))
		requireResultV2ErrorCode(t, encode(identityOverLimit), artifact.ErrorStringLimit)
	})

	t.Run("Property observation diagnostic alternatives", func(t *testing.T) {
		encode := func(alternative string) error {
			document := propertyNonSuccessResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
				"unknown", "observation-evaluation-failure",
			)
			diagnostic := observationDiagnosticV2(&evidence, "compatible-alternatives")
			diagnostic.Alternatives = []string{alternative}
			discriminator := "switch.observation.discriminator.power"
			diagnostic.MissingDiscriminatorDefinitionID = &discriminator
			document.PropertyVerdicts[0].Diagnostic.ObservationDiagnostic = &diagnostic
			document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
			document = sealedResultV2Document(t, document)
			_, err := artifact.EncodeResultV2(document)
			return err
		}

		require.NoError(t, encode(identityAtLimit))
		requireResultV2ErrorCode(t, encode(identityOverLimit), artifact.ErrorStringLimit)
	})

	t.Run("Implementation Link Known Gap reason", func(t *testing.T) {
		encode := func(reason string) error {
			document := unresolvedResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
			)
			document.ImplementationLinkStatus = "unsupported"
			diagnostic := implementationLinkDiagnosticV2("known-gap")
			diagnostic.KnownGapReason = &reason
			document.ImplementationLink.Diagnostic = diagnostic
			sealImplementationLinkDiagnosticV2(t, &document.ImplementationLink)
			document = sealedResultV2Document(t, document)
			_, err := artifact.EncodeResultV2(document)
			return err
		}

		require.NoError(t, encode(diagnosticAtLimit))
		requireResultV2ErrorCode(t, encode(diagnosticOverLimit), artifact.ErrorStringLimit)
	})

	t.Run("Implementation Link diagnostic identity", func(t *testing.T) {
		encode := func(identity string) error {
			document := unresolvedResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
			)
			document.ImplementationLinkStatus = "invalid"
			diagnostic := implementationLinkDiagnosticV2("stale-source-target")
			diagnostic.Identity = identity
			document.ImplementationLink.Diagnostic = diagnostic
			document = sealedResultV2Document(t, document)
			_, err := artifact.EncodeResultV2(document)
			return err
		}

		requireResultV2ErrorCode(t, encode(identityAtLimit), artifact.ErrorMalformedValue)
		requireResultV2ErrorCode(t, encode(identityOverLimit), artifact.ErrorStringLimit)
	})
}

func TestResultV2EvidenceRejectsIncompleteLinksStaleReferencesAndRawLeakage(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)

	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.Evidence)
	}{
		{
			name: "partial trace",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceBackedModelTrace.Trace.Steps = []artifactv2.ModelTraceStep{}
			},
		},
		{
			name: "duplicate coordinate link",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceLinks[1] = document.EvidenceLinks[0]
			},
		},
		{
			name: "noncanonical coordinate link order",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceLinks[0], document.EvidenceLinks[1] =
					document.EvidenceLinks[1], document.EvidenceLinks[0]
			},
		},
		{
			name: "unconsumed Evidence identity",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceBackedModelTrace.EvidenceDefinitionIDs = append(
					document.EvidenceBackedModelTrace.EvidenceDefinitionIDs,
					"switch.evidence.history.3",
				)
			},
		},
		{
			name: "trace value absent from vocabulary",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceBackedModelTrace.Vocabulary =
					document.EvidenceBackedModelTrace.Vocabulary[1:]
			},
		},
		{
			name: "stale mapping version",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceLinks[0].MappingVersion = artifactv2.NaturalFromUint64(2)
			},
		},
		{
			name: "Evidence identities disagree with ordering support",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceLinks[0].EvidenceDefinitionIDs,
					document.EvidenceLinks[1].EvidenceDefinitionIDs =
					document.EvidenceLinks[1].EvidenceDefinitionIDs,
					document.EvidenceLinks[0].EvidenceDefinitionIDs
			},
		},
		{
			name: "inconsistent closure support",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceLinks[1].ClosureSupport[0].LastOrdinal = artifactv2.NaturalFromUint64(2)
			},
		},
		{
			name: "empty closure support",
			mutate: func(document *artifactv2.Evidence) {
				for index := range document.EvidenceLinks {
					document.EvidenceLinks[index].ClosureSupport = []artifactv2.EvidenceClosureFact{}
				}
			},
		},
		{
			name: "closure support misses an ordering kind",
			mutate: func(document *artifactv2.Evidence) {
				document.EvidenceLinks[0].OrderingSupport[0].KindDefinitionID =
					"umpire.evidence.kind.other"
			},
		},
		{
			name: "uniformly stale closure ordinal",
			mutate: func(document *artifactv2.Evidence) {
				for index := range document.EvidenceLinks {
					document.EvidenceLinks[index].ClosureSupport[0].LastOrdinal =
						artifactv2.NaturalFromUint64(2)
				}
			},
		},
	} {
		t.Run("transport rejects "+test.name, func(t *testing.T) {
			document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
			test.mutate(&document)
			document = sealedEvidenceV2Document(t, document)
			_, err := artifact.EncodeEvidenceV2(document)
			requireResultV2ErrorCode(t, err, artifact.ErrorMalformedValue)
		})
	}

	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.Evidence)
	}{
		{
			name: "stale observation program",
			mutate: func(document *artifactv2.Evidence) {
				document.ObservationProgram.DefinitionID = "switch.observation.program.stale"
				document.EvidenceBackedModelTrace.ObservationPlan = document.ObservationProgram
			},
		},
		{
			name: "stale RawEvidence binding",
			mutate: func(document *artifactv2.Evidence) {
				document.RawEvidence = document.Run
				document.RawEvidence.FormatVersion = artifactv2.RawEvidenceFormat
			},
		},
		{
			name: "prohibited redacted raw field value",
			mutate: func(document *artifactv2.Evidence) {
				field := artifactv2.FieldReference{
					KindDefinitionID:  "umpire.evidence.kind.participant-output",
					FieldDefinitionID: "umpire.evidence.field.secret",
				}
				document.Dispositions = append(document.Dispositions, artifactv2.FieldDispositionRecord{
					Field: field, Disposition: "retain",
				})
				slices.SortFunc(document.Dispositions, func(left, right artifactv2.FieldDispositionRecord) int {
					if left.Field.KindDefinitionID != right.Field.KindDefinitionID {
						return strings.Compare(left.Field.KindDefinitionID, right.Field.KindDefinitionID)
					}
					return strings.Compare(left.Field.FieldDefinitionID, right.Field.FieldDefinitionID)
				})
				value := "secret"
				document.EvidenceLinks[0].AppliedDispositions = append(
					document.EvidenceLinks[0].AppliedDispositions,
					artifactv2.AppliedFieldDisposition{Field: field, Kind: "retained", NormalizedValue: &value},
				)
				slices.SortFunc(document.EvidenceLinks[0].AppliedDispositions,
					func(left, right artifactv2.AppliedFieldDisposition) int {
						if left.Field.KindDefinitionID != right.Field.KindDefinitionID {
							return strings.Compare(left.Field.KindDefinitionID, right.Field.KindDefinitionID)
						}
						return strings.Compare(left.Field.FieldDefinitionID, right.Field.FieldDefinitionID)
					})
			},
		},
	} {
		t.Run("closure rejects "+test.name, func(t *testing.T) {
			document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
			test.mutate(&document)
			document = sealedEvidenceV2Document(t, document)
			err := artifact.ValidateEvidenceV2Closure(
				document, experiment, runtimeConfiguration, run, rawEvidence,
			)
			requireResultV2ErrorCode(t, err, artifact.ErrorClosure)
		})
	}
}

func TestResultV2EvidenceAdmitsOpaqueDigestTokenWithoutRawValue(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	field := artifactv2.FieldReference{
		KindDefinitionID:  "umpire.evidence.kind.participant-output",
		FieldDefinitionID: "umpire.evidence.field.digest",
	}
	policy := "switch.observation.digest-policy.v1"
	document.Dispositions = append(document.Dispositions, artifactv2.FieldDispositionRecord{
		Field: field, Disposition: "hash", DigestPolicyDefinitionID: &policy,
	})
	slices.SortFunc(document.Dispositions, func(left, right artifactv2.FieldDispositionRecord) int {
		if left.Field.KindDefinitionID != right.Field.KindDefinitionID {
			return strings.Compare(left.Field.KindDefinitionID, right.Field.KindDefinitionID)
		}
		return strings.Compare(left.Field.FieldDefinitionID, right.Field.FieldDefinitionID)
	})
	token := "opaque-digest-token/v1"
	document.EvidenceLinks[0].AppliedDispositions = append(
		document.EvidenceLinks[0].AppliedDispositions,
		artifactv2.AppliedFieldDisposition{
			Field: field, Kind: "digest-token", DigestPolicyDefinitionID: &policy, DigestToken: &token,
		},
	)
	slices.SortFunc(document.EvidenceLinks[0].AppliedDispositions,
		func(left, right artifactv2.AppliedFieldDisposition) int {
			if left.Field.KindDefinitionID != right.Field.KindDefinitionID {
				return strings.Compare(left.Field.KindDefinitionID, right.Field.KindDefinitionID)
			}
			return strings.Compare(left.Field.FieldDefinitionID, right.Field.FieldDefinitionID)
		})
	document = sealedEvidenceV2Document(t, document)

	encoded, err := artifact.EncodeEvidenceV2(document)
	require.NoError(t, err)
	decoded, err := artifact.DecodeEvidenceV2(encoded)
	require.NoError(t, err)
	require.NoError(t, artifact.ValidateEvidenceV2Closure(
		decoded, experiment, runtimeConfiguration, run, rawEvidence,
	))

	stalePolicy := "switch.observation.digest-policy.stale"
	document.EvidenceLinks[0].AppliedDispositions[1].DigestPolicyDefinitionID = &stalePolicy
	document = sealedEvidenceV2Document(t, document)
	err = artifact.ValidateEvidenceV2Closure(
		document, experiment, runtimeConfiguration, run, rawEvidence,
	)
	requireResultV2ErrorCode(t, err, artifact.ErrorClosure)
}

func TestResultV2AdmitsOpaqueObservationTraceIdentity(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	traceID := "sha256:6c1d77f7e7e15b89e6ea6d132b28ab1f4eb59db41ef4cc7c70130294d8162c93"
	evidence.EvidenceBackedModelTrace.TraceID = traceID
	evidence.EvidenceBackedModelTrace.Trace.TraceID = traceID
	evidence = sealedEvidenceV2Document(t, evidence)

	evidenceBytes, err := artifact.EncodeEvidenceV2(evidence)
	require.NoError(t, err)
	decodedEvidence, err := artifact.DecodeEvidenceV2(evidenceBytes)
	require.NoError(t, err)
	require.NoError(t, artifact.ValidateEvidenceV2Closure(
		decodedEvidence, experiment, runtimeConfiguration, run, rawEvidence,
	))

	result := resolvedResultV2Document(t, experiment, runtimeConfiguration, run, rawEvidence, evidence)
	resultBytes, err := artifact.EncodeResultV2(result)
	require.NoError(t, err)
	decodedResult, err := artifact.DecodeResultV2(resultBytes)
	require.NoError(t, err)
	require.NoError(t, artifact.ValidateResultV2Closure(
		decodedResult, experiment, runtimeConfiguration, run, rawEvidence, decodedEvidence,
	))
}

func TestResultV2EvidenceRequiresDispositionForEveryRejectedRawField(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)

	for _, status := range []string{"accepted", "unknown"} {
		t.Run(status, func(t *testing.T) {
			document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
			if status == "unknown" {
				document = nonAcceptedEvidenceV2Document(
					t, experiment, runtimeConfiguration, run, rawEvidence, status, "empty-evidence",
				)
			}
			document.Dispositions = document.Dispositions[:1]
			document = sealedEvidenceV2Document(t, document)

			err := artifact.ValidateEvidenceV2Closure(
				document, experiment, runtimeConfiguration, run, rawEvidence,
			)
			requireResultV2ErrorCode(t, err, artifact.ErrorClosure)
		})
	}
}

func TestResultV2ImplementationPropertySemanticAndChecksumMatrices(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)

	for _, test := range []struct {
		name   string
		status string
		kind   string
	}{
		{name: "invalid", status: "invalid", kind: "stale-source-target"},
		{name: "unknown", status: "unknown", kind: "absent-coordinate"},
		{name: "conflict", status: "conflict", kind: "duplicate-coordinate"},
		{name: "unsupported", status: "unsupported", kind: "known-gap"},
	} {
		t.Run("admits Implementation Link "+test.name, func(t *testing.T) {
			document := unresolvedResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
			)
			document.ImplementationLinkStatus = test.status
			document.ImplementationLink.Diagnostic = implementationLinkDiagnosticV2(test.kind)
			sealImplementationLinkDiagnosticV2(t, &document.ImplementationLink)
			document = sealedResultV2Document(t, document)
			encoded, err := artifact.EncodeResultV2(document)
			require.NoError(t, err)
			_, err = artifact.DecodeResultV2(encoded)
			require.NoError(t, err)
		})
	}

	for _, test := range []struct {
		name   string
		status string
		kind   string
	}{
		{name: "unknown", status: "unknown", kind: "invalid-evidence-bound"},
		{name: "unsupported", status: "unsupported", kind: "missing-capability"},
	} {
		t.Run("admits Property "+test.name, func(t *testing.T) {
			document := propertyNonSuccessResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence, test.status, test.kind,
			)
			encoded, err := artifact.EncodeResultV2(document)
			require.NoError(t, err)
			decoded, err := artifact.DecodeResultV2(encoded)
			require.NoError(t, err)
			require.Equal(t, "incomplete", decoded.SemanticStatus)
			require.Nil(t, decoded.EvaluationOutcomeChecksum)
			require.NoError(t, artifact.ValidateResultV2Closure(
				decoded, experiment, runtimeConfiguration, run, rawEvidence, evidence,
			))
		})
	}

	for _, test := range []struct {
		name   string
		status string
		kind   string
		mutate func(*artifactv2.PropertyVerdict)
	}{
		{
			name: "required trace and Evidence Limit both missing", status: "unknown",
			kind: "invalid-evidence-bound",
			mutate: func(verdict *artifactv2.PropertyVerdict) {
				verdict.TraceID = nil
				verdict.EvidenceLimit = nil
			},
		},
		{
			name: "observation failure trace and Evidence Limit both missing", status: "unknown",
			kind: "observation-evaluation-failure",
			mutate: func(verdict *artifactv2.PropertyVerdict) {
				diagnostic := observationDiagnosticV2(&evidence, "empty-evidence")
				verdict.Diagnostic.ObservationDiagnostic = &diagnostic
				verdict.TraceID = nil
				verdict.EvidenceLimit = nil
			},
		},
		{
			name: "trace and Evidence Limit half present", status: "unknown",
			kind: "invalid-evidence-bound",
			mutate: func(verdict *artifactv2.PropertyVerdict) {
				verdict.TraceID = nil
			},
		},
		{
			name: "query mismatch carries invented trace context", status: "unsupported",
			kind: "query-property-mismatch",
			mutate: func(verdict *artifactv2.PropertyVerdict) {
				traceID := evidence.EvidenceBackedModelTrace.TraceID
				limit := evidence.EvidenceBackedModelTrace.AppliedLimit
				verdict.TraceID = &traceID
				verdict.EvidenceLimit = &limit
			},
		},
	} {
		t.Run("rejects Property nullability "+test.name, func(t *testing.T) {
			document := propertyNonSuccessResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence, test.status, test.kind,
			)
			test.mutate(&document.PropertyVerdicts[0])
			document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
			document.QuerySummary.TraceIDs = document.QuerySummary.TraceIDs[:0]
			if document.PropertyVerdicts[0].TraceID != nil {
				document.QuerySummary.TraceIDs = append(
					document.QuerySummary.TraceIDs, *document.PropertyVerdicts[0].TraceID,
				)
			}
			document = sealedResultV2Document(t, document)
			_, err := artifact.EncodeResultV2(document)
			requireResultV2ErrorCode(t, err, artifact.ErrorMalformedValue)
		})
	}

	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.Result)
	}{
		{
			name: "applied link with diagnostic",
			mutate: func(document *artifactv2.Result) {
				document.ImplementationLink.Diagnostic = implementationLinkDiagnosticV2("stale-source-target")
			},
		},
		{
			name: "failed link without diagnostic",
			mutate: func(document *artifactv2.Result) {
				document.ImplementationLinkStatus = "invalid"
				document.PropertyVerdicts = []artifactv2.PropertyVerdict{}
				document.QuerySummary = incompleteQuerySummaryV2(experiment)
				document.SemanticStatus = "incomplete"
				document.EvaluationOutcomeChecksum = nil
			},
		},
		{
			name: "link diagnostic class mismatch",
			mutate: func(document *artifactv2.Result) {
				document.ImplementationLinkStatus = "conflict"
				document.ImplementationLink.Diagnostic = implementationLinkDiagnosticV2("absent-coordinate")
				document.PropertyVerdicts = []artifactv2.PropertyVerdict{}
				document.QuerySummary = incompleteQuerySummaryV2(experiment)
				document.SemanticStatus = "incomplete"
				document.EvaluationOutcomeChecksum = nil
			},
		},
		{
			name: "not accepted observation evaluates link",
			mutate: func(document *artifactv2.Result) {
				document.ObservationEvaluationStatus = "unknown"
			},
		},
		{
			name: "non-applied link retains verdicts",
			mutate: func(document *artifactv2.Result) {
				document.ImplementationLinkStatus = "invalid"
				document.ImplementationLink.Diagnostic = implementationLinkDiagnosticV2("stale-source-target")
			},
		},
		{
			name: "applied link omits verdicts",
			mutate: func(document *artifactv2.Result) {
				document.PropertyVerdicts = []artifactv2.PropertyVerdict{}
				document.QuerySummary = incompleteQuerySummaryV2(experiment)
				document.SemanticStatus = "incomplete"
				document.EvaluationOutcomeChecksum = nil
			},
		},
		{
			name: "resolved Property has null trace",
			mutate: func(document *artifactv2.Result) {
				document.PropertyVerdicts[0].TraceID = nil
				document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
			},
		},
		{
			name: "resolved Property has diagnostic",
			mutate: func(document *artifactv2.Result) {
				document.PropertyVerdicts[0].Diagnostic = &artifactv2.SemanticVerdictDiagnostic{
					Kind: "missing-capability", RelatedDefinitionIDs: []string{},
				}
				document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
			},
		},
		{
			name: "Property status disagrees with clauses",
			mutate: func(document *artifactv2.Result) {
				document.PropertyVerdicts[0].Clauses[0].Status = "violated"
				document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
			},
		},
		{
			name: "Query summary verdict differs",
			mutate: func(document *artifactv2.Result) {
				document.QuerySummary.PropertyVerdicts[0].Status = "violated"
			},
		},
		{
			name: "semantic status differs from Query summary",
			mutate: func(document *artifactv2.Result) {
				document.SemanticStatus = "violated"
			},
		},
		{
			name: "resolved semantics omit evaluation checksum",
			mutate: func(document *artifactv2.Result) {
				document.EvaluationOutcomeChecksum = nil
			},
		},
		{
			name: "incomplete semantics carry evaluation checksum",
			mutate: func(document *artifactv2.Result) {
				document.PropertyVerdicts[0].Status = "unknown"
				document.PropertyVerdicts[0].Clauses = []artifactv2.SemanticClauseVerdict{}
				document.PropertyVerdicts[0].Diagnostic = &artifactv2.SemanticVerdictDiagnostic{
					Kind: "invalid-evidence-bound", RelatedDefinitionIDs: []string{},
				}
				document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
				document.QuerySummary.Status = "incomplete"
				document.SemanticStatus = "incomplete"
			},
		},
	} {
		t.Run("rejects "+test.name, func(t *testing.T) {
			document := resolvedResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
			)
			test.mutate(&document)
			document = sealedResultV2Document(t, document)
			_, err := artifact.EncodeResultV2(document)
			requireResultV2ErrorCode(t, err, artifact.ErrorMalformedValue)
		})
	}
}

func TestResultV2RejectsImplementationLinkDiagnosticIdentityDrift(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	document := unresolvedResultV2Document(
		t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
	)
	document.ImplementationLinkStatus = "invalid"
	document.ImplementationLink.Diagnostic = implementationLinkDiagnosticV2("stale-source-target")
	document = sealedResultV2Document(t, document)

	_, err := artifact.EncodeResultV2(document)
	requireResultV2ErrorCode(t, err, artifact.ErrorMalformedValue)
}

func TestResultV2ImplementationLinkDiagnosticIdentityUsesExactPrettyPreimage(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	document := unresolvedResultV2Document(
		t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
	)
	document.ImplementationLinkStatus = "invalid"
	document.ImplementationLink.Diagnostic = implementationLinkDiagnosticV2("stale-source-target")
	record := document.ImplementationLink
	diagnostic := record.Diagnostic

	type targetIdentity struct {
		ID                  string `json:"id"`
		Kind                string `json:"kind"`
		BehaviorFingerprint string `json:"behaviorFingerprint"`
	}
	preimage, err := artifact.CanonicalPretty(struct {
		ImplementationLinkID                  string              `json:"implementationLinkId"`
		ImplementationLinkBehaviorFingerprint string              `json:"implementationLinkBehaviorFingerprint"`
		SourceTarget                          targetIdentity      `json:"sourceTarget"`
		DestinationTarget                     targetIdentity      `json:"destinationTarget"`
		Kind                                  string              `json:"kind"`
		Status                                string              `json:"status"`
		Coordinate                            *string             `json:"coordinate"`
		RelatedDefinitionIDs                  []string            `json:"relatedDefinitionIds"`
		SourceSetupBehaviorFingerprint        *string             `json:"sourceSetupBehaviorFingerprint"`
		AppliedLimit                          *artifactv2.Limit   `json:"appliedLimit"`
		ObservedCount                         *artifactv2.Natural `json:"observedCount"`
		KnownGapCode                          *string             `json:"knownGapCode"`
		KnownGapReason                        *string             `json:"knownGapReason"`
		UnsupportedVocabularyKind             *string             `json:"unsupportedVocabularyKind"`
		EvidenceLinkBehaviorFingerprint       *string             `json:"evidenceLinkBehaviorFingerprint"`
	}{
		ImplementationLinkID:                  record.DefinitionID,
		ImplementationLinkBehaviorFingerprint: record.BehaviorFingerprint,
		SourceTarget: targetIdentity{
			ID: record.SourceTarget.DefinitionID, Kind: record.SourceTarget.Kind,
			BehaviorFingerprint: record.SourceTarget.BehaviorFingerprint,
		},
		DestinationTarget: targetIdentity{
			ID: record.DestinationTarget.DefinitionID, Kind: record.DestinationTarget.Kind,
			BehaviorFingerprint: record.DestinationTarget.BehaviorFingerprint,
		},
		Kind:                            diagnostic.Kind,
		Status:                          "invalid",
		Coordinate:                      nil,
		RelatedDefinitionIDs:            diagnostic.RelatedDefinitionIDs,
		SourceSetupBehaviorFingerprint:  diagnostic.SourceSetupBehaviorFingerprint,
		AppliedLimit:                    diagnostic.AppliedLimit,
		ObservedCount:                   diagnostic.ObservedCount,
		KnownGapCode:                    diagnostic.KnownGapCode,
		KnownGapReason:                  diagnostic.KnownGapReason,
		UnsupportedVocabularyKind:       diagnostic.UnsupportedVocabularyKind,
		EvidenceLinkBehaviorFingerprint: diagnostic.EvidenceLinkBehaviorFingerprint,
	})
	require.NoError(t, err)
	requireCanonicalJSONLine(t, preimage)
	expected := independentExperimentV2Checksum("umpire.behavior-fingerprint/v1", preimage)
	actual, err := artifactv2.ExpectedImplementationLinkDiagnosticIdentity(record)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestResultV2RejectsCanonicalChecksumAndClosureMutations(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	result := resolvedResultV2Document(t, experiment, runtimeConfiguration, run, rawEvidence, evidence)
	evidenceBytes, err := artifactv2.CanonicalEvidenceBytes(evidence)
	require.NoError(t, err)
	resultBytes, err := artifactv2.CanonicalResultBytes(result)
	require.NoError(t, err)

	for _, test := range []struct {
		name    string
		decode  func([]byte) error
		encoded []byte
		code    artifact.ErrorCode
	}{
		{
			name: "compact Evidence",
			decode: func(encoded []byte) error {
				_, decodeErr := artifact.DecodeEvidenceV2(encoded)
				return decodeErr
			},
			encoded: compactResultV2JSON(t, evidenceBytes),
			code:    artifact.ErrorNoncanonical,
		},
		{
			name: "alternate Result whitespace",
			decode: func(encoded []byte) error {
				_, decodeErr := artifact.DecodeResultV2(encoded)
				return decodeErr
			},
			encoded: replaceResultV2First(t, resultBytes,
				"{\n  \"formatVersion\":", "{\n    \"formatVersion\":"),
			code: artifact.ErrorNoncanonical,
		},
		{
			name: "Evidence field order",
			decode: func(encoded []byte) error {
				_, decodeErr := artifact.DecodeEvidenceV2(encoded)
				return decodeErr
			},
			encoded: replaceResultV2First(t, evidenceBytes,
				"  \"formatVersion\": \"umpire-evidence/v2\",\n  \"runIdentity\": "+
					jsonStringV2(t, evidence.RunIdentity)+",",
				"  \"runIdentity\": "+jsonStringV2(t, evidence.RunIdentity)+
					",\n  \"formatVersion\": \"umpire-evidence/v2\","),
			code: artifact.ErrorNoncanonical,
		},
		{
			name: "Result field order",
			decode: func(encoded []byte) error {
				_, decodeErr := artifact.DecodeResultV2(encoded)
				return decodeErr
			},
			encoded: replaceResultV2First(t, resultBytes,
				"  \"formatVersion\": \"umpire-result/v2\",\n  \"runIdentity\": "+
					jsonStringV2(t, result.RunIdentity)+",",
				"  \"runIdentity\": "+jsonStringV2(t, result.RunIdentity)+
					",\n  \"formatVersion\": \"umpire-result/v2\","),
			code: artifact.ErrorNoncanonical,
		},
		{
			name: "Evidence Artifact checksum drift",
			decode: func(encoded []byte) error {
				_, decodeErr := artifact.DecodeEvidenceV2(encoded)
				return decodeErr
			},
			encoded: replaceResultV2First(t, evidenceBytes, evidence.ArtifactChecksum,
				"sha256:"+strings.Repeat("0", 64)),
			code: artifact.ErrorArtifactChecksum,
		},
		{
			name: "Result provenance checksum drift",
			decode: func(encoded []byte) error {
				_, decodeErr := artifact.DecodeResultV2(encoded)
				return decodeErr
			},
			encoded: replaceResultV2First(t, resultBytes, result.ProvenanceChecksum,
				"sha256:"+strings.Repeat("f", 64)),
			code: artifact.ErrorProvenanceChecksum,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			requireResultV2ErrorCode(t, test.decode(test.encoded), test.code)
		})
	}

	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.Result)
	}{
		{
			name: "stale Evidence binding",
			mutate: func(document *artifactv2.Result) {
				document.Evidence.ArtifactChecksum = document.RawEvidence.ArtifactChecksum
			},
		},
		{
			name: "stale Property fingerprint",
			mutate: func(document *artifactv2.Result) {
				document.PropertyVerdicts[0].PropertyBehaviorFingerprint = document.BehaviorFingerprint
				document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
			},
		},
		{
			name: "clause link absent from Evidence",
			mutate: func(document *artifactv2.Result) {
				document.PropertyVerdicts[0].Clauses[0].EvidenceLinks[0].RuleDefinitionID =
					"switch.observation.rule.stale"
				document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
			},
		},
		{
			name: "Property verdict trace differs from Evidence",
			mutate: func(document *artifactv2.Result) {
				traceID := "sha256:" + strings.Repeat("b", 64)
				document.PropertyVerdicts[0].TraceID = &traceID
				document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
				document.QuerySummary.TraceIDs = []string{traceID}
				checksum, checksumErr := artifactv2.ExpectedEvaluationOutcomeChecksum(
					*document, evidence, experiment,
				)
				require.NoError(t, checksumErr)
				document.EvaluationOutcomeChecksum = &checksum
			},
		},
		{
			name: "Property verdict Evidence Limit differs from Evidence",
			mutate: func(document *artifactv2.Result) {
				limit := artifactv2.Limit{
					Value: artifactv2.NaturalFromUint64(1), Unit: "evidence-records",
				}
				document.PropertyVerdicts[0].EvidenceLimit = &limit
				document.PropertyVerdicts[0].Clauses[0].EvidenceLimit = limit
				document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
				checksum, checksumErr := artifactv2.ExpectedEvaluationOutcomeChecksum(
					*document, evidence, experiment,
				)
				require.NoError(t, checksumErr)
				document.EvaluationOutcomeChecksum = &checksum
			},
		},
		{
			name: "evaluation outcome checksum drift",
			mutate: func(document *artifactv2.Result) {
				checksum := "sha256:" + strings.Repeat("a", 64)
				document.EvaluationOutcomeChecksum = &checksum
			},
		},
	} {
		t.Run("closure rejects "+test.name, func(t *testing.T) {
			document := resolvedResultV2Document(
				t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
			)
			test.mutate(&document)
			document = sealedResultV2Document(t, document)
			err := artifact.ValidateResultV2Closure(
				document, experiment, runtimeConfiguration, run, rawEvidence, evidence,
			)
			requireResultV2ErrorCode(t, err, artifact.ErrorClosure)
		})
	}
}

func TestResultV2RejectsStaleQuerySummaryPartition(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	document := propertyNonSuccessResultV2Document(
		t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
		"unknown", "invalid-evidence-bound",
	)
	document.QuerySummary.MissingPropertyDefinitionIDs =
		document.QuerySummary.RequiredPropertyDefinitionIDs
	document = sealedResultV2Document(t, document)

	err := artifact.ValidateResultV2Closure(
		document, experiment, runtimeConfiguration, run, rawEvidence, evidence,
	)
	requireResultV2ErrorCode(t, err, artifact.ErrorClosure)
}

func TestResultV2EvaluationChecksumUsesExactPrettyPreimage(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	result := resolvedResultV2Document(t, experiment, runtimeConfiguration, run, rawEvidence, evidence)

	preimage, err := artifact.CanonicalPretty(struct {
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
		EvidenceBackedModelTrace: *evidence.EvidenceBackedModelTrace,
		EvidenceLinks:            evidence.EvidenceLinks,
		ObservationProgram:       evidence.ObservationProgram,
		Mapping:                  evidence.Mapping,
		ImplementationLink:       result.ImplementationLink,
		QuerySummary:             result.QuerySummary,
		Properties:               experiment.Properties,
		PropertyVerdicts:         result.PropertyVerdicts,
		Limits:                   result.Limits,
	})
	require.NoError(t, err)
	requireCanonicalJSONLine(t, preimage)
	require.Equal(t,
		independentExperimentV2Checksum("umpire.evaluation-outcome/v2", preimage),
		*result.EvaluationOutcomeChecksum,
	)
}

func TestResultV2AdmitsOperationalFailureIndependentlyFromResolvedSemantics(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	failureCode := "switch.phase.realization-failed"
	run.PhaseOutcomes[1].Status = "failed"
	run.PhaseOutcomes[1].Code = &failureCode
	run.OperationalStatus = "failed"
	var err error
	run, err = artifactv2.SealExperimentRun(run)
	require.NoError(t, err)
	_, err = artifact.EncodeExperimentRunV2(run)
	require.NoError(t, err)

	rawEvidence := rawEvidenceV2Document(t)
	rawEvidence.Run = artifactv2.ExperimentRunArtifactBinding(run)
	rawEvidence, err = artifactv2.SealRawEvidence(rawEvidence)
	require.NoError(t, err)
	require.NoError(t, artifact.ValidateRawEvidenceV2Closure(
		rawEvidence, experiment, runtimeConfiguration, run,
	))
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	result := resolvedResultV2Document(t, experiment, runtimeConfiguration, run, rawEvidence, evidence)

	encoded, err := artifact.EncodeResultV2(result)
	require.NoError(t, err)
	decoded, err := artifact.DecodeResultV2(encoded)
	require.NoError(t, err)
	require.Equal(t, "failed", decoded.OperationalStatus)
	require.Equal(t, "satisfied", decoded.SemanticStatus)
	require.NoError(t, artifact.ValidateResultV2Closure(
		decoded, experiment, runtimeConfiguration, run, rawEvidence, evidence,
	))
}

func TestResultV2RejectsEveryNestedProjectionFieldOrderMutation(t *testing.T) {
	experiment, runtimeConfiguration, run := rawEvidenceV2ClosureInputs(t)
	rawEvidence := rawEvidenceV2Document(t)
	evidence := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	result := resolvedResultV2Document(t, experiment, runtimeConfiguration, run, rawEvidence, evidence)
	failedEvidence := nonAcceptedEvidenceV2Document(
		t, experiment, runtimeConfiguration, run, rawEvidence, "unknown", "empty-evidence",
	)
	failedResult := unresolvedResultV2Document(
		t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
	)
	failedResult.ImplementationLinkStatus = "invalid"
	failedResult.ImplementationLink.Diagnostic = implementationLinkDiagnosticV2("stale-source-target")
	sealImplementationLinkDiagnosticV2(t, &failedResult.ImplementationLink)
	failedResult = sealedResultV2Document(t, failedResult)
	propertyFailure := propertyNonSuccessResultV2Document(
		t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
		"unknown", "invalid-evidence-bound",
	)

	canonical := func(value any) []byte {
		t.Helper()
		encoded, err := artifact.CanonicalPretty(value)
		require.NoError(t, err)
		return encoded
	}
	documents := map[string][]byte{
		"evidence":         canonical(evidence),
		"failed-evidence":  canonical(failedEvidence),
		"result":           canonical(result),
		"failed-result":    canonical(failedResult),
		"property-failure": canonical(propertyFailure),
	}

	for _, test := range []struct {
		name       string
		document   string
		anchor     string
		occurrence int
		first      string
		second     string
	}{
		{name: "Evidence", document: "evidence", anchor: `"formatVersion": "umpire-evidence/v2"`, first: "formatVersion", second: "runIdentity"},
		{name: "ArtifactBinding", document: "evidence", anchor: `"formatVersion": "umpire-experiment/v2"`, first: "formatVersion", second: "artifactChecksum"},
		{name: "ObservationPlanReference", document: "evidence", anchor: `"definitionId": "switch.observation.program"`, first: "definitionId", second: "behaviorFingerprint"},
		{name: "EvidenceBackedModelTrace", document: "evidence", anchor: `"sourceClosed": true`, first: "traceId", second: "observationPlan"},
		{name: "SourceLocation", document: "evidence", anchor: `"path": "Umpire/Artifact/Tests/Result.lean"`, first: "path", second: "line"},
		{name: "MeaningProvision", document: "evidence", anchor: `"canonicalBehavior": "switch.action.flip/v1"`, first: "definitionId", second: "kind"},
		{name: "Limit", document: "evidence", anchor: `"unit": "evidence-records"`, first: "value", second: "unit"},
		{name: "ModelTrace", document: "evidence", anchor: `"initialState": {`, first: "traceId", second: "initialState"},
		{name: "ModelTraceStep", document: "evidence", anchor: `"selectedAction": {`, first: "position", second: "selectedAction"},
		{name: "ModelValue", document: "evidence", anchor: `"value": "off"`, first: "definitionId", second: "value"},
		{name: "ModelCoordinate", document: "evidence", anchor: `"kind": "initial-state"`, first: "kind", second: "step"},
		{name: "EvidenceLink", document: "evidence", anchor: `"ruleDefinitionId": "switch.observation.rule.initial-state"`, first: "coordinate", second: "mappingDefinitionId"},
		{name: "EvidenceOrderingFact", document: "evidence", anchor: `"factDefinitionId": "switch.evidence.history.1"`, first: "factDefinitionId", second: "kindDefinitionId"},
		{name: "EvidenceClosureFact", document: "evidence", anchor: `"lastOrdinal": 1`, first: "kindDefinitionId", second: "lastOrdinal"},
		{name: "AppliedFieldDisposition", document: "evidence", anchor: `"normalizedValue": "flip-requested"`, first: "field", second: "kind"},
		{name: "FieldReference", document: "evidence", anchor: `"fieldDefinitionId": "umpire.evidence.field.event"`, first: "kindDefinitionId", second: "fieldDefinitionId"},
		{name: "FieldDispositionRecord", document: "evidence", anchor: `"disposition": "retain"`, first: "field", second: "disposition"},
		{name: "Provenance", document: "evidence", anchor: `"sourceDefinitionIds": [`, first: "sourceDefinitionIds", second: "sourceLocations"},
		{name: "ObservationDiagnostic", document: "failed-evidence", anchor: `"kind": "empty-evidence"`, first: "kind", second: "observationPlanDefinitionId"},
		{name: "Result", document: "result", anchor: `"formatVersion": "umpire-result/v2"`, first: "formatVersion", second: "runIdentity"},
		{name: "ImplementationLinkRecord", document: "result", anchor: `"definitionId": "switch.implementation-link.system-to-feature"`, first: "definitionId", second: "behaviorFingerprint"},
		{name: "ImplementationTargetReference", document: "result", anchor: `"definitionId": "switch.target.feature"`, first: "definitionId", second: "kind"},
		{name: "SemanticClauseVerdict", document: "result", anchor: `"clauseDefinitionId":`, first: "propertyDefinitionId", second: "clauseDefinitionId"},
		{name: "PropertyVerdict", document: "result", anchor: `"propertyBehaviorFingerprint":`, first: "queryDefinitionId", second: "propertyDefinitionId"},
		{name: "QueryLimits", document: "result", anchor: `"behavior": {`, first: "behavior", second: "search"},
		{name: "BehaviorLimits", document: "result", anchor: `"transitions": {`, first: "transitions", second: "selectedActions"},
		{name: "QuerySummary", document: "result", anchor: `"requiredPropertyDefinitionIds":`, first: "queryDefinitionId", second: "status"},
		{name: "StagedLimit", document: "result", anchor: `"stage": "observation-evaluation"`, first: "stage", second: "limit"},
		{name: "ImplementationLinkDiagnostic", document: "failed-result", anchor: `"kind": "stale-source-target"`, first: "kind", second: "coordinate"},
		{name: "SemanticVerdictDiagnostic", document: "property-failure", anchor: `"kind": "invalid-evidence-bound"`, first: "kind", second: "relatedDefinitionIds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := swapResultV2ObjectMembers(
				t, documents[test.document], test.anchor, test.occurrence, test.first, test.second,
			)
			var err error
			if strings.Contains(test.document, "evidence") {
				_, err = artifact.DecodeEvidenceV2(mutated)
			} else {
				_, err = artifact.DecodeResultV2(mutated)
			}
			requireResultV2ErrorCode(t, err, artifact.ErrorNoncanonical)
		})
	}
}

func acceptedEvidenceV2Document(
	t *testing.T,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
) artifactv2.Evidence {
	t.Helper()
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)
	mapping := artifactv2.DefinitionReference{
		DefinitionID:        runtimeConfiguration.Observation.MappingDefinitionID,
		BehaviorFingerprint: runtimeConfiguration.Observation.MappingBehaviorFingerprint,
	}
	program := artifactv2.DefinitionReference{
		DefinitionID:        runtimeConfiguration.Observation.ProgramDefinitionID,
		BehaviorFingerprint: runtimeConfiguration.Observation.ProgramBehaviorFingerprint,
	}
	evidenceLimit := artifactv2.Limit{
		Value: artifactv2.NaturalFromUint64(2),
		Unit:  "evidence-records",
	}
	field := artifactv2.FieldReference{
		KindDefinitionID:  "umpire.evidence.kind.history",
		FieldDefinitionID: "umpire.evidence.field.event",
	}
	meaning := artifactv2.MeaningProvision{
		DefinitionID:      experiment.Plan.InitialState.DefinitionID,
		Kind:              "state",
		CanonicalBehavior: "switch.state.off/v1",
	}
	initialLink := acceptedEvidenceV2Link(
		artifactv2.ModelCoordinate{Kind: "initial-state"},
		mapping,
		runtimeConfiguration,
		"switch.evidence.history.1",
		artifactv2.NaturalFromUint64(0),
		[]string{},
		"switch.observation.rule.initial-state",
		field,
		"flip-requested",
		evidenceLimit,
		artifactv2.BehaviorFingerprint([]byte(meaning.CanonicalBehavior)),
	)
	step := artifactv2.NaturalFromUint64(1)
	stepCoordinates := []artifactv2.ModelCoordinate{
		{Kind: "selected-action", Step: &step},
		{Kind: "model-outcome", Step: &step},
		{Kind: "resulting-state", Step: &step},
		{Kind: "observation", Step: &step, Position: naturalV2Pointer(1)},
	}
	evidenceLinks := []artifactv2.EvidenceLink{initialLink}
	for index, coordinate := range stepCoordinates {
		evidenceLinks = append(evidenceLinks, acceptedEvidenceV2Link(
			coordinate,
			mapping,
			runtimeConfiguration,
			"switch.evidence.history.2",
			artifactv2.NaturalFromUint64(1),
			[]string{"switch.evidence.history.1"},
			[]string{
				"switch.observation.rule.selected-action",
				"switch.observation.rule.model-outcome",
				"switch.observation.rule.resulting-state",
				"switch.observation.rule.observation",
			}[index],
			field,
			"flip-completed",
			evidenceLimit,
			artifactv2.BehaviorFingerprint([]byte("switch.meaning.step/v1")),
		))
	}
	document := artifactv2.Evidence{
		FormatVersion:               artifactv2.EvidenceFormat,
		RunIdentity:                 run.RunIdentity,
		BehaviorFingerprint:         "sha256:0aa42f873839132836c028886c9be5ad63e5dc66dbc967182ae139159501c8ab",
		Experiment:                  experimentBinding,
		RuntimeConfiguration:        artifactv2.RuntimeConfigurationArtifactBinding(runtimeConfiguration),
		Run:                         artifactv2.ExperimentRunArtifactBinding(run),
		RawEvidence:                 artifactv2.RawEvidenceArtifactBinding(rawEvidence),
		ObservationProgram:          program,
		Mapping:                     mapping,
		ObservationEvaluationStatus: "accepted",
		EvidenceBackedModelTrace: &artifactv2.EvidenceBackedModelTrace{
			TraceID:                    "switch.trace.accepted",
			ObservationPlan:            program,
			MappingDefinitionID:        mapping.DefinitionID,
			MappingVersion:             artifactv2.NaturalFromUint64(1),
			MappingBehaviorFingerprint: mapping.BehaviorFingerprint,
			Source: artifactv2.SourceLocation{
				Path: "Umpire/Artifact/Tests/Result.lean", Line: artifactv2.NaturalFromUint64(1),
				Column: artifactv2.NaturalFromUint64(1), Provenance: "lean-model",
			},
			ProfileDefinitionID: runtimeConfiguration.Observation.ProfileDefinitionID,
			ProfileVersion:      artifactv2.NaturalFromUint64(1),
			SourceClosed:        true,
			Vocabulary: []artifactv2.MeaningProvision{
				{DefinitionID: "switch.action.flip", Kind: "action", CanonicalBehavior: "switch.action.flip/v1"},
				{DefinitionID: "switch.observation.power", Kind: "observation", CanonicalBehavior: "switch.observation.power/v1"},
				{DefinitionID: "switch.outcome.accepted", Kind: "outcome", CanonicalBehavior: "switch.outcome.accepted/v1"},
				{DefinitionID: "switch.state.on", Kind: "state", CanonicalBehavior: "switch.state.on/v1"},
				meaning,
			},
			AppliedLimit:          evidenceLimit,
			EvidenceDefinitionIDs: []string{"switch.evidence.history.1", "switch.evidence.history.2"},
			Trace: artifactv2.ModelTrace{
				TraceID:      "switch.trace.accepted",
				InitialState: experiment.Plan.InitialState,
				Steps: []artifactv2.ModelTraceStep{{
					Position:       artifactv2.NaturalFromUint64(1),
					SelectedAction: artifactv2.ModelValue{DefinitionID: "switch.action.flip", Value: "requested"},
					ModelOutcome:   artifactv2.ModelValue{DefinitionID: "switch.outcome.accepted", Value: "accepted"},
					ResultingState: artifactv2.ModelValue{DefinitionID: "switch.state.on", Value: "on"},
					Observations:   []artifactv2.ModelValue{{DefinitionID: "switch.observation.power", Value: "on"}},
				}},
			},
		},
		EvidenceLinks: evidenceLinks,
		Dispositions: []artifactv2.FieldDispositionRecord{
			{Field: field, Disposition: "retain", DigestPolicyDefinitionID: nil},
			{
				Field: artifactv2.FieldReference{
					KindDefinitionID:  "umpire.evidence.kind.participant-output",
					FieldDefinitionID: "umpire.evidence.field.rejected",
				},
				Disposition: "reject", DigestPolicyDefinitionID: nil,
			},
		},
		Diagnostics: []artifactv2.ObservationDiagnostic{},
		KnownGaps:   []artifactv2.KnownGap{},
		Provenance: artifactv2.Provenance{
			SourceDefinitionIDs: []string{"switch.evidence.interpreted"},
			SourceLocations: []artifactv2.SourceLocation{{
				Path: "Umpire/Artifact/Tests/Result.lean", Line: artifactv2.NaturalFromUint64(1),
				Column: artifactv2.NaturalFromUint64(1), Provenance: "lean-model",
			}},
		},
	}
	document, err = artifactv2.SealEvidence(document)
	require.NoError(t, err)
	return document
}

func leanFixtureEvidenceV2Document(
	t *testing.T,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
) artifactv2.Evidence {
	t.Helper()
	document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	document.EvidenceBackedModelTrace.Vocabulary = []artifactv2.MeaningProvision{{
		DefinitionID:      experiment.Plan.InitialState.DefinitionID,
		Kind:              "state",
		CanonicalBehavior: "switch.state.off/v1",
	}}
	document.EvidenceBackedModelTrace.EvidenceDefinitionIDs = []string{"switch.evidence.history.1"}
	document.EvidenceBackedModelTrace.Trace.Steps = []artifactv2.ModelTraceStep{}
	document.EvidenceLinks = document.EvidenceLinks[:1]
	document.EvidenceLinks[0].ClosureSupport[0].LastOrdinal = artifactv2.NaturalFromUint64(0)
	return sealedEvidenceV2Document(t, document)
}

func acceptedEvidenceV2Link(
	coordinate artifactv2.ModelCoordinate,
	mapping artifactv2.DefinitionReference,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	factDefinitionID string,
	ordinal artifactv2.Natural,
	causalFactDefinitionIDs []string,
	ruleDefinitionID string,
	field artifactv2.FieldReference,
	normalizedValue string,
	evidenceLimit artifactv2.Limit,
	meaningBehaviorFingerprint string,
) artifactv2.EvidenceLink {
	return artifactv2.EvidenceLink{
		Coordinate:                 coordinate,
		MappingDefinitionID:        mapping.DefinitionID,
		MappingVersion:             artifactv2.NaturalFromUint64(1),
		MappingBehaviorFingerprint: mapping.BehaviorFingerprint,
		ProfileDefinitionID:        runtimeConfiguration.Observation.ProfileDefinitionID,
		ProfileVersion:             artifactv2.NaturalFromUint64(1),
		EvidenceDefinitionIDs:      []string{factDefinitionID},
		RuleDefinitionID:           ruleDefinitionID,
		BindingDefinitionIDs:       []string{},
		OrderingSupport: []artifactv2.EvidenceOrderingFact{{
			FactDefinitionID: factDefinitionID, KindDefinitionID: "umpire.evidence.kind.history",
			Ordinal: ordinal, CausalFactDefinitionIDs: causalFactDefinitionIDs,
		}},
		ClosureSupport: []artifactv2.EvidenceClosureFact{{
			KindDefinitionID: "umpire.evidence.kind.history", LastOrdinal: artifactv2.NaturalFromUint64(1),
		}},
		AppliedDispositions: []artifactv2.AppliedFieldDisposition{{
			Field: field, Kind: "retained", NormalizedValue: &normalizedValue,
			DigestPolicyDefinitionID: nil, DigestToken: nil,
		}},
		AppliedLimit:               evidenceLimit,
		MeaningBehaviorFingerprint: meaningBehaviorFingerprint,
	}
}

func resolvedResultV2Document(
	t *testing.T,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
	evidence artifactv2.Evidence,
) artifactv2.Result {
	t.Helper()
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)
	traceID := evidence.EvidenceBackedModelTrace.TraceID
	evidenceLimit := evidence.EvidenceBackedModelTrace.AppliedLimit
	queryLimits := experiment.Plan.ExpandedLimits
	verdicts := make([]artifactv2.PropertyVerdict, len(experiment.Properties))
	for index, property := range experiment.Properties {
		propertyLimit := artifactv2.Limit{
			Value: artifactv2.NaturalFromUint64(1), Unit: "observation-positions",
		}
		provenanceDefinitionIDs := []string{experiment.Plan.QueryDefinitionID, property.DefinitionID}
		slices.Sort(provenanceDefinitionIDs)
		verdicts[index] = artifactv2.PropertyVerdict{
			QueryDefinitionID:           experiment.Plan.QueryDefinitionID,
			PropertyDefinitionID:        property.DefinitionID,
			PropertyBehaviorFingerprint: property.BehaviorFingerprint,
			TraceID:                     &traceID,
			Status:                      "satisfied",
			QueryLimits:                 queryLimits,
			EvidenceLimit:               &evidenceLimit,
			ProvenanceDefinitionIDs:     provenanceDefinitionIDs,
			Clauses: []artifactv2.SemanticClauseVerdict{{
				PropertyDefinitionID:    property.DefinitionID,
				ClauseDefinitionID:      property.DefinitionID + ".clause",
				Status:                  "satisfied",
				Coordinates:             []artifactv2.ModelCoordinate{{Kind: "initial-state"}},
				QueryLimits:             queryLimits,
				PropertyLimit:           &propertyLimit,
				EvidenceLimit:           evidenceLimit,
				ProvenanceDefinitionIDs: []string{property.DefinitionID},
				EvidenceLinks:           []artifactv2.EvidenceLink{evidence.EvidenceLinks[0]},
			}},
			Diagnostic: nil,
		}
	}
	requiredPropertyDefinitionIDs := make([]string, len(experiment.Properties))
	for index, property := range experiment.Properties {
		requiredPropertyDefinitionIDs[index] = property.DefinitionID
	}
	result := artifactv2.Result{
		FormatVersion:               artifactv2.ResultFormat,
		RunIdentity:                 run.RunIdentity,
		BehaviorFingerprint:         "sha256:f6fbf2847d73f198dd50a9c466e6f1834f67042db0df0a54965c2bcb6b4f7a41",
		Experiment:                  experimentBinding,
		RuntimeConfiguration:        artifactv2.RuntimeConfigurationArtifactBinding(runtimeConfiguration),
		Run:                         artifactv2.ExperimentRunArtifactBinding(run),
		RawEvidence:                 artifactv2.RawEvidenceArtifactBinding(rawEvidence),
		Evidence:                    artifactv2.EvidenceArtifactBinding(evidence),
		OperationalStatus:           run.OperationalStatus,
		ObservationEvaluationStatus: evidence.ObservationEvaluationStatus,
		ImplementationLink: artifactv2.ImplementationLinkRecord{
			DefinitionID:        "switch.implementation-link.system-to-feature",
			BehaviorFingerprint: "sha256:0ec0f5e52dc5ed18516f1ffb9ae2973a98c5c7469a5482e2f0ef53f522f37d69",
			SourceTarget: artifactv2.ImplementationTargetReference{
				DefinitionID: experiment.Plan.TargetDefinitionID, Kind: "target",
				BehaviorFingerprint: experiment.Plan.TargetBehaviorFingerprint,
			},
			DestinationTarget: artifactv2.ImplementationTargetReference{
				DefinitionID: "switch.target.feature", Kind: "target",
				BehaviorFingerprint: "sha256:bf5ea7369835e8267f27e21cc1fb185505c83a6558905fb82b57fb55bd014828",
			},
			Diagnostic: nil,
		},
		ImplementationLinkStatus: "applied",
		PropertyVerdicts:         verdicts,
		QuerySummary: artifactv2.QuerySummary{
			QueryDefinitionID:               experiment.Plan.QueryDefinitionID,
			Status:                          "satisfied",
			QueryLimits:                     queryLimits,
			RequiredPropertyDefinitionIDs:   requiredPropertyDefinitionIDs,
			PropertyVerdicts:                verdicts,
			MissingPropertyDefinitionIDs:    []string{},
			DuplicatePropertyDefinitionIDs:  []string{},
			UnexpectedPropertyDefinitionIDs: []string{},
			DivergentPropertyDefinitionIDs:  []string{},
			WrongQueryResultDefinitionIDs:   []string{},
			TraceIDs:                        []string{traceID},
		},
		SemanticStatus: "satisfied",
		Limits: []artifactv2.StagedLimit{
			{Stage: "observation-evaluation", Limit: evidenceLimit},
			{Stage: "query", Limit: queryLimits.Search},
		},
		KnownGaps:     []artifactv2.KnownGap{},
		CleanupStatus: run.Cleanup.Status,
		Provenance: artifactv2.Provenance{
			SourceDefinitionIDs: []string{"switch.result.interpreted"},
			SourceLocations: []artifactv2.SourceLocation{{
				Path: "Umpire/Artifact/Tests/Result.lean", Line: artifactv2.NaturalFromUint64(1),
				Column: artifactv2.NaturalFromUint64(1), Provenance: "lean-model",
			}},
		},
	}
	checksum, err := artifactv2.ExpectedEvaluationOutcomeChecksum(result, evidence, experiment)
	require.NoError(t, err)
	result.EvaluationOutcomeChecksum = &checksum
	result, err = artifactv2.SealResult(result)
	require.NoError(t, err)
	return result
}

func naturalV2Pointer(value uint64) *artifactv2.Natural {
	natural := artifactv2.NaturalFromUint64(value)
	return &natural
}

func nonAcceptedEvidenceV2Document(
	t *testing.T,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
	status string,
	diagnosticKind string,
) artifactv2.Evidence {
	t.Helper()
	document := acceptedEvidenceV2Document(t, experiment, runtimeConfiguration, run, rawEvidence)
	document.ObservationEvaluationStatus = status
	document.EvidenceBackedModelTrace = nil
	document.EvidenceLinks = []artifactv2.EvidenceLink{}
	document.Diagnostics = []artifactv2.ObservationDiagnostic{
		observationDiagnosticV2(&document, diagnosticKind),
	}
	return sealedEvidenceV2Document(t, document)
}

func observationDiagnosticV2(document *artifactv2.Evidence, kind string) artifactv2.ObservationDiagnostic {
	return artifactv2.ObservationDiagnostic{
		Kind:                        kind,
		ObservationPlanDefinitionID: document.ObservationProgram.DefinitionID,
		RelatedDefinitionIDs:        []string{},
		Alternatives:                []string{},
	}
}

func unresolvedResultV2Document(
	t *testing.T,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
	evidence artifactv2.Evidence,
) artifactv2.Result {
	t.Helper()
	document := resolvedResultV2Document(
		t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
	)
	document.PropertyVerdicts = []artifactv2.PropertyVerdict{}
	document.QuerySummary = incompleteQuerySummaryV2(experiment)
	document.SemanticStatus = "incomplete"
	document.EvaluationOutcomeChecksum = nil
	return sealedResultV2Document(t, document)
}

func propertyNonSuccessResultV2Document(
	t *testing.T,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
	evidence artifactv2.Evidence,
	status string,
	diagnosticKind string,
) artifactv2.Result {
	t.Helper()
	document := unresolvedResultV2Document(
		t, experiment, runtimeConfiguration, run, rawEvidence, evidence,
	)
	property := experiment.Properties[0]
	verdict := artifactv2.PropertyVerdict{
		QueryDefinitionID:           experiment.Plan.QueryDefinitionID,
		PropertyDefinitionID:        property.DefinitionID,
		PropertyBehaviorFingerprint: property.BehaviorFingerprint,
		Status:                      status,
		QueryLimits:                 experiment.Plan.ExpandedLimits,
		ProvenanceDefinitionIDs:     []string{property.DefinitionID},
		Clauses:                     []artifactv2.SemanticClauseVerdict{},
		Diagnostic: &artifactv2.SemanticVerdictDiagnostic{
			Kind: diagnosticKind, RelatedDefinitionIDs: []string{property.DefinitionID},
		},
	}
	if diagnosticKind != "query-property-mismatch" {
		traceID := evidence.EvidenceBackedModelTrace.TraceID
		evidenceLimit := evidence.EvidenceBackedModelTrace.AppliedLimit
		verdict.TraceID = &traceID
		verdict.EvidenceLimit = &evidenceLimit
		document.QuerySummary.TraceIDs = []string{traceID}
	}
	document.PropertyVerdicts = []artifactv2.PropertyVerdict{verdict}
	document.QuerySummary.PropertyVerdicts = document.PropertyVerdicts
	document.QuerySummary.MissingPropertyDefinitionIDs = []string{}
	return sealedResultV2Document(t, document)
}

func incompleteQuerySummaryV2(experiment artifactv2.Experiment) artifactv2.QuerySummary {
	required := make([]string, len(experiment.Properties))
	for index, property := range experiment.Properties {
		required[index] = property.DefinitionID
	}
	return artifactv2.QuerySummary{
		QueryDefinitionID:               experiment.Plan.QueryDefinitionID,
		Status:                          "incomplete",
		QueryLimits:                     experiment.Plan.ExpandedLimits,
		RequiredPropertyDefinitionIDs:   required,
		PropertyVerdicts:                []artifactv2.PropertyVerdict{},
		MissingPropertyDefinitionIDs:    required,
		DuplicatePropertyDefinitionIDs:  []string{},
		UnexpectedPropertyDefinitionIDs: []string{},
		DivergentPropertyDefinitionIDs:  []string{},
		WrongQueryResultDefinitionIDs:   []string{},
		TraceIDs:                        []string{},
	}
}

func implementationLinkDiagnosticV2(kind string) *artifactv2.ImplementationLinkDiagnostic {
	diagnostic := &artifactv2.ImplementationLinkDiagnostic{
		Kind:                 kind,
		RelatedDefinitionIDs: []string{},
		Identity:             "sha256:6639cd4b341550eed8e729afe5a56fab11d7e363bc0ebbcbe0e4bf72752b698e",
	}
	if kind == "known-gap" {
		code := "switch.known-gap.unsupported-link"
		reason := "destination vocabulary is intentionally unavailable"
		diagnostic.KnownGapCode = &code
		diagnostic.KnownGapReason = &reason
	}
	return diagnostic
}

func sealImplementationLinkDiagnosticV2(t *testing.T, record *artifactv2.ImplementationLinkRecord) {
	t.Helper()
	identity, err := artifactv2.ExpectedImplementationLinkDiagnosticIdentity(*record)
	require.NoError(t, err)
	record.Diagnostic.Identity = identity
}

func sealedEvidenceV2Document(t *testing.T, document artifactv2.Evidence) artifactv2.Evidence {
	t.Helper()
	sealed, err := artifactv2.SealEvidence(document)
	require.NoError(t, err)
	return sealed
}

func sealedResultV2Document(t *testing.T, document artifactv2.Result) artifactv2.Result {
	t.Helper()
	sealed, err := artifactv2.SealResult(document)
	require.NoError(t, err)
	return sealed
}

func compactResultV2JSON(t *testing.T, encoded []byte) []byte {
	t.Helper()
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, encoded))
	return compact.Bytes()
}

func replaceResultV2First(t *testing.T, encoded []byte, old, replacement string) []byte {
	t.Helper()
	require.Contains(t, string(encoded), old)
	return bytes.Replace(encoded, []byte(old), []byte(replacement), 1)
}

func jsonStringV2(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

func requireResultV2ErrorCode(t *testing.T, err error, expected artifact.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	code, ok := artifact.CodeOf(err)
	require.True(t, ok, err)
	require.Equal(t, expected, code)
}

func swapResultV2ObjectMembers(
	t *testing.T,
	encoded []byte,
	anchor string,
	occurrence int,
	firstKey string,
	secondKey string,
) []byte {
	t.Helper()
	anchorOffset := -1
	searchFrom := 0
	for index := 0; index <= occurrence; index++ {
		relative := bytes.Index(encoded[searchFrom:], []byte(anchor))
		require.NotEqual(t, -1, relative, "anchor %q occurrence %d", anchor, occurrence)
		anchorOffset = searchFrom + relative
		searchFrom = anchorOffset + len(anchor)
	}

	type delimiter struct {
		kind  byte
		index int
	}
	stack := make([]delimiter, 0, 16)
	inString := false
	escaped := false
	for index, character := range encoded[:anchorOffset] {
		if inString {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inString = false
			}
			continue
		}
		switch character {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, delimiter{kind: character, index: index})
		case '}', ']':
			require.NotEmpty(t, stack)
			stack = stack[:len(stack)-1]
		}
	}
	objectStart := -1
	for index := len(stack) - 1; index >= 0; index-- {
		if stack[index].kind == '{' {
			objectStart = stack[index].index
			break
		}
	}
	require.NotEqual(t, -1, objectStart, "anchor %q is not inside an object", anchor)

	objectEnd := -1
	depth := 0
	inString = false
	escaped = false
	for index := objectStart; index < len(encoded); index++ {
		character := encoded[index]
		if inString {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inString = false
			}
			continue
		}
		switch character {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				objectEnd = index
				index = len(encoded)
			}
		}
	}
	require.NotEqual(t, -1, objectEnd)

	content := encoded[objectStart+1 : objectEnd]
	members := make([]string, 0, 16)
	memberStart := 0
	containerDepth := 0
	inString = false
	escaped = false
	for index, character := range content {
		if inString {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inString = false
			}
			continue
		}
		switch character {
		case '"':
			inString = true
		case '{', '[':
			containerDepth++
		case '}', ']':
			containerDepth--
		case ',':
			if containerDepth == 0 {
				members = append(members, strings.TrimSpace(string(content[memberStart:index])))
				memberStart = index + 1
			}
		}
	}
	members = append(members, strings.TrimSpace(string(content[memberStart:])))
	firstIndex := -1
	secondIndex := -1
	for index, member := range members {
		if strings.HasPrefix(member, jsonStringV2(t, firstKey)+":") {
			firstIndex = index
		}
		if strings.HasPrefix(member, jsonStringV2(t, secondKey)+":") {
			secondIndex = index
		}
	}
	require.NotEqual(t, -1, firstIndex, "member %q absent from object containing %q", firstKey, anchor)
	require.NotEqual(t, -1, secondIndex, "member %q absent from object containing %q", secondKey, anchor)
	members[firstIndex], members[secondIndex] = members[secondIndex], members[firstIndex]

	lineStart := bytes.LastIndex(encoded[:objectEnd], []byte{'\n'}) + 1
	closingIndent := string(encoded[lineStart:objectEnd])
	memberIndent := closingIndent + "  "
	replacement := []byte("\n" + memberIndent + strings.Join(members, ",\n"+memberIndent) +
		"\n" + closingIndent)
	mutated := make([]byte, 0, len(encoded))
	mutated = append(mutated, encoded[:objectStart+1]...)
	mutated = append(mutated, replacement...)
	mutated = append(mutated, encoded[objectEnd:]...)
	return mutated
}
