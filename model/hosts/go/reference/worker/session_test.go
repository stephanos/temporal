package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	genericworker "github.com/temporalio/umpire/hosts/go/worker"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func activitySessionFixture(t *testing.T, p *Program, mutate func(*TaskIdentity)) (*bridgeFixture, *ActivitySession, TaskIdentity) {
	t.Helper()
	f := newBridgeFixture(t)
	request := &workflowservice.PollActivityTaskQueueRequest{Namespace: "ns", TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"}}
	response := &workflowservice.PollActivityTaskQueueResponse{TaskToken: []byte("session-token"), Attempt: 2, ActivityId: "job", ActivityRunId: "run", WorkflowNamespace: "ns", ActivityType: &commonpb.ActivityType{Name: "activity"}, Header: &commonpb.Header{Fields: map[string]*commonpb.Payload{ScenarioHeader: {Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("history")}}}}
	start := f.start(t.Context(), workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, request)
	f.complete(t.Context(), start, response, "")
	identity := IdentityFromPoll("ns", "queue", response)
	if mutate != nil {
		mutate(&identity)
	}
	s, err := p.NewSession(f.bridge, identity, "executor", nil)
	require.NoError(t, err)
	return f, s, identity
}

func completeActivityTicket(t *testing.T, f *bridgeFixture, s *ActivitySession, ticket *ActivityTicket, bit bool) {
	t.Helper()
	request, err := ResponseMessage(ticket.Request())
	require.NoError(t, err)
	var response proto.Message
	switch request.(type) {
	case *workflowservice.RecordActivityTaskHeartbeatRequest:
		response = &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: bit}
	case *workflowservice.RespondActivityTaskCompletedRequest:
		response = &workflowservice.RespondActivityTaskCompletedResponse{}
	case *workflowservice.RespondActivityTaskFailedRequest:
		response = &workflowservice.RespondActivityTaskFailedResponse{}
	case *workflowservice.RespondActivityTaskCanceledRequest:
		response = &workflowservice.RespondActivityTaskCanceledResponse{}
	default:
		t.Fatalf("unexpected ticket request %T", request)
	}
	ctx := s.Context(t.Context())
	start := f.start(ctx, ticket.Method(), request)
	f.complete(ctx, start, response, "")
	require.NoError(t, ticket.Wait(ctx))
}

func TestActivitySessionExactBranchAndImmutableBinding(t *testing.T) {
	for _, path := range []struct {
		name     string
		bits     []bool
		terminal string
	}{
		{name: "early", bits: []bool{true}, terminal: workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName},
		{name: "false-false", bits: []bool{false, false}, terminal: workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName},
		{name: "singleton", bits: []bool{false, true, true}, terminal: workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName},
	} {
		t.Run(path.name, func(t *testing.T) {
			p, err := Decode(programBytes(t, historyProgram(t)))
			require.NoError(t, err)
			f, s, identity := activitySessionFixture(t, p, nil)
			identity.Token[0] = 'x'
			for _, bit := range path.bits {
				ticket, err := s.Begin()
				require.NoError(t, err)
				require.Equal(t, genericworker.Continue, ticket.Role())
				request := new(workflowservice.RecordActivityTaskHeartbeatRequest)
				require.NoError(t, ticket.Request().UnmarshalTo(request))
				require.Equal(t, []byte("session-token"), request.TaskToken)
				require.Equal(t, "ns", request.Namespace)
				require.Equal(t, "executor", request.Identity)
				ticket.Request().Value = nil
				ticket.Template().Value = nil
				completeActivityTicket(t, f, s, ticket, bit)
			}
			ticket, err := s.Begin()
			require.NoError(t, err)
			require.Equal(t, path.terminal, ticket.Method())
			require.Equal(t, genericworker.Finish, ticket.Role())
			completeActivityTicket(t, f, s, ticket, false)
			_, err = s.Begin()
			require.Error(t, err)
			require.NoError(t, f.bridge.Close())
		})
	}
}

func TestActivitySessionSingletonMissCannotUseInitialFallback(t *testing.T) {
	p, err := Decode(programBytes(t, historyProgram(t)))
	require.NoError(t, err)
	f, s, _ := activitySessionFixture(t, p, nil)
	for _, bit := range []bool{false, true} {
		ticket, err := s.Begin()
		require.NoError(t, err)
		completeActivityTicket(t, f, s, ticket, bit)
	}
	ticket, err := s.Begin()
	require.NoError(t, err)
	request, err := ResponseMessage(ticket.Request())
	require.NoError(t, err)
	ctx := s.Context(t.Context())
	start := f.start(ctx, ticket.Method(), request)
	f.complete(ctx, start, &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: false}, "")
	require.ErrorContains(t, ticket.Wait(ctx), "outside admitted domain")
	_, err = s.Begin()
	require.Error(t, err)
	require.NoError(t, f.bridge.Close())
}

func TestActivitySessionInitialFallbackAndActualResponseRequired(t *testing.T) {
	p, err := Decode(programBytes(t, &temporalv1.WorkerProgram{Namespace: "ns", TaskQueue: "queue"}))
	require.NoError(t, err)
	f, s, _ := activitySessionFixture(t, p, nil)
	ticket, err := s.Begin()
	require.NoError(t, err)
	request := new(workflowservice.RespondActivityTaskFailedRequest)
	require.NoError(t, ticket.Request().UnmarshalTo(request))
	require.Equal(t, Unprogrammed, request.GetFailure().GetApplicationFailureInfo().GetType())
	ctx := s.Context(t.Context())
	start := f.start(ctx, ticket.Method(), request)
	f.complete(ctx, start, &workflowservice.RespondActivityTaskCompletedResponse{}, "")
	require.Error(t, ticket.Wait(ctx))
	_, err = s.Begin()
	require.Error(t, err)
	require.NoError(t, f.bridge.Close())
}

func TestActivitySessionNoCaptureCannotEarnObservation(t *testing.T) {
	p, err := Decode(programBytes(t, historyProgram(t)))
	require.NoError(t, err)
	f, s, _ := activitySessionFixture(t, p, nil)
	ticket, err := s.Begin()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, ticket.Wait(ctx), context.DeadlineExceeded)
	_, err = s.Begin()
	require.Error(t, err)
	require.ErrorContains(t, f.bridge.Close(), "outstanding")
}
