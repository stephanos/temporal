package artifact_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestRuntimeConfigurationV2CanonicalFixtureRoundTrip(t *testing.T) {
	authoritative := readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/RuntimeConfigurationV2.json")
	encoded := readExperimentV2Fixture(t,
		"tools/umpire/artifact/testdata/runtime-configuration-v2.json")
	// The Go testdata remains a byte-exact mirror of its Lean-owned fixture.
	//nolint:testifylint
	require.Equal(t, authoritative, encoded)

	document, err := artifact.DecodeRuntimeConfigurationV2(encoded)
	require.NoError(t, err)
	reencoded, err := artifact.EncodeRuntimeConfigurationV2(document)
	require.NoError(t, err)
	require.Equal(t, encoded, reencoded)
	require.True(t, bytes.HasSuffix(reencoded, []byte{'\n'}))
	require.False(t, bytes.HasSuffix(reencoded, []byte("\n\n")))
}

func TestRuntimeV2ExperimentRunCanonicalFixtureRoundTrip(t *testing.T) {
	authoritative := readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/ExperimentRunV2.json")
	encoded := readExperimentV2Fixture(t,
		"tools/umpire/artifact/testdata/experiment-run-v2.json")
	// The Go testdata remains a byte-exact mirror of its Lean-owned fixture.
	//nolint:testifylint
	require.Equal(t, authoritative, encoded)

	document, err := artifact.DecodeExperimentRunV2(encoded)
	require.NoError(t, err)
	reencoded, err := artifact.EncodeExperimentRunV2(document)
	require.NoError(t, err)
	require.Equal(t, encoded, reencoded)
	require.True(t, bytes.HasSuffix(reencoded, []byte{'\n'}))
	require.False(t, bytes.HasSuffix(reencoded, []byte("\n\n")))
}

func TestRuntimeV2ArtifactBindingsCloseAgainstExperiment(t *testing.T) {
	experiment, err := artifact.DecodeExperimentV2(readExperimentV2Fixture(t,
		"tools/umpire/artifact/testdata/switch-experiment-v2.json"))
	require.NoError(t, err)
	runtimeConfiguration, err := artifact.DecodeRuntimeConfigurationV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/RuntimeConfigurationV2.json"))
	require.NoError(t, err)
	run, err := artifact.DecodeExperimentRunV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/ExperimentRunV2.json"))
	require.NoError(t, err)

	require.NoError(t, artifact.ValidateRuntimeConfigurationV2Closure(runtimeConfiguration, experiment))
	require.NoError(t, artifact.ValidateExperimentRunV2Closure(run, experiment, runtimeConfiguration))
}

func TestRuntimeV2ChecksumsUseExactPrettyPreimages(t *testing.T) {
	for _, test := range []struct {
		name               string
		provenanceChecksum string
		artifactChecksum   string
		provenance         func() artifactv2.Provenance
		preimage           func() []byte
	}{
		{
			name:               "RuntimeConfiguration",
			provenanceChecksum: "sha256:09745642d54e6faf89fd0c5a1a848d62fab3d8e472cc653db4fd02a96ff9e34e",
			artifactChecksum:   "sha256:454acc851c5c1638166b1a334eaaedc97e4515b5ebe6614d5a57672ddbd9d1c2",
			provenance: func() artifactv2.Provenance {
				return runtimeConfigurationV2Fixture(t).Provenance
			},
			preimage: func() []byte {
				document := runtimeConfigurationV2Fixture(t)
				document.ArtifactChecksum = ""
				encoded, err := artifact.CanonicalPretty(document)
				require.NoError(t, err)
				return encoded
			},
		},
		{
			name:               "ExperimentRun",
			provenanceChecksum: "sha256:b879d5eba0c02a60c52e59a009c79f953310a6c49e3453ea863fddcbb07a75a9",
			artifactChecksum:   "sha256:f1e9bce053d7ab53f9e9259187395456dc026934a317144785b1dcbe7475868e",
			provenance: func() artifactv2.Provenance {
				return experimentRunV2Fixture(t).Provenance
			},
			preimage: func() []byte {
				document := experimentRunV2Fixture(t)
				document.ArtifactChecksum = ""
				encoded, err := artifact.CanonicalPretty(document)
				require.NoError(t, err)
				return encoded
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			provenancePreimage, err := artifact.CanonicalPretty(test.provenance())
			require.NoError(t, err)
			requireCanonicalJSONLine(t, provenancePreimage)
			require.Equal(t, test.provenanceChecksum,
				independentExperimentV2Checksum("umpire.provenance/v2", provenancePreimage))

			artifactPreimage := test.preimage()
			requireCanonicalJSONLine(t, artifactPreimage)
			domain := "umpire.runtime-configuration/v2"
			if test.name == "ExperimentRun" {
				domain = "umpire.experiment-run/v2"
			}
			require.Equal(t, test.artifactChecksum,
				independentExperimentV2Checksum(domain, artifactPreimage))
		})
	}
}

