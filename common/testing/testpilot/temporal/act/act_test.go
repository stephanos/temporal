package act

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/umpire/model/v1"
	realizationv1 "umpire.local/model/umpire/realization/v1"
	runv1 "umpire.local/model/umpire/run/v1"
	"umpire.local/model/umpire/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// fakeServer answers WorkflowService calls with scripted bytes or statuses and records the exact
// bytes it received.
type fakeServer struct {
	address string
	respond func(method string) ([]byte, error)

	mu       sync.Mutex
	received map[string][][]byte
}

func newFakeServer(t *testing.T, respond func(method string) ([]byte, error)) *fakeServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	f := &fakeServer{address: listener.Addr().String(), respond: respond, received: map[string][][]byte{}}
	server := grpc.NewServer(grpc.ForceServerCodecV2(rawCodec{}), grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
		method, _ := grpc.MethodFromServerStream(stream)
		in := new(raw)
		if err := stream.RecvMsg(in); err != nil {
			return err
		}
		f.mu.Lock()
		f.received[method] = append(f.received[method], in.data)
		f.mu.Unlock()
		data, err := f.respond(method)
		if err != nil {
			return err
		}
		return stream.SendMsg(&raw{data: data})
	}))
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	return f
}

func (f *fakeServer) calls(method string) [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.received[method]...)
}

// unreachable is an address nothing listens on.
func unreachable(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	return address
}

func marshal(t *testing.T, m proto.Message) []byte {
	t.Helper()
	data, err := proto.Marshal(m)
	require.NoError(t, err)
	return data
}

func packed(t *testing.T, m proto.Message) *anypb.Any {
	t.Helper()
	value, err := anypb.New(m)
	require.NoError(t, err)
	return value
}

func testDriver(t *testing.T, address string, opts Options) *Driver {
	t.Helper()
	opts.Address, opts.Namespace, opts.TaskQueue = address, "namespace", "queue"
	d, err := newDriver(opts.withDefaults())
	require.NoError(t, err)
	d.ensureWorker = func(context.Context) error { return nil }
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func start(t *testing.T, d *Driver, request uint64, payload *anypb.Any) runner.Exchange {
	t.Helper()
	x, err := d.Start(t.Context(), request, payload)
	require.NoError(t, err)
	t.Cleanup(func() { _ = x.Close() })
	return x
}

func next(t *testing.T, x runner.Exchange) runner.Event {
	t.Helper()
	select {
	case event, open := <-x.Events():
		require.True(t, open, "the event stream closed")
		return event
	case <-time.After(30 * time.Second):
		t.Fatal("no event")
		return runner.Event{}
	}
}

func observation(t *testing.T, event runner.Event) *realizationv1.Observation {
	t.Helper()
	require.Equal(t, runner.EventObservation, event.Kind, "event: %+v", event)
	value := new(realizationv1.Observation)
	require.NoError(t, event.Payload.UnmarshalTo(value))
	return value
}

func explained(continuePolling bool) runner.Decision {
	return runner.Decision{Judgment: &modelv1.JudgeResponse{
		Judgment:        &modelv1.JudgeResponse_Explained{Explained: &modelv1.Explanation{}},
		ContinuePolling: continuePolling,
	}}
}

var cancelRequest = &workflowservice.RequestCancelActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity", RunId: "run"}

func describeInstruction(t *testing.T, readTimeout time.Duration, workerTasks uint32) *anypb.Any {
	t.Helper()
	return packed(t, &temporalv1.ObserveActivity{
		Request:        packed(t, &workflowservice.DescribeActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity", RunId: "run"}),
		PollIntervalMs: 1, ReadTimeoutMs: uint64(readTimeout.Milliseconds()), BackoffMs: 1, WorkerTasks: workerTasks,
	})
}

func describeResponse(t *testing.T) []byte {
	return marshal(t, &workflowservice.DescribeActivityExecutionResponse{RunId: "run"})
}

func TestUnaryObservesTheActualInvocation(t *testing.T) {
	response := []byte{} // an empty RequestCancelActivityExecutionResponse
	server := newFakeServer(t, func(string) ([]byte, error) { return response, nil })
	proposal := packed(t, cancelRequest)
	x := start(t, testDriver(t, server.address, Options{}), 7, proposal)
	event := next(t, x)
	require.Equal(t, uint64(1), event.ID)
	require.Equal(t, uint64(7), event.Request)
	got := observation(t, event)
	require.Equal(t, workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName, got.GetMethod())
	require.True(t, proto.Equal(proposal, got.GetRequest()), "the observed request is the proposal")
	require.Equal(t, "type.googleapis.com/temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse", got.GetResponse().GetTypeUrl())
	require.Equal(t, [][]byte{proposal.GetValue()}, server.calls(got.GetMethod()), "the proposal's bytes are sent unchanged")
	require.NoError(t, x.Respond(t.Context(), event.ID, explained(false)))
	done := next(t, x)
	require.Equal(t, runner.EventDone, done.Kind)
	require.Equal(t, uint64(2), done.ID)
	_, open := <-x.Events()
	require.False(t, open)
}

func TestAServerStatusIsEvidence(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return nil, status.Error(codes.NotFound, "activity not found") })
	x := start(t, testDriver(t, server.address, Options{}), 1, packed(t, cancelRequest))
	got := observation(t, next(t, x))
	require.Equal(t, int32(codes.NotFound), got.GetError().GetCode())
	require.Equal(t, "activity not found", got.GetError().GetMessage())
}

