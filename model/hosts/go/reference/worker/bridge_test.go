package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type bridgeFixture struct {
	t                    *testing.T
	bridge               *Bridge
	sequence, invocation uint64
}

func newBridgeFixture(t *testing.T) *bridgeFixture {
	t.Helper()
	return &bridgeFixture{t: t, bridge: NewBridge(t.Context())}
}
func bridgePack(t *testing.T, message proto.Message) *anypb.Any {
	t.Helper()
	a, err := anypb.New(message)
	require.NoError(t, err)
	return a
}
func (f *bridgeFixture) start(ctx context.Context, method string, request proto.Message) RPC {
	f.t.Helper()
	f.sequence++
	f.invocation++
	r := RPC{Invocation: f.invocation, Sequence: f.sequence, Stage: RPCStarted, Method: method, Request: bridgePack(f.t, request)}
	require.NoError(f.t, f.bridge.Observe(ctx, r))
	return r
}
func (f *bridgeFixture) complete(ctx context.Context, start RPC, response proto.Message, transportError string) RPC {
	f.t.Helper()
	f.sequence++
	r := start
	r.Sequence, r.Stage, r.TransportError = f.sequence, RPCCompleted, transportError
	r.Sent, r.Answered = true, transportError == ""
	if response != nil {
		r.Response = bridgePack(f.t, response)
	}
	require.NoError(f.t, f.bridge.Observe(ctx, r))
	return r
}
func (f *bridgeFixture) poll(token string, attempt int32) TaskIdentity {
	f.t.Helper()
	request := &workflowservice.PollActivityTaskQueueRequest{Namespace: "bridge-ns", TaskQueue: &taskqueuepb.TaskQueue{Name: "bridge-queue"}}
	response := &workflowservice.PollActivityTaskQueueResponse{TaskToken: []byte(token), Attempt: attempt, ActivityId: "bridge-activity", ActivityRunId: "bridge-run", WorkflowNamespace: "bridge-ns", ActivityType: &commonpb.ActivityType{Name: "bridge-type"}}
	r := f.start(f.t.Context(), workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, request)
	f.complete(f.t.Context(), r, response, "")
	return IdentityFromPoll(request.GetNamespace(), request.GetTaskQueue().GetName(), response)
}

func TestBridgeRetainsAssignmentAcrossRetryAndTerminal(t *testing.T) {
	f := newBridgeFixture(t)
	identity := f.poll("first", 7)
	a, err := f.bridge.Bind(identity)
	require.NoError(t, err)
	ctx := a.HeartbeatContext(t.Context())
	for _, bit := range []bool{false, false, true, true} {
		w, err := a.BeginHeartbeat()
		require.NoError(t, err)
		request := &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: identity.Namespace, TaskToken: identity.Token}
		retry := f.start(ctx, workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, request)
		f.complete(ctx, retry, nil, "Unavailable")
		r := f.start(ctx, workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, request)
		completed := f.complete(ctx, r, &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: bit}, "")
		got, err := w.Wait(t.Context())
		require.NoError(t, err)
		require.Equal(t, completed.Invocation, got.RPC().Invocation)
		exposed := got.RPC()
		exposed.Request.Value = []byte("mutated")
		require.True(t, proto.Equal(completed.Request, got.RPC().Request))
	}
	w, err := a.BeginTerminal(workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName)
	require.NoError(t, err)
	r := f.start(t.Context(), workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName, &workflowservice.RespondActivityTaskCompletedRequest{Namespace: identity.Namespace, TaskToken: identity.Token})
	f.complete(t.Context(), r, &workflowservice.RespondActivityTaskCompletedResponse{}, "")
	_, err = w.Wait(t.Context())
	require.NoError(t, err)
	require.NoError(t, f.bridge.Close())
}

