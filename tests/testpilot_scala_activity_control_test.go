//go:build test_dep && integration

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/chasm"
	"go.temporal.io/server/common/rpc/grpcfaults"
	serviceerrors "go.temporal.io/server/common/serviceerror"
	"go.temporal.io/server/common/testing/testhooks"
	"go.temporal.io/server/common/testing/testpilot/temporal/control"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/durationpb"
)

// This qualifies the server cut independently of Case lowering. The post-handler hook observes
// the authoritative refusal, so an empty poll alone cannot make this test pass.
func TestTestpilotActivityDeliveryCut(t *testing.T) {
	env := scalaActivityEnvironment(t)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	id := uuid.NewString()
	queue := &taskqueuepb.TaskQueue{Name: "held-" + id}
	gate := control.NewDeliveryGate(id)
	defer gate.Close()
	held := make(chan testhooks.ActivityDelivery, 1)
	env.InjectHook(testhooks.NewHook(testhooks.ActivityDispatch, func(ctx context.Context, delivery testhooks.ActivityDelivery) error {
		if delivery.Execution.BusinessID == id {
			select {
			case held <- delivery:
			default:
			}
		}
		return gate.Arrive(ctx, delivery.Execution.BusinessID)
	}))
	type admission struct {
		request *historyservice.RecordActivityTaskStartedRequest
		err     error
	}
	observed := make(chan admission, 1)
	env.InjectHook(testhooks.NewHook(testhooks.GRPCResponseFaultGeneratorByNamespaceID,
		grpcfaults.ResponseCallback(func(_ context.Context, _ string, request, response any, err error) *grpcfaults.Outcome {
			if req, ok := request.(*historyservice.RecordActivityTaskStartedRequest); ok {
				ref, decodeErr := chasm.DeserializeComponentRef(req.GetComponentRef())
				if decodeErr == nil && ref.BusinessID == id {
					select {
					case observed <- admission{request: req, err: err}:
					default:
					}
				}
			}
			return nil
		})))
	start, err := env.FrontendClient().StartActivityExecution(ctx, &workflowservice.StartActivityExecutionRequest{
		Namespace: env.Namespace().String(), ActivityId: id, RequestId: uuid.NewString(),
		ActivityType: &commonpb.ActivityType{Name: "held"}, TaskQueue: queue, StartToCloseTimeout: durationpb.New(time.Minute),
	})
	require.NoError(t, err)
	require.NoError(t, gate.WaitHeld(ctx))
	delivery := <-held
	require.Equal(t, chasm.ExecutionKey{NamespaceID: env.NamespaceID().String(), BusinessID: id, RunID: start.GetRunId()}, delivery.Execution)
	_, err = env.FrontendClient().PauseActivityExecution(ctx, &workflowservice.PauseActivityExecutionRequest{
		Namespace: env.Namespace().String(), ActivityId: id, RunId: start.GetRunId(), RequestId: uuid.NewString(),
	})
	require.NoError(t, err)
	require.NoError(t, gate.Release())
	pollCtx, stopPoll := context.WithCancel(ctx)
	defer stopPoll()
	type pollResult struct {
		response *workflowservice.PollActivityTaskQueueResponse
		err      error
	}
	polled := make(chan pollResult, 1)
	go func() {
		response, pollErr := env.FrontendClient().PollActivityTaskQueue(pollCtx, &workflowservice.PollActivityTaskQueueRequest{
			Namespace: env.Namespace().String(), TaskQueue: queue, Identity: "held-delivery-poller",
		})
		polled <- pollResult{response: response, err: pollErr}
	}()
	select {
	case result := <-observed:
		require.Equal(t, delivery.Stamp, result.request.GetStamp())
		require.NotEmpty(t, result.request.GetRequestId())
		var obsolete *serviceerrors.ObsoleteMatchingTask
		require.ErrorAs(t, result.err, &obsolete)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	stopPoll()
	poll := <-polled
	require.Empty(t, poll.response.GetTaskToken())
	if poll.err != nil {
		require.Equal(t, codes.Canceled, serviceerror.ToStatus(poll.err).Code())
	}
	description, err := env.FrontendClient().DescribeActivityExecution(ctx, &workflowservice.DescribeActivityExecutionRequest{
		Namespace: env.Namespace().String(), ActivityId: id, RunId: start.GetRunId(),
	})
	require.NoError(t, err)
	require.Equal(t, enumspb.ACTIVITY_EXECUTION_STATUS_PAUSED, description.GetInfo().GetStatus())
	require.Nil(t, description.GetInfo().GetLastStartedTime())
}