func TestAnUnansweredCallIsAFault(t *testing.T) {
	x := start(t, testDriver(t, unreachable(t), Options{}), 1, packed(t, cancelRequest))
	event := next(t, x)
	require.Equal(t, runner.EventFault, event.Kind)
	require.Contains(t, event.Fault, "the server did not answer")
}

func TestMalformedEvidenceIsRefusedBeforeJudge(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return []byte{0xff}, nil })
	x := start(t, testDriver(t, server.address, Options{}), 1, packed(t, cancelRequest))
	event := next(t, x)
	require.Equal(t, runner.EventFault, event.Kind, "no Observation reaches the Runner")
	require.Contains(t, event.Fault, "malformed evidence")
}

func TestACallThatIsNotAReadCannotContinue(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return nil, nil })
	x := start(t, testDriver(t, server.address, Options{}), 1, packed(t, cancelRequest))
	event := next(t, x)
	require.NoError(t, x.Respond(t.Context(), event.ID, explained(true)))
	fault := next(t, x)
	require.Equal(t, runner.EventFault, fault.Kind)
	require.Contains(t, fault.Fault, "not a read")
}

func TestStartRefusesWhatNoMethodTakes(t *testing.T) {
	d := testDriver(t, unreachable(t), Options{})
	_, err := d.Start(t.Context(), 1, packed(t, &commonpb.Payload{}))
	require.ErrorContains(t, err, "input of no WorkflowService method")
	_, err = d.Start(t.Context(), 1, packed(t, &temporalv1.ObserveActivity{}))
	require.ErrorContains(t, err, "names no read")
}

func TestReadRepeatsUntilTheModelAccepts(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return describeResponse(t), nil })
	x := start(t, testDriver(t, server.address, Options{}), 3, describeInstruction(t, 30*time.Second, 0))
	for i := 1; i <= 3; i++ {
		event := next(t, x)
		require.Equal(t, uint64(i), event.ID)
		require.Equal(t, workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName, observation(t, event).GetMethod())
		require.NoError(t, x.Respond(t.Context(), event.ID, explained(i < 3)))
	}
	require.Equal(t, runner.EventDone, next(t, x).Kind)
	reads := server.calls(workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName)
	require.Len(t, reads, 3)
}

func TestAReadTimeoutWithoutAcceptanceIsAFault(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return describeResponse(t), nil })
	x := start(t, testDriver(t, server.address, Options{}), 1, describeInstruction(t, 200*time.Millisecond, 0))
	for {
		event := next(t, x)
		if event.Kind == runner.EventFault {
			require.Equal(t, "read timeout without Model acceptance", event.Fault)
			return
		}
		require.NoError(t, x.Respond(t.Context(), event.ID, explained(true)))
	}
}

func TestAnUnansweredReadIsRetriedUntilTheReadTimeout(t *testing.T) {
	x := start(t, testDriver(t, unreachable(t), Options{}), 1, describeInstruction(t, 200*time.Millisecond, 0))
	event := next(t, x)
	require.Equal(t, runner.EventFault, event.Kind)
	require.Equal(t, "read timeout without Model acceptance", event.Fault)
}

func TestTheDriverCapsTheInstructionsReadTimeout(t *testing.T) {
	x := start(t, testDriver(t, unreachable(t), Options{MaxReadTimeout: 100 * time.Millisecond}), 1, describeInstruction(t, time.Hour, 0))
	require.Equal(t, "read timeout without Model acceptance", next(t, x).Fault)
}

func TestRespondRefusesWhatNoEventAwaits(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return nil, nil })
	x := start(t, testDriver(t, server.address, Options{}), 1, packed(t, cancelRequest))
	event := next(t, x)
	require.ErrorContains(t, x.Respond(t.Context(), 9, explained(false)), "awaits no decision")
	require.ErrorContains(t, x.Respond(t.Context(), event.ID, runner.Decision{Reply: packed(t, &realizationv1.Reply{})}), "does not answer")
	require.NoError(t, x.Respond(t.Context(), event.ID, explained(false)))
	require.ErrorContains(t, x.Respond(t.Context(), event.ID, explained(false)), "awaits no decision")
	require.NoError(t, x.Close())
	require.ErrorContains(t, x.Respond(t.Context(), event.ID, explained(false)), "closed")
}