func TestRuntimeV2ExperimentRunClosedStatusMatrices(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.ExperimentRun)
	}{
		{
			name: "phase not started",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "not-started"
				document.PhaseOutcomes[1].StartedAtUnixMillis = nil
				document.PhaseOutcomes[1].FinishedAtUnixMillis = nil
				document.OperationalStatus = "incomplete"
			},
		},
		{
			name: "phase failed",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "failed"
				document.PhaseOutcomes[1].Code = stringPointer("switch.phase.failed")
				document.OperationalStatus = "failed"
			},
		},
		{
			name: "phase timed out",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "timed-out"
				document.PhaseOutcomes[1].Code = stringPointer("switch.phase.timed-out")
				document.OperationalStatus = "incomplete"
			},
		},
		{
			name: "phase canceled",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "canceled"
				document.PhaseOutcomes[1].Code = stringPointer("switch.phase.canceled")
				document.OperationalStatus = "incomplete"
			},
		},
		{
			name: "control not attempted",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = "not-attempted"
				document.ControlAttempts[0].ReceiptFactDefinitionID = nil
				document.OperationalStatus = "incomplete"
			},
		},
		{
			name: "control rejected",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = "rejected"
				document.ControlAttempts[0].Code = stringPointer("switch.control.rejected")
				document.OperationalStatus = "failed"
			},
		},
		{
			name: "control unsupported",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = "unsupported"
				document.ControlAttempts[0].Code = stringPointer("switch.control.unsupported")
				document.OperationalStatus = "failed"
			},
		},
		{
			name: "control failed",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = "failed"
				document.ControlAttempts[0].Code = stringPointer("switch.control.failed")
				document.OperationalStatus = "failed"
			},
		},
		{
			name: "control canceled",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = "canceled"
				document.ControlAttempts[0].Code = stringPointer("switch.control.canceled")
				document.OperationalStatus = "incomplete"
			},
		},
		{
			name: "source partial",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.SourceClosures[0].Status = "partial"
				document.OperationalStatus = "incomplete"
			},
		},
		{
			name: "source failed",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.SourceClosures[0].Status = "failed"
				document.OperationalStatus = "failed"
			},
		},
		{
			name: "cleanup incomplete",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.Cleanup.Status = "incomplete"
				document.Cleanup.Code = stringPointer("switch.cleanup.incomplete")
				document.OperationalStatus = "incomplete"
			},
		},
		{
			name: "cleanup failed",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.Cleanup.Status = "failed"
				document.Cleanup.Code = stringPointer("switch.cleanup.failed")
				document.OperationalStatus = "failed"
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded := resealedExperimentRunV2Mutation(t, test.mutate)
			_, err := artifact.DecodeExperimentRunV2(encoded)
			require.NoError(t, err)
		})
	}
}

