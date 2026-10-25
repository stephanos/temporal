package sdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	sdklog "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	sdkworker "go.temporal.io/sdk/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	"umpire.local/temporal/reference/worker"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestCanceledConversionPreservesPayload(t *testing.T) {
	payload := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("canceled-details")}
	response, err := anypb.New(&workflowservice.RespondActivityTaskCanceledRequest{Details: &commonpb.Payloads{Payloads: []*commonpb.Payload{payload}}})
	require.NoError(t, err)
	got, err := convert(response)
	require.NoError(t, err)
	require.True(t, temporal.IsCanceledError(got.err))
	failure := temporal.GetDefaultFailureConverter().ErrorToFailure(got.err)
	require.True(t, proto.Equal(payload, failure.GetCanceledFailureInfo().GetDetails().GetPayloads()[0]))
	failed, err := anypb.New(&workflowservice.RespondActivityTaskFailedRequest{Failure: &failurepb.Failure{FailureInfo: &failurepb.Failure_CanceledFailureInfo{CanceledFailureInfo: &failurepb.CanceledFailureInfo{}}}})
	require.NoError(t, err)
	_, err = convert(failed)
	require.ErrorContains(t, err, "canceled")
}

type sdkProgressKey struct{}
type sdkInvocationValue struct{}
type sdkProgress struct{ sent, answered atomic.Bool }
type sdkProgressStats struct{}

func (sdkProgressStats) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context { return ctx }
func (sdkProgressStats) HandleRPC(ctx context.Context, event stats.RPCStats) {
	progress, ok := ctx.Value(sdkProgressKey{}).(*sdkProgress)
	if !ok {
		return
	}
	switch event.(type) {
	case *stats.OutHeader:
		progress.sent.Store(true)
	case *stats.InTrailer:
		progress.answered.Store(true)
	default:
	}
}
func (sdkProgressStats) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}
func (sdkProgressStats) HandleConn(context.Context, stats.ConnStats) {}

type sdkCapture struct {
	bridge                                       *worker.Bridge
	mu                                           sync.Mutex
	invocation, sequence                         uint64
	completed                                    chan worker.RPC
	mutate                                       func(string, proto.Message)
	run                                          context.Context
	hold                                         string
	blocked                                      chan worker.RPC
	release                                      chan struct{}
	publishers                                   sync.WaitGroup
	inFlight                                     sync.WaitGroup
	rewriteTerminal                              bool
	terminalOriginalContext, terminalCallContext context.Context
	terminalCanceledAtCapture                    bool
}

func (c *sdkCapture) intercept(ctx context.Context, method string, request, response any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	switch method {
	case workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName,
		workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName:
	default:
		return invoke(ctx, method, request, response, cc, opts...)
	}
	c.inFlight.Add(1)
	defer c.inFlight.Done()
	originalContext := context.WithValue(ctx, sdkInvocationValue{}, c)
	ctx, cancelCall := context.WithCancel(originalContext)
	cancelJoined := make(chan struct{})
	stopRun := context.AfterFunc(c.run, func() { cancelCall(); close(cancelJoined) })
	defer func() {
		cancelCall()
		if !stopRun() {
			<-cancelJoined
		}
	}()
	if c.rewriteTerminal && method == workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName {
		completed := request.(*workflowservice.RespondActivityTaskCompletedRequest)
		method = workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName
		request = &workflowservice.RespondActivityTaskFailedRequest{Namespace: completed.Namespace, TaskToken: append([]byte(nil), completed.TaskToken...), Failure: worker.UnprogrammedFailure()}
		response = new(workflowservice.RespondActivityTaskFailedResponse)
	}
	if c.mutate != nil {
		c.mutate(method, request.(proto.Message))
	}
	actual, err := anypb.New(proto.Clone(request.(proto.Message)))
	if err != nil {
		return err
	}
	reached := new(sdkProgress)
	ctx = context.WithValue(ctx, sdkProgressKey{}, reached)
	c.mu.Lock()
	c.invocation++
	invocation := c.invocation
	c.sequence++
	_ = c.bridge.Observe(ctx, worker.RPC{Invocation: invocation, Sequence: c.sequence, Stage: worker.RPCStarted, Method: method, Request: actual})
	c.mu.Unlock()
	if c.hold == "terminal" && method == workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName {
		c.blocked <- worker.RPC{Invocation: invocation, Method: method, Request: actual}
		select {
		case <-c.release:
		case <-c.run.Done():
			<-cancelJoined
		}
	}
	if err == nil {
		err = invoke(ctx, method, request, response, cc, opts...)
	}
	result, packErr := anypb.New(proto.Clone(response.(proto.Message)))
	if packErr != nil {
		return packErr
	}
	completed := worker.RPC{Invocation: invocation, Stage: worker.RPCCompleted, Method: method, Request: actual, Response: result, Sent: reached.sent.Load(), Answered: reached.answered.Load()}
	if err == nil {
		completed.Answered = true
	} else {
		completed.CallError, packErr = anypb.New(status.Convert(err).Proto())
		if packErr != nil {
			return packErr
		}
	}
	if err != nil && completed.Answered && status.Code(err) != codes.Internal {
		completed.ServerError = &realizationv1.Error{Code: int32(status.Code(err)), Message: status.Convert(err).Message()}
	} else if err != nil {
		completed.TransportError = err.Error()
	}
	c.mu.Lock()
	if c.hold == "terminal" && method == workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName {
		c.terminalOriginalContext, c.terminalCallContext = originalContext, ctx
		c.terminalCanceledAtCapture = ctx.Err() != nil && c.run.Err() != nil
	}
	c.sequence++
	completed.Sequence = c.sequence
	if c.hold == "publication" && method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName && err == nil {
		c.blocked <- completed
		c.publishers.Add(1)
		go func() {
			defer c.publishers.Done()
			select {
			case <-c.release:
				c.mu.Lock()
				_ = c.bridge.Observe(ctx, completed)
				c.completed <- completed
				c.mu.Unlock()
			case <-c.run.Done():
			}
		}()
	} else {
		_ = c.bridge.Observe(ctx, completed)
		c.completed <- completed
	}
	c.mu.Unlock()
	if c.hold == "delivery" && method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName && err == nil {
		c.blocked <- completed
		select {
		case <-c.release:
		case <-c.run.Done():
		case <-ctx.Done():
		}
	}
	return err
}

