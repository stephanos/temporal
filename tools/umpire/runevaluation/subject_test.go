package runevaluation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestCheckSubjectProvesLocalAndCISubjectParity(t *testing.T) {
	localBytes, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "model", "Temporal", "Feature", "Nexus", "Experimental",
		"testdata", "nexus-caller-closure-experiment-spec.json",
	))
	require.NoError(t, err)
	ciBytes := readCallerClosureInput(t, "experiment.json")
	// This is deliberately byte-exact; semantic JSON equality would hide wire drift.
	//nolint:testifylint
	require.Equal(t, localBytes, ciBytes)

	expected := callerClosureSubjectGolden()
	require.Equal(t, expected.ExperimentSHA256, independentSubjectSHA256(localBytes))
	var experiment artifactv2.Experiment
	require.NoError(t, json.Unmarshal(localBytes, &experiment))
	planChecksum := experiment.Plan.ArtifactChecksum
	experiment.Plan.ArtifactChecksum = ""
	require.Equal(t, expected.DrivePlanArtifactChecksum,
		independentSubjectChecksum("umpire.drive-plan/v2", independentSubjectJSON(t, experiment.Plan)))
	experiment.Plan.ArtifactChecksum = planChecksum
	experiment.ArtifactChecksum = ""
	require.Equal(t, expected.ExperimentArtifactChecksum,
		independentSubjectChecksum("umpire.experiment-spec/v2", independentSubjectJSON(t, experiment)))
	require.Equal(t, expected.BehaviorFingerprints, []string{
		experiment.QueryBehaviorFingerprint,
		experiment.Plan.QueryBehaviorFingerprint,
		experiment.Plan.BehaviorFingerprint,
		experiment.Plan.TargetBehaviorFingerprint,
		experiment.Plan.KernelBehaviorFingerprint,
		experiment.Properties[0].BehaviorFingerprint,
	})

	uses := []struct {
		name    string
		encoded []byte
	}{
		{name: "local", encoded: localBytes},
		{name: "CI", encoded: ciBytes},
	}
	for _, use := range uses {
		t.Run(use.name, func(t *testing.T) {
			actual, err := PinSubject(use.encoded)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
			require.NoError(t, CheckSubject(use.encoded, expected))
		})
	}
}

func TestCheckSubjectRejectsIndependentArtifactAndModelMutations(t *testing.T) {
	experimentBytes := readCallerClosureInput(t, "experiment.json")
	expected := callerClosureSubjectGolden()
	semanticFingerprint := independentSubjectSHA256([]byte("recompiled-query"))
	closureFingerprint := independentSubjectSHA256([]byte("incomplete-query-closure"))
	propertyFingerprint := independentSubjectSHA256([]byte("drifted-property"))

	mutations := []struct {
		name         string
		mutate       func([]byte) []byte
		artifactCode artifact.ErrorCode
		failureClass []string
		admitted     bool
	}{
		{
			name: "canonical byte",
			mutate: func(encoded []byte) []byte {
				return bytes.Replace(encoded, []byte("{\n"), []byte("{ \n"), 1)
			},
			artifactCode: artifact.ErrorNoncanonical,
		},
		{
			name: "format version",
			mutate: func(encoded []byte) []byte {
				return replaceSubjectBytesOnce(t, encoded,
					"umpire-experiment/v2", "umpire-experiment/v3")
			},
			artifactCode: artifact.ErrorUnsupportedFormat,
		},
		{
			name: "Artifact Checksum",
			mutate: func(encoded []byte) []byte {
				return replaceSubjectBytesOnce(t, encoded, expected.ExperimentArtifactChecksum,
					"sha256:0000000000000000000000000000000000000000000000000000000000000000")
			},
			artifactCode: artifact.ErrorArtifactChecksum,
		},
		{
			name: "Behavior Fingerprint",
			mutate: func(encoded []byte) []byte {
				return independentlyResealSubject(t, encoded, func(experiment *artifactv2.Experiment) {
					experiment.Properties[0].BehaviorFingerprint = propertyFingerprint
				})
			},
			failureClass: []string{
				"subject-binding", "admission", "umpire.run-evaluation.subject.binding-drift",
			},
			admitted: true,
		},
		{
			name: "incomplete closure",
			mutate: func(encoded []byte) []byte {
				return independentlyResealSubject(t, encoded, func(experiment *artifactv2.Experiment) {
					experiment.QueryBehaviorFingerprint = closureFingerprint
				})
			},
			artifactCode: artifact.ErrorClosure,
		},
		{
			name: "generated semantic difference",
			mutate: func(encoded []byte) []byte {
				return independentlyResealSubject(t, encoded, func(experiment *artifactv2.Experiment) {
					experiment.QueryBehaviorFingerprint = semanticFingerprint
					experiment.Plan.QueryBehaviorFingerprint = semanticFingerprint
				})
			},
			failureClass: []string{
				"subject-binding", "admission", "umpire.run-evaluation.subject.binding-drift",
			},
			admitted: true,
		},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			mutated := mutation.mutate(experimentBytes)
			require.NotEqual(t, experimentBytes, mutated)
			if mutation.admitted {
				_, err := artifact.DecodeExperimentV2(mutated)
				require.NoError(t, err)
			}
			err := CheckSubject(mutated, expected)
			require.Error(t, err)
			if mutation.artifactCode != "" {
				code, ok := artifact.CodeOf(err)
				require.True(t, ok)
				require.Equal(t, mutation.artifactCode, code)
			}
			if mutation.failureClass != nil {
				requireSubjectFailureClassification(t, err, mutation.failureClass)
			}
		})
	}
}

