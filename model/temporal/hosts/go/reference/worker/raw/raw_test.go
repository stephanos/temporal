package raw

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	"umpire.local/temporal/reference/worker"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

type rawService struct {
	workflowservice.UnimplementedWorkflowServiceServer
	mu               sync.Mutex
	polls            int
	heartbeats       []*workflowservice.RecordActivityTaskHeartbeatRequest
	terminals        []proto.Message
	bits             []bool
	badHeartbeat     string
	failedValidation bool
	mutateTerminal   bool
	mutateHeartbeat  bool
	holdHeartbeat    bool
	holdPublication  bool
	entered          chan struct{}
	cancel           context.CancelFunc
	task             *workflowservice.PollActivityTaskQueueResponse
}

func (s *rawService) PollActivityTaskQueue(ctx context.Context, _ *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	s.mu.Lock()
	s.polls++
	first := s.polls == 1
	s.mu.Unlock()
	if first {
		return proto.CloneOf(s.task), nil
	}
	s.cancel()
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s *rawService) RecordActivityTaskHeartbeat(ctx context.Context, request *workflowservice.RecordActivityTaskHeartbeatRequest) (*workflowservice.RecordActivityTaskHeartbeatResponse, error) {
	s.mu.Lock()
	index := len(s.heartbeats)
	s.heartbeats = append(s.heartbeats, proto.CloneOf(request))
	s.mu.Unlock()
	if s.holdHeartbeat {
		s.entered <- struct{}{}
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	}
	if s.badHeartbeat == "transport" {
		return nil, status.Error(codes.PermissionDenied, "proof-denied")
	}
	response := &workflowservice.RecordActivityTaskHeartbeatResponse{}
	if index < len(s.bits) {
		response.CancelRequested = s.bits[index]
	}
	switch s.badHeartbeat {
	case "paused":
		response.ActivityPaused = true
	case "reset":
		response.ActivityReset = true
	case "unknown":
		response.ProtoReflect().SetUnknown([]byte{0x20, 1})
	case "malformed":
		response.ProtoReflect().SetUnknown([]byte{0xff})
	default:
	}
	return response, nil
}
func (s *rawService) terminal(message proto.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.terminals = append(s.terminals, proto.CloneOf(message))
}
func (s *rawService) RespondActivityTaskCompleted(_ context.Context, request *workflowservice.RespondActivityTaskCompletedRequest) (*workflowservice.RespondActivityTaskCompletedResponse, error) {
	s.terminal(request)
	return &workflowservice.RespondActivityTaskCompletedResponse{}, nil
}
func (s *rawService) RespondActivityTaskFailed(_ context.Context, request *workflowservice.RespondActivityTaskFailedRequest) (*workflowservice.RespondActivityTaskFailedResponse, error) {
	s.terminal(request)
	response := &workflowservice.RespondActivityTaskFailedResponse{}
	if s.failedValidation {
		response.Failures = []*failurepb.Failure{{Message: "invalid terminal"}}
	}
	return response, nil
}
func (s *rawService) RespondActivityTaskCanceled(_ context.Context, request *workflowservice.RespondActivityTaskCanceledRequest) (*workflowservice.RespondActivityTaskCanceledResponse, error) {
	s.terminal(request)
	return &workflowservice.RespondActivityTaskCanceledResponse{}, nil
}

