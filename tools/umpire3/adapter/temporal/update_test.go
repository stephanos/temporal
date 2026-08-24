package temporal

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	umpire3execution "go.temporal.io/server/tools/umpire3/execution"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
	"go.temporal.io/server/tools/umpire3/regression"
	"go.temporal.io/server/tools/umpire3/scenario"
	"go.temporal.io/server/tools/umpire3/scenario/workflow"
)

func TestUpdateExperimentUsesSameRuntimeSeam(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	factory := newUpdateFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{
			BuildID: "build", Namespace: "namespace", MintedWorkflowID: "workflow", MintedUpdateID: "update",
		}, nil
	})

	result, err := umpire3execution.Run(context.Background(), umpire3execution.Request{
		Experiment: experiment, Environment: factory,
	})
	require.NoError(t, err)
	require.Equal(t, umpire3execution.ClaimConforming, result.Claim.Kind)
	require.Equal(t, "workflow", result.Bindings["workflow"])
	require.Equal(t, "update", result.Bindings["update"])
}

func TestTypedUpdateRegressionFacadeRuns(t *testing.T) {
	update := workflow.Update("update")
	authored := workflow.Scenario("typed-update-lifecycle", update,
		scenario.OnePath(update.Lifecycle(), update.CompletionThroughHistory()))
	factory := newUpdateFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{
			BuildID: "build", Namespace: "namespace", MintedWorkflowID: "workflow", MintedUpdateID: "update",
		}, nil
	})

	regression.RequireRegression(t, authored, regression.WithEnvironment(factory))
}
