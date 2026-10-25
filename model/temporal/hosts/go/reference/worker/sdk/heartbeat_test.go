package sdk

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	sdklog "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const heartbeatNamespace = "heartbeat-proof"

var heartbeatToken = []byte("opaque-heartbeat-proof-token")

type heartbeatConverter struct{ converter.DataConverter }

func (c heartbeatConverter) ToPayloads(values ...any) (*commonpb.Payloads, error) {
	payloads, err := c.DataConverter.ToPayloads(values...)
	if err != nil {
		return nil, err
	}
	for _, payload := range payloads.GetPayloads() {
		payload.Metadata["proof-converter"] = []byte("retained")
	}
	return payloads, nil
}

type heartbeatReply struct {
	response *workflowservice.RecordActivityTaskHeartbeatResponse
	err      error
}

type heartbeatCall struct {
	request *workflowservice.RecordActivityTaskHeartbeatRequest
	reply   chan heartbeatReply
}

type heartbeatCompletion struct {
	invocation uint64
	request    *workflowservice.RecordActivityTaskHeartbeatRequest
	response   *workflowservice.RecordActivityTaskHeartbeatResponse
	err        error
}

type heartbeatService struct {
	workflowservice.UnimplementedWorkflowServiceServer
	delivered  atomic.Int32
	heartbeats chan heartbeatCall
}