func rawPack(t *testing.T, message proto.Message) *anypb.Any {
	t.Helper()
	packed, err := anypb.New(message)
	require.NoError(t, err)
	return packed
}
func rawProgram(t *testing.T, bits []bool, terminal string, matched bool) []byte {
	t.Helper()
	p := &temporalv1.WorkerProgram{Namespace: "ns", TaskQueue: "q"}
	if matched {
		history := []bool{}
		for i, bit := range bits {
			key := &temporalv1.ActivityTaskKey{Scenario: "history", Attempt: 2, ActivityType: "activity", HeartbeatHistory: append([]bool(nil), history...)}
			request := &workflowservice.RecordActivityTaskHeartbeatRequest{Details: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte{byte(i)}}}}}
			p.Rules = append(p.Rules, &temporalv1.Rule{Key: rawPack(t, key), Respond: []*anypb.Any{rawPack(t, request)}})
			p.ContinuationDomains = append(p.ContinuationDomains, rawPack(t, &temporalv1.ActivityTaskSuccessors{Key: key, CancelRequested: []bool{bit}}))
			history = append(history, bit)
		}
		var request proto.Message
		switch terminal {
		case "completed":
			request = &workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("exact")}}}}
		case "failed":
			request = &workflowservice.RespondActivityTaskFailedRequest{Failure: &failurepb.Failure{Message: "chosen", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "proof", NonRetryable: true}}}}
		case "canceled":
			request = &workflowservice.RespondActivityTaskCanceledRequest{Details: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("ack")}}}}
		default:
			t.Fatalf("unknown terminal %q", terminal)
		}
		key := &temporalv1.ActivityTaskKey{Scenario: "history", Attempt: 2, ActivityType: "activity", HeartbeatHistory: history}
		p.Rules = append(p.Rules, &temporalv1.Rule{Key: rawPack(t, key), Respond: []*anypb.Any{rawPack(t, request)}})
	}
	data, err := proto.Marshal(rawPack(t, p))
	require.NoError(t, err)
	return data
}

type rawFixture struct {
	ctx         context.Context
	cancel      context.CancelFunc
	bridge      *worker.Bridge
	service     *rawService
	done        chan struct{}
	err         error
	observeErr  error
	publication chan worker.RPC
	publishers  sync.WaitGroup
}

func newRawFixture(t *testing.T, program []byte, configure func(*rawService)) *rawFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	f := &rawFixture{ctx: ctx, cancel: cancel, bridge: worker.NewBridge(ctx), done: make(chan struct{}), publication: make(chan worker.RPC, 1)}
	f.service = &rawService{cancel: cancel, entered: make(chan struct{}, 1), task: &workflowservice.PollActivityTaskQueueResponse{TaskToken: []byte("opaque-token"), WorkflowNamespace: "ns", Attempt: 2, ActivityId: "activity-id", ActivityRunId: "activity-run", ActivityType: &commonpb.ActivityType{Name: "activity"}, StartedTime: timestamppb.Now(), ScheduledTime: timestamppb.Now(), StartToCloseTimeout: durationpb.New(time.Hour), ScheduleToCloseTimeout: durationpb.New(time.Hour), Header: &commonpb.Header{Fields: map[string]*commonpb.Payload{worker.ScenarioHeader: {Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("history")}}}}}
	if configure != nil {
		configure(f.service)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(server, f.service)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	var invocation, sequence uint64
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithChainUnaryInterceptor(func(callCtx context.Context, method string, request, response any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if f.service.mutateHeartbeat {
			if heartbeat, ok := request.(*workflowservice.RecordActivityTaskHeartbeatRequest); ok {
				heartbeat.Details.Payloads[0].Data = []byte("mutated")
			}
		}
		if f.service.mutateTerminal {
			if completed, ok := request.(*workflowservice.RespondActivityTaskCompletedRequest); ok {
				completed.Result.Payloads[0].Data = []byte("mutated")
			}
		}
		invocation++
		id := invocation
		sequence++
		requestMessage, ok := request.(proto.Message)
		if !ok {
			return fmt.Errorf("raw fixture request has unexpected type %T", request)
		}
		actualRequest := rawPack(t, proto.CloneOf(requestMessage))
		observeErr := f.bridge.Observe(callCtx, worker.RPC{Invocation: id, Sequence: sequence, Stage: worker.RPCStarted, Method: method, Request: actualRequest})
		if observeErr != nil {
			f.observeErr = observeErr
		}
		callErr := invoke(callCtx, method, request, response, cc, opts...)
		sequence++
		rpc := worker.RPC{Invocation: id, Sequence: sequence, Stage: worker.RPCCompleted, Method: method, Request: actualRequest}
		if callErr == nil {
			rpc.Sent = true
			rpc.Answered = true
			responseMessage, ok := response.(proto.Message)
			if !ok {
				return fmt.Errorf("raw fixture response has unexpected type %T", response)
			}
			rpc.Response = rawPack(t, proto.CloneOf(responseMessage))
		} else {
			rpc.CallError = rawPack(t, status.Convert(callErr).Proto())
			if callCtx.Err() == nil && status.Code(callErr) != codes.Internal {
				rpc.Sent = true
				rpc.Answered = true
				rpc.ServerError = &realizationv1.Error{Code: int32(status.Code(callErr)), Message: status.Convert(callErr).Message()}
			} else if callCtx.Err() != nil {
				rpc.TransportError = callErr.Error()
			}
		}
		if f.service.holdPublication && method == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
			f.publishers.Add(1)
			go func() { defer f.publishers.Done(); f.publication <- rpc; <-ctx.Done() }()
		} else if observeErr := f.bridge.Observe(callCtx, rpc); observeErr != nil {
			f.observeErr = observeErr
		}
		return callErr
	}))
	require.NoError(t, err)
	go func() {
		defer close(f.done)
		f.err = Follow(ctx, workflowservice.NewWorkflowServiceClient(conn), program, f.bridge)
	}()
	t.Cleanup(func() {
		cancel()
		<-f.done
		f.publishers.Wait()
		closeErr := f.bridge.Close()
		if closeErr != nil && f.err == nil {
			t.Error(closeErr)
		}
		require.NoError(t, conn.Close())
		server.Stop()
		<-served
	})
	return f
}
func (f *rawFixture) wait(t *testing.T) error {
	t.Helper()
	select {
	case <-f.done:
		return f.err
	case <-f.ctx.Done():
		select {
		case <-f.done:
			return f.err
		case <-time.After(time.Second):
			t.Fatal("raw follower did not join")
			return nil
		}
	}
}