func TestRuntimeV2RejectsCrossBoundaryInconsistency(t *testing.T) {
	experiment, err := artifact.DecodeExperimentV2(readExperimentV2Fixture(t,
		"tools/umpire/artifact/testdata/switch-experiment-v2.json"))
	require.NoError(t, err)

	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.RuntimeConfiguration)
	}{
		{
			name: "experiment Artifact Checksum",
			mutate: func(document *artifactv2.RuntimeConfiguration) {
				document.Experiment.ArtifactChecksum =
					"sha256:0000000000000000000000000000000000000000000000000000000000000000"
			},
		},
		{
			name: "experiment Behavior Fingerprint",
			mutate: func(document *artifactv2.RuntimeConfiguration) {
				document.Experiment.BehaviorFingerprint =
					"sha256:0000000000000000000000000000000000000000000000000000000000000000"
			},
		},
		{
			name: "experiment provenance checksum",
			mutate: func(document *artifactv2.RuntimeConfiguration) {
				document.Experiment.ProvenanceChecksum =
					"sha256:0000000000000000000000000000000000000000000000000000000000000000"
			},
		},
		{
			name: "missing capability",
			mutate: func(document *artifactv2.RuntimeConfiguration) {
				document.ParticipantBindings[0].CapabilityDefinitionIDs = []string{}
			},
		},
		{
			name: "extra capability",
			mutate: func(document *artifactv2.RuntimeConfiguration) {
				document.ParticipantBindings[0].CapabilityDefinitionIDs = append(
					document.ParticipantBindings[0].CapabilityDefinitionIDs,
					"switch.capability.unexpected")
			},
		},
	} {
		t.Run("RuntimeConfiguration "+test.name, func(t *testing.T) {
			document := runtimeConfigurationV2Fixture(t)
			test.mutate(&document)
			err := artifact.ValidateRuntimeConfigurationV2Closure(document, experiment)
			requireRuntimeV2ErrorCode(t, err, artifact.ErrorClosure)
		})
	}

	runtimeConfiguration := runtimeConfigurationV2Fixture(t)
	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.ExperimentRun)
	}{
		{
			name: "experiment binding",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.Experiment.ArtifactChecksum =
					"sha256:0000000000000000000000000000000000000000000000000000000000000000"
			},
		},
		{
			name: "runtime configuration binding",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.RuntimeConfiguration.BehaviorFingerprint =
					"sha256:0000000000000000000000000000000000000000000000000000000000000000"
			},
		},
		{
			name: "cross-staged limits",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.Limits[0].MaxBytes = "2"
			},
		},
		{
			name: "missing planned control",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts = []artifactv2.ControlAttempt{}
				document.OperationalStatus = "succeeded"
			},
		},
		{
			name: "crossed action",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].ActionDefinitionID = "switch.action.unexpected"
			},
		},
		{
			name: "unknown occurrence",
			mutate: func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].OccurrenceDefinitionID = "switch.occurrence.unexpected"
			},
		},
	} {
		t.Run("ExperimentRun "+test.name, func(t *testing.T) {
			document := experimentRunV2Fixture(t)
			test.mutate(&document)
			err := artifact.ValidateExperimentRunV2Closure(document, experiment, runtimeConfiguration)
			requireRuntimeV2ErrorCode(t, err, artifact.ErrorClosure)
		})
	}
}