func (*heartbeatService) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{}, nil
}
func (*heartbeatService) DescribeNamespace(context.Context, *workflowservice.DescribeNamespaceRequest) (*workflowservice.DescribeNamespaceResponse, error) {
	return &workflowservice.DescribeNamespaceResponse{NamespaceInfo: &namespacepb.NamespaceInfo{State: enumspb.NAMESPACE_STATE_REGISTERED}}, nil
}
func (s *heartbeatService) PollActivityTaskQueue(ctx context.Context, _ *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	if s.delivered.CompareAndSwap(0, 1) {
		return &workflowservice.PollActivityTaskQueueResponse{
			TaskToken: append([]byte(nil), heartbeatToken...), Attempt: 7,
			ScheduledTime: timestamppb.Now(), StartedTime: timestamppb.Now(),
			StartToCloseTimeout: durationpb.New(time.Hour), ScheduleToCloseTimeout: durationpb.New(time.Hour),
			ActivityId: "proof-activity", ActivityType: &commonpb.ActivityType{Name: "heartbeat-proof"},
			Header: &commonpb.Header{Fields: map[string]*commonpb.Payload{"umpire-scenario": {Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("proof")}}},
		}, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s *heartbeatService) RecordActivityTaskHeartbeat(ctx context.Context, request *workflowservice.RecordActivityTaskHeartbeatRequest) (*workflowservice.RecordActivityTaskHeartbeatResponse, error) {
	call := heartbeatCall{request: proto.Clone(request).(*workflowservice.RecordActivityTaskHeartbeatRequest), reply: make(chan heartbeatReply, 1)}
	select {
	case s.heartbeats <- call:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case reply := <-call.reply:
		return reply.response, reply.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (*heartbeatService) RespondActivityTaskCompleted(context.Context, *workflowservice.RespondActivityTaskCompletedRequest) (*workflowservice.RespondActivityTaskCompletedResponse, error) {
	return &workflowservice.RespondActivityTaskCompletedResponse{}, nil
}
func (*heartbeatService) RespondActivityTaskFailed(context.Context, *workflowservice.RespondActivityTaskFailedRequest) (*workflowservice.RespondActivityTaskFailedResponse, error) {
	return &workflowservice.RespondActivityTaskFailedResponse{}, nil
}
func (*heartbeatService) RespondActivityTaskCanceled(context.Context, *workflowservice.RespondActivityTaskCanceledRequest) (*workflowservice.RespondActivityTaskCanceledResponse, error) {
	return &workflowservice.RespondActivityTaskCanceledResponse{}, nil
}

type heartbeatCommand struct {
	details  []int
	detached bool
	terminal string
}

type heartbeatReturn struct{ cause error }

type heartbeatWorker struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	service              *heartbeatService
	client               client.Client
	commands             chan heartbeatCommand
	returned             chan heartbeatReturn
	started              chan activity.Info
	canceled             chan error
	handlerDone          chan struct{}
	handlerEntered       atomic.Bool
	heartbeatInvocations atomic.Uint64
	publishers           sync.WaitGroup
	captured             chan heartbeatCompletion
	blocked              chan struct{}
	release              chan struct{}
	completions          chan heartbeatCompletion
	terminals            chan proto.Message
	beginStop            func()
	joinStop             func()
	stop                 func()
}

func newHeartbeatWorker(t *testing.T, throttle time.Duration, hold string) *heartbeatWorker {
	t.Helper()
	run, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	f := &heartbeatWorker{ctx: run, cancel: cancel, captured: make(chan heartbeatCompletion, 32), blocked: make(chan struct{}, 32), release: make(chan struct{}), service: &heartbeatService{heartbeats: make(chan heartbeatCall, 32)}, commands: make(chan heartbeatCommand), returned: make(chan heartbeatReturn, 32), started: make(chan activity.Info, 1), canceled: make(chan error, 1), handlerDone: make(chan struct{}), completions: make(chan heartbeatCompletion, 32), terminals: make(chan proto.Message, 32)}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(server, f.service)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	conn, err := client.NewLazyClient(client.Options{
		HostPort: listener.Addr().String(), Namespace: heartbeatNamespace, Identity: "heartbeat-proof-worker",
		DataConverter:      heartbeatConverter{DataConverter: converter.GetDefaultDataConverter()},
		ContextPropagators: []workflow.ContextPropagator{ScenarioPropagator()},
		Logger:             sdklog.NewStructuredLogger(slog.New(slog.DiscardHandler)),
		ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, request, response any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			isTerminal := method == workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName || method == workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName || method == workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName
			if hold == "poll" && method == workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName || hold == "terminal" && isTerminal {
				f.blocked <- struct{}{}
				select {
				case <-f.release:
				case <-run.Done():
					return run.Err()
				}
			}
			var invocation uint64
			var heartbeatRequest *workflowservice.RecordActivityTaskHeartbeatRequest
			if method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
				invocation = f.heartbeatInvocations.Add(1)
				heartbeatRequest = proto.Clone(request.(*workflowservice.RecordActivityTaskHeartbeatRequest)).(*workflowservice.RecordActivityTaskHeartbeatRequest)
			}
			err := invoke(ctx, method, request, response, cc, opts...)
			switch method {
			case workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName:
				completion := heartbeatCompletion{invocation: invocation, request: heartbeatRequest, response: proto.Clone(response.(*workflowservice.RecordActivityTaskHeartbeatResponse)).(*workflowservice.RecordActivityTaskHeartbeatResponse), err: err}
				if hold == "publication" {
					f.captured <- completion
					f.publishers.Add(1)
					go func() {
						defer f.publishers.Done()
						select {
						case <-f.release:
							f.completions <- completion
						case <-run.Done():
						}
					}()
				} else {
					f.completions <- completion
				}
				if hold == "heartbeat-completion" {
					f.blocked <- struct{}{}
					select {
					case <-f.release:
					case <-run.Done():
						return run.Err()
					}
				}
			case workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName, workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName, workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName:
				if err == nil {
					f.terminals <- proto.Clone(request.(proto.Message))
				}
			default:
			}
			return err
		})}},
	})
	require.NoError(t, err)
	f.client = conn
	worker := sdkworker.New(conn, "heartbeat-proof-queue", sdkworker.Options{DisableWorkflowWorker: true, BackgroundActivityContext: run, WorkerStopTimeout: time.Second, DefaultHeartbeatThrottleInterval: throttle, MaxHeartbeatThrottleInterval: throttle})
	worker.RegisterDynamicActivity(func(ctx context.Context, _ converter.EncodedValues) (string, error) {
		defer close(f.handlerDone)
		f.handlerEntered.Store(true)
		f.started <- activity.GetInfo(ctx)
		// Preserve SDK values and the activity deadline, but let the Run own cancellation.
		deadline, ok := ctx.Deadline()
		if !ok {
			return "", errors.New("SDK activity has no deadline")
		}
		hbCtx, hbCancel := context.WithDeadline(context.WithoutCancel(ctx), deadline)
		joined := make(chan struct{})
		stopRun := context.AfterFunc(run, func() { hbCancel(); close(joined) })
		defer func() {
			hbCancel()
			if !stopRun() {
				<-joined
			}
		}()
		cancellationJoined := make(chan struct{})
		stopCancellation := context.AfterFunc(ctx, func() { f.canceled <- context.Cause(ctx); close(cancellationJoined) })
		defer func() {
			if !stopCancellation() {
				<-cancellationJoined
			}
		}()
		for {
			select {
			case command := <-f.commands:
				switch command.terminal {
				case "completed":
					return "proof-completed", nil
				case "failed":
					return "", temporal.NewNonRetryableApplicationError("proof-failed", "proof", nil)
				case "canceled":
					return "", temporal.NewCanceledError("proof-canceled")
				}
				heartbeatCtx := ctx
				if command.detached {
					heartbeatCtx = hbCtx
				}
				if activity.GetInfo(heartbeatCtx).Attempt != 7 || heartbeatCtx.Value(scenarioKey{}) != "proof" || activity.GetLogger(heartbeatCtx) != activity.GetLogger(ctx) {
					return "", errors.New("detached context lost activity values")
				}
				for _, detail := range command.details {
					activity.RecordHeartbeat(heartbeatCtx, detail)
				}
				f.returned <- heartbeatReturn{cause: context.Cause(ctx)}
			case <-run.Done():
				return "", run.Err()
			}
		}
	}, activity.DynamicRegisterOptions{})
	require.NoError(t, worker.Start())
	var sdkStopOnce sync.Once
	stopped := make(chan struct{})
	f.beginStop = func() {
		sdkStopOnce.Do(func() {
			go func() { worker.Stop(); close(stopped) }()
		})
	}
	f.joinStop = func() {
		f.beginStop()
		heartbeatReceive(f.ctxWithoutCancellation(t), t, stopped)
	}
	var cleanupOnce sync.Once
	f.stop = func() {
		cleanupOnce.Do(func() {
			cancel()
			f.joinStop()
			conn.Close()
			server.Stop()
			<-served
			f.publishers.Wait()
			if f.handlerEntered.Load() {
				heartbeatReceive(f.ctxWithoutCancellation(t), t, f.handlerDone)
			}
		})
	}
	t.Cleanup(f.stop)
	return f
}