// poll plays the SDK worker's poll through the gate, returning the task with token when the
// gate admits it.
func poll(ctx context.Context, d *Driver, token string) error {
	invoke := func(_ context.Context, _ string, _, reply any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		if token != "" {
			task := reply.(*workflowservice.PollActivityTaskQueueResponse)
			task.TaskToken, task.ActivityId, task.Attempt = []byte(token), "activity", 1
		}
		return nil
	}
	return d.gate.intercept(ctx, workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName,
		new(workflowservice.PollActivityTaskQueueRequest), new(workflowservice.PollActivityTaskQueueResponse), nil, invoke)
}

func completedReply(t *testing.T, payloads ...*commonpb.Payload) *anypb.Any {
	return packed(t, &realizationv1.Reply{Request: packed(t, &workflowservice.RespondActivityTaskCompletedRequest{
		Result: &commonpb.Payloads{Payloads: payloads},
	})})
}

func TestAWorkerTaskIsTakenOnlyBetweenReadsAndPausesThem(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return describeResponse(t), nil })
	d := testDriver(t, server.address, Options{})
	polled := make(chan error, 1)
	go func() { polled <- poll(t.Context(), d, "token") }()
	x := start(t, d, 2, describeInstruction(t, 30*time.Second, 1))
	first := next(t, x)
	require.Equal(t, runner.EventObservation, first.Kind)
	select {
	case err := <-polled:
		t.Fatalf("the gate admitted a poll before the Model asked to continue: %v", err)
	default:
	}
	require.NoError(t, x.Respond(t.Context(), first.ID, explained(true)))
	require.NoError(t, <-polled)
	type result struct {
		payload *commonpb.Payload
		err     error
	}
	delivered := make(chan result, 1)
	go func() {
		value, err := d.deliver(t.Context(), []byte("token"))
		delivered <- result{value.Payload(), err}
	}()
	callback := next(t, x)
	require.Equal(t, runner.EventCallback, callback.Kind)
	require.Equal(t, uint64(1), callback.Call)
	envelope := new(realizationv1.Callback)
	require.NoError(t, callback.Payload.UnmarshalTo(envelope))
	task := new(workflowservice.PollActivityTaskQueueResponse)
	require.NoError(t, envelope.GetTask().UnmarshalTo(task))
	require.Equal(t, []byte("token"), task.GetTaskToken(), "the callback carries the captured task")
	require.Len(t, server.calls(workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName), 1, "reads pause while a task is unanswered")
	payload := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Complete")}
	require.NoError(t, x.Respond(t.Context(), callback.ID, runner.Decision{Reply: completedReply(t, payload)}))
	got := <-delivered
	require.NoError(t, got.err)
	require.True(t, proto.Equal(payload, got.payload))
	second := next(t, x)
	require.Equal(t, runner.EventObservation, second.Kind)
	require.NoError(t, x.Respond(t.Context(), second.ID, explained(true)))
	third := next(t, x)
	require.Equal(t, runner.EventObservation, third.Kind, "one task per worker poll instruction")
	require.NoError(t, x.Respond(t.Context(), third.ID, explained(false)))
	require.Equal(t, runner.EventDone, next(t, x).Kind)
}

func TestAReadWithoutWorkerTasksNeverAdmitsAPoll(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return describeResponse(t), nil })
	d := testDriver(t, server.address, Options{})
	ctx, cancel := context.WithCancel(t.Context())
	polled := make(chan error, 1)
	go func() { polled <- poll(ctx, d, "token") }()
	x := start(t, d, 1, describeInstruction(t, 30*time.Second, 0))
	for i := 0; i < 3; i++ {
		event := next(t, x)
		require.NoError(t, x.Respond(t.Context(), event.ID, explained(i < 2)))
	}
	require.Equal(t, runner.EventDone, next(t, x).Kind)
	cancel()
	require.ErrorIs(t, <-polled, context.Canceled)
}

func TestAnUncapturedTaskIsRefusedBeforeAnswer(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return describeResponse(t), nil })
	d := testDriver(t, server.address, Options{})
	polled := make(chan error, 1)
	go func() { polled <- poll(t.Context(), d, "token") }()
	x := start(t, d, 1, describeInstruction(t, 30*time.Second, 1))
	require.NoError(t, x.Respond(t.Context(), next(t, x).ID, explained(true)))
	require.NoError(t, <-polled)
	go func() { _, _ = d.deliver(t.Context(), []byte("another token")) }()
	event := next(t, x)
	require.Equal(t, runner.EventFault, event.Kind)
	require.Contains(t, event.Fault, "malformed evidence")
}