func TestRuntimeV2StringBounds(t *testing.T) {
	identityAtLimit := "a." + strings.Repeat("x", artifact.MaximumIdentityBytes-2)
	identityOverLimit := identityAtLimit + "x"
	detailAtLimit := strings.Repeat("x", artifact.MaximumDiagnosticBytes)
	detailOverLimit := detailAtLimit + "x"

	for _, test := range []struct {
		name      string
		atLimit   []byte
		overLimit []byte
		decode    func([]byte) error
	}{
		{
			name: "RuntimeConfiguration identity",
			atLimit: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.ConfigurationDefinitionID = identityAtLimit
			}),
			overLimit: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.ConfigurationDefinitionID = identityOverLimit
			}),
			decode: func(encoded []byte) error {
				_, err := artifact.DecodeRuntimeConfigurationV2(encoded)
				return err
			},
		},
		{
			name: "ExperimentRun identity",
			atLimit: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.RunIdentity = identityAtLimit
			}),
			overLimit: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.RunIdentity = identityOverLimit
			}),
			decode: func(encoded []byte) error {
				_, err := artifact.DecodeExperimentRunV2(encoded)
				return err
			},
		},
		{
			name: "Known Gap detail",
			atLimit: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.KnownGaps = []artifactv2.KnownGap{{
					Kind: "input", Code: "switch.gap.capacity", Detail: stringPointer(detailAtLimit),
				}}
				document.OperationalStatus = "incomplete"
			}),
			overLimit: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.KnownGaps = []artifactv2.KnownGap{{
					Kind: "input", Code: "switch.gap.capacity", Detail: stringPointer(detailOverLimit),
				}}
				document.OperationalStatus = "incomplete"
			}),
			decode: func(encoded []byte) error {
				_, err := artifact.DecodeExperimentRunV2(encoded)
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, test.decode(test.atLimit))
			requireRuntimeV2ErrorCode(t, test.decode(test.overLimit), artifact.ErrorStringLimit)
		})
	}
}

