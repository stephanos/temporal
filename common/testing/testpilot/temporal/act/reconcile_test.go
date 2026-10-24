package act

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/await"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	"umpire.local/model/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func cancelWithRead(t *testing.T, timeout time.Duration) *temporalv1.TemporalInstruction {
	read := new(temporalv1.ObserveActivity)
	require.NoError(t, describeInstruction(t, timeout).UnmarshalTo(read))
	return &temporalv1.TemporalInstruction{Request: packed(t, cancelRequest), NotFoundRead: packed(t, read)}
}

func TestCancellationReconcilesActualHistoryBeforeJudgingNotFound(t *testing.T) {
	server := newFakeServer(t, func(method string) ([]byte, error) {
		if method == workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName {
			return nil, status.Error(codes.NotFound, "already closed")
		}
		return describeResponse(t), nil
	})
	d := testDriver(t, server.address, Options{})
	x := start(t, d, 1, packed(t, cancelWithRead(t, time.Second)))
	read := next(t, x)
	require.Equal(t, runner.EventObservation, read.Kind)
	compound := new(temporalv1.ReconciledObservation)
	require.NoError(t, read.Payload.UnmarshalTo(compound))
	current := new(realizationv1.Observation)
	require.NoError(t, compound.GetCurrent().UnmarshalTo(current))
	require.Equal(t, workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName, current.Method)
	previous := new(realizationv1.Observation)
	require.NoError(t, compound.GetAcknowledged().UnmarshalTo(previous))
	require.Equal(t, int32(codes.NotFound), previous.GetError().Code)
	require.Len(t, server.calls(workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName), 1)
	require.NoError(t, x.Respond(t.Context(), read.ID, explained(true)))
	ack := next(t, x)
	actual := observation(t, ack)
	require.Equal(t, workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName, actual.Method)
	require.Equal(t, int32(codes.NotFound), actual.GetError().Code)
	require.NoError(t, x.Respond(t.Context(), ack.ID, explained(false)))
	require.Equal(t, runner.EventDone, next(t, x).Kind)
}

func TestReconciliationRetainsAcknowledgmentOnTimeoutOrClose(t *testing.T) {
	for _, closeEarly := range []bool{false, true} {
		t.Run(map[bool]string{false: "timeout", true: "close"}[closeEarly], func(t *testing.T) {
			release := make(chan struct{})
			defer close(release)
			server := newFakeServer(t, func(method string) ([]byte, error) {
				if method == workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName {
					return nil, status.Error(codes.NotFound, "already closed")
				}
				<-release
				return describeResponse(t), nil
			})
			d := testDriver(t, server.address, Options{})
			x := start(t, d, 1, packed(t, cancelWithRead(t, 100*time.Millisecond)))
			var evidence interface{ GetTypeUrl() string }
			if closeEarly {
				await.RequireTrue(t, func() bool {
					return len(server.calls(workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName)) > 0
				}, time.Second, time.Millisecond)
				err := x.Close()
				require.Error(t, err)
				retained, ok := err.(interface{ Evidence() *anypb.Any })
				require.True(t, ok)
				evidence = retained.Evidence()
			} else {
				fault := next(t, x)
				require.Equal(t, runner.EventFault, fault.Kind)
				require.Contains(t, fault.Fault, "read timeout")
				evidence = fault.Payload
			}
			actual := new(realizationv1.Observation)
			packed, ok := evidence.(*anypb.Any)
			require.True(t, ok)
			require.NoError(t, packed.UnmarshalTo(actual))
			require.Equal(t, workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName, actual.Method)
			require.Equal(t, int32(codes.NotFound), actual.GetError().Code)
			require.Equal(t, "already closed", actual.GetError().Message)
		})
	}
}