func TestFollowRetainsAssignmentThroughActualHeartbeatHistory(t *testing.T) {
	bits := []bool{false, false, true, true}
	for _, terminal := range []string{"completed", "failed", "canceled"} {
		t.Run(terminal, func(t *testing.T) {
			f := newRawFixture(t, rawProgram(t, bits, terminal, true), func(service *rawService) { service.bits = bits })
			require.NoError(t, f.wait(t))
			f.service.mu.Lock()
			defer f.service.mu.Unlock()
			require.Len(t, f.service.heartbeats, len(bits))
			require.Len(t, f.service.terminals, 1)
			for i, request := range f.service.heartbeats {
				require.Equal(t, "ns", request.GetNamespace())
				require.Equal(t, []byte("opaque-token"), request.GetTaskToken())
				require.Equal(t, []byte{byte(i)}, request.GetDetails().GetPayloads()[0].GetData())
			}
			require.Equal(t, 2, f.service.polls)
			switch request := f.service.terminals[0].(type) {
			case *workflowservice.RespondActivityTaskCompletedRequest:
				require.Equal(t, "completed", terminal)
				require.Equal(t, []byte("exact"), request.GetResult().GetPayloads()[0].GetData())
			case *workflowservice.RespondActivityTaskFailedRequest:
				require.Equal(t, "failed", terminal)
				require.Equal(t, "chosen", request.GetFailure().GetMessage())
			case *workflowservice.RespondActivityTaskCanceledRequest:
				require.Equal(t, "canceled", terminal)
				require.Equal(t, []byte("ack"), request.GetDetails().GetPayloads()[0].GetData())
			default:
				t.Fatalf("unexpected terminal %T", request)
			}
		})
	}
}
func TestFollowInitialUnprogrammedRemainsFailure(t *testing.T) {
	f := newRawFixture(t, rawProgram(t, nil, "completed", false), nil)
	require.NoError(t, f.wait(t))
	f.service.mu.Lock()
	defer f.service.mu.Unlock()
	require.Len(t, f.service.terminals, 1)
	failed, ok := f.service.terminals[0].(*workflowservice.RespondActivityTaskFailedRequest)
	require.True(t, ok)
	require.Equal(t, worker.Unprogrammed, failed.GetFailure().GetApplicationFailureInfo().GetType())
	require.True(t, failed.GetFailure().GetApplicationFailureInfo().GetNonRetryable())
}
func TestFollowBadHeartbeatNeverAdvancesOrSendsTerminal(t *testing.T) {
	for _, bad := range []string{"transport", "paused", "reset", "unknown", "malformed", "off-domain"} {
		t.Run(bad, func(t *testing.T) {
			f := newRawFixture(t, rawProgram(t, []bool{false}, "completed", true), func(service *rawService) {
				service.badHeartbeat = bad
				if bad == "off-domain" {
					service.bits = []bool{true}
				}
			})
			require.Error(t, f.wait(t))
			f.service.mu.Lock()
			defer f.service.mu.Unlock()
			require.Len(t, f.service.heartbeats, 1)
			require.Empty(t, f.service.terminals)
			require.Equal(t, 1, f.service.polls)
		})
	}
}
func TestFollowFailedAcknowledgmentValidationFailure(t *testing.T) {
	f := newRawFixture(t, rawProgram(t, nil, "failed", true), func(service *rawService) { service.failedValidation = true })
	require.ErrorContains(t, f.wait(t), "validation failures")
	f.service.mu.Lock()
	defer f.service.mu.Unlock()
	require.Len(t, f.service.terminals, 1)
	require.Equal(t, 1, f.service.polls)
}
func TestFollowCancellationAtActualHeartbeatDoesNotAdvance(t *testing.T) {
	for _, closeBridge := range []bool{false, true} {
		t.Run(map[bool]string{false: "run-cancel", true: "bridge-close"}[closeBridge], func(t *testing.T) {
			f := newRawFixture(t, rawProgram(t, []bool{false}, "completed", true), func(service *rawService) { service.holdHeartbeat = true })
			select {
			case <-f.service.entered:
			case <-f.ctx.Done():
				t.Fatal(f.ctx.Err())
			}
			if closeBridge {
				require.ErrorContains(t, f.bridge.Close(), "outstanding assignment")
				require.Error(t, f.wait(t))
			} else {
				f.cancel()
				require.NoError(t, f.wait(t))
			}
			f.service.mu.Lock()
			defer f.service.mu.Unlock()
			require.Len(t, f.service.heartbeats, 1)
			require.Empty(t, f.service.terminals)
		})
	}
}