func TestRuntimeConfigurationV2RejectsOneAtATimeMutations(t *testing.T) {
	canonical := readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/RuntimeConfigurationV2.json")
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, canonical))
	compact.WriteByte('\n')

	cases := map[string]struct {
		encoded []byte
		code    artifact.ErrorCode
	}{
		"malformed JSON": {
			encoded: []byte("{\n"),
			code:    artifact.ErrorSyntax,
		},
		"unsupported format": {
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-runtime-configuration/v2"`, `"umpire-runtime-configuration/v1"`),
			code: artifact.ErrorUnsupportedFormat,
		},
		"wrong family": {
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-runtime-configuration/v2"`, `"umpire-experiment-run/v2"`),
			code: artifact.ErrorWrongFamily,
		},
		"unsupported experiment binding format": {
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-experiment/v2"`, `"umpire-experiment/`+`v1"`),
			code: artifact.ErrorUnsupportedFormat,
		},
		"wrong experiment binding family": {
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-experiment/v2"`, `"umpire-result/v2"`),
			code: artifact.ErrorWrongFamily,
		},
		"unknown authority material": {
			encoded: replaceExperimentV2Once(t, canonical,
				"\n  \"authorityProfile\": {", "\n  \"endpoint\": \"localhost\",\n  \"authorityProfile\": {"),
			code: artifact.ErrorUnknownField,
		},
		"compact JSON": {
			encoded: compact.Bytes(),
			code:    artifact.ErrorNoncanonical,
		},
		"alternate indentation": {
			encoded: replaceExperimentV2Once(t, canonical,
				"{\n  \"formatVersion\"", "{\n    \"formatVersion\""),
			code: artifact.ErrorNoncanonical,
		},
		"reordered fields": {
			encoded: replaceExperimentV2Once(t, canonical,
				"{\n  \"formatVersion\": \"umpire-runtime-configuration/v2\",\n  \"configurationDefinitionId\": \"switch.runtime.configuration\",",
				"{\n  \"configurationDefinitionId\": \"switch.runtime.configuration\",\n  \"formatVersion\": \"umpire-runtime-configuration/v2\","),
			code: artifact.ErrorNoncanonical,
		},
		"missing terminal LF": {
			encoded: bytes.TrimSuffix(canonical, []byte{'\n'}),
			code:    artifact.ErrorNoncanonical,
		},
		"extra terminal LF": {
			encoded: append(bytes.Clone(canonical), '\n'),
			code:    artifact.ErrorNoncanonical,
		},
		"malformed configuration definition ID": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.ConfigurationDefinitionID = "unnamespaced"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"malformed behavior fingerprint": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.BehaviorFingerprint = "sha256:ABC"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"missing phase": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.PhaseLimits = document.PhaseLimits[:4]
			}),
			code: artifact.ErrorMalformedValue,
		},
		"phase order": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.PhaseLimits[0], document.PhaseLimits[1] = document.PhaseLimits[1], document.PhaseLimits[0]
			}),
			code: artifact.ErrorMalformedValue,
		},
		"zero phase bound": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.PhaseLimits[0].MaxAttempts = "0"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"zero authority profile version": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.AuthorityProfile.Version = "0"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"unsorted authority capabilities": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.AuthorityProfile.RequiredCapabilityDefinitionIDs = []string{
					"switch.capability.z", "switch.capability.a",
				}
			}),
			code: artifact.ErrorMalformedValue,
		},
		"malformed observation": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.Observation.MappingDefinitionID = "unnamespaced"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"null participants": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.ParticipantBindings = nil
			}),
			code: artifact.ErrorMalformedValue,
		},
		"duplicate participant": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.ParticipantBindings = append(document.ParticipantBindings, document.ParticipantBindings[0])
			}),
			code: artifact.ErrorMalformedValue,
		},
		"unsorted participant capabilities": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.ParticipantBindings[0].CapabilityDefinitionIDs = []string{
					"switch.capability.z", "switch.capability.a",
				}
			}),
			code: artifact.ErrorMalformedValue,
		},
		"invalid Known Gap": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.KnownGaps = []artifactv2.KnownGap{{Kind: "free-form", Code: "switch.gap.invalid"}}
			}),
			code: artifact.ErrorMalformedValue,
		},
		"null Known Gaps": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.KnownGaps = nil
			}),
			code: artifact.ErrorMalformedValue,
		},
		"malformed provenance": {
			encoded: resealedRuntimeConfigurationV2Mutation(t, func(document *artifactv2.RuntimeConfiguration) {
				document.Provenance.SourceLocations[0].Line = "0"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"provenance checksum drift": {
			encoded: replaceExperimentV2Once(t, canonical,
				"sha256:09745642d54e6faf89fd0c5a1a848d62fab3d8e472cc653db4fd02a96ff9e34e",
				"sha256:19745642d54e6faf89fd0c5a1a848d62fab3d8e472cc653db4fd02a96ff9e34e"),
			code: artifact.ErrorProvenanceChecksum,
		},
		"artifact checksum drift": {
			encoded: replaceExperimentV2Once(t, canonical,
				"sha256:454acc851c5c1638166b1a334eaaedc97e4515b5ebe6614d5a57672ddbd9d1c2",
				"sha256:554acc851c5c1638166b1a334eaaedc97e4515b5ebe6614d5a57672ddbd9d1c2"),
			code: artifact.ErrorArtifactChecksum,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := artifact.DecodeRuntimeConfigurationV2(test.encoded)
			requireRuntimeV2ErrorCode(t, err, test.code)
		})
	}
}

