package nexus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/regress"
)

func TestCancellationRegressionCompilesThroughTypedFacade(t *testing.T) {
	t.Parallel()

	operation := Operation("operation")
	scenario := Regression("nexus-cancellation", operation,
		regress.OnePath(operation.CancelWithRetry(), operation.CancellationSafety()))

	suite, err := compiler.Compile(context.Background(), scenario, compiler.Limits{
		MaxPaths: 1, MaxActions: 12, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 1)
	require.Len(t, suite.Explain.Identities, 1)
}