func TestFollowActivityDeadlineAbortsActualHeartbeat(t *testing.T) {
	f := newRawFixture(t, rawProgram(t, []bool{false}, "completed", true), func(service *rawService) {
		service.holdHeartbeat = true
		service.task.StartToCloseTimeout = durationpb.New(250 * time.Millisecond)
	})
	select {
	case <-f.service.entered:
	case <-f.done:
		t.Fatal("task deadline expired before the actual heartbeat reached the server")
	case <-f.ctx.Done():
		t.Fatal(f.ctx.Err())
	}
	require.Error(t, f.wait(t))
	require.NoError(t, f.ctx.Err())
	f.service.mu.Lock()
	defer f.service.mu.Unlock()
	require.Len(t, f.service.heartbeats, 1)
	require.Empty(t, f.service.terminals)
	require.Equal(t, 1, f.service.polls)
}

func TestFollowTransportMutationCannotRewriteChosenTerminal(t *testing.T) {
	f := newRawFixture(t, rawProgram(t, nil, "completed", true), func(service *rawService) { service.mutateTerminal = true })
	require.ErrorContains(t, f.wait(t), "payload or assignment")
	f.service.mu.Lock()
	defer f.service.mu.Unlock()
	require.Len(t, f.service.terminals, 1)
	require.Equal(t, 1, f.service.polls)
}

func TestFollowTransportMutationCannotRewriteChosenHeartbeat(t *testing.T) {
	f := newRawFixture(t, rawProgram(t, []bool{false}, "completed", true), func(service *rawService) { service.mutateHeartbeat = true })
	require.ErrorContains(t, f.wait(t), "heartbeat request differs")
	f.service.mu.Lock()
	defer f.service.mu.Unlock()
	require.Len(t, f.service.heartbeats, 1)
	require.Equal(t, []byte("mutated"), f.service.heartbeats[0].GetDetails().GetPayloads()[0].GetData())
	require.Empty(t, f.service.terminals)
	require.Equal(t, 1, f.service.polls)
}