func TestRuntimeV2ExperimentRunRejectsOneAtATimeMutations(t *testing.T) {
	canonical := readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/ExperimentRunV2.json")
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, canonical))
	compact.WriteByte('\n')

	cases := []struct {
		name    string
		encoded []byte
		code    artifact.ErrorCode
	}{
		{name: "malformed JSON", encoded: []byte("{\n"), code: artifact.ErrorSyntax},
		{
			name: "unsupported format",
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-experiment-run/v2"`, `"umpire-experiment-run/v1"`),
			code: artifact.ErrorUnsupportedFormat,
		},
		{
			name: "wrong family",
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-experiment-run/v2"`, `"umpire-result/v2"`),
			code: artifact.ErrorWrongFamily,
		},
		{
			name: "unsupported experiment binding",
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-experiment/v2"`, `"umpire-experiment/`+`v1"`),
			code: artifact.ErrorUnsupportedFormat,
		},
		{
			name: "unsupported runtime binding",
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-runtime-configuration/v2"`, `"umpire-runtime-configuration/v1"`),
			code: artifact.ErrorUnsupportedFormat,
		},
		{
			name: "Property field",
			encoded: replaceExperimentV2Once(t, canonical,
				"\n  \"operationalStatus\":", "\n  \"propertyVerdicts\": [],\n  \"operationalStatus\":"),
			code: artifact.ErrorUnknownField,
		},
		{
			name: "Claim Assessment field",
			encoded: replaceExperimentV2Once(t, canonical,
				"\n  \"operationalStatus\":", "\n  \"claimAssessment\": null,\n  \"operationalStatus\":"),
			code: artifact.ErrorUnknownField,
		},
		{name: "compact JSON", encoded: compact.Bytes(), code: artifact.ErrorNoncanonical},
		{
			name: "reordered fields",
			encoded: replaceExperimentV2Once(t, canonical,
				"{\n  \"formatVersion\": \"umpire-experiment-run/v2\",\n  \"runIdentity\": \"switch.run.1\",",
				"{\n  \"runIdentity\": \"switch.run.1\",\n  \"formatVersion\": \"umpire-experiment-run/v2\","),
			code: artifact.ErrorNoncanonical,
		},
		{
			name:    "missing terminal LF",
			encoded: bytes.TrimSuffix(canonical, []byte{'\n'}),
			code:    artifact.ErrorNoncanonical,
		},
		{
			name:    "extra terminal LF",
			encoded: append(bytes.Clone(canonical), '\n'),
			code:    artifact.ErrorNoncanonical,
		},
		{
			name: "malformed run identity",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.RunIdentity = "unnamespaced"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "zero attempt",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.Attempt = "0"
				document.ControlAttempts[0].Attempt = "0"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "unknown operational status",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.OperationalStatus = "unknown"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "missing phase outcome",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes = document.PhaseOutcomes[:4]
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "phase order",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[0], document.PhaseOutcomes[1] =
					document.PhaseOutcomes[1], document.PhaseOutcomes[0]
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "unknown phase status",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "active"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "not-started phase has timestamp",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "not-started"
				document.PhaseOutcomes[1].FinishedAtUnixMillis = nil
				document.OperationalStatus = "incomplete"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "not-started phase has code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "not-started"
				document.PhaseOutcomes[1].StartedAtUnixMillis = nil
				document.PhaseOutcomes[1].FinishedAtUnixMillis = nil
				document.PhaseOutcomes[1].Code = stringPointer("switch.phase.skipped")
				document.OperationalStatus = "incomplete"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "succeeded phase missing timestamp",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].StartedAtUnixMillis = nil
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "succeeded phase has code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Code = stringPointer("switch.phase.unexpected")
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "terminal timestamps reversed",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].StartedAtUnixMillis = naturalPointer("1201")
				document.PhaseOutcomes[1].FinishedAtUnixMillis = naturalPointer("1200")
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "failed phase missing code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "failed"
				document.OperationalStatus = "failed"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "timed-out phase missing code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "timed-out"
				document.OperationalStatus = "incomplete"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "canceled phase missing code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.PhaseOutcomes[1].Status = "canceled"
				document.OperationalStatus = "incomplete"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "unknown control status",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = "unknown"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "control attempt differs from Run",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Attempt = "2"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "not-attempted control has receipt",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = "not-attempted"
				document.OperationalStatus = "incomplete"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "not-attempted control has code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = "not-attempted"
				document.ControlAttempts[0].ReceiptFactDefinitionID = nil
				document.ControlAttempts[0].Code = stringPointer("switch.control.skipped")
				document.OperationalStatus = "incomplete"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "attempted control missing receipt",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].ReceiptFactDefinitionID = nil
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "accepted control has code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Code = stringPointer("switch.control.unexpected")
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "rejected control missing code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = "rejected"
				document.OperationalStatus = "failed"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "duplicate control receipt",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				second := document.ControlAttempts[0]
				second.OccurrenceDefinitionID = "switch.occurrence.second"
				document.ControlAttempts = append(document.ControlAttempts, second)
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "null source closures",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.SourceClosures = nil
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "source closure order",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.SourceClosures[0], document.SourceClosures[1] =
					document.SourceClosures[1], document.SourceClosures[0]
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "duplicate source closure",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.SourceClosures = append(document.SourceClosures, document.SourceClosures[3])
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "unknown source status",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.SourceClosures[0].Status = "unknown"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "complete cleanup has open handle",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.Cleanup.OpenHandleCount = "1"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "complete cleanup has code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.Cleanup.Code = stringPointer("switch.cleanup.unexpected")
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "incomplete cleanup missing code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.Cleanup.Status = "incomplete"
				document.OperationalStatus = "incomplete"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "failed cleanup missing code",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.Cleanup.Status = "failed"
				document.OperationalStatus = "failed"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "unknown cleanup status",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.Cleanup.Status = "unknown"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "missing staged Limit",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.Limits = document.Limits[:4]
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "cross-staged Limit",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.Limits[0].Phase = "realization"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "invalid Known Gap",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.KnownGaps = []artifactv2.KnownGap{{Kind: "free-form", Code: "switch.gap.invalid"}}
				document.OperationalStatus = "incomplete"
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "null Known Gaps",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.KnownGaps = nil
			}),
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "provenance checksum drift",
			encoded: replaceExperimentV2Once(t, canonical,
				"sha256:b879d5eba0c02a60c52e59a009c79f953310a6c49e3453ea863fddcbb07a75a9",
				"sha256:c879d5eba0c02a60c52e59a009c79f953310a6c49e3453ea863fddcbb07a75a9"),
			code: artifact.ErrorProvenanceChecksum,
		},
		{
			name: "artifact checksum drift",
			encoded: replaceExperimentV2Once(t, canonical,
				"sha256:f1e9bce053d7ab53f9e9259187395456dc026934a317144785b1dcbe7475868e",
				"sha256:e1e9bce053d7ab53f9e9259187395456dc026934a317144785b1dcbe7475868e"),
			code: artifact.ErrorArtifactChecksum,
		},
	}

	for _, status := range []struct {
		name              string
		code              *string
		operationalStatus string
	}{
		{name: "succeeded", operationalStatus: "succeeded"},
		{name: "failed", code: stringPointer("switch.phase.failed"), operationalStatus: "failed"},
		{name: "timed-out", code: stringPointer("switch.phase.timed-out"), operationalStatus: "incomplete"},
		{name: "canceled", code: stringPointer("switch.phase.canceled"), operationalStatus: "incomplete"},
	} {
		for _, timestamp := range []struct {
			name   string
			mutate func(*artifactv2.PhaseOutcome)
		}{
			{name: "start", mutate: func(outcome *artifactv2.PhaseOutcome) {
				outcome.StartedAtUnixMillis = nil
			}},
			{name: "finish", mutate: func(outcome *artifactv2.PhaseOutcome) {
				outcome.FinishedAtUnixMillis = nil
			}},
		} {
			status := status
			timestamp := timestamp
			cases = append(cases, struct {
				name    string
				encoded []byte
				code    artifact.ErrorCode
			}{
				name: status.name + " phase missing " + timestamp.name + " timestamp",
				encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
					document.PhaseOutcomes[1].Status = status.name
					document.PhaseOutcomes[1].Code = status.code
					document.OperationalStatus = status.operationalStatus
					timestamp.mutate(&document.PhaseOutcomes[1])
				}),
				code: artifact.ErrorMalformedValue,
			})
		}
	}

	for _, status := range []struct {
		name              string
		code              *string
		operationalStatus string
	}{
		{name: "accepted", operationalStatus: "succeeded"},
		{name: "rejected", code: stringPointer("switch.control.rejected"), operationalStatus: "failed"},
		{name: "unsupported", code: stringPointer("switch.control.unsupported"), operationalStatus: "failed"},
		{name: "failed", code: stringPointer("switch.control.failed"), operationalStatus: "failed"},
		{name: "canceled", code: stringPointer("switch.control.canceled"), operationalStatus: "incomplete"},
	} {
		status := status
		cases = append(cases, struct {
			name    string
			encoded []byte
			code    artifact.ErrorCode
		}{
			name: status.name + " control missing receipt",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				document.ControlAttempts[0].Status = status.name
				document.ControlAttempts[0].Code = status.code
				document.ControlAttempts[0].ReceiptFactDefinitionID = nil
				document.OperationalStatus = status.operationalStatus
			}),
			code: artifact.ErrorMalformedValue,
		})
		if status.code != nil {
			cases = append(cases, struct {
				name    string
				encoded []byte
				code    artifact.ErrorCode
			}{
				name: status.name + " control missing code",
				encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
					document.ControlAttempts[0].Status = status.name
					document.ControlAttempts[0].Code = nil
					document.OperationalStatus = status.operationalStatus
				}),
				code: artifact.ErrorMalformedValue,
			})
		}
	}

	cases = append(cases,
		struct {
			name    string
			encoded []byte
			code    artifact.ErrorCode
		}{
			name: "duplicate control occurrence attempt",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				duplicate := document.ControlAttempts[0]
				duplicate.ReceiptFactDefinitionID = stringPointer("switch.evidence.control-receipt.2")
				document.ControlAttempts = append(document.ControlAttempts, duplicate)
			}),
			code: artifact.ErrorMalformedValue,
		},
		struct {
			name    string
			encoded []byte
			code    artifact.ErrorCode
		}{
			name: "control attempt order",
			encoded: resealedExperimentRunV2Mutation(t, func(document *artifactv2.ExperimentRun) {
				first := document.ControlAttempts[0]
				first.OccurrenceDefinitionID = "switch.occurrence.a"
				first.ReceiptFactDefinitionID = stringPointer("switch.evidence.control-receipt.2")
				document.ControlAttempts = append(document.ControlAttempts, first)
			}),
			code: artifact.ErrorMalformedValue,
		},
	)

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := artifact.DecodeExperimentRunV2(test.encoded)
			requireRuntimeV2ErrorCode(t, err, test.code)
		})
	}
}

