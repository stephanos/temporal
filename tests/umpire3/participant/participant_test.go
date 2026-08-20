package participant

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestCompileMapsEveryProgramCommandExhaustively(t *testing.T) {
	t.Parallel()

	commands := make([]Command, len(AllCommandKinds()))
	for index, kind := range AllCommandKinds() {
		commands[index] = Command{Identifier: string(kind), Kind: kind, Response: ResponseSynchronous}
	}
	plan, err := Compile(Program{FormatVersion: FormatVersion, Identifier: "all-commands", Commands: commands})
	require.NoError(t, err)
	require.Len(t, plan.Operations, len(AllCommandKinds()))
	for index, operation := range plan.Operations {
		require.Equal(t, commands[index].Identifier, operation.CommandID)
		require.NotEmpty(t, operation.SDKOperation)
	}
}

func TestExperimentActionsCompileToParticipantProgramWithoutBespokeWorkflow(t *testing.T) {
	t.Parallel()

	file, err := os.Open("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	experiment, err := protocol.DecodeExperiment(file, protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	program, plan, err := CompileExperiment(experiment)
	require.NoError(t, err)
	require.Equal(t, experiment.ExperimentID, program.Identifier)
	require.Len(t, plan.Operations, len(experiment.Actions))
	for index, action := range experiment.Actions {
		require.Equal(t, action.Identifier, plan.Operations[index].CommandID)
	}
}

func TestExperimentResponseSemanticsCompileToParticipantProgram(t *testing.T) {
	t.Parallel()

	file, err := os.Open("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	experiment, err := protocol.DecodeExperiment(file, protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	experiment.Actions[0].ResponseMode = protocol.ResponseDeferred
	program, _, err := CompileExperiment(experiment)
	require.NoError(t, err)
	require.Equal(t, ResponseDeferred, program.Commands[0].Response)
}

func TestParticipantActionMappingCoversGeneratedCatalog(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidateActionMappings())
}

func TestCallbackActionsCompileToDedicatedSDKOperations(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		action    protocol.ActionKind
		command   CommandKind
		operation SDKOperation
	}{
		{protocol.ActionKindRegisterCallback, CommandCallbackRegister, SDKRegisterCallback},
		{protocol.ActionKindRecordCallbackResponse, CommandCallbackComplete, SDKCompleteCallback},
	} {
		kind, known := actionCommandKind(test.action)
		require.True(t, known)
		require.Equal(t, test.command, kind)
		compiled, capability, err := compileCommand(Command{
			Identifier: string(test.action), Kind: kind, SemanticAction: string(test.action), Response: ResponseSynchronous,
		})
		require.NoError(t, err)
		require.Equal(t, "callback", capability)
		require.Equal(t, test.operation, compiled.SDKOperation)
	}
}

func TestCompileRejectsMalformedAndUnsupportedPrograms(t *testing.T) {
	t.Parallel()

	_, err := Compile(Program{FormatVersion: FormatVersion, Identifier: "malformed", Commands: []Command{{
		Identifier: "missing-kind", Response: ResponseSynchronous,
	}}})
	require.ErrorContains(t, err, "unknown command")

	_, err = Compile(Program{FormatVersion: FormatVersion, Identifier: "unsupported", Commands: []Command{{
		Identifier: "command", Kind: CommandWorkflow, Response: "future-response",
	}}})
	require.ErrorContains(t, err, "unknown response")
}

func TestStartValidatesEntireProgramBeforeRunnerAllocation(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	_, err := Start(context.Background(), Program{
		FormatVersion: FormatVersion,
		Identifier:    "invalid",
		Commands: []Command{
			{Identifier: "valid", Kind: CommandWorkflow, Response: ResponseSynchronous},
			{Identifier: "invalid", Kind: "future-command", Response: ResponseSynchronous},
		},
	}, runner)
	require.Error(t, err)
	require.Zero(t, runner.startCount)
}

func TestSessionExecutesAndCleansUpIdempotently(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	session, err := Start(context.Background(), Program{
		FormatVersion: FormatVersion, Identifier: "program",
		Commands: []Command{{Identifier: "workflow", Kind: CommandWorkflow, Response: ResponseDeferred}},
	}, runner)
	require.NoError(t, err)

	result, err := session.Execute(context.Background(), "workflow")
	require.NoError(t, err)
	require.Equal(t, "workflow", result.CommandID)
	require.NoError(t, session.Cleanup(context.Background()))
	require.NoError(t, session.Cleanup(context.Background()))
	require.Equal(t, 1, runner.stopCount)
}

type fakeRunner struct {
	startCount int
	stopCount  int
}

func (r *fakeRunner) Start(context.Context, Plan) error {
	r.startCount++
	return nil
}

func (r *fakeRunner) Execute(_ context.Context, operation Operation) (Result, error) {
	return Result{CommandID: operation.CommandID, Status: "completed"}, nil
}

func (r *fakeRunner) Stop(context.Context) error {
	r.stopCount++
	return nil
}
