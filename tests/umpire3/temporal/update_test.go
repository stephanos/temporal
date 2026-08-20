package temporal

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

func TestUpdateExperimentUsesSameRuntimeSeam(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	factory := NewUpdateFactory(func(context.Context) (ClusterInfo, error) {
		return ClusterInfo{
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