func heartbeatReceive[T any](ctx context.Context, t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-ctx.Done():
		t.Fatal("heartbeat proof wait: ", ctx.Err())
		var zero T
		return zero
	}
}

func (f *heartbeatWorker) command(t *testing.T, command heartbeatCommand) {
	t.Helper()
	select {
	case f.commands <- command:
	case <-f.ctx.Done():
		t.Fatal(f.ctx.Err())
	}
}

func checkHeartbeatIdentity(t *testing.T, request *workflowservice.RecordActivityTaskHeartbeatRequest, detail int) {
	t.Helper()
	require.Equal(t, heartbeatNamespace, request.GetNamespace())
	require.Equal(t, heartbeatToken, request.GetTaskToken())
	require.Equal(t, "heartbeat-proof-worker", request.GetIdentity())
	require.Len(t, request.GetDetails().GetPayloads(), 1)
	require.Equal(t, []byte("retained"), request.GetDetails().GetPayloads()[0].GetMetadata()["proof-converter"])
	var got int
	require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(request.GetDetails(), &got))
	require.Equal(t, detail, got)
}

func (f *heartbeatWorker) heartbeat(t *testing.T, detail int, canceled bool, detached bool) {
	t.Helper()
	f.command(t, heartbeatCommand{details: []int{detail}, detached: detached})
	call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	checkHeartbeatIdentity(t, call.request, detail)
	call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: canceled}}
	completion := heartbeatReceive(f.ctx, t, f.completions)
	require.NoError(t, completion.err)
	checkHeartbeatIdentity(t, completion.request, detail)
	require.Equal(t, canceled, completion.response.GetCancelRequested())
	heartbeatReceive(f.ctx, t, f.returned)
}

