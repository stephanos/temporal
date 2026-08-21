package scenario

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestSemanticTraceScenarioPreservesObservedOutcomes(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	view, found, err := protocol.DefaultAttemptExecutionView(experiment)
	require.NoError(t, err)
	require.True(t, found)
	attempts := make([]protocol.ObservedAttempt, len(experiment.Actions))
	for index, action := range experiment.Actions {
		attempts[index] = protocol.ObservedAttempt{
			Action: protocol.ActionKind(action.Kind), Outcome: protocol.ActionOutcomeApplied,
		}
	}
	trace, err := protocol.NewLiveSemanticTrace(experiment, view, attempts)
	require.NoError(t, err)

	authored, err := FromSemanticTrace(SemanticTraceIdentifier(trace), trace)
	require.NoError(t, err)
	suite, err := Compile(context.Background(), authored, Limits{
		MaxPaths: 1, MaxActions: 16, MaxStates: 64,
		MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 1)
	require.Len(t, suite.Experiments[0].Actions, len(attempts))
	for _, action := range suite.Experiments[0].Actions {
		require.Equal(t, []protocol.ActionOutcome{protocol.ActionOutcomeApplied}, action.AllowedOutcomes)
	}
}
