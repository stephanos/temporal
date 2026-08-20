package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/scenario"
)

func TestUpdateRegressionCompilesThroughTypedFacade(t *testing.T) {
	t.Parallel()

	update := Update("update")
	authored := Regression("update-lifecycle", update,
		scenario.OnePath(update.Lifecycle(), update.CompletionThroughHistory()))

	suite, err := scenario.Compile(context.Background(), authored, scenario.Limits{
		MaxPaths: 1, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 1)
	require.Len(t, suite.Explain.Identities, 1)
}