func TestBridgeRejectsCrossedAssignmentAndPublication(t *testing.T) {
	for _, mutation := range []string{"attempt", "run", "token", "namespace", "marker", "duplicate", "crossed-request", "reordered-sequence", "overlap", "missing-response", "duplicate-bind"} {
		t.Run(mutation, func(t *testing.T) {
			f := newBridgeFixture(t)
			identity := f.poll("first", 7)
			switch mutation {
			case "attempt":
				identity.Attempt++
			case "run":
				identity.ActivityRunID = "other"
			case "token":
				identity.Token = []byte("other")
			case "namespace":
				identity.Namespace = "other"
			default:
			}
			a, err := f.bridge.Bind(identity)
			if mutation == "attempt" || mutation == "run" || mutation == "token" || mutation == "namespace" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if mutation == "duplicate-bind" {
				_, err = f.bridge.Bind(identity)
				require.Error(t, err)
				return
			}
			w, err := a.BeginHeartbeat()
			require.NoError(t, err)
			if mutation == "overlap" {
				_, err = a.BeginHeartbeat()
				require.Error(t, err)
				return
			}
			ctx := a.HeartbeatContext(t.Context())
			request := &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: identity.Namespace, TaskToken: identity.Token}
			if mutation == "marker" {
				f.sequence++
				f.invocation++
				require.Error(t, f.bridge.Observe(t.Context(), RPC{Sequence: f.sequence, Invocation: f.invocation, Stage: RPCStarted, Method: workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, Request: bridgePack(t, request)}))
				return
			}
			r := f.start(ctx, workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, request)
			f.sequence++
			r.Sequence, r.Stage = f.sequence, RPCCompleted
			r.Response = bridgePack(t, &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true})
			r.Sent, r.Answered = true, true
			switch mutation {
			case "duplicate":
				require.NoError(t, f.bridge.Observe(ctx, r))
				f.sequence++
				r.Sequence = f.sequence
			case "crossed-request":
				r.Request = bridgePack(t, &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: identity.Namespace, TaskToken: []byte("other")})
			case "reordered-sequence":
				r.Sequence--
			case "missing-response":
				r.Response = nil
			default:
				t.Fatal("unknown mutation")
			}
			require.Error(t, f.bridge.Observe(ctx, r))
			_, err = w.Wait(t.Context())
			require.Error(t, err)
		})
	}
}

func TestBridgeIndependentAssignmentsMayCompleteOutOfOrder(t *testing.T) {
	f := newBridgeFixture(t)
	one, err := f.bridge.Bind(f.poll("first", 1))
	require.NoError(t, err)
	two, err := f.bridge.Bind(f.poll("second", 2))
	require.NoError(t, err)
	w1, err := one.BeginHeartbeat()
	require.NoError(t, err)
	w2, err := two.BeginHeartbeat()
	require.NoError(t, err)
	c1, c2 := one.HeartbeatContext(t.Context()), two.HeartbeatContext(t.Context())
	r1 := f.start(c1, workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: "bridge-ns", TaskToken: []byte("first")})
	r2 := f.start(c2, workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: "bridge-ns", TaskToken: []byte("second")})
	f.complete(c2, r2, &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}, "")
	f.complete(c1, r1, &workflowservice.RecordActivityTaskHeartbeatResponse{}, "")
	got, err := w2.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, r2.Invocation, got.RPC().Invocation)
	got, err = w1.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, r1.Invocation, got.RPC().Invocation)
}

func TestBridgeCancellationAndCloseWakeOutstandingWaits(t *testing.T) {
	for _, shutdown := range []string{"wait-context", "run", "close"} {
		t.Run(shutdown, func(t *testing.T) {
			run, cancelRun := context.WithCancel(t.Context())
			defer cancelRun()
			f := newBridgeFixture(t)
			f.bridge = NewBridge(run)
			a, err := f.bridge.Bind(f.poll("first", 1))
			require.NoError(t, err)
			w, err := a.BeginHeartbeat()
			require.NoError(t, err)
			ctx, cancelWait := context.WithCancel(t.Context())
			defer cancelWait()
			done := make(chan error, 1)
			go func() { _, err := w.Wait(ctx); done <- err }()
			switch shutdown {
			case "wait-context":
				cancelWait()
			case "run":
				cancelRun()
			case "close":
				require.Error(t, f.bridge.Close())
			default:
				t.Fatal("unknown shutdown")
			}
			select {
			case err := <-done:
				require.Error(t, err)
			case <-time.After(time.Second):
				t.Fatal("bridge waiter did not join")
			}
			if shutdown == "run" {
				require.NoError(t, f.bridge.Err())
				require.NoError(t, f.bridge.Close())
			}
		})
	}
}