type continuationService struct {
	heartbeatService
	terminal         chan proto.Message
	failures         []*failurepb.Failure
	taskTimeout      time.Duration
	heartbeatTimeout time.Duration
	tasksMu          sync.Mutex
	tasks            []*workflowservice.PollActivityTaskQueueResponse
	customTasks      bool
}

func (s *continuationService) PollActivityTaskQueue(ctx context.Context, request *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	var task *workflowservice.PollActivityTaskQueueResponse
	var err error
	if s.customTasks {
		s.tasksMu.Lock()
		if len(s.tasks) != 0 {
			task = s.tasks[0]
			s.tasks = s.tasks[1:]
			s.delivered.Add(1)
		}
		s.tasksMu.Unlock()
		if task == nil {
			<-ctx.Done()
			return nil, ctx.Err()
		}
	} else {
		task, err = s.heartbeatService.PollActivityTaskQueue(ctx, request)
	}
	if task != nil && len(task.GetTaskToken()) != 0 {
		task.HeartbeatTimeout = durationpb.New(s.heartbeatTimeout)
		task.StartToCloseTimeout = durationpb.New(s.taskTimeout)
		task.ScheduleToCloseTimeout = durationpb.New(s.taskTimeout)
		if task.ActivityRunId == "" {
			task.ActivityRunId = "activity-run-proof"
		}
		task.WorkflowNamespace = request.GetNamespace()
	}
	return task, err
}
func (s *continuationService) RespondActivityTaskCompleted(_ context.Context, request *workflowservice.RespondActivityTaskCompletedRequest) (*workflowservice.RespondActivityTaskCompletedResponse, error) {
	s.terminal <- proto.Clone(request)
	return &workflowservice.RespondActivityTaskCompletedResponse{}, nil
}
func (s *continuationService) RespondActivityTaskFailed(_ context.Context, request *workflowservice.RespondActivityTaskFailedRequest) (*workflowservice.RespondActivityTaskFailedResponse, error) {
	s.terminal <- proto.Clone(request)
	return &workflowservice.RespondActivityTaskFailedResponse{Failures: s.failures}, nil
}
func (s *continuationService) RespondActivityTaskCanceled(_ context.Context, request *workflowservice.RespondActivityTaskCanceledRequest) (*workflowservice.RespondActivityTaskCanceledResponse, error) {
	s.terminal <- proto.Clone(request)
	return &workflowservice.RespondActivityTaskCanceledResponse{}, nil
}

type sdkContinuationFixture struct {
	ctx      context.Context
	cancel   context.CancelFunc
	bridge   *worker.Bridge
	capture  *sdkCapture
	service  *continuationService
	done     chan struct{}
	err      error
	closeErr error
	stop     func()
}

type sdkContinuationOptions struct {
	hold            string
	batch           bool
	taskTimeout     time.Duration
	interceptor     *sdkActivityProbe
	tasks           []*workflowservice.PollActivityTaskQueueResponse
	rewriteTerminal bool
}

func startSDKContinuation(t *testing.T, program []byte, mutate func(string, proto.Message), failures []*failurepb.Failure, options ...sdkContinuationOptions) *sdkContinuationFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	bridge := worker.NewBridge(ctx)
	var optionsValue sdkContinuationOptions
	if len(options) != 0 {
		optionsValue = options[0]
	}
	capture := &sdkCapture{bridge: bridge, completed: make(chan worker.RPC, 128), mutate: mutate, run: ctx, hold: optionsValue.hold, blocked: make(chan worker.RPC, 32), release: make(chan struct{}), rewriteTerminal: optionsValue.rewriteTerminal}
	service := &continuationService{heartbeatService: heartbeatService{heartbeats: make(chan heartbeatCall, 32)}, terminal: make(chan proto.Message, 32), failures: failures, taskTimeout: time.Hour, heartbeatTimeout: 25 * time.Millisecond}
	if optionsValue.tasks != nil {
		service.customTasks = true
		service.tasks = optionsValue.tasks
	}
	if optionsValue.batch {
		service.heartbeatTimeout = 0
	}
	if optionsValue.taskTimeout != 0 {
		service.taskTimeout = optionsValue.taskTimeout
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(server, service)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	if optionsValue.interceptor != nil {
		optionsValue.interceptor.run = ctx
	}
	conn, err := client.NewLazyClient(client.Options{
		HostPort: listener.Addr().String(), Namespace: "outer-namespace", Identity: "production-sdk",
		Logger:            sdklog.NewStructuredLogger(slog.New(slog.DiscardHandler)),
		ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{grpc.WithStatsHandler(sdkProgressStats{}), grpc.WithChainUnaryInterceptor(capture.intercept)}},
	})
	require.NoError(t, err)
	fixture := &sdkContinuationFixture{ctx: ctx, cancel: cancel, bridge: bridge, capture: capture, service: service, done: make(chan struct{})}
	go func() {
		if optionsValue.interceptor == nil {
			fixture.err = Follow(ctx, conn, program, bridge)
		} else {
			fixture.err = followWithOptions(ctx, conn, program, bridge, sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{optionsValue.interceptor}})
		}
		close(fixture.done)
	}()
	var once sync.Once
	fixture.stop = func() {
		once.Do(func() {
			cancel()
			cleanup, cleanupCancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
			defer cleanupCancel()
			heartbeatReceive(cleanup, t, fixture.done)
			conn.Close()
			server.Stop()
			<-served
			capture.publishers.Wait()
			capture.inFlight.Wait()
			fixture.closeErr = bridge.Close()
		})
	}
	t.Cleanup(fixture.stop)
	return fixture
}

