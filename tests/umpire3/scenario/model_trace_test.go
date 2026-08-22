package scenario

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/checker/finite"
	checkertrace "go.temporal.io/server/tests/umpire3/checker/trace"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestSemanticTraceScenarioPreservesObservedOutcomes(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/generated/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	view, found, err := finite.DefaultAttemptExecutionView(experiment)
	require.NoError(t, err)
	require.True(t, found)
	attempts := make([]finite.ObservedAttempt, len(experiment.Actions))
	for index, action := range experiment.Actions {
		attempts[index] = finite.ObservedAttempt{
			Action: protocolcatalog.ActionKind(action.Kind), Outcome: protocolexperiment.ActionOutcomeApplied,
		}
	}
	trace, err := checkertrace.NewLive(experiment, view, attempts)
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
		require.Equal(t, []protocolexperiment.ActionOutcome{protocolexperiment.ActionOutcomeApplied}, action.AllowedOutcomes)
	}
}