func TestBridgeRejectsReusedPollAndCrossedResponseNamespace(t *testing.T) {
	for _, mutation := range []string{"live-token", "retired-token", "response-namespace"} {
		t.Run(mutation, func(t *testing.T) {
			f := newBridgeFixture(t)
			identity := f.poll("first", 1)
			if mutation == "retired-token" {
				a, err := f.bridge.Bind(identity)
				require.NoError(t, err)
				wait, err := a.BeginTerminal(workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName)
				require.NoError(t, err)
				rpc := f.start(t.Context(), workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName, &workflowservice.RespondActivityTaskCompletedRequest{Namespace: identity.Namespace, TaskToken: identity.Token})
				f.complete(t.Context(), rpc, &workflowservice.RespondActivityTaskCompletedResponse{}, "")
				_, err = wait.Wait(t.Context())
				require.NoError(t, err)
			}
			request := &workflowservice.PollActivityTaskQueueRequest{Namespace: identity.Namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: identity.TaskQueue}}
			response := &workflowservice.PollActivityTaskQueueResponse{TaskToken: identity.Token, Attempt: 2, WorkflowNamespace: identity.Namespace, ActivityType: &commonpb.ActivityType{Name: identity.ActivityType}}
			if mutation == "response-namespace" {
				response.TaskToken = []byte("second")
				response.WorkflowNamespace = "crossed"
			}
			rpc := f.start(t.Context(), workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, request)
			f.sequence++
			rpc.Sequence, rpc.Stage = f.sequence, RPCCompleted
			rpc.Response, rpc.Sent, rpc.Answered = bridgePack(t, response), true, true
			require.Error(t, f.bridge.Observe(t.Context(), rpc))
			require.Error(t, f.bridge.Err())
		})
	}
}

func TestBridgeCloseSignalsLifetimeWithoutInventingFault(t *testing.T) {
	bridge := NewBridge(t.Context())
	require.NoError(t, bridge.Close())
	select {
	case <-bridge.Done():
	default:
		t.Fatal("bridge lifetime did not close")
	}
	select {
	case <-bridge.Failed():
		t.Fatal("idle Close invented a correlation failure")
	default:
	}
	require.NoError(t, bridge.Err())
	require.NoError(t, bridge.Close())
}

func TestBridgeDelayedCompletionPublicationCanFollowIndependentPoll(t *testing.T) {
	f := newBridgeFixture(t)
	identity := f.poll("first", 7)
	a, err := f.bridge.Bind(identity)
	require.NoError(t, err)
	wait, err := a.BeginHeartbeat()
	require.NoError(t, err)
	ctx := a.HeartbeatContext(t.Context())
	completion := f.start(ctx, workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName, &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: identity.Namespace, TaskToken: identity.Token})
	f.sequence++
	completion.Sequence, completion.Stage = f.sequence, RPCCompleted
	completion.Response = bridgePack(t, &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true})
	completion.Sent, completion.Answered = true, true
	poll := f.start(t.Context(), workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, &workflowservice.PollActivityTaskQueueRequest{Namespace: identity.Namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: identity.TaskQueue}})
	f.complete(t.Context(), poll, &workflowservice.PollActivityTaskQueueResponse{}, "")
	require.NoError(t, f.bridge.Observe(ctx, completion))
	got, err := wait.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, completion.Invocation, got.RPC().Invocation)
}

func TestBridgeShutdownRetainsTerminalButCannotAdvanceHeartbeat(t *testing.T) {
	for _, operation := range []string{"heartbeat", "terminal", "failed-terminal"} {
		t.Run(operation, func(t *testing.T) {
			terminal := operation != "heartbeat"
			run, cancel := context.WithCancel(t.Context())
			defer cancel()
			f := newBridgeFixture(t)
			f.bridge = NewBridge(run)
			identity := f.poll("shutdown", 7)
			a, err := f.bridge.Bind(identity)
			require.NoError(t, err)
			var wait interface {
				Wait(context.Context) (ActualCompletion, error)
			}
			method := workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName
			var request proto.Message = &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: identity.Namespace, TaskToken: identity.Token}
			var response proto.Message = &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}
			if terminal {
				method = workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName
				request = &workflowservice.RespondActivityTaskCompletedRequest{Namespace: identity.Namespace, TaskToken: identity.Token}
				response = &workflowservice.RespondActivityTaskCompletedResponse{}
				wait, err = a.BeginTerminal(method)
			} else {
				wait, err = a.BeginHeartbeat()
			}
			require.NoError(t, err)
			rpc := f.start(a.HeartbeatContext(run), method, request)
			transportError := ""
			if operation == "failed-terminal" {
				transportError, response = "actual-unavailable", nil
			}
			completed := f.complete(run, rpc, response, transportError)
			cancel()
			got, err := wait.Wait(run)
			if terminal {
				require.NoError(t, err)
				require.Equal(t, completed.Invocation, got.RPC().Invocation)
				require.Equal(t, completed.TransportError, got.RPC().TransportError)
				require.True(t, proto.Equal(completed.Request, got.RPC().Request))
				require.True(t, proto.Equal(completed.Response, got.RPC().Response))
			} else {
				require.ErrorIs(t, err, context.Canceled)
				require.Zero(t, got.RPC().Invocation)
			}
			require.NoError(t, f.bridge.Close())
		})
	}
}