func productionProgram(t *testing.T, bits []bool, terminal string) []byte {
	t.Helper()
	program := &temporalv1.WorkerProgram{Namespace: heartbeatNamespace, TaskQueue: "production-queue"}
	history := []bool{}
	for index, bit := range bits {
		key := &temporalv1.ActivityTaskKey{Scenario: "proof", Attempt: 7, ActivityType: "heartbeat-proof", HeartbeatHistory: append([]bool(nil), history...)}
		packedKey, err := anypb.New(key)
		require.NoError(t, err)
		details, err := converter.GetDefaultDataConverter().ToPayloads(index)
		require.NoError(t, err)
		response, err := anypb.New(&workflowservice.RecordActivityTaskHeartbeatRequest{Details: details})
		require.NoError(t, err)
		program.Rules = append(program.Rules, &temporalv1.Rule{Key: packedKey, Respond: []*anypb.Any{response}})
		domain, err := anypb.New(&temporalv1.ActivityTaskSuccessors{Key: key, CancelRequested: []bool{bit}})
		require.NoError(t, err)
		program.ContinuationDomains = append(program.ContinuationDomains, domain)
		history = append(history, bit)
	}
	key, err := anypb.New(&temporalv1.ActivityTaskKey{Scenario: "proof", Attempt: 7, ActivityType: "heartbeat-proof", HeartbeatHistory: history})
	require.NoError(t, err)
	var response proto.Message
	switch terminal {
	case "completed":
		details, err := converter.GetDefaultDataConverter().ToPayloads("proof-completed")
		require.NoError(t, err)
		response = &workflowservice.RespondActivityTaskCompletedRequest{Result: details}
	case "failed":
		response = &workflowservice.RespondActivityTaskFailedRequest{Failure: &failurepb.Failure{Message: "proof-failed", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "proof", NonRetryable: true}}}}
	case "canceled":
		details, err := converter.GetDefaultDataConverter().ToPayloads("proof-canceled")
		require.NoError(t, err)
		response = &workflowservice.RespondActivityTaskCanceledRequest{Details: details}
	default:
		t.Fatalf("unknown terminal %q", terminal)
	}
	packedResponse, err := anypb.New(response)
	require.NoError(t, err)
	program.Rules = append(program.Rules, &temporalv1.Rule{Key: key, Respond: []*anypb.Any{packedResponse}})
	envelope, err := anypb.New(program)
	require.NoError(t, err)
	data, err := proto.Marshal(envelope)
	require.NoError(t, err)
	return data
}

func TestProductionSDKFollowsActualHeartbeatHistory(t *testing.T) {
	for _, terminal := range []string{"completed", "failed", "canceled"} {
		t.Run(terminal, func(t *testing.T) {
			bits := []bool{false, false, true, true}
			fixture := startSDKContinuation(t, productionProgram(t, bits, terminal), nil, nil)
			for index, bit := range bits {
				call := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
				require.Equal(t, heartbeatNamespace, call.request.GetNamespace())
				require.Equal(t, heartbeatToken, call.request.GetTaskToken())
				var detail int
				require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(call.request.GetDetails(), &detail))
				require.Equal(t, index, detail)
				require.Empty(t, fixture.service.terminal)
				call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: bit}}
			}
			checkHeartbeatTerminal(t, heartbeatReceive(fixture.ctx, t, fixture.service.terminal), terminal)
			require.Equal(t, int32(1), fixture.service.delivered.Load())
			fixture.stop()
			require.NoError(t, fixture.err)
			require.NoError(t, fixture.closeErr)
		})
	}
}

func TestProductionSDKRefusesUnnotifiedCanceledLeaf(t *testing.T) {
	require.ErrorContains(t, Validate(productionProgram(t, nil, "canceled")), "true heartbeat")
}

func TestProductionSDKTerminalCoordinatorRejectsActualMismatch(t *testing.T) {
	for _, test := range []struct {
		name, terminal, want string
		mutate               func(string, proto.Message)
		failures             []*failurepb.Failure
	}{
		{name: "mutated-completion", terminal: "completed", want: "payload", mutate: func(method string, request proto.Message) {
			if method == workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName {
				completed := request.(*workflowservice.RespondActivityTaskCompletedRequest)
				completed.Result.Payloads[0].Data = []byte("mutated")
			}
		}},
		{name: "failed-validation", terminal: "failed", want: "validation failures", failures: []*failurepb.Failure{{Message: "rejected"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := startSDKContinuation(t, productionProgram(t, nil, test.terminal), test.mutate, test.failures)
			heartbeatReceive(fixture.ctx, t, fixture.service.terminal)
			heartbeatReceive(fixture.ctx, t, fixture.done)
			require.ErrorContains(t, fixture.err, test.want)
			fixture.stop()
		})
	}
}

func TestProductionSDKRetryWaitsForActualSuccess(t *testing.T) {
	fixture := startSDKContinuation(t, productionProgram(t, []bool{true}, "canceled"), nil, nil)
	first := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
	first.reply <- heartbeatReply{err: status.Error(codes.Unavailable, "retry actual heartbeat")}
	require.Empty(t, fixture.service.terminal)
	second := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
	require.True(t, proto.Equal(first.request, second.request))
	second.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
	checkHeartbeatTerminal(t, heartbeatReceive(fixture.ctx, t, fixture.service.terminal), "canceled")
	fixture.stop()
	require.NoError(t, fixture.err)
	require.NoError(t, fixture.closeErr)
	var attempts []worker.RPC
	for len(fixture.capture.completed) != 0 {
		rpc := <-fixture.capture.completed
		if rpc.Method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
			attempts = append(attempts, rpc)
		}
	}
	require.Len(t, attempts, 2)
	require.NotNil(t, attempts[0].CallError)
	require.NotEqual(t, attempts[0].Invocation, attempts[1].Invocation)
	require.Nil(t, attempts[1].CallError)
}

