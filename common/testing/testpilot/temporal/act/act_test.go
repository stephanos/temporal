package act

import (
	"context"
	"maps"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	runv1 "umpire.local/model/gen/umpire/run/v1"
	"umpire.local/model/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// fakeServer answers WorkflowService calls with scripted bytes or statuses and records the exact
// bytes it received.
type fakeServer struct {
	address string
	respond func(method string) ([]byte, error)

	mu       sync.Mutex
	received map[string][][]byte
	metadata []metadata.MD
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
		incoming, _ := metadata.FromIncomingContext(stream.Context())
		f.mu.Lock()
		f.received[method] = append(f.received[method], in.data)
		f.metadata = append(f.metadata, incoming)
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

// A Transport's dial options replace the plaintext default: the Driver's calls carry what they add,
// and options with no transport security of their own cannot dial at all.
func TestATransportReplacesThePlaintextDefault(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return []byte{}, nil })
	probe := func(ctx context.Context, method string, request, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
		return invoke(metadata.AppendToOutgoingContext(ctx, "x-transport", "probe"), method, request, reply, conn, options...)
	}
	transport := Transport{DialOptions: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithChainUnaryInterceptor(probe)}}
	x := start(t, testDriver(t, server.address, Options{Transport: transport}), 1, packed(t, cancelRequest))
	event := next(t, x)
	observation(t, event)
	server.mu.Lock()
	require.Equal(t, []string{"probe"}, server.metadata[0].Get("x-transport"), "the call went through the Transport's options")
	server.mu.Unlock()
	require.NoError(t, x.Respond(t.Context(), event.ID, explained(false)))

	_, err := newDriver(Options{Address: server.address, Namespace: "namespace", TaskQueue: "queue",
		Transport: Transport{DialOptions: []grpc.DialOption{grpc.WithChainUnaryInterceptor(probe)}}}.withDefaults())
	require.Error(t, err, "the Transport's options are used alone, never beside the plaintext default")
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
	owned := packed(t, &realizationv1.Reply{Request: packed(t, &workflowservice.RespondActivityTaskCompletedRequest{
		TaskToken: []byte("token"), Namespace: "namespace", Identity: "worker",
		Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{payload}},
	})})
	_, err = deliverable(owned)
	require.NoError(t, err, "the SDK supplies task_token, namespace and identity itself")
	versioned := packed(t, &realizationv1.Reply{Request: packed(t, &workflowservice.RespondActivityTaskCompletedRequest{
		Result:            &commonpb.Payloads{Payloads: []*commonpb.Payload{payload}},
		DeploymentOptions: &deploymentpb.WorkerDeploymentOptions{BuildId: "build"},
	})})
	_, err = deliverable(versioned)
	require.ErrorContains(t, err, "the SDK cannot send the reply's deployment_options")
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
	token := "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse.task_token"
	queue := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.task_queue.name"
	env := map[string]string{
		EnvAddress: "localhost:7233", EnvNamespace: "namespace", EnvTaskQueue: "queue", EnvCreate: "true",
		EnvRPCTimeout: "3s", EnvWorkerPollTimeout: "4s", EnvRedact: token + ", " + queue,
	}
	opts, err := Configure(func(name string) string { return env[name] })
	require.NoError(t, err)
	require.Equal(t, Options{
		Address: "localhost:7233", Namespace: "namespace", TaskQueue: "queue", Create: true,
		RPCTimeout: 3 * time.Second, WorkerPollTimeout: 4 * time.Second, Redact: []string{token, queue},
	}, opts)
	for _, bad := range []struct{ name, value, want string }{
		{EnvAddress, "", "required"},
		{EnvTaskQueue, "", "required"},
		{EnvCreate, "maybe", EnvCreate},
		{EnvRPCTimeout, "soon", EnvRPCTimeout},
		{EnvWorkerPollTimeout, "1s", "at least 2s"},
		{EnvRedact, "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse.tasktoken", `has no field "tasktoken"`},
		{EnvRedact, "temporal.api.workflowservice.v1.NoSuchResponse.task_token", "names no known message"},
		{EnvRedact, queue + ".kind", "is not a message"},
	} {
		changed := maps.Clone(env)
		changed[bad.name] = bad.value
		_, err := Configure(func(name string) string { return changed[name] })
		require.ErrorContains(t, err, bad.want, "%s=%s", bad.name, bad.value)
	}
}

func TestAnOutOfRangeStatusIsMalformed(t *testing.T) {
	server := newFakeServer(t, func(string) ([]byte, error) { return nil, status.Error(codes.Code(20), "no such code") })
	x := start(t, testDriver(t, server.address, Options{}), 1, packed(t, cancelRequest))
	event := next(t, x)
	require.Equal(t, runner.EventFault, event.Kind)
	require.Contains(t, event.Fault, "malformed evidence: status code 20")
}

func TestStartRefusesAReadWithoutItsMechanics(t *testing.T) {
	d := testDriver(t, unreachable(t), Options{})
	for _, instruction := range []*temporalv1.ObserveActivity{
		{PollIntervalMs: 0, BackoffMs: 1, ReadTimeoutMs: 1},
		{PollIntervalMs: 1, BackoffMs: 0, ReadTimeoutMs: 1},
		{PollIntervalMs: 1, BackoffMs: 1, ReadTimeoutMs: 0},
	} {
		instruction.Request = packed(t, &workflowservice.DescribeActivityExecutionRequest{})
		_, err := d.Start(t.Context(), 1, packed(t, instruction))
		require.ErrorContains(t, err, "needs a poll interval, a backoff and a read timeout")
	}
}

func completedReply(t *testing.T, payloads ...*commonpb.Payload) *anypb.Any {
	return packed(t, &realizationv1.Reply{Request: packed(t, &workflowservice.RespondActivityTaskCompletedRequest{
		Result: &commonpb.Payloads{Payloads: payloads},
	})})
}