func checkHeartbeatTerminal(t *testing.T, request proto.Message, kind string) {
	t.Helper()
	var namespace string
	var token []byte
	var payloads *commonpb.Payloads
	switch request := request.(type) {
	case *workflowservice.RespondActivityTaskCompletedRequest:
		require.Equal(t, "completed", kind)
		namespace, token, payloads = request.GetNamespace(), request.GetTaskToken(), request.GetResult()
	case *workflowservice.RespondActivityTaskFailedRequest:
		require.Equal(t, "failed", kind)
		namespace, token = request.GetNamespace(), request.GetTaskToken()
		require.Equal(t, "proof-failed", request.GetFailure().GetMessage())
		require.Equal(t, "proof", request.GetFailure().GetApplicationFailureInfo().GetType())
		require.True(t, request.GetFailure().GetApplicationFailureInfo().GetNonRetryable())
	case *workflowservice.RespondActivityTaskCanceledRequest:
		require.Equal(t, "canceled", kind)
		namespace, token, payloads = request.GetNamespace(), request.GetTaskToken(), request.GetDetails()
	default:
		t.Fatalf("unexpected terminal %T", request)
	}
	require.Equal(t, heartbeatNamespace, namespace)
	require.Equal(t, heartbeatToken, token)
	if kind != "failed" {
		var detail string
		require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(payloads, &detail))
		require.Equal(t, "proof-"+kind, detail)
	}
}

func TestNativeHeartbeatDetachedFromFirstWindow(t *testing.T) {
	for _, terminal := range []string{"completed", "failed", "canceled"} {
		t.Run(terminal, func(t *testing.T) {
			f := newHeartbeatWorker(t, 20*time.Millisecond, "")
			info := heartbeatReceive(f.ctx, t, f.started)
			require.Equal(t, int32(7), info.Attempt)
			require.Equal(t, heartbeatToken, info.TaskToken)
			for detail, bit := range []bool{false, false, true, true} {
				f.heartbeat(t, detail, bit, true)
				if detail == 2 {
					cause := heartbeatReceive(f.ctx, t, f.canceled)
					require.True(t, temporal.IsCanceledError(cause), "SDK cancellation cause = %T", cause)
				}
			}
			f.command(t, heartbeatCommand{terminal: terminal})
			checkHeartbeatTerminal(t, heartbeatReceive(f.ctx, t, f.terminals), terminal)
			heartbeatReceive(f.ctx, t, f.handlerDone)
			f.stop()
			require.Equal(t, int32(1), f.service.delivered.Load())
			require.Empty(t, f.completions)
			require.Empty(t, f.terminals)
			require.Empty(t, f.service.heartbeats)
		})
	}
}

func TestNativeHeartbeatStopFlushUsesOpeningContext(t *testing.T) {
	for _, opening := range []struct {
		name     string
		detached bool
	}{{name: "original"}, {name: "detached-from-first", detached: true}} {
		t.Run(opening.name, func(t *testing.T) {
			f := newHeartbeatWorker(t, time.Hour, "")
			heartbeatReceive(f.ctx, t, f.started)
			f.heartbeat(t, 0, true, opening.detached)
			require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.canceled)))
			f.command(t, heartbeatCommand{details: []int{1}, detached: true})
			require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.returned).cause))
			require.Empty(t, f.service.heartbeats)
			require.Empty(t, f.completions)
			require.NoError(t, f.ctx.Err())
			f.beginStop()
			if opening.detached {
				call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
				checkHeartbeatIdentity(t, call.request, 1)
				require.Empty(t, f.terminals)
				call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
			}
			completion := heartbeatReceive(f.ctx, t, f.completions)
			checkHeartbeatIdentity(t, completion.request, 1)
			if opening.detached {
				require.NoError(t, completion.err)
				require.True(t, completion.response.GetCancelRequested())
			} else {
				require.Equal(t, codes.Canceled, status.Code(completion.err))
				require.False(t, completion.response.GetCancelRequested())
				require.Empty(t, f.service.heartbeats)
			}
			require.NoError(t, f.ctx.Err())
			f.command(t, heartbeatCommand{terminal: "completed"})
			checkHeartbeatTerminal(t, heartbeatReceive(f.ctx, t, f.terminals), "completed")
			heartbeatReceive(f.ctx, t, f.handlerDone)
			f.joinStop()
			require.NoError(t, f.ctx.Err())
			f.stop()
			require.Equal(t, int32(1), f.service.delivered.Load())
			require.Empty(t, f.completions)
			require.Empty(t, f.terminals)
			require.Empty(t, f.service.heartbeats)
		})
	}
}

