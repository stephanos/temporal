package command

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/qualification"
	umpire3temporal "go.temporal.io/server/tests/umpire3/temporal"
)

func TestExplainUsesStableDiagnosticData(t *testing.T) {
	result, err := execute(context.Background(), []string{
		"explain", "-experiment", "../../testdata/update-lifecycle.json",
	})
	require.NoError(t, err)
	value, ok := result.(explanation)
	require.True(t, ok)
	require.Equal(t, "workflow-update-lifecycle-v1", value.ExperimentID)
	require.Equal(t, "workflow-update.accepted-completes-through-history", value.Property)
	require.Contains(t, value.RequiredCapabilities, "update")
	require.NotEmpty(t, value.ExperimentDigest)
}

func TestCommandsFailBeforeAllocationWhenRequiredInputsAreMissing(t *testing.T) {
	for _, command := range []string{"run", "replay", "campaign", "qualify"} {
		_, err := execute(context.Background(), []string{command})
		require.Error(t, err, command)
	}
}

func TestUnknownCommandListsStableSurface(t *testing.T) {
	_, err := execute(context.Background(), []string{"unknown"})
	require.EqualError(t, err, `unknown command "unknown": expected explain, run, replay, campaign, or qualify`)
}

func TestRunDispatchesThroughInjectedBackend(t *testing.T) {
	backend := &recordingBackend{}
	value, err := Execute(context.Background(), []string{
		"run", "-experiment", "../../testdata/update-lifecycle.json",
	}, backend)
	require.NoError(t, err)
	require.Equal(t, 1, backend.executions)
	_, ok := value.(execution.Result)
	require.True(t, ok)
}

func TestSubcommandFlagsRemainStable(t *testing.T) {
	connection := []string{
		"-address", "localhost:7233", "-namespace", "namespace", "-task-queue", "queue",
		"-build-id", "build", "-profile", "remote-deployment", "-nexus-endpoint", "endpoint",
		"-nexus-service", "service", "-nexus-operation", "operation", "-timeout", "1s",
	}
	tests := []struct {
		name      string
		arguments []string
		error     string
	}{
		{name: "explain", arguments: []string{"explain", "-experiment", ""}, error: "experiment path is required"},
		{
			name: "run",
			arguments: append([]string{
				"run", "-experiment", "", "-output", "result.json", "-bundle-output", "bundle.json",
			}, connection...),
			error: "experiment path is required",
		},
		{
			name: "replay", arguments: append([]string{"replay", "-bundle", ""}, connection...),
			error: "replay bundle path is required",
		},
		{
			name: "campaign",
			arguments: append([]string{
				"campaign", "-experiment", "", "-seed", "7", "-max-candidates", "4",
			}, connection...),
			error: "experiment path is required",
		},
		{
			name: "qualify",
			arguments: []string{
				"qualify", "-release", "", "-experiment", "", "-result", "", "-profile", "remote-deployment",
			},
			error: "release path is required",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := execute(context.Background(), test.arguments)
			require.EqualError(t, err, test.error)
		})
	}
}

func TestCommandOutputEnvelopeRemainsStable(t *testing.T) {
	command := exec.Command("go", "run", "-tags", "test_dep", "../../cmd/umpire3", "explain",
		"-experiment", "../../testdata/update-lifecycle.json")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))

	var envelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(output, &envelope))
	require.ElementsMatch(t, []string{"formatVersion", "command", "status", "data"}, mapKeys(envelope))
	require.JSONEq(t, `"umpire3/diagnostic/v1"`, string(envelope["formatVersion"]))
	require.JSONEq(t, `"explain"`, string(envelope["command"]))
	require.JSONEq(t, `"ok"`, string(envelope["status"]))
	require.NotEmpty(t, envelope["data"])
}

func mapKeys(values map[string]json.RawMessage) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	return result
}

type recordingBackend struct {
	executions int
}

func (b *recordingBackend) Execute(
	context.Context,
	protocol.Experiment,
	umpire3temporal.Options,
) (execution.Result, error) {
	b.executions++
	return execution.Result{}, nil
}

func (*recordingBackend) Qualify(qualification.Request) (qualification.Receipt, error) {
	return qualification.Receipt{}, nil
}
