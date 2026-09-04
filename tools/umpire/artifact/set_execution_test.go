package artifact

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestExecutableSetAdmitExecutionReusesExactInputBytes(t *testing.T) {
	runBytes := readExecutionFixture(t, "experiment-run-v2.json")
	rawEvidenceBytes := readExecutionFixture(t, "raw-evidence-v2.json")
	members := []SetMember{
		{Path: artifactSetPaths[0], Encoded: readExecutionFixture(t, "switch-experiment-v2.json")},
		{Path: artifactSetPaths[1], Encoded: readExecutionFixture(t, "runtime-configuration-v2.json")},
	}
	original := cloneSetMembers(members)
	admitted, err := AdmitSet(members)
	require.NoError(t, err)
	executable, ok := admitted.Executable()
	require.True(t, ok)

	run, err := DecodeExperimentRunV2(runBytes)
	require.NoError(t, err)
	rawEvidence, err := DecodeRawEvidenceV2(rawEvidenceBytes)
	require.NoError(t, err)
	execution, err := executable.AdmitExecution(run, rawEvidence)
	require.NoError(t, err)
	executionProjection, ok := execution.Execution()
	require.True(t, ok)

	run.PhaseOutcomes[0].Phase = "mutated"
	*run.PhaseOutcomes[0].StartedAtUnixMillis = "0"
	rawEvidence.Sources[0].Status = "mutated"
	rawEvidence.Facts[3].CausalFactDefinitionIDs[0] = "mutated"
	rawEvidence.Facts[1].Fields[0].FieldDefinitionID = "mutated"

	retainedRunBytes, err := EncodeExperimentRunV2(executionProjection.ExperimentRun())
	require.NoError(t, err)
	require.Equal(t, runBytes, retainedRunBytes)
	retainedRawEvidenceBytes, err := EncodeRawEvidenceV2(executionProjection.RawEvidence())
	require.NoError(t, err)
	require.Equal(t, rawEvidenceBytes, retainedRawEvidenceBytes)

	returnedRun := executionProjection.ExperimentRun()
	returnedRawEvidence := executionProjection.RawEvidence()
	returnedRun.PhaseOutcomes[0].Phase = "mutated"
	*returnedRun.PhaseOutcomes[0].StartedAtUnixMillis = "0"
	returnedRawEvidence.Sources[0].Status = "mutated"
	returnedRawEvidence.Facts[3].CausalFactDefinitionIDs[0] = "mutated"
	returnedRawEvidence.Facts[1].Fields[0].FieldDefinitionID = "mutated"

	againRunBytes, err := EncodeExperimentRunV2(executionProjection.ExperimentRun())
	require.NoError(t, err)
	require.Equal(t, runBytes, againRunBytes)
	againRawEvidence := executionProjection.RawEvidence()
	againRawEvidenceBytes, err := EncodeRawEvidenceV2(againRawEvidence)
	require.NoError(t, err)
	require.Equal(t, rawEvidenceBytes, againRawEvidenceBytes)
	requireRawEvidenceScalarDomain(t, againRawEvidence)

	require.Len(t, execution.members, 4)
	require.Equal(t, []string{
		execution.members[0].Path,
		execution.members[1].Path,
		execution.members[2].Path,
		execution.members[3].Path,
	}, artifactSetPaths[:4])
	require.True(t, bytes.Equal(original[0].Encoded, execution.members[0].Encoded))
	require.True(t, bytes.Equal(original[1].Encoded, execution.members[1].Encoded))
	require.True(t, bytes.Equal(original[0].Encoded, admitted.members[0].Encoded))
	require.True(t, bytes.Equal(original[1].Encoded, admitted.members[1].Encoded))
	require.Nil(t, execution.executable)
}

func requireRawEvidenceScalarDomain(t *testing.T, rawEvidence artifactv2.RawEvidence) {
	t.Helper()

	var sawNil, sawBoolean, sawNumber, sawString bool
	for _, fact := range rawEvidence.Facts {
		for _, field := range fact.Fields {
			switch field.Value.(type) {
			case nil:
				sawNil = true
			case bool:
				sawBoolean = true
			case json.Number:
				sawNumber = true
			case string:
				sawString = true
			default:
				require.Failf(t, "unexpected Raw Evidence field value", "%T", field.Value)
			}
		}
	}
	require.True(t, sawNil)
	require.True(t, sawBoolean)
	require.True(t, sawNumber)
	require.True(t, sawString)
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
