package scenario_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/scenario"
	"go.temporal.io/server/tests/umpire3/scenario/nexus"
)

func TestPublicFacadeCompilesArgumentsAlternativesAndScopedFaultsWithoutProtocol(t *testing.T) {
	t.Parallel()

	operation := nexus.Operation("operation")
	authored := nexus.Regression("public-facade", operation, scenario.OnePath(
		operation.Schedule(),
		operation.Dispatch(),
		operation.RequestCancellation(scenario.WithReason("retry cancellation")),
		operation.CommitCancellation(),
		operation.AcquireOwnership(),
		scenario.During(
			scenario.Drop("drop-retry",
				scenario.OnServices("nexus"),
				scenario.OnRoutes("/service/operation"),
				scenario.AtOccurrence(2, 1),
			),
			scenario.OnePath(
				operation.Retry(),
				operation.WorkerReturnsSuccess(),
				operation.PersistSuccess(scenario.Outcomes(scenario.Applied, scenario.Suppressed)),
			),
		),
		operation.CancellationSafety(),
	))

	suite, err := scenario.Compile(context.Background(), authored, scenario.Limits{
		MaxPaths: 1, MaxActions: 16, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 1)
	experiment := suite.Experiments[0]
	require.Equal(t, "retry cancellation", *experiment.Actions[2].Arguments[0].Value.Text)
	require.Equal(t, []string{"applied", "suppressed"}, []string{
		string(experiment.Actions[7].AllowedOutcomes[0]),
		string(experiment.Actions[7].AllowedOutcomes[1]),
	})
	require.Equal(t, []string{"nexus"}, experiment.Faults[0].Scope.Services)
	require.Equal(t, []string{"/service/operation"}, experiment.Faults[0].Scope.Routes)
	require.Equal(t, 2, experiment.Faults[0].Occurrence.First)
	require.Equal(t, 1, experiment.Faults[0].Occurrence.Count)
}

func TestPublicFacadeCompilesTypedNexusCompletionModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		term scenario.Term
		want scenario.NexusCompletionMode
	}{
		{
			name: "ordinary",
			term: scenario.ScheduleOperation("schedule",
				scenario.WithNexusCompletion(scenario.NexusCompletionOrdinary)),
			want: scenario.NexusCompletionOrdinary,
		},
		{
			name: "completion before start",
			term: scenario.WorkerReturnsSuccess("complete",
				scenario.WithNexusCompletion(scenario.NexusCompletionBeforeStart)),
			want: scenario.NexusCompletionBeforeStart,
		},
		{
			name: "failed",
			term: scenario.CloseNexusOperation("close",
				scenario.WithNexusCompletion(scenario.NexusCompletionFailed)),
			want: scenario.NexusCompletionFailed,
		},
		{
			name: "open at caller close",
			term: scenario.CloseNexusOperation("close",
				scenario.WithNexusCompletion(scenario.NexusCompletionOpenAtCallerClose)),
			want: scenario.NexusCompletionOpenAtCallerClose,
		},
		{
			name: "retry then success",
			term: scenario.CloseNexusOperation("close",
				scenario.WithNexusCompletion(scenario.NexusCompletionRetryThenSuccess)),
			want: scenario.NexusCompletionRetryThenSuccess,
		},
		{
			name: "retry stuck",
			term: scenario.CloseNexusOperation("close",
				scenario.WithNexusCompletion(scenario.NexusCompletionRetryStuck)),
			want: scenario.NexusCompletionRetryStuck,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authored := scenario.FeatureNexusRegression("typed-nexus-completion", []scenario.Resource{
				scenario.NexusOperation("operation"), scenario.Workflow("workflow"),
			}, scenario.OnePath(test.term, scenario.RequireNexusOperationClosure()))
			suite, err := scenario.Compile(context.Background(), authored, scenario.Limits{
				MaxPaths: 1, MaxActions: 4, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
			})
			require.NoError(t, err)
			require.Len(t, suite.Experiments, 1)
			var argumentNames []string
			var argumentValues []string
			for _, action := range suite.Experiments[0].Actions {
				for _, argument := range action.Arguments {
					if argument.Name == "nexus-completion" && argument.Value.Text != nil {
						argumentNames = append(argumentNames, argument.Name)
						argumentValues = append(argumentValues, *argument.Value.Text)
					}
				}
			}
			require.Equal(t, []string{"nexus-completion"}, argumentNames)
			require.Equal(t, []string{string(test.want)}, argumentValues)
		})
	}
}

func TestPublicFacadeRejectsUnsupportedNexusCompletionMode(t *testing.T) {
	t.Parallel()

	authored := scenario.FeatureNexusRegression("unsupported-nexus-completion", []scenario.Resource{
		scenario.NexusOperation("operation"), scenario.Workflow("workflow"),
	}, scenario.OnePath(
		scenario.CloseNexusOperation("close",
			scenario.WithNexusCompletion(scenario.NexusCompletionMode("invented"))),
		scenario.RequireNexusOperationClosure(),
	))
	_, err := scenario.Compile(context.Background(), authored, scenario.Limits{
		MaxPaths: 1, MaxActions: 4, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.ErrorContains(t, err, "unsupported value")
}