func naturalPointer(value artifactv2.Natural) *artifactv2.Natural {
	return &value
}

func stringPointer(value string) *string {
	return &value
}

func runtimeConfigurationV2Fixture(t *testing.T) artifactv2.RuntimeConfiguration {
	t.Helper()
	document, err := artifact.DecodeRuntimeConfigurationV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/RuntimeConfigurationV2.json"))
	require.NoError(t, err)
	return document
}

func experimentRunV2Fixture(t *testing.T) artifactv2.ExperimentRun {
	t.Helper()
	document, err := artifact.DecodeExperimentRunV2(readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/ExperimentRunV2.json"))
	require.NoError(t, err)
	return document
}

func resealedRuntimeConfigurationV2Mutation(
	t *testing.T,
	mutate func(*artifactv2.RuntimeConfiguration),
) []byte {
	t.Helper()
	document := runtimeConfigurationV2Fixture(t)
	mutate(&document)
	document, err := artifactv2.SealRuntimeConfiguration(document)
	require.NoError(t, err)
	encoded, err := artifactv2.CanonicalRuntimeConfigurationBytes(document)
	require.NoError(t, err)
	return encoded
}

func resealedExperimentRunV2Mutation(t *testing.T, mutate func(*artifactv2.ExperimentRun)) []byte {
	t.Helper()
	document := experimentRunV2Fixture(t)
	mutate(&document)
	document, err := artifactv2.SealExperimentRun(document)
	require.NoError(t, err)
	encoded, err := artifactv2.CanonicalExperimentRunBytes(document)
	require.NoError(t, err)
	return encoded
}

func requireRuntimeV2ErrorCode(t *testing.T, err error, expected artifact.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	code, ok := artifact.CodeOf(err)
	require.True(t, ok, err)
	require.Equal(t, expected, code)
}