func TestCheckSubjectRejectsCanonicalByteAndGeneratedBindingDrift(t *testing.T) {
	experimentBytes := readCallerClosureInput(t, "experiment.json")
	pinned := callerClosureSubjectGolden()
	require.NoError(t, CheckSubject(experimentBytes, pinned))

	mutatedBytes := bytes.Clone(experimentBytes)
	mutatedBytes[len(mutatedBytes)-2] ^= 1
	require.Error(t, CheckSubject(mutatedBytes, pinned))
	for _, testCase := range []struct {
		name string
		from string
		to   string
	}{
		{name: "format version", from: "umpire-experiment/v2", to: "umpire-experiment/v3"},
		{name: "artifact checksum", from: pinned.ExperimentArtifactChecksum, to: testDigest("0")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mutated := bytes.Replace(experimentBytes, []byte(testCase.from), []byte(testCase.to), 1)
			require.NotEqual(t, experimentBytes, mutated)
			require.Error(t, CheckSubject(mutated, pinned))
		})
	}
	for _, testCase := range []struct {
		name   string
		mutate func(*artifactv2.Experiment)
	}{
		{
			name: "behavior fingerprint",
			mutate: func(experiment *artifactv2.Experiment) {
				experiment.QueryBehaviorFingerprint = testDigest("1")
				experiment.Plan.QueryBehaviorFingerprint = testDigest("1")
			},
		},
		{
			name: "definition closure",
			mutate: func(experiment *artifactv2.Experiment) {
				experiment.QueryBehaviorFingerprint = testDigest("2")
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			experiment, err := artifact.DecodeExperimentV2(experimentBytes)
			require.NoError(t, err)
			testCase.mutate(&experiment)
			experiment, err = artifactv2.SealExperiment(experiment)
			require.NoError(t, err)
			mutated, err := artifactv2.CanonicalExperimentBytes(experiment)
			require.NoError(t, err)
			require.Error(t, CheckSubject(mutated, pinned))
		})
	}

	for _, testCase := range []struct {
		name   string
		mutate func(*SubjectBinding)
	}{
		{name: "ExperimentSpec digest", mutate: func(binding *SubjectBinding) { binding.ExperimentSHA256 = testDigest("1") }},
		{name: "ExperimentSpec version", mutate: func(binding *SubjectBinding) { binding.ExperimentFormatVersion = "umpire-experiment/v3" }},
		{name: "Drive Plan version", mutate: func(binding *SubjectBinding) { binding.DrivePlanFormatVersion = "umpire-drive-plan/v3" }},
		{name: "ExperimentSpec checksum", mutate: func(binding *SubjectBinding) { binding.ExperimentArtifactChecksum = testDigest("2") }},
		{name: "Drive Plan checksum", mutate: func(binding *SubjectBinding) { binding.DrivePlanArtifactChecksum = testDigest("3") }},
		{name: "definition ID", mutate: func(binding *SubjectBinding) { binding.DefinitionIDs[0] += ".drift" }},
		{name: "behavior fingerprint", mutate: func(binding *SubjectBinding) { binding.BehaviorFingerprints[0] = testDigest("4") }},
		{name: "limit", mutate: func(binding *SubjectBinding) { binding.Limits[0].Value = "2" }},
		{name: "known gap", mutate: func(binding *SubjectBinding) { binding.KnownGaps[0].Code += ".drift" }},
		{name: "query Definition ID", mutate: func(binding *SubjectBinding) { binding.Query.DefinitionID += ".drift" }},
		{name: "query Behavior Fingerprint", mutate: func(binding *SubjectBinding) { binding.Query.BehaviorFingerprint = testDigest("5") }},
		{name: "property Definition ID", mutate: func(binding *SubjectBinding) { binding.Properties[0].DefinitionID += ".drift" }},
		{name: "property Behavior Fingerprint", mutate: func(binding *SubjectBinding) { binding.Properties[0].BehaviorFingerprint = testDigest("6") }},
		{name: "observation requirement", mutate: func(binding *SubjectBinding) { binding.ObservationRequirementDefinitionIDs[0] += ".drift" }},
		{name: "observation program Definition ID", mutate: func(binding *SubjectBinding) { binding.ObservationProgram.DefinitionID += ".drift" }},
		{name: "observation program Behavior Fingerprint", mutate: func(binding *SubjectBinding) { binding.ObservationProgram.BehaviorFingerprint = testDigest("7") }},
		{name: "Implementation Link Definition ID", mutate: func(binding *SubjectBinding) { binding.ImplementationLinkID += ".drift" }},
		{name: "Implementation Link Behavior Fingerprint", mutate: func(binding *SubjectBinding) { binding.ImplementationLinkBehaviorFingerprint = testDigest("8") }},
		{name: "Implementation Link source target Definition ID", mutate: func(binding *SubjectBinding) { binding.ImplementationLinkSourceTarget.DefinitionID += ".drift" }},
		{name: "Implementation Link source target Behavior Fingerprint", mutate: func(binding *SubjectBinding) {
			binding.ImplementationLinkSourceTarget.BehaviorFingerprint = testDigest("9")
		}},
		{name: "Implementation Link destination target Definition ID", mutate: func(binding *SubjectBinding) { binding.ImplementationLinkDestinationTarget.DefinitionID += ".drift" }},
		{name: "Implementation Link destination target Behavior Fingerprint", mutate: func(binding *SubjectBinding) {
			binding.ImplementationLinkDestinationTarget.BehaviorFingerprint = testDigest("a")
		}},
		{name: "Implementation Link diagnostic", mutate: func(binding *SubjectBinding) { binding.ImplementationLinkDiagnosticPresent = true }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mutated := cloneSubjectBinding(pinned)
			testCase.mutate(&mutated)
			err := CheckSubject(experimentBytes, mutated)
			require.Error(t, err)
			requireSubjectFailureClassification(t, err, []string{
				"subject-binding", "admission", "umpire.run-evaluation.subject.binding-drift",
			})
		})
	}
}