func TestARefusedCallbackFailsTheActivity(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return describeResponse(t), nil })
	d := testDriver(t, server.address, Options{})
	polled := make(chan error, 1)
	go func() { polled <- poll(t.Context(), d, "token") }()
	x := start(t, d, 1, describeInstruction(t, 30*time.Second, 1))
	require.NoError(t, x.Respond(t.Context(), next(t, x).ID, explained(true)))
	require.NoError(t, <-polled)
	delivered := make(chan error, 1)
	go func() {
		_, err := d.deliver(t.Context(), []byte("token"))
		delivered <- err
	}()
	callback := next(t, x)
	require.NoError(t, x.Respond(t.Context(), callback.ID, runner.Decision{Refusal: &modelv1.Refusal{Code: modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED, Message: "unsupported callback policy"}}))
	var failure *temporal.ApplicationError
	require.ErrorAs(t, <-delivered, &failure)
	require.True(t, failure.NonRetryable())
	_, open := <-x.Events()
	require.False(t, open, "a refusal ends the exchange")
}

func TestRepliesTheSDKCanDeliver(t *testing.T) {
	payload := &commonpb.Payload{Data: []byte("Complete")}
	got, err := deliverable(completedReply(t, payload))
	require.NoError(t, err)
	require.True(t, proto.Equal(payload, got.value.Payload()))
	_, err = deliverable(completedReply(t, payload, payload))
	require.ErrorContains(t, err, "one result payload")
	failed := packed(t, &realizationv1.Reply{Request: packed(t, &workflowservice.RespondActivityTaskFailedRequest{
		Failure: &failurepb.Failure{Message: "Fail", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{
			ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "MiniActivityFailure", NonRetryable: true},
		}},
	})})
	got, err = deliverable(failed)
	require.NoError(t, err)
	var failure *temporal.ApplicationError
	require.ErrorAs(t, got.err, &failure)
	require.Equal(t, "MiniActivityFailure", failure.Type())
	require.True(t, failure.NonRetryable())
	_, err = deliverable(packed(t, &realizationv1.Reply{Request: packed(t, &workflowservice.RespondActivityTaskCanceledRequest{})}))
	require.ErrorContains(t, err, "cannot send")
}

func TestRedactionRemovesNamedFieldsThroughPackedEvidence(t *testing.T) {
	task := packed(t, &workflowservice.PollActivityTaskQueueResponse{TaskToken: []byte("secret"), ActivityId: "activity"})
	event := &runv1.Event{Kind: &runv1.Event_Answer{Answer: &runv1.Answer{Request: &modelv1.AnswerRequest{
		Callback: packed(t, &realizationv1.Callback{Task: task}),
	}}}}
	d := &Driver{opts: Options{Redact: []string{"temporal.api.workflowservice.v1.PollActivityTaskQueueResponse.task_token"}}}
	require.True(t, d.Redact(event))
	callback := new(realizationv1.Callback)
	require.NoError(t, event.GetAnswer().GetRequest().GetCallback().UnmarshalTo(callback))
	got := new(workflowservice.PollActivityTaskQueueResponse)
	require.NoError(t, callback.GetTask().UnmarshalTo(got))
	require.Empty(t, got.GetTaskToken())
	require.Equal(t, "activity", got.GetActivityId())
	require.False(t, d.Redact(event), "nothing is left to remove")
	require.False(t, (&Driver{}).Redact(event))
}

func TestConfigureReadsTheEnvironment(t *testing.T) {
	env := map[string]string{
		EnvAddress: "localhost:7233", EnvNamespace: "namespace", EnvTaskQueue: "queue", EnvCreate: "true",
		EnvRPCTimeout: "3s", EnvWorkerPollTimeout: "4s", EnvRedact: "a.B.c, d.E.f",
	}
	opts, err := Configure(func(name string) string { return env[name] })
	require.NoError(t, err)
	require.Equal(t, Options{
		Address: "localhost:7233", Namespace: "namespace", TaskQueue: "queue", Create: true,
		RPCTimeout: 3 * time.Second, WorkerPollTimeout: 4 * time.Second, Redact: []string{"a.B.c", "d.E.f"},
	}, opts)
	for name, value := range map[string]string{EnvAddress: "", EnvTaskQueue: "", EnvCreate: "maybe", EnvRPCTimeout: "soon", EnvRedact: "field"} {
		changed := map[string]string{}
		for k, v := range env {
			changed[k] = v
		}
		changed[name] = value
		_, err := Configure(func(name string) string { return changed[name] })
		require.Error(t, err, name)
	}
}