func (f *heartbeatWorker) ctxWithoutCancellation(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestNativeHeartbeatCoalescesAndFlushesLatestBeforeFailure(t *testing.T) {
	f := newHeartbeatWorker(t, time.Hour, "")
	heartbeatReceive(f.ctx, t, f.started)
	f.heartbeat(t, 0, false, true)
	f.command(t, heartbeatCommand{details: []int{1, 2, 3}, detached: true})
	require.NoError(t, heartbeatReceive(f.ctx, t, f.returned).cause)
	require.Empty(t, f.service.heartbeats)
	require.Empty(t, f.completions)
	f.command(t, heartbeatCommand{terminal: "failed"})
	call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	checkHeartbeatIdentity(t, call.request, 3)
	require.Empty(t, f.terminals)
	call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{}}
	completion := heartbeatReceive(f.ctx, t, f.completions)
	require.NoError(t, completion.err)
	checkHeartbeatIdentity(t, completion.request, 3)
	checkHeartbeatTerminal(t, heartbeatReceive(f.ctx, t, f.terminals), "failed")
	heartbeatReceive(f.ctx, t, f.handlerDone)
	f.stop()
	require.Empty(t, f.service.heartbeats)
	require.Empty(t, f.completions)
	require.Empty(t, f.terminals)
}

func TestNativeHeartbeatTerminalSelection(t *testing.T) {
	for _, history := range []string{"unknown", "false", "true"} {
		for _, terminal := range []string{"completed", "failed", "canceled"} {
			if history != "true" && terminal == "canceled" {
				continue
			}
			t.Run(history+"/"+terminal, func(t *testing.T) {
				f := newHeartbeatWorker(t, time.Hour, "")
				heartbeatReceive(f.ctx, t, f.started)
				if history != "unknown" {
					f.heartbeat(t, 0, history == "true", true)
					if history == "true" {
						require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.canceled)))
					}
				}
				f.command(t, heartbeatCommand{terminal: terminal})
				checkHeartbeatTerminal(t, heartbeatReceive(f.ctx, t, f.terminals), terminal)
				heartbeatReceive(f.ctx, t, f.handlerDone)
				f.stop()
				require.Equal(t, int32(1), f.service.delivered.Load())
				require.Empty(t, f.terminals)
				require.Empty(t, f.completions)
			})
		}
	}
}

func TestClientHeartbeatDoesNotCancelNativeHandler(t *testing.T) {
	f := newHeartbeatWorker(t, time.Hour, "")
	heartbeatReceive(f.ctx, t, f.started)
	returned := make(chan error, 1)
	go func() { returned <- f.client.RecordActivityHeartbeat(f.ctx, heartbeatToken, 0) }()
	call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	checkHeartbeatIdentity(t, call.request, 0)
	call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
	completion := heartbeatReceive(f.ctx, t, f.completions)
	require.NoError(t, completion.err)
	require.True(t, completion.response.GetCancelRequested())
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, returned)))
	f.command(t, heartbeatCommand{})
	require.NoError(t, heartbeatReceive(f.ctx, t, f.returned).cause)
	require.Empty(t, f.canceled)
	f.command(t, heartbeatCommand{terminal: "completed"})
	checkHeartbeatTerminal(t, heartbeatReceive(f.ctx, t, f.terminals), "completed")
	heartbeatReceive(f.ctx, t, f.handlerDone)
	f.stop()
}