func TestProductionSDKCancelOutstandingHeartbeatJoins(t *testing.T) {
	fixture := startSDKContinuation(t, productionProgram(t, []bool{false}, "completed"), nil, nil)
	heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
	fixture.stop()
	require.NoError(t, fixture.err)
	require.NoError(t, fixture.closeErr)
	for len(fixture.service.terminal) != 0 {
		require.IsType(t, &workflowservice.RespondActivityTaskFailedRequest{}, <-fixture.service.terminal)
	}
	for len(fixture.capture.completed) != 0 {
		rpc := <-fixture.capture.completed
		if rpc.Method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
			require.NotNil(t, rpc.CallError)
		}
	}
}

type sdkActivityProbe struct {
	interceptor.InterceptorBase
	started          chan struct{}
	returned         chan int
	canceled         chan error
	calls            atomic.Int32
	holdRegistration bool
	release          chan struct{}
	run              context.Context
}

func newSDKActivityProbe() *sdkActivityProbe {
	return &sdkActivityProbe{started: make(chan struct{}, 16), returned: make(chan int, 16), canceled: make(chan error, 16), release: make(chan struct{})}
}
func (p *sdkActivityProbe) InterceptActivity(_ context.Context, next interceptor.ActivityInboundInterceptor) interceptor.ActivityInboundInterceptor {
	return &sdkActivityInbound{ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next}, probe: p}
}

type sdkActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	probe *sdkActivityProbe
}

func (i *sdkActivityInbound) Init(next interceptor.ActivityOutboundInterceptor) error {
	return i.Next.Init(&sdkActivityOutbound{ActivityOutboundInterceptorBase: interceptor.ActivityOutboundInterceptorBase{Next: next}, probe: i.probe})
}
func (i *sdkActivityInbound) ExecuteActivity(ctx context.Context, input *interceptor.ExecuteActivityInput) (any, error) {
	joined := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { i.probe.canceled <- context.Cause(ctx); close(joined) })
	defer func() {
		if !stop() {
			<-joined
		}
	}()
	i.probe.started <- struct{}{}
	if i.probe.holdRegistration {
		select {
		case <-i.probe.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-i.probe.run.Done():
			return nil, i.probe.run.Err()
		}
	}
	return i.Next.ExecuteActivity(ctx, input)
}

type sdkActivityOutbound struct {
	interceptor.ActivityOutboundInterceptorBase
	probe *sdkActivityProbe
}

func (o *sdkActivityOutbound) RecordHeartbeat(ctx context.Context, details ...any) {
	index := int(o.probe.calls.Add(1) - 1)
	o.Next.RecordHeartbeat(ctx, details...)
	o.probe.returned <- index
}

func TestProductionSDKJoinsNotificationAndNativeCancellationInEitherOrder(t *testing.T) {
	for _, hold := range []string{"delivery", "publication"} {
		t.Run(hold, func(t *testing.T) {
			probe := newSDKActivityProbe()
			fixture := startSDKContinuation(t, productionProgram(t, []bool{true}, "canceled"), nil, nil, sdkContinuationOptions{hold: hold, interceptor: probe})
			call := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
			call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
			captured := heartbeatReceive(fixture.ctx, t, fixture.capture.blocked)
			require.Nil(t, captured.CallError)
			if hold == "delivery" {
				require.Empty(t, probe.canceled)
			} else {
				require.True(t, temporal.IsCanceledError(heartbeatReceive(fixture.ctx, t, probe.canceled)))
			}
			require.Empty(t, fixture.service.terminal)
			close(fixture.capture.release)
			terminal := heartbeatReceive(fixture.ctx, t, fixture.service.terminal)
			if _, ok := terminal.(*workflowservice.RespondActivityTaskCanceledRequest); !ok {
				t.Logf("unexpected terminal %T, bridge error %v", terminal, fixture.bridge.Err())
				select {
				case <-fixture.done:
					t.Logf("follower error %v", fixture.err)
				default:
				}
			}
			checkHeartbeatTerminal(t, terminal, "canceled")
			fixture.stop()
			require.NoError(t, fixture.err)
			require.NoError(t, fixture.closeErr)
		})
	}
}

func blockedSDKStage(t *testing.T, stage string, deadline bool) *sdkContinuationFixture {
	t.Helper()
	probe := newSDKActivityProbe()
	opts := sdkContinuationOptions{interceptor: probe}
	if deadline {
		opts.taskTimeout = time.Second
	}
	bits := []bool{false}
	terminal := "completed"
	switch stage {
	case "registration":
		probe.holdRegistration = true
	case "native-rpc":
	case "batch":
		opts.batch = true
		bits = []bool{false, false}
	case "true-join":
		opts.hold = "delivery"
		bits = []bool{true}
		terminal = "canceled"
	case "publication":
		opts.hold = "publication"
		bits = []bool{true}
		terminal = "canceled"
	case "terminal":
		opts.hold = "terminal"
		bits = nil
	default:
		t.Fatalf("unexpected stage %q", stage)
	}
	fixture := startSDKContinuation(t, productionProgram(t, bits, terminal), nil, nil, opts)
	switch stage {
	case "registration":
		heartbeatReceive(fixture.ctx, t, probe.started)
	case "terminal":
		heartbeatReceive(fixture.ctx, t, fixture.capture.blocked)
	default:
		call := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
		switch stage {
		case "native-rpc":
		case "batch":
			call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{}}
			require.Equal(t, 0, heartbeatReceive(fixture.ctx, t, probe.returned))
			require.Equal(t, 1, heartbeatReceive(fixture.ctx, t, probe.returned))
			require.Empty(t, fixture.service.heartbeats)
		case "true-join", "publication":
			call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
			heartbeatReceive(fixture.ctx, t, fixture.capture.blocked)
			if stage == "true-join" {
				require.Empty(t, probe.canceled)
			} else {
				require.True(t, temporal.IsCanceledError(heartbeatReceive(fixture.ctx, t, probe.canceled)))
			}
		default:
			t.Fatalf("unexpected stage %q", stage)
		}
	}
	return fixture
}

