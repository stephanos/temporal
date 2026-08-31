package runevaluation

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestCheckSubjectRejectsCanonicalByteAndGeneratedBindingDrift(t *testing.T) {
	experimentBytes := readCallerClosureInput(t, "experiment.json")
	pinned, err := PinSubject(experimentBytes)
	require.NoError(t, err)
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
		{name: "query", mutate: func(binding *SubjectBinding) { binding.Query.DefinitionID += ".drift" }},
		{name: "property", mutate: func(binding *SubjectBinding) { binding.Properties[0].DefinitionID += ".drift" }},
		{name: "observation requirement", mutate: func(binding *SubjectBinding) { binding.ObservationRequirementDefinitionIDs[0] += ".drift" }},
		{name: "observation program", mutate: func(binding *SubjectBinding) { binding.ObservationProgram.DefinitionID += ".drift" }},
		{name: "Implementation Link", mutate: func(binding *SubjectBinding) { binding.ImplementationLinkID += ".drift" }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mutated := cloneSubjectBinding(pinned)
			testCase.mutate(&mutated)
			require.Error(t, CheckSubject(experimentBytes, mutated))
		})
	}
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