func TestFollowCancellationWhileActualCompletionPublicationIsHeld(t *testing.T) {
	f := newRawFixture(t, rawProgram(t, []bool{false}, "completed", true), func(service *rawService) { service.holdPublication = true })
	var captured worker.RPC
	select {
	case captured = <-f.publication:
	case <-f.ctx.Done():
		t.Fatal(f.ctx.Err())
	}
	require.True(t, captured.Sent)
	require.True(t, captured.Answered)
	response := new(workflowservice.RecordActivityTaskHeartbeatResponse)
	require.NoError(t, captured.Response.UnmarshalTo(response))
	require.False(t, response.GetCancelRequested())
	f.cancel()
	require.NoError(t, f.wait(t))
	f.publishers.Wait()
	f.service.mu.Lock()
	defer f.service.mu.Unlock()
	require.Len(t, f.service.heartbeats, 1)
	require.Empty(t, f.service.terminals)
	require.Equal(t, 1, f.service.polls)
}

func TestFollowExplicitAfterTrueFalseTransportDomain(t *testing.T) {
	bits := []bool{true, false}
	f := newRawFixture(t, rawProgram(t, bits, "completed", true), func(service *rawService) { service.bits = bits })
	require.NoError(t, f.wait(t))
	f.service.mu.Lock()
	defer f.service.mu.Unlock()
	require.Len(t, f.service.heartbeats, 2)
	require.Len(t, f.service.terminals, 1)
	require.IsType(t, &workflowservice.RespondActivityTaskCompletedRequest{}, f.service.terminals[0])
}

func TestFollowTerminalOnlyResponseSemantics(t *testing.T) {
	for _, terminal := range []string{"completed", "failed"} {
		t.Run(terminal, func(t *testing.T) {
			f := newRawFixture(t, rawProgram(t, nil, terminal, true), nil)
			require.NoError(t, f.wait(t))
			f.service.mu.Lock()
			defer f.service.mu.Unlock()
			require.Empty(t, f.service.heartbeats)
			require.Len(t, f.service.terminals, 1)
			require.Equal(t, 2, f.service.polls)
		})
	}
}

func TestFollowOriginalBinaryOneHeartbeatSubset(t *testing.T) {
	for _, test := range []struct {
		name     string
		bit      bool
		terminal string
	}{
		{name: "false-complete", terminal: "completed"},
		{name: "false-fail", terminal: "failed"},
		{name: "true-complete", bit: true, terminal: "completed"},
		{name: "true-fail", bit: true, terminal: "failed"},
		{name: "true-canceled", bit: true, terminal: "canceled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := new(temporalv1.WorkerProgram)
			envelope := new(anypb.Any)
			falseTerminal := test.terminal
			if falseTerminal == "canceled" {
				falseTerminal = "completed"
			}
			require.NoError(t, proto.Unmarshal(rawProgram(t, []bool{false}, falseTerminal, true), envelope))
			require.NoError(t, envelope.UnmarshalTo(program))
			alternate := new(temporalv1.WorkerProgram)
			require.NoError(t, proto.Unmarshal(rawProgram(t, []bool{true}, test.terminal, true), envelope))
			require.NoError(t, envelope.UnmarshalTo(alternate))
			program.Rules = append(program.Rules, alternate.Rules[1])
			domain := new(temporalv1.ActivityTaskSuccessors)
			require.NoError(t, program.ContinuationDomains[0].UnmarshalTo(domain))
			domain.CancelRequested = []bool{false, true}
			program.ContinuationDomains[0] = rawPack(t, domain)
			data, err := proto.Marshal(rawPack(t, program))
			require.NoError(t, err)
			f := newRawFixture(t, data, func(service *rawService) { service.bits = []bool{test.bit} })
			require.NoError(t, f.wait(t))
			f.service.mu.Lock()
			defer f.service.mu.Unlock()
			require.Len(t, f.service.heartbeats, 1)
			require.Len(t, f.service.terminals, 1)
			require.Equal(t, 2, f.service.polls)
		})
	}
}
