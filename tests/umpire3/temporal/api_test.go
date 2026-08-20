package temporal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
)

func TestInterpretCancelRequestPreservesSemanticFields(t *testing.T) {
	command, err := InterpretCancelRequest(&workflowservice.RequestCancelNexusOperationExecutionRequest{
		Namespace:   "namespace",
		OperationId: "operation",
		RunId:       "run",
		Identity:    "transport-only-identity",
		RequestId:   "request",
		Reason:      "reason",
	})
	require.NoError(t, err)
	require.Equal(t, CancelCommand{
		Namespace:   "namespace",
		OperationID: "operation",
		RunID:       "run",
		RequestID:   "request",
		Reason:      "reason",
	}, command)
}

func FuzzInterpretCancelRequest(f *testing.F) {
	f.Add("namespace", "operation", "request")
	f.Add("", "operation", "request")
	f.Fuzz(func(t *testing.T, namespace, operation, requestID string) {
		_, err := InterpretCancelRequest(&workflowservice.RequestCancelNexusOperationExecutionRequest{
			Namespace: namespace, OperationId: operation, RequestId: requestID,
		})
		require.Equal(t, namespace != "" && operation != "" && requestID != "", err == nil)
	})
}