func TestNativeHeartbeatCompletionPrecedesSDKCancellationDelivery(t *testing.T) {
	f := newHeartbeatWorker(t, time.Hour, "heartbeat-completion")
	heartbeatReceive(f.ctx, t, f.started)
	f.command(t, heartbeatCommand{details: []int{0}, detached: true})
	call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
	completion := heartbeatReceive(f.ctx, t, f.completions)
	require.NoError(t, completion.err)
	require.True(t, completion.response.GetCancelRequested())
	heartbeatReceive(f.ctx, t, f.blocked)
	require.Empty(t, f.canceled)
	require.Empty(t, f.returned)
	require.Empty(t, f.terminals)
	close(f.release)
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.canceled)))
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.returned).cause))
	f.command(t, heartbeatCommand{terminal: "canceled"})
	checkHeartbeatTerminal(t, heartbeatReceive(f.ctx, t, f.terminals), "canceled")
	heartbeatReceive(f.ctx, t, f.handlerDone)
	f.stop()
}

func TestNativeHeartbeatRunCancellationJoinsBlockedWork(t *testing.T) {
	for _, wait := range []string{"poll", "handler", "heartbeat-rpc", "batch", "heartbeat-completion", "terminal"} {
		t.Run(wait, func(t *testing.T) {
			hold := ""
			if wait == "poll" || wait == "heartbeat-completion" || wait == "terminal" {
				hold = wait
			}
			f := newHeartbeatWorker(t, time.Hour, hold)
			cleanupCtx := f.ctxWithoutCancellation(t)
			if wait == "poll" {
				heartbeatReceive(f.ctx, t, f.blocked)
			} else {
				heartbeatReceive(f.ctx, t, f.started)
				switch wait {
				case "heartbeat-rpc", "heartbeat-completion":
					f.command(t, heartbeatCommand{details: []int{0}, detached: true})
					call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
					if wait == "heartbeat-completion" {
						call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
						completed := heartbeatReceive(f.ctx, t, f.completions)
						require.NoError(t, completed.err)
						require.True(t, completed.response.GetCancelRequested())
						heartbeatReceive(f.ctx, t, f.blocked)
						require.Empty(t, f.canceled)
					}
				case "batch":
					f.heartbeat(t, 0, false, true)
					f.command(t, heartbeatCommand{details: []int{1}, detached: true})
					require.NoError(t, heartbeatReceive(f.ctx, t, f.returned).cause)
				case "terminal":
					f.command(t, heartbeatCommand{terminal: "completed"})
					heartbeatReceive(f.ctx, t, f.blocked)
				default:
					require.Equal(t, "handler", wait)
				}
			}
			f.cancel()
			if wait != "poll" {
				heartbeatReceive(cleanupCtx, t, f.handlerDone)
			}
			f.stop()
			require.Empty(t, f.service.heartbeats)
			for len(f.completions) != 0 {
				completion := <-f.completions
				require.Error(t, completion.err)
			}
			for len(f.terminals) != 0 {
				terminal := <-f.terminals
				require.IsType(t, &workflowservice.RespondActivityTaskFailedRequest{}, terminal)
			}
			if wait == "poll" {
				require.Zero(t, f.service.delivered.Load())
			}
		})
	}
}

func TestNativeHeartbeatRetryCapturesEveryTransportAttempt(t *testing.T) {
	f := newHeartbeatWorker(t, time.Hour, "")
	heartbeatReceive(f.ctx, t, f.started)
	f.command(t, heartbeatCommand{details: []int{0}, detached: true})
	first := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	first.reply <- heartbeatReply{err: status.Error(codes.Unavailable, "retry-proof")}
	failed := heartbeatReceive(f.ctx, t, f.completions)
	require.Equal(t, uint64(1), failed.invocation)
	require.Equal(t, codes.Unavailable, status.Code(failed.err))
	require.Empty(t, f.returned)
	require.Empty(t, f.canceled)
	require.Empty(t, f.terminals)
	second := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	require.True(t, proto.Equal(first.request, second.request))
	second.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
	completed := heartbeatReceive(f.ctx, t, f.completions)
	require.Equal(t, uint64(2), completed.invocation)
	require.NoError(t, completed.err)
	require.True(t, completed.response.GetCancelRequested())
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.canceled)))
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.returned).cause))
	f.command(t, heartbeatCommand{terminal: "canceled"})
	checkHeartbeatTerminal(t, heartbeatReceive(f.ctx, t, f.terminals), "canceled")
	heartbeatReceive(f.ctx, t, f.handlerDone)
	f.stop()
	require.Empty(t, f.completions)
	require.Empty(t, f.terminals)
}

