package act

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	modelv1 "umpire.local/model/umpire/model/v1"
	"umpire.local/model/umpire/runner"
	"umpire.local/temporal"
)

// world is a Start request the Temporal Model admits.
func world(t *testing.T) *anypb.Any {
	t.Helper()
	return packed(t, &workflowservice.StartActivityExecutionRequest{
		Namespace: "namespace", ActivityId: "activity",
		ActivityType: &commonpb.ActivityType{Name: "ModelSlice"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"},
		StartToCloseTimeout: durationpb.New(20 * time.Second),
		RetryPolicy:         &commonpb.RetryPolicy{MaximumAttempts: 1},
		Input: &commonpb.Payloads{Payloads: []*commonpb.Payload{{
			Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Complete"),
		}}},
	})
}

func explore(t *testing.T, address string) *runner.Run {
	t.Helper()
	factory, err := NewFactory(Options{Address: address, Namespace: "namespace", TaskQueue: "queue"})
	require.NoError(t, err)
	run, err := runner.New(temporal.Guest(), factory.New).Run(t.Context(), runner.ModeExplore, runner.Options{World: world(t), Seed: 1})
	require.NoError(t, err)
	return run
}

// requireFaultAfterBootstrap checks the Run ended at a Driver fault right after the bootstrap
// proposal: nothing reached Judge.
func requireFaultAfterBootstrap(t *testing.T, run *runner.Run, reason string) {
	t.Helper()
	events := run.GetEvents()
	require.Len(t, events, 3, "events: %v", events)
	require.NotNil(t, events[0].GetReset_())
	require.Equal(t, modelv1.ProposalKind_PROPOSAL_KIND_BOOTSTRAP, events[1].GetProposal().GetResponse().GetKind())
	require.Contains(t, events[2].GetDriverFault().GetReason(), reason)
	require.Equal(t, uint64(2), run.GetVerdict().GetRefused().GetEvent())
}

func TestMalformedStartEvidenceRefusesTheRunBeforeJudge(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return []byte{0xff}, nil })
	run := explore(t, server.address)
	requireFaultAfterBootstrap(t, run, "malformed evidence")
	sent := server.calls(workflowservice.WorkflowService_StartActivityExecution_FullMethodName)
	require.Equal(t, [][]byte{run.GetEvents()[1].GetProposal().GetResponse().GetRequest().GetValue()}, sent,
		"the bootstrap proposal's bytes are what the server received")
}

func TestAnUnreachableEndpointRefusesTheRun(t *testing.T) {
	requireFaultAfterBootstrap(t, explore(t, unreachable(t)), "the server did not answer")
}
