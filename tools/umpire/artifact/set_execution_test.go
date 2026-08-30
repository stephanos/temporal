package artifact

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestExecutableSetAdmitExecutionReusesExactInputBytes(t *testing.T) {
	members := []SetMember{
		{Path: artifactSetPaths[0], Encoded: readExecutionFixture(t, "switch-experiment-v2.json")},
		{Path: artifactSetPaths[1], Encoded: readExecutionFixture(t, "runtime-configuration-v2.json")},
	}
	original := cloneSetMembers(members)
	admitted, err := AdmitSet(members)
	require.NoError(t, err)
	executable, ok := admitted.Executable()
	require.True(t, ok)

	run, err := DecodeExperimentRunV2(readExecutionFixture(t, "experiment-run-v2.json"))
	require.NoError(t, err)
	rawEvidence, err := DecodeRawEvidenceV2(readExecutionFixture(t, "raw-evidence-v2.json"))
	require.NoError(t, err)
	execution, err := executable.AdmitExecution(run, rawEvidence)
	require.NoError(t, err)

	require.Len(t, execution.members, 4)
	require.Equal(t, artifactSetPaths[:4], []string{
		execution.members[0].Path,
		execution.members[1].Path,
		execution.members[2].Path,
		execution.members[3].Path,
	})
	require.True(t, bytes.Equal(original[0].Encoded, execution.members[0].Encoded))
	require.True(t, bytes.Equal(original[1].Encoded, execution.members[1].Encoded))
	require.True(t, bytes.Equal(original[0].Encoded, admitted.members[0].Encoded))
	require.True(t, bytes.Equal(original[1].Encoded, admitted.members[1].Encoded))
	require.Nil(t, execution.executable)
}

func TestExecutionSetAdmitEvaluationReusesExactInputBytes(t *testing.T) {
	members := []SetMember{
		{Path: artifactSetPaths[0], Encoded: readExecutionFixture(t, "switch-experiment-v2.json")},
		{Path: artifactSetPaths[1], Encoded: readExecutionFixture(t, "runtime-configuration-v2.json")},
		{Path: artifactSetPaths[2], Encoded: readExecutionFixture(t, "experiment-run-v2.json")},
		{Path: artifactSetPaths[3], Encoded: readExecutionFixture(t, "raw-evidence-v2.json")},
	}
	original := cloneSetMembers(members)
	admitted, err := AdmitSet(members)
	require.NoError(t, err)
	execution, ok := admitted.Execution()
	require.True(t, ok)

	evidence, err := DecodeEvidenceV2(readExecutionFixture(t, "evidence-v2.json"))
	require.NoError(t, err)
	result, err := DecodeResultV2(readExecutionFixture(t, "result-v2.json"))
	require.NoError(t, err)
	evaluation, err := execution.AdmitEvaluation(evidence, result)
	require.NoError(t, err)

	require.Len(t, evaluation.members, 6)
	for index := range original {
		require.True(t, bytes.Equal(original[index].Encoded, evaluation.members[index].Encoded))
		require.True(t, bytes.Equal(original[index].Encoded, admitted.members[index].Encoded))
	}
	require.False(t, bytes.Equal(evaluation.members[4].Encoded, evaluation.members[5].Encoded))

	run := execution.ExperimentRun()
	run.RunIdentity = "mutated"
	rawEvidence := execution.RawEvidence()
	rawEvidence.Facts = nil
	require.NotEqual(t, run.RunIdentity, execution.ExperimentRun().RunIdentity)
	require.NotNil(t, execution.RawEvidence().Facts)

	_, executable := admitted.Executable()
	require.False(t, executable)
	_, completeExecution := evaluation.Execution()
	require.False(t, completeExecution)
}

func TestExecutionSetAdmitEvaluationRequiresPlanTargetAtOneLinkEndpoint(t *testing.T) {
	execution := admittedExecutionFixture(t)
	evidence, err := DecodeEvidenceV2(readExecutionFixture(t, "evidence-v2.json"))
	require.NoError(t, err)
	result, err := DecodeResultV2(readExecutionFixture(t, "result-v2.json"))
	require.NoError(t, err)

	planTarget := result.ImplementationLink.SourceTarget
	unrelatedTarget := result.ImplementationLink.DestinationTarget
	result.ImplementationLink.SourceTarget = unrelatedTarget
	result.ImplementationLink.DestinationTarget = planTarget
	result = sealExecutionFixtureResult(t, result, evidence, execution.Experiment())
	_, err = execution.AdmitEvaluation(evidence, result)
	require.NoError(t, err)

	result.ImplementationLink.DestinationTarget = unrelatedTarget
	result = sealExecutionFixtureResult(t, result, evidence, execution.Experiment())
	_, err = execution.AdmitEvaluation(evidence, result)
	require.ErrorContains(t, err, "do not match ExperimentSpec")
}

func admittedExecutionFixture(t *testing.T) ExecutionSet {
	t.Helper()
	members := []SetMember{
		{Path: artifactSetPaths[0], Encoded: readExecutionFixture(t, "switch-experiment-v2.json")},
		{Path: artifactSetPaths[1], Encoded: readExecutionFixture(t, "runtime-configuration-v2.json")},
		{Path: artifactSetPaths[2], Encoded: readExecutionFixture(t, "experiment-run-v2.json")},
		{Path: artifactSetPaths[3], Encoded: readExecutionFixture(t, "raw-evidence-v2.json")},
	}
	admitted, err := AdmitSet(members)
	require.NoError(t, err)
	execution, ok := admitted.Execution()
	require.True(t, ok)
	return execution
}

func sealExecutionFixtureResult(
	t *testing.T,
	result artifactv2.Result,
	evidence artifactv2.Evidence,
	experiment artifactv2.Experiment,
) artifactv2.Result {
	t.Helper()
	outcomeChecksum, err := artifactv2.ExpectedEvaluationOutcomeChecksum(result, evidence, experiment)
	require.NoError(t, err)
	result.EvaluationOutcomeChecksum = &outcomeChecksum
	result, err = artifactv2.SealResult(result)
	require.NoError(t, err)
	return result
}

func readExecutionFixture(t *testing.T, name string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return encoded
}