func TestNativeHeartbeatOtherCancellationCausesAreNotServerCancel(t *testing.T) {
	for _, test := range []struct {
		name         string
		reply        heartbeatReply
		cancellation bool
	}{
		{name: "permission-denied", reply: heartbeatReply{err: status.Error(codes.PermissionDenied, "proof-denied")}, cancellation: true},
		{name: "not-found", reply: heartbeatReply{err: status.Error(codes.NotFound, "proof-missing")}, cancellation: true},
		{name: "paused", reply: heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{ActivityPaused: true}}, cancellation: true},
		{name: "reset", reply: heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{ActivityReset: true}}, cancellation: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newHeartbeatWorker(t, time.Hour, "")
			heartbeatReceive(f.ctx, t, f.started)
			f.command(t, heartbeatCommand{details: []int{0}, detached: true})
			call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
			call.reply <- test.reply
			completion := heartbeatReceive(f.ctx, t, f.completions)
			checkHeartbeatIdentity(t, completion.request, 0)
			if test.reply.err != nil {
				require.Equal(t, status.Code(test.reply.err), status.Code(completion.err))
			} else {
				require.NoError(t, completion.err)
			}
			require.False(t, completion.response.GetCancelRequested())
			cause := heartbeatReceive(f.ctx, t, f.returned).cause
			if test.cancellation {
				require.Error(t, cause)
				require.False(t, temporal.IsCanceledError(cause))
				switch test.name {
				case "permission-denied":
					var denied *serviceerror.PermissionDenied
					require.ErrorAs(t, cause, &denied)
				case "not-found":
					var missing *serviceerror.NotFound
					require.ErrorAs(t, cause, &missing)
				case "paused":
					require.ErrorIs(t, cause, activity.ErrActivityPaused)
				case "reset":
					require.ErrorIs(t, cause, activity.ErrActivityReset)
				default:
					t.Fatalf("unexpected control %q", test.name)
				}
				require.Error(t, heartbeatReceive(f.ctx, t, f.canceled))
			} else {
				require.NoError(t, cause)
				require.Empty(t, f.canceled)
			}
			require.Empty(t, f.terminals)
			f.cancel()
			heartbeatReceive(f.ctxWithoutCancellation(t), t, f.handlerDone)
			f.stop()
		})
	}
}

func TestNativeHeartbeatSDKCancellationPrecedesCompletionPublication(t *testing.T) {
	f := newHeartbeatWorker(t, time.Hour, "publication")
	heartbeatReceive(f.ctx, t, f.started)
	f.command(t, heartbeatCommand{details: []int{0}, detached: true})
	call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
	captured := heartbeatReceive(f.ctx, t, f.captured)
	require.NoError(t, captured.err)
	require.True(t, captured.response.GetCancelRequested())
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.canceled)))
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.returned).cause))
	require.Empty(t, f.completions)
	require.Empty(t, f.terminals)
	close(f.release)
	published := heartbeatReceive(f.ctx, t, f.completions)
	require.Equal(t, captured.invocation, published.invocation)
	require.True(t, proto.Equal(captured.request, published.request))
	require.True(t, proto.Equal(captured.response, published.response))
	f.command(t, heartbeatCommand{terminal: "canceled"})
	checkHeartbeatTerminal(t, heartbeatReceive(f.ctx, t, f.terminals), "canceled")
	heartbeatReceive(f.ctx, t, f.handlerDone)
	f.stop()
}

