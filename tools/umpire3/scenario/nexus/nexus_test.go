package nexus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire3/scenario"
)

func TestCancellationRegressionCompilesThroughTypedFacade(t *testing.T) {
	t.Parallel()

	operation := Operation("operation")
	authored := Scenario("nexus-cancellation", operation,
		scenario.OnePath(operation.CancelWithRetry(), operation.CancellationSafety()))

	suite, err := scenario.Compile(context.Background(), authored, scenario.Limits{
		MaxPaths: 1, MaxActions: 12, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 1)
	require.Len(t, suite.Explain.Identities, 1)
}