func requireNoCompletedTerminal(t *testing.T, fixture *sdkContinuationFixture) {
	t.Helper()
	for len(fixture.service.terminal) != 0 {
		require.NotContains(t, fmt.Sprintf("%T", <-fixture.service.terminal), "Completed")
	}
}

func TestProductionSDKRunCancellationJoinsBlockedStages(t *testing.T) {
	for _, stage := range []string{"registration", "native-rpc", "batch", "true-join", "publication", "terminal"} {
		t.Run(stage, func(t *testing.T) {
			fixture := blockedSDKStage(t, stage, false)
			fixture.stop()
			require.NoError(t, fixture.err)
			require.NoError(t, fixture.closeErr)
			requireNoCompletedTerminal(t, fixture)
			if stage == "terminal" {
				original, invoked := fixture.capture.terminalOriginalContext, fixture.capture.terminalCallContext
				require.NotNil(t, invoked)
				require.True(t, fixture.capture.terminalCanceledAtCapture, "actual invocation context was Run-canceled before capture")
				require.ErrorIs(t, invoked.Err(), context.Canceled)
				require.Same(t, fixture.capture, invoked.Value(sdkInvocationValue{}))
				originalDeadline, originalHasDeadline := original.Deadline()
				invokedDeadline, invokedHasDeadline := invoked.Deadline()
				require.Equal(t, originalHasDeadline, invokedHasDeadline)
				require.Equal(t, originalDeadline, invokedDeadline)
			}
		})
	}
}

func TestProductionSDKInitialUnprogrammedFailureRemainsTerminal(t *testing.T) {
	envelope := new(anypb.Any)
	require.NoError(t, proto.Unmarshal(productionProgram(t, nil, "completed"), envelope))
	program := new(temporalv1.WorkerProgram)
	require.NoError(t, envelope.UnmarshalTo(program))
	key := new(temporalv1.ActivityTaskKey)
	require.NoError(t, program.Rules[0].Key.UnmarshalTo(key))
	key.Scenario = "other"
	var err error
	program.Rules[0].Key, err = anypb.New(key)
	require.NoError(t, err)
	envelope, err = anypb.New(program)
	require.NoError(t, err)
	data, err := proto.Marshal(envelope)
	require.NoError(t, err)
	fixture := startSDKContinuation(t, data, nil, nil)
	terminal := heartbeatReceive(fixture.ctx, t, fixture.service.terminal)
	failed, ok := terminal.(*workflowservice.RespondActivityTaskFailedRequest)
	require.True(t, ok)
	require.Equal(t, worker.Unprogrammed, failed.GetFailure().GetApplicationFailureInfo().GetType())
	fixture.stop()
	require.NoError(t, fixture.err)
	require.NoError(t, fixture.closeErr)
}

func TestProductionSDKActivityDeadlineJoinsBlockedStages(t *testing.T) {
	for _, stage := range []string{"native-rpc", "batch", "true-join", "publication", "terminal"} {
		t.Run(stage, func(t *testing.T) {
			fixture := blockedSDKStage(t, stage, true)
			heartbeatReceive(fixture.ctx, t, fixture.done)
			require.ErrorContains(t, fixture.err, "deadline")
			fixture.stop()
			requireNoCompletedTerminal(t, fixture)
		})
	}
}

func TestProductionSDKBridgeCloseJoinsBlockedStages(t *testing.T) {
	for _, stage := range []string{"registration", "native-rpc", "batch", "true-join", "publication", "terminal"} {
		t.Run(stage, func(t *testing.T) {
			fixture := blockedSDKStage(t, stage, false)
			closeErr := fixture.bridge.Close()
			if closeErr != nil {
				require.ErrorContains(t, closeErr, "outstanding")
			}
			heartbeatReceive(fixture.ctx, t, fixture.done)
			require.ErrorContains(t, fixture.err, "closed")
			fixture.stop()
			requireNoCompletedTerminal(t, fixture)
		})
	}
}

func TestProductionSDKMalformedHeartbeatCannotSelectFalse(t *testing.T) {
	fixture := startSDKContinuation(t, productionProgram(t, []bool{false}, "completed"), nil, nil)
	call := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
	malformed := &workflowservice.RecordActivityTaskHeartbeatResponse{}
	malformed.ProtoReflect().SetUnknown([]byte{0xff})
	call.reply <- heartbeatReply{response: malformed}
	for {
		rpc := heartbeatReceive(fixture.ctx, t, fixture.capture.completed)
		if rpc.Method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
			require.NotNil(t, rpc.CallError)
			require.NotEmpty(t, rpc.TransportError)
			require.Empty(t, fixture.service.terminal)
			break
		}
	}
	fixture.stop()
	require.NoError(t, fixture.err)
}

func TestProductionSDKCrossedHeartbeatRequestFaults(t *testing.T) {
	fixture := startSDKContinuation(t, productionProgram(t, []bool{false}, "completed"), func(method string, request proto.Message) {
		if method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
			request.(*workflowservice.RecordActivityTaskHeartbeatRequest).TaskToken = []byte("crossed-token")
		}
	}, nil)
	heartbeatReceive(fixture.ctx, t, fixture.done)
	require.ErrorContains(t, fixture.err, "assignment")
	fixture.stop()
	for len(fixture.service.terminal) != 0 {
		require.NotContains(t, fmt.Sprintf("%T", <-fixture.service.terminal), "Completed")
	}
}

func productionTask(token []byte, attempt int32, runID string) *workflowservice.PollActivityTaskQueueResponse {
	return &workflowservice.PollActivityTaskQueueResponse{
		TaskToken: append([]byte(nil), token...), Attempt: attempt, ActivityId: "same-activity", ActivityRunId: runID,
		ActivityType: &commonpb.ActivityType{Name: "heartbeat-proof"}, ScheduledTime: timestamppb.Now(), StartedTime: timestamppb.Now(),
		Header: &commonpb.Header{Fields: map[string]*commonpb.Payload{worker.ScenarioHeader: {Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("proof")}}},
	}
}