func TestNativeHeartbeatRunCancellationJoinsUnpublishedCompletion(t *testing.T) {
	f := newHeartbeatWorker(t, time.Hour, "publication")
	heartbeatReceive(f.ctx, t, f.started)
	f.command(t, heartbeatCommand{details: []int{0}, detached: true})
	call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
	captured := heartbeatReceive(f.ctx, t, f.captured)
	require.NoError(t, captured.err)
	require.True(t, captured.response.GetCancelRequested())
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.canceled)))
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, f.returned).cause))
	f.cancel()
	heartbeatReceive(f.ctxWithoutCancellation(t), t, f.handlerDone)
	f.stop()
	require.Empty(t, f.completions)
	for len(f.terminals) != 0 {
		require.IsType(t, &workflowservice.RespondActivityTaskCanceledRequest{}, <-f.terminals)
	}
}

func TestNativeHeartbeatMalformedResponseIsNotFalse(t *testing.T) {
	f := newHeartbeatWorker(t, time.Hour, "")
	heartbeatReceive(f.ctx, t, f.started)
	f.command(t, heartbeatCommand{details: []int{0}, detached: true})
	call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	malformed := &workflowservice.RecordActivityTaskHeartbeatResponse{}
	malformed.ProtoReflect().SetUnknown([]byte{0xff})
	call.reply <- heartbeatReply{response: malformed}
	completion := heartbeatReceive(f.ctx, t, f.completions)
	require.Equal(t, codes.Internal, status.Code(completion.err))
	require.Contains(t, completion.err.Error(), "unmarshal")
	require.Empty(t, f.terminals)
	f.cancel()
	heartbeatReceive(f.ctxWithoutCancellation(t), t, f.handlerDone)
	f.stop()
	for len(f.completions) != 0 {
		require.Error(t, (<-f.completions).err)
	}
	for len(f.terminals) != 0 {
		require.IsType(t, &workflowservice.RespondActivityTaskFailedRequest{}, <-f.terminals)
	}
}

func TestNativeHeartbeatCrossedClientCompletionDoesNotCancelHandler(t *testing.T) {
	f := newHeartbeatWorker(t, time.Hour, "")
	heartbeatReceive(f.ctx, t, f.started)
	f.command(t, heartbeatCommand{details: []int{0}, detached: true})
	native := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	checkHeartbeatIdentity(t, native.request, 0)
	crossed, err := client.NewClientFromExistingWithContext(f.ctx, f.client, client.Options{Namespace: "crossed-namespace"})
	require.NoError(t, err)
	defer crossed.Close()
	returned := make(chan error, 1)
	go func() { returned <- crossed.RecordActivityHeartbeat(f.ctx, []byte("crossed-token"), 99) }()
	call := heartbeatReceive(f.ctx, t, f.service.heartbeats)
	require.Equal(t, "crossed-namespace", call.request.GetNamespace())
	require.Equal(t, []byte("crossed-token"), call.request.GetTaskToken())
	call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
	completion := heartbeatReceive(f.ctx, t, f.completions)
	require.Equal(t, uint64(2), completion.invocation)
	require.NoError(t, completion.err)
	require.Equal(t, "crossed-namespace", completion.request.GetNamespace())
	require.Equal(t, []byte("crossed-token"), completion.request.GetTaskToken())
	require.True(t, completion.response.GetCancelRequested())
	require.True(t, temporal.IsCanceledError(heartbeatReceive(f.ctx, t, returned)))
	require.Empty(t, f.canceled)
	require.Empty(t, f.returned)
	native.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{}}
	completion = heartbeatReceive(f.ctx, t, f.completions)
	require.Equal(t, uint64(1), completion.invocation)
	require.NoError(t, completion.err)
	require.False(t, completion.response.GetCancelRequested())
	checkHeartbeatIdentity(t, completion.request, 0)
	require.NoError(t, heartbeatReceive(f.ctx, t, f.returned).cause)
	f.command(t, heartbeatCommand{terminal: "completed"})
	checkHeartbeatTerminal(t, heartbeatReceive(f.ctx, t, f.terminals), "completed")
	heartbeatReceive(f.ctx, t, f.handlerDone)
	f.stop()
}