func requireSubjectFailureClassification(t *testing.T, err error, want []string) {
	t.Helper()
	var classified interface {
		error
		Kind() string
		Phase() string
		Code() string
	}
	require.ErrorAs(t, err, &classified)
	require.Equal(t, want, []string{
		classified.Kind(), classified.Phase(), classified.Code(),
	})
}

func callerClosureSubjectGolden() SubjectBinding {
	return SubjectBinding{
		ExperimentSHA256:           "sha256:528c23e7807ee9833af65baeb32a8ec2d38ffacc1fae829600692d3d3eb93fd1",
		ExperimentFormatVersion:    "umpire-experiment/v2",
		DrivePlanFormatVersion:     "umpire-drive-plan/v2",
		ExperimentArtifactChecksum: "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8",
		DrivePlanArtifactChecksum:  "sha256:328a90c67ca91a885a31b1e146d36af09a73cba7f729eab69a6028041a8b0bb8",
		DefinitionIDs: []string{
			"workflow-nexus.query.exact-action-caller-closure",
			"workflow-nexus.behavior.exact-action",
			"workflow-nexus.target.caller-closure",
			"workflow-nexus.kernel.caller-closure",
			"workflow-nexus.role.operation",
			"workflow-nexus.state.config",
			"workflow-nexus.setup.operation-is-clash",
			"workflow-nexus.role.operation",
			"workflow-nexus.state.config",
			"workflow-nexus.state.config",
			"workflow.action.force-close",
			"nexus.outcome.cancellation-upgraded",
			"workflow-nexus.state.config",
			"workflow-nexus.occurrence.force-close",
			"workflow.action.force-close",
			"workflow-nexus.occurrence.force-close",
			"nexus.capability.cancellation",
			"workflow-nexus.capability.ownership",
			"workflow.capability.lifecycle",
			"nexus.observation.cancellation-delivered",
			"nexus.observation.pending-cancellation-count",
			"workflow-nexus.relation.owns-operation",
			"workflow-nexus.property.caller-closure",
			"nexus.capability.cancellation",
			"workflow-nexus.capability.ownership",
			"workflow.capability.lifecycle",
			"nexus.observation.cancellation-delivered",
			"nexus.observation.pending-cancellation-count",
			"workflow-nexus.relation.owns-operation",
			"workflow-nexus.behavior.exact-action",
			"workflow-nexus.kernel.caller-closure",
			"workflow-nexus.property.caller-closure",
			"workflow-nexus.query.exact-action-caller-closure",
			"workflow-nexus.target.caller-closure",
			"workflow-nexus.behavior.exact-action",
			"workflow-nexus.kernel.caller-closure",
			"workflow-nexus.property.caller-closure",
			"workflow-nexus.query.exact-action-caller-closure",
			"workflow-nexus.target.caller-closure",
		},
		BehaviorFingerprints: []string{
			"sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af",
			"sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af",
			"sha256:322893fbbe0a80ca186aa1f10268df45966bda212db37c725ea71fd75903b703",
			"sha256:22e49d60fb38ec52fd44f09549f28329d169605168dd6dc828f43941445faacd",
			"sha256:22e49d60fb38ec52fd44f09549f28329d169605168dd6dc828f43941445faacd",
			"sha256:b7a6e89d79e40dad31a7f96c281a05ca8af74996fbc2f8a6f302b379d609192f",
		},
		Limits: []SubjectLimit{
			{Path: "behavior.transitions", Value: "1", Unit: "semantic-transitions"},
			{Path: "behavior.selectedActions", Value: "1", Unit: "selected-actions"},
			{Path: "search", Value: "8", Unit: "candidate-evaluations"},
		},
		KnownGaps: []SubjectKnownGap{
			{Kind: "input", Code: "umpire.known-gap.execution-evidence"},
			{Kind: "interpretation", Code: "umpire.known-gap.artifact-migrations"},
			{Kind: "interpretation", Code: "umpire.known-gap.artifact-reading"},
			{Kind: "interpretation", Code: "umpire.known-gap.evidence-evaluation"},
			{Kind: "interpretation", Code: "umpire.known-gap.runtime-scheduler-order"},
			{Kind: "interpretation", Code: "umpire.known-gap.runtime-storage-order"},
			{Kind: "interpretation", Code: "umpire.known-gap.runtime-transport-order"},
			{Kind: "claim", Code: "umpire.known-gap.promotion"},
		},
		Query: SubjectDefinition{
			DefinitionID:        "workflow-nexus.query.exact-action-caller-closure",
			BehaviorFingerprint: "sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af",
		},
		Properties: []SubjectDefinition{
			{
				DefinitionID:        "workflow-nexus.property.caller-closure",
				BehaviorFingerprint: "sha256:b7a6e89d79e40dad31a7f96c281a05ca8af74996fbc2f8a6f302b379d609192f",
			},
		},
		ObservationRequirementDefinitionIDs: []string{
			"nexus.observation.cancellation-delivered",
			"nexus.observation.pending-cancellation-count",
			"workflow-nexus.relation.owns-operation",
		},
		ObservationProgram: SubjectDefinition{
			DefinitionID:        "temporal.nexus.observation-program.basic-lifecycle",
			BehaviorFingerprint: "sha256:1ab36fdcd2978dec901678491646ec67fe0fc1d3bd1883e599bc2c53810b3480",
		},
		ImplementationLinkID:                  "temporal.system.nexus.caller-closure.implementation-link",
		ImplementationLinkBehaviorFingerprint: "sha256:96b55d0e5a782099f66479c6ced603c08c8046b565f89435b5b2a54848aed777",
		ImplementationLinkSourceTarget: SubjectDefinition{
			DefinitionID:        "temporal.system.nexus.caller-closure.target",
			Kind:                "target",
			BehaviorFingerprint: "sha256:6729e790d336a96173ffd0ebe0b2b2d2406e6c5444596924f0c06c4ba9652bf8",
		},
		ImplementationLinkDestinationTarget: SubjectDefinition{
			DefinitionID:        "workflow-nexus.target.caller-closure",
			Kind:                "target",
			BehaviorFingerprint: "sha256:22e49d60fb38ec52fd44f09549f28329d169605168dd6dc828f43941445faacd",
		},
		ImplementationLinkDiagnosticPresent: false,
	}
}