func combinedProductionProgram(t *testing.T) []byte {
	t.Helper()
	combined := new(temporalv1.WorkerProgram)
	for index, choice := range []struct {
		bits     []bool
		terminal string
	}{{[]bool{false, true}, "completed"}, {[]bool{true, true}, "canceled"}} {
		envelope := new(anypb.Any)
		require.NoError(t, proto.Unmarshal(productionProgram(t, choice.bits, choice.terminal), envelope))
		program := new(temporalv1.WorkerProgram)
		require.NoError(t, envelope.UnmarshalTo(program))
		for _, rule := range program.Rules {
			key := new(temporalv1.ActivityTaskKey)
			require.NoError(t, rule.Key.UnmarshalTo(key))
			key.Attempt += int32(index)
			var err error
			rule.Key, err = anypb.New(key)
			require.NoError(t, err)
		}
		for i, packed := range program.ContinuationDomains {
			domain := new(temporalv1.ActivityTaskSuccessors)
			require.NoError(t, packed.UnmarshalTo(domain))
			domain.Key.Attempt += int32(index)
			var err error
			program.ContinuationDomains[i], err = anypb.New(domain)
			require.NoError(t, err)
		}
		combined.Namespace, combined.TaskQueue = program.Namespace, program.TaskQueue
		combined.Rules = append(combined.Rules, program.Rules...)
		combined.ContinuationDomains = append(combined.ContinuationDomains, program.ContinuationDomains...)
	}
	envelope, err := anypb.New(combined)
	require.NoError(t, err)
	data, err := proto.Marshal(envelope)
	require.NoError(t, err)
	return data
}

func TestProductionSDKConcurrentAssignmentsKeepAttemptAndRun(t *testing.T) {
	tokens := [][]byte{[]byte("first-token"), []byte("second-token")}
	fixture := startSDKContinuation(t, combinedProductionProgram(t), nil, nil, sdkContinuationOptions{tasks: []*workflowservice.PollActivityTaskQueueResponse{productionTask(tokens[0], 7, "first-run"), productionTask(tokens[1], 8, "second-run")}})
	next := map[string]int{}
	bits := map[string][]bool{string(tokens[0]): {false, true}, string(tokens[1]): {true, true}}
	for count := 0; count < 4; count++ {
		call := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
		token := string(call.request.GetTaskToken())
		sequence, ok := bits[token]
		require.True(t, ok)
		index := next[token]
		require.Less(t, index, len(sequence))
		var detail int
		require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(call.request.GetDetails(), &detail))
		require.Equal(t, index, detail)
		call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: sequence[index]}}
		next[token]++
	}
	seen := map[string]bool{}
	for count := 0; count < 2; count++ {
		terminal := heartbeatReceive(fixture.ctx, t, fixture.service.terminal)
		var token []byte
		switch terminal := terminal.(type) {
		case *workflowservice.RespondActivityTaskCompletedRequest:
			token = terminal.GetTaskToken()
			require.Equal(t, tokens[0], token)
		case *workflowservice.RespondActivityTaskCanceledRequest:
			token = terminal.GetTaskToken()
			require.Equal(t, tokens[1], token)
		default:
			t.Fatalf("unexpected terminal %T", terminal)
		}
		require.False(t, seen[string(token)])
		seen[string(token)] = true
	}
	require.Equal(t, int32(2), fixture.service.delivered.Load())
	fixture.stop()
	require.NoError(t, fixture.err)
	require.NoError(t, fixture.closeErr)
}

func TestProductionSDKCollidingTokenAssignmentsRefuse(t *testing.T) {
	fixture := startSDKContinuation(t, combinedProductionProgram(t), nil, nil, sdkContinuationOptions{tasks: []*workflowservice.PollActivityTaskQueueResponse{productionTask(heartbeatToken, 7, "first-run"), productionTask(heartbeatToken, 8, "second-run")}})
	heartbeatReceive(fixture.ctx, t, fixture.done)
	require.ErrorContains(t, fixture.err, "token")
	fixture.stop()
}

func TestProductionSDKOriginalOneHeartbeatSubset(t *testing.T) {
	for _, bit := range []bool{false, true} {
		for _, terminal := range []string{"completed", "failed", "canceled"} {
			if !bit && terminal == "canceled" {
				continue
			}
			t.Run(fmt.Sprintf("%t/%s", bit, terminal), func(t *testing.T) {
				envelope := new(anypb.Any)
				require.NoError(t, proto.Unmarshal(productionProgram(t, []bool{true}, terminal), envelope))
				program := new(temporalv1.WorkerProgram)
				require.NoError(t, envelope.UnmarshalTo(program))
				domain := new(temporalv1.ActivityTaskSuccessors)
				require.NoError(t, program.ContinuationDomains[0].UnmarshalTo(domain))
				domain.CancelRequested = []bool{false, true}
				var err error
				program.ContinuationDomains[0], err = anypb.New(domain)
				require.NoError(t, err)
				falseTerminal := terminal
				if terminal == "canceled" {
					falseTerminal = "completed"
				}
				require.NoError(t, proto.Unmarshal(productionProgram(t, []bool{false}, falseTerminal), envelope))
				falseProgram := new(temporalv1.WorkerProgram)
				require.NoError(t, envelope.UnmarshalTo(falseProgram))
				program.Rules = append(program.Rules, falseProgram.Rules[1])
				envelope, err = anypb.New(program)
				require.NoError(t, err)
				data, err := proto.Marshal(envelope)
				require.NoError(t, err)
				fixture := startSDKContinuation(t, data, nil, nil)
				call := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
				call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: bit}}
				checkHeartbeatTerminal(t, heartbeatReceive(fixture.ctx, t, fixture.service.terminal), terminal)
				fixture.stop()
				require.NoError(t, fixture.err)
				require.NoError(t, fixture.closeErr)
				require.Empty(t, fixture.service.heartbeats)
			})
		}
	}
}

