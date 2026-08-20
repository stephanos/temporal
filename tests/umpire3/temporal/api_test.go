package temporal

import (
	"encoding/json"
	"os"
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

func TestInterpretCancelRequestCoversGeneratedInterpretationDisposition(t *testing.T) {
	encoded, err := os.ReadFile("../model/Temporal/API/Generated/field-dispositions.json")
	require.NoError(t, err)
	var report struct {
		Messages []struct {
			FullName string `json:"fullName"`
			Fields   []struct {
				Name        string `json:"name"`
				Disposition string `json:"disposition"`
			} `json:"fields"`
		} `json:"messages"`
	}
	require.NoError(t, json.Unmarshal(encoded, &report))
	var interpreted []string
	for _, message := range report.Messages {
		if message.FullName != "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest" {
			continue
		}
		for _, field := range message.Fields {
			if field.Disposition == "interpreted" {
				interpreted = append(interpreted, field.Name)
			}
		}
	}
	require.ElementsMatch(t, []string{"namespace", "operation_id", "run_id", "request_id", "reason"}, interpreted)
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
