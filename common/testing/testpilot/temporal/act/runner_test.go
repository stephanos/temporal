package act

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	"umpire.local/model/runner"
	"umpire.local/temporal"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
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
	server := newFakeServer(t, func(method string) ([]byte, error) {
		if method == workflowservice.WorkflowService_StartActivityExecution_FullMethodName {
			return []byte{0xff}, nil
		}
		return nil, nil
	})
	run := explore(t, server.address)
	requireFaultAfterBootstrap(t, run, "malformed evidence")
	sent := server.calls(workflowservice.WorkflowService_StartActivityExecution_FullMethodName)
	require.Equal(t, [][]byte{run.GetEvents()[1].GetProposal().GetResponse().GetRequest().GetValue()}, sent,
		"the bootstrap proposal's bytes are what the server received")
}

func TestAnUnreachableEndpointRefusesTheRun(t *testing.T) {
	requireFaultAfterBootstrap(t, explore(t, unreachable(t)), "the server did not answer")
}

// TestAHostRefusedRunRedactsItsEvidenceAndIsTerminatedByClose is the HostRefusal shape: the
// Model refuses the worker's callback, the Run is Refused, the configured fields are gone from
// every event including the refused Answer's request bytes and the World, the Driver answers
// nothing for the Model, and Close terminates the activity.
func TestAHostRefusedRunRedactsItsEvidenceAndIsTerminatedByClose(t *testing.T) {
	const runID = "3f1c2a64-5d1e-4c39-9a51-0b6f1c9e2d7a"
	server := newFakeServer(t, func(method string) ([]byte, error) {
		switch method {
		case workflowservice.WorkflowService_StartActivityExecution_FullMethodName:
			return marshal(t, &workflowservice.StartActivityExecutionResponse{RunId: runID, Started: true}), nil
		case workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName:
			return marshal(t, &workflowservice.DescribeActivityExecutionResponse{RunId: runID, Info: &activitypb.ActivityExecutionInfo{
				ActivityId: "activity", RunId: runID, Attempt: 1,
				Status: enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, RunState: enumspb.PENDING_ACTIVITY_STATE_SCHEDULED,
			}}), nil
		default:
			return nil, nil
		}
	})
	input := &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("HostRefusal")}}}
	start := &workflowservice.StartActivityExecutionRequest{
		Namespace: "namespace", ActivityId: "activity", Identity: "secret-identity",
		ActivityType: &commonpb.ActivityType{Name: "ModelSlice"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"},
		StartToCloseTimeout: durationpb.New(20 * time.Second), RetryPolicy: &commonpb.RetryPolicy{MaximumAttempts: 1}, Input: input,
	}
	taken := &workflowservice.PollActivityTaskQueueResponse{
		TaskToken: []byte("secret-token"), WorkflowNamespace: "namespace", ActivityId: "activity", ActivityRunId: runID,
		ActivityType: &commonpb.ActivityType{Name: "ModelSlice"}, Input: input, Attempt: 1,
	}
	seed := uint64(1)
	for runner.ProposalSeed(seed, 2)%2 == 0 {
		seed++
	}
	factory, err := NewFactory(Options{Address: server.address, Namespace: "namespace", TaskQueue: "queue", Redact: []string{
		"temporal.api.workflowservice.v1.PollActivityTaskQueueResponse.task_token",
		"temporal.api.workflowservice.v1.StartActivityExecutionRequest.identity",
	}})
	require.NoError(t, err)
	delivered := make(chan error, seed)
	newDriver := func(ctx context.Context) (runner.Driver, error) {
		driver, err := factory.New(ctx)
		if err != nil {
			return nil, err
		}
		d := driver.(*Driver)
		d.ensureWorker = func(context.Context) error { return nil }
		go func() {
			if poll(context.Background(), d, taken) == nil {
				_, err := d.deliver(context.Background(), taken.GetTaskToken())
				delivered <- err
			}
		}()
		return &callbackDriver{Driver: d, t: t}, nil
	}
	run, err := runner.New(temporal.Guest(), newDriver).Run(t.Context(), runner.ModeExplore, runner.Options{World: packed(t, start), Seed: seed, Budget: 3})
	require.NoError(t, err)
	refused := run.GetVerdict().GetRefused()
	require.NotNil(t, refused, "verdict: %v", run.GetVerdict())
	event := run.GetEvents()[refused.GetEvent()]
	require.Equal(t, modelv1.Method_METHOD_ANSWER, event.GetRefusal().GetMethod())
	require.Equal(t, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED, event.GetRefusal().GetRefusal().GetCode())
	require.True(t, event.GetRedacted())
	request := new(modelv1.AnswerRequest)
	require.NoError(t, proto.Unmarshal(event.GetRefusal().GetRequest(), request), "the refused request stays decodable")
	require.NotNil(t, request.GetCallback())
	recorded, err := proto.Marshal(run)
	require.NoError(t, err)
	require.NotContains(t, string(recorded), "secret-token", "the refused Answer keeps the task token")
	require.NotContains(t, string(recorded), "secret-identity", "the World keeps its identity")
	require.ErrorIs(t, <-delivered, errReleased, "the Driver answered the refused task itself")
	terminations := server.calls(workflowservice.WorkflowService_TerminateActivityExecution_FullMethodName)
	require.NotEmpty(t, terminations, "Close terminated the activity")
	termination := new(workflowservice.TerminateActivityExecutionRequest)
	require.NoError(t, proto.Unmarshal(terminations[len(terminations)-1], termination))
	require.Equal(t, runID, termination.GetRunId())
}

type callbackDriver struct {
	*Driver
	t *testing.T
}

func (d *callbackDriver) Start(ctx context.Context, request uint64, payload *anypb.Any) (runner.Exchange, error) {
	if payload.MessageIs(&temporalv1.ObserveActivity{}) {
		return workerExchange(d.t, d.Driver, request, 30*time.Second), nil
	}
	return d.Driver.Start(ctx, request, payload)
}
