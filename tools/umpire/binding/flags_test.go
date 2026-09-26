package binding

import (
	"flag"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterFlagsNamesTheDeploymentAndMissingNamesWhatIsRequired(t *testing.T) {
	var deployment Deployment
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	RegisterFlags(flags, &deployment, "the Case")
	require.Equal(t, []string{"--grpc", "--http", "--namespace", "--task-queue"}, Missing(deployment))
	require.NoError(t, flags.Parse([]string{"--grpc", "a", "--http", "b", "--namespace", "c", "--create"}))
	require.Equal(t, Deployment{GRPCAddress: "a", HTTPAddress: "b", Namespace: "c", Create: true}, deployment)
	require.Equal(t, []string{"--task-queue"}, Missing(deployment))
	require.NoError(t, flags.Parse([]string{"--task-queue", "d", "--nexus-endpoint", "e", "--handler-task-queue", "f"}))
	require.Empty(t, Missing(deployment))
	require.Equal(t, "e", deployment.NexusEndpoint)
	require.Equal(t, "f", deployment.HandlerTaskQueue)
}