func TestProductionSDKMutatedHeartbeatDetailsCannotAdvance(t *testing.T) {
	fixture := startSDKContinuation(t, productionProgram(t, []bool{false}, "completed"), func(method string, request proto.Message) {
		if method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
			request.(*workflowservice.RecordActivityTaskHeartbeatRequest).Details.Payloads[0].Data = []byte("mutated-details")
		}
	}, nil)
	call := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
	require.Equal(t, []byte("mutated-details"), call.request.GetDetails().GetPayloads()[0].GetData())
	call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{}}
	heartbeatReceive(fixture.ctx, t, fixture.done)
	require.ErrorContains(t, fixture.err, "heartbeat")
	fixture.stop()
	for len(fixture.service.terminal) != 0 {
		require.NotContains(t, fmt.Sprintf("%T", <-fixture.service.terminal), "Completed")
	}
}

func TestProductionSDKWrongActualTerminalMethodFaults(t *testing.T) {
	fixture := startSDKContinuation(t, productionProgram(t, nil, "completed"), nil, nil, sdkContinuationOptions{rewriteTerminal: true})
	actual := heartbeatReceive(fixture.ctx, t, fixture.service.terminal)
	require.IsType(t, &workflowservice.RespondActivityTaskFailedRequest{}, actual)
	heartbeatReceive(fixture.ctx, t, fixture.done)
	require.ErrorContains(t, fixture.err, "matching outstanding")
	fixture.stop()
}

func TestProductionSDKRefusesPresentEmptyHeartbeatDetails(t *testing.T) {
	envelope := new(anypb.Any)
	require.NoError(t, proto.Unmarshal(productionProgram(t, []bool{false}, "completed"), envelope))
	program := new(temporalv1.WorkerProgram)
	require.NoError(t, envelope.UnmarshalTo(program))
	var err error
	program.Rules[0].Respond[0], err = anypb.New(&workflowservice.RecordActivityTaskHeartbeatRequest{Details: &commonpb.Payloads{}})
	require.NoError(t, err)
	envelope, err = anypb.New(program)
	require.NoError(t, err)
	data, err := proto.Marshal(envelope)
	require.NoError(t, err)
	require.ErrorContains(t, Validate(data), "present empty heartbeat details")
}

type terminalQueueContext struct {
	context.Context
	registered chan struct{}
}

func (c terminalQueueContext) Deadline() (time.Time, bool) {
	select {
	case c.registered <- struct{}{}:
	default:
	}
	return c.Context.Deadline()
}

func TestTerminalEnqueueBackpressureJoinsWithoutPublishing(t *testing.T) {
	for _, ending := range []string{"run-cancel", "bridge-close"} {
		t.Run(ending, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			bridge := worker.NewBridge(ctx)
			request, err := anypb.New(&workflowservice.PollActivityTaskQueueRequest{Namespace: heartbeatNamespace, TaskQueue: &taskqueuepb.TaskQueue{Name: "production-queue"}})
			require.NoError(t, err)
			task := productionTask(heartbeatToken, 7, "queue-run")
			task.WorkflowNamespace = heartbeatNamespace
			response, err := anypb.New(task)
			require.NoError(t, err)
			require.NoError(t, bridge.Observe(ctx, worker.RPC{Invocation: 1, Sequence: 1, Stage: worker.RPCStarted, Method: workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, Request: request}))
			require.NoError(t, bridge.Observe(ctx, worker.RPC{Invocation: 1, Sequence: 2, Stage: worker.RPCCompleted, Method: workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, Request: request, Response: response, Sent: true, Answered: true}))
			assignment, err := bridge.Bind(worker.IdentityFromPoll(heartbeatNamespace, "production-queue", task))
			require.NoError(t, err)
			payloads, err := converter.GetDefaultDataConverter().ToPayloads("proof-completed")
			require.NoError(t, err)
			terminal, err := anypb.New(&workflowservice.RespondActivityTaskCompletedRequest{Result: payloads})
			require.NoError(t, err)
			queue := make(chan terminalExpectation, 1)
			queue <- terminalExpectation{}
			registered := make(chan struct{}, 1)
			finished := make(chan error, 1)
			go func() {
				_, err := enqueueTerminal(terminalQueueContext{Context: ctx, registered: registered}, bridge, assignment, terminal, heartbeatNamespace, heartbeatToken, queue)
				finished <- err
			}()
			heartbeatReceive(ctx, t, registered)
			require.Len(t, queue, 1)
			select {
			case <-finished:
				t.Fatal("full terminal queue did not apply backpressure")
			default:
			}
			if ending == "run-cancel" {
				cancel()
			} else {
				require.ErrorContains(t, bridge.Close(), "outstanding")
			}
			cleanup, cleanupCancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Second)
			defer cleanupCancel()
			err = heartbeatReceive(cleanup, t, finished)
			if ending == "run-cancel" {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.ErrorContains(t, err, "closed")
			}
			require.Len(t, queue, 1)
			require.Nil(t, (<-queue).wait)
			cancel()
			if ending == "run-cancel" {
				require.NoError(t, bridge.Close())
			} else {
				require.ErrorContains(t, bridge.Close(), "outstanding")
			}
		})
	}
}

