package temporal

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/scenario"
	"go.temporal.io/server/tests/umpire3/scenario/workflow"
	"go.temporal.io/server/tests/umpire3/umpire3test"
)

func TestUpdateExperimentUsesSameRuntimeSeam(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	factory := newUpdateFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{
			BuildID: "build", Namespace: "namespace", MintedWorkflowID: "workflow", MintedUpdateID: "update",
		}, nil
	})

	result, err := umpire3runtime.Run(context.Background(), umpire3runtime.Request{
		Experiment: experiment, Environment: factory,
	})
	require.NoError(t, err)
	require.Equal(t, umpire3runtime.ClaimConforming, result.Claim.Kind)
	require.Equal(t, "workflow", result.Bindings["workflow"])
	require.Equal(t, "update", result.Bindings["update"])
}

func TestTypedUpdateRegressionFacadeRuns(t *testing.T) {
	update := workflow.Update("update")
	authored := workflow.Regression("typed-update-lifecycle", update,
		scenario.OnePath(update.Lifecycle(), update.CompletionThroughHistory()))
	factory := newUpdateFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{
			BuildID: "build", Namespace: "namespace", MintedWorkflowID: "workflow", MintedUpdateID: "update",
		}, nil
	})

	umpire3test.RequireRegression(t, authored, umpire3test.WithEnvironment(factory))
}
