package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/canary"
	"go.temporal.io/server/tests/umpire3/profile"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestDecodeRequestIsStrictAndBounded(t *testing.T) {
	experiment := loadCanaryExperiment(t)
	request := canary.WorkerRequest{
		FormatVersion: canary.FormatVersion, Operation: canary.OperationCleanup, Experiment: experiment,
		Profile:  profile.Definition{Endpoint: "https://temporal.example", Namespace: "namespace", TaskQueue: "queue"},
		Approval: canary.Approval{Identifier: "approval"},
	}
	encoded, err := json.Marshal(request)
	require.NoError(t, err)
	decoded, err := decodeRequest(bytes.NewReader(encoded))
	require.NoError(t, err)
	require.Equal(t, request.Operation, decoded.Operation)

	unknown := append(encoded[:len(encoded)-1], []byte(`,"unknown":true}`)...)
	_, err = decodeRequest(bytes.NewReader(unknown))
	require.ErrorContains(t, err, "unknown field")

	_, err = decodeRequest(strings.NewReader(strings.Repeat("x", maxRequestBytes+1)))
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

func loadCanaryExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	input, err := os.Open("../../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	defer func() { require.NoError(t, input.Close()) }()
	experiment, err := protocol.DecodeExperiment(input, protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}