func TestTerminalCoordinatorValidatesCapturedResultsDuringShutdown(t *testing.T) {
	for _, ending := range []string{"run", "owner-first", "active-deadline"} {
		t.Run(ending, func(t *testing.T) {
			for _, outcome := range []string{"valid", "mutated-request", "nonempty-failed-ack", "malformed-response", "transport-error", "ambiguous-status", "malformed-status", "server-canceled", "server-deadline", "local-canceled", "retry-success", "failure-then-local-canceled", "unfinished"} {
				if ending == "active-deadline" && outcome != "unfinished" {
					continue
				}
				t.Run(outcome, func(t *testing.T) {
					run, cancel := context.WithCancel(t.Context())
					defer cancel()
					owner, end := run, cancel
					if ending == "owner-first" {
						var cancelOwner context.CancelFunc
						owner, cancelOwner = context.WithCancel(t.Context())
						defer cancelOwner()
						end = cancelOwner
					}
					callCtx, cancelCall := context.WithCancel(run)
					defer cancelCall()
					bridge := worker.NewBridge(owner)
					pack := func(message proto.Message) *anypb.Any {
						value, err := anypb.New(message)
						require.NoError(t, err)
						return value
					}
					poll := &workflowservice.PollActivityTaskQueueRequest{Namespace: heartbeatNamespace, TaskQueue: &taskqueuepb.TaskQueue{Name: "production-queue"}}
					task := productionTask(heartbeatToken, 7, "close-run")
					task.WorkflowNamespace = heartbeatNamespace
					require.NoError(t, bridge.Observe(run, worker.RPC{Invocation: 1, Sequence: 1, Stage: worker.RPCStarted, Method: workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, Request: pack(poll)}))
					require.NoError(t, bridge.Observe(run, worker.RPC{Invocation: 1, Sequence: 2, Stage: worker.RPCCompleted, Method: workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName, Request: pack(poll), Response: pack(task), Sent: true, Answered: true}))
					assignment, err := bridge.Bind(worker.IdentityFromPoll(heartbeatNamespace, "production-queue", task))
					require.NoError(t, err)
					method := workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName
					expected := pack(&workflowservice.RespondActivityTaskCompletedRequest{Namespace: heartbeatNamespace, TaskToken: heartbeatToken})
					actual := proto.CloneOf(expected)
					response := pack(&workflowservice.RespondActivityTaskCompletedResponse{})
					if outcome == "mutated-request" {
						actual = pack(&workflowservice.RespondActivityTaskCompletedRequest{Namespace: heartbeatNamespace, TaskToken: heartbeatToken, Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("mutated")}}}})
					}
					if outcome == "nonempty-failed-ack" {
						method = workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName
						expected = pack(&workflowservice.RespondActivityTaskFailedRequest{Namespace: heartbeatNamespace, TaskToken: heartbeatToken})
						actual = proto.CloneOf(expected)
						response = pack(&workflowservice.RespondActivityTaskFailedResponse{Failures: []*failurepb.Failure{{Message: "rejected"}}})
					}
					wait, err := assignment.BeginTerminal(method)
					require.NoError(t, err)
					require.NoError(t, bridge.Observe(callCtx, worker.RPC{Invocation: 2, Sequence: 3, Stage: worker.RPCStarted, Method: method, Request: actual}))
					completed := worker.RPC{Invocation: 2, Sequence: 4, Stage: worker.RPCCompleted, Method: method, Request: actual, Response: response, Sent: true, Answered: true}
					switch outcome {
					case "transport-error", "retry-success", "failure-then-local-canceled":
						completed.CallError = pack(status.New(codes.Unavailable, "retryable terminal failure").Proto())
					case "ambiguous-status":
						completed.CallError = pack(status.New(codes.Internal, "unattributed failure").Proto())
					case "malformed-status":
						completed.CallError = pack(status.New(codes.Canceled, "malformed").Proto())
						completed.CallError.Value = []byte{0xff}
					case "server-deadline":
						completed.CallError = pack(status.New(codes.DeadlineExceeded, "server deadline with live call context").Proto())
					case "server-canceled", "local-canceled":
						completed.CallError = pack(status.New(codes.Canceled, "canceled terminal").Proto())
						if outcome == "local-canceled" {
							cancelCall()
						}
					case "malformed-response":
						completed.Response.Value = []byte{0xff}
					case "valid", "mutated-request", "nonempty-failed-ack", "unfinished":
					default:
						t.Fatal("unknown terminal shutdown outcome")
					}
					if outcome != "unfinished" {
						require.NoError(t, bridge.Observe(callCtx, completed))
					}
					if outcome == "retry-success" || outcome == "failure-then-local-canceled" {
						require.NoError(t, bridge.Observe(callCtx, worker.RPC{Invocation: 3, Sequence: 5, Stage: worker.RPCStarted, Method: method, Request: actual}))
						completed.Invocation, completed.Sequence, completed.CallError = 3, 6, nil
						if outcome == "failure-then-local-canceled" {
							cancelCall()
							completed.CallError = pack(status.New(codes.Canceled, "local shutdown").Proto())
						}
						require.NoError(t, bridge.Observe(callCtx, completed))
					}
					queue := make(chan terminalExpectation, 1)
					deadline := time.Now().Add(time.Second)
					if ending == "active-deadline" {
						deadline = time.Now().Add(-time.Second)
					}
					queue <- terminalExpectation{wait: wait, expected: expected, deadline: deadline}
					resume, joined := make(chan struct{}), make(chan struct{})
					var failures []error
					go func() {
						defer close(joined)
						<-resume
						coordinateTerminals(run, owner, bridge, queue, func(err error) { failures = append(failures, err) })
					}()
					if ending != "active-deadline" {
						end()
					}
					if ending == "owner-first" {
						require.NoError(t, run.Err(), "derived Run has not yet observed owner cancellation")
					}
					close(queue)
					close(resume)
					heartbeatReceive(t.Context(), t, joined)
					if ending == "active-deadline" {
						require.Len(t, failures, 1)
						require.ErrorIs(t, failures[0], context.DeadlineExceeded)
					} else {
						switch outcome {
						case "valid", "unfinished", "local-canceled", "retry-success":
							require.Empty(t, failures)
						default:
							require.Len(t, failures, 1)
							require.ErrorContains(t, failures[0], "SDK terminal completion")
							if outcome == "mutated-request" {
								require.ErrorContains(t, failures[0], "actual terminal payload or assignment")
							}
							if outcome == "nonempty-failed-ack" {
								require.ErrorContains(t, failures[0], "failures")
							}
						}
					}
					end()
					require.NoError(t, bridge.Close())
				})
			}
		})
	}
}

func TestShutdownCancellationRetainsJoinedValidationFailure(t *testing.T) {
	run, cancel := context.WithCancel(t.Context())
	cancel()
	validation := errors.New("terminal acknowledgment validation failed")
	require.False(t, shutdownCancellation(run, run, errors.Join(context.Canceled, validation)))
	require.True(t, shutdownCancellation(run, run, errors.Join(context.Canceled, context.DeadlineExceeded)))
}