func independentlyResealSubject(
	t *testing.T,
	encoded []byte,
	mutate func(*artifactv2.Experiment),
) []byte {
	t.Helper()
	var experiment artifactv2.Experiment
	require.NoError(t, json.Unmarshal(encoded, &experiment))
	mutate(&experiment)
	experiment.Plan.ArtifactChecksum = ""
	planPreimage := independentSubjectJSON(t, experiment.Plan)
	experiment.Plan.ArtifactChecksum = independentSubjectChecksum("umpire.drive-plan/v2", planPreimage)
	experiment.ArtifactChecksum = ""
	experimentPreimage := independentSubjectJSON(t, experiment)
	experiment.ArtifactChecksum = independentSubjectChecksum("umpire.experiment-spec/v2", experimentPreimage)
	return independentSubjectJSON(t, experiment)
}

func independentSubjectJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	return append(encoded, '\n')
}

func independentSubjectChecksum(domain string, preimage []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{'\n'})
	_, _ = hasher.Write(preimage)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

func independentSubjectSHA256(encoded []byte) string {
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func replaceSubjectBytesOnce(t *testing.T, encoded []byte, old, replacement string) []byte {
	t.Helper()
	require.Equal(t, 1, bytes.Count(encoded, []byte(old)))
	return bytes.Replace(encoded, []byte(old), []byte(replacement), 1)
}

func cloneSubjectBinding(binding SubjectBinding) SubjectBinding {
	binding.DefinitionIDs = append([]string(nil), binding.DefinitionIDs...)
	binding.BehaviorFingerprints = append([]string(nil), binding.BehaviorFingerprints...)
	binding.Limits = append([]SubjectLimit(nil), binding.Limits...)
	binding.KnownGaps = append([]SubjectKnownGap(nil), binding.KnownGaps...)
	binding.Properties = append([]SubjectDefinition(nil), binding.Properties...)
	binding.ObservationRequirementDefinitionIDs = append(
		[]string(nil), binding.ObservationRequirementDefinitionIDs...,
	)
	return binding
}
