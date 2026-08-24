package command

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire3/deployment"
	"go.temporal.io/server/tools/umpire3/deployment/canary"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func TestDecodeCanaryWorkerRequestIsStrictAndBounded(t *testing.T) {
	experiment := loadCanaryExperiment(t)
	request := canary.WorkerRequest{
		FormatVersion: canary.FormatVersion, Operation: canary.OperationCleanup, Experiment: experiment,
		Profile:  deployment.Profile{Endpoint: "https://temporal.example", Namespace: "namespace", TaskQueue: "queue"},
		Approval: canary.Approval{Identifier: "approval"},
	}
	encoded, err := json.Marshal(request)
	require.NoError(t, err)
	decoded, err := decodeCanaryWorkerRequest(bytes.NewReader(encoded))
	require.NoError(t, err)
	require.Equal(t, request.Operation, decoded.Operation)

	unknown := append(encoded[:len(encoded)-1], []byte(`,"unknown":true}`)...)
	_, err = decodeCanaryWorkerRequest(bytes.NewReader(unknown))
	require.ErrorContains(t, err, "unknown field")

	_, err = decodeCanaryWorkerRequest(strings.NewReader(strings.Repeat("x", maxCanaryWorkerRequestBytes+1)))
	require.EqualError(t, err, "canary worker request exceeds input budget")
}

func TestCanaryWorkflowIdentityIsSemanticAndStable(t *testing.T) {
	experiment := loadCanaryExperiment(t)
	first, err := canaryWorkflowID(experiment)
	require.NoError(t, err)
	second, err := canaryWorkflowID(experiment)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, first, len("umpire3-canary-")+32)
}

func TestCanaryWorkerOptionsEnforceApprovedConcurrencyAndRate(t *testing.T) {
	options := canaryWorkerOptions(canary.Approval{MaxConcurrent: 3, MaxRatePerSecond: 7})
	require.Equal(t, 3, options.MaxConcurrentActivityExecutionSize)
	require.Equal(t, 3, options.MaxConcurrentLocalActivityExecutionSize)
	require.Equal(t, 3, options.MaxConcurrentWorkflowTaskExecutionSize)
	require.InDelta(t, 7, options.WorkerActivitiesPerSecond, 0)
	require.InDelta(t, 7, options.TaskQueueActivitiesPerSecond, 0)
}

func loadCanaryExperiment(t *testing.T) protocolexperiment.Experiment {
	t.Helper()
	input, err := os.Open("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	defer func() { require.NoError(t, input.Close()) }()
	experiment, err := protocolexperiment.DecodeExperiment(input, protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}
