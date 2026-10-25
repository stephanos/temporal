package temporaldriver

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	"umpire.local/model/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

type continuationServer struct {
	workflowservice.UnimplementedWorkflowServiceServer
	delivered  atomic.Bool
	heartbeats chan *workflowservice.RecordActivityTaskHeartbeatRequest
	canceled   chan *workflowservice.RespondActivityTaskCanceledRequest
}

func (*continuationServer) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{}, nil
}
func (*continuationServer) DescribeNamespace(context.Context, *workflowservice.DescribeNamespaceRequest) (*workflowservice.DescribeNamespaceResponse, error) {
	return &workflowservice.DescribeNamespaceResponse{NamespaceInfo: &namespacepb.NamespaceInfo{State: enumspb.NAMESPACE_STATE_REGISTERED}}, nil
}
func (s *continuationServer) PollActivityTaskQueue(ctx context.Context, request *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	if request.GetNamespace() != "namespace" || request.GetTaskQueue().GetName() != "queue" {
		return nil, status.Error(codes.InvalidArgument, "crossed composed poll")
	}
	if !s.delivered.Swap(true) {
		return &workflowservice.PollActivityTaskQueueResponse{
			TaskToken: []byte("composed-token"), WorkflowNamespace: request.GetNamespace(), Attempt: 7, ActivityId: "activity", ActivityRunId: "composed-run",
			ActivityType: &commonpb.ActivityType{Name: "Composed"}, ScheduledTime: timestamppb.Now(), StartedTime: timestamppb.Now(),
			StartToCloseTimeout: durationpb.New(time.Hour), ScheduleToCloseTimeout: durationpb.New(time.Hour),
			Header: &commonpb.Header{Fields: map[string]*commonpb.Payload{"umpire-scenario": {Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("History")}}},
		}, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}
func (*continuationServer) DescribeActivityExecution(context.Context, *workflowservice.DescribeActivityExecutionRequest) (*workflowservice.DescribeActivityExecutionResponse, error) {
	return &workflowservice.DescribeActivityExecutionResponse{}, nil
}
func (s *continuationServer) RecordActivityTaskHeartbeat(_ context.Context, request *workflowservice.RecordActivityTaskHeartbeatRequest) (*workflowservice.RecordActivityTaskHeartbeatResponse, error) {
	s.heartbeats <- proto.Clone(request).(*workflowservice.RecordActivityTaskHeartbeatRequest)
	return &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}, nil
}
func (s *continuationServer) RespondActivityTaskCanceled(_ context.Context, request *workflowservice.RespondActivityTaskCanceledRequest) (*workflowservice.RespondActivityTaskCanceledResponse, error) {
	s.canceled <- proto.Clone(request).(*workflowservice.RespondActivityTaskCanceledRequest)
	return &workflowservice.RespondActivityTaskCanceledResponse{}, nil
}

func continuationAny(t *testing.T, message proto.Message) *anypb.Any {
	t.Helper()
	result, err := anypb.New(message)
	require.NoError(t, err)
	return result
}

func continuationProgram(t *testing.T) []byte {
	t.Helper()
	root := &temporalv1.ActivityTaskKey{Scenario: "History", Attempt: 7, ActivityType: "Composed"}
	response := continuationAny(t, &workflowservice.RecordActivityTaskHeartbeatRequest{Details: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("chosen-heartbeat")}}}})
	rows := []*temporalv1.Rule{{Key: continuationAny(t, root), Respond: []*anypb.Any{response}}}
	for _, bit := range []bool{false, true} {
		key := proto.Clone(root).(*temporalv1.ActivityTaskKey)
		key.HeartbeatHistory = []bool{bit}
		var terminal proto.Message = &workflowservice.RespondActivityTaskCanceledRequest{}
		if !bit {
			terminal = &workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("completed")}}}}
		}
		rows = append(rows, &temporalv1.Rule{Key: continuationAny(t, key), Respond: []*anypb.Any{continuationAny(t, terminal)}})
	}
	program := &temporalv1.WorkerProgram{Namespace: "namespace", TaskQueue: "queue", Rules: rows, ContinuationDomains: []*anypb.Any{continuationAny(t, &temporalv1.ActivityTaskSuccessors{Key: root, CancelRequested: []bool{false, true}})}}
	result, err := proto.Marshal(continuationAny(t, program))
	require.NoError(t, err)
	return result
}

func continuationReceive[T any](ctx context.Context, t *testing.T, events <-chan T) T {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-ctx.Done():
		t.Fatal(ctx.Err())
		var zero T
		return zero
	}
}

func TestFactoryComposesActualHeartbeatContinuationAndFaultCapture(t *testing.T) {
	for _, executor := range []string{"raw", "sdk"} {
		for _, mutate := range []bool{false, true} {
			t.Run(executor+map[bool]string{false: "/actual-true", true: "/mutated-heartbeat"}[mutate], func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				service := &continuationServer{heartbeats: make(chan *workflowservice.RecordActivityTaskHeartbeatRequest, 1), canceled: make(chan *workflowservice.RespondActivityTaskCanceledRequest, 1)}
				server := grpc.NewServer()
				workflowservice.RegisterWorkflowServiceServer(server, service)
				served := make(chan error, 1)
				go func() { served <- server.Serve(listener) }()
				defer func() { server.Stop(); require.NoError(t, <-served) }()
				terminalReturned := make(chan struct{}, 1)
				transport := act.Transport{DialOptions: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, request, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
					if request, ok := request.(*workflowservice.RecordActivityTaskHeartbeatRequest); ok && mutate {
						request.Details.Payloads[0].Data = []byte("actual-mutated-heartbeat")
					}
					err := invoke(ctx, method, request, reply, conn, options...)
					if method == workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName && err == nil {
						terminalReturned <- struct{}{}
					}
					return err
				})}}
				factory, err := NewFactory(act.Options{Address: listener.Addr().String(), Namespace: "namespace", TaskQueue: "queue", ExecutorKind: executor, Program: continuationProgram(t), Transport: transport})
				require.NoError(t, err)
				defer func() { require.NoError(t, factory.Close(ctx)) }()
				driver, err := factory.New(ctx)
				require.NoError(t, err)
				closer := driver.(interface{ Close() error })
				defer func() { _ = closer.Close() }()
				instruction := continuationAny(t, &temporalv1.ObserveActivity{Request: continuationAny(t, &workflowservice.DescribeActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity"}), PollIntervalMs: 1, BackoffMs: 1, ReadTimeoutMs: 5000})
				exchange, err := driver.Start(ctx, 1, instruction)
				require.NoError(t, err)
				defer func() { _ = exchange.Close() }()
				var fault runner.Event
				for {
					event := continuationReceive(ctx, t, exchange.Events())
					if event.Kind == runner.EventFault {
						fault = event
						break
					}
					if event.Kind == runner.EventDone {
						break
					}
					require.Equal(t, runner.EventObservation, event.Kind)
					require.NoError(t, exchange.Respond(ctx, event.ID, runner.Decision{Judgment: &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Explained{Explained: &modelv1.Explanation{}}, ContinuePolling: mutate}}))
				}
				if fault.Kind == runner.EventFault && !mutate {
					t.Fatalf("composed executor fault before heartbeat: %s; assigned=%t", fault.Fault, service.delivered.Load())
				}
				var heartbeat *workflowservice.RecordActivityTaskHeartbeatRequest
				select {
				case heartbeat = <-service.heartbeats:
				case <-ctx.Done():
					t.Fatalf("heartbeat wait: %v; assigned=%t; close=%v", ctx.Err(), service.delivered.Load(), closer.Close())
				}
				require.Equal(t, "namespace", heartbeat.GetNamespace())
				require.Equal(t, []byte("composed-token"), heartbeat.GetTaskToken())
				if mutate {
					require.Equal(t, []byte("actual-mutated-heartbeat"), heartbeat.GetDetails().GetPayloads()[0].GetData())
					require.Equal(t, runner.EventFault, fault.Kind)
					require.Contains(t, fault.Fault, "worker executor")
					captured := new(temporalv1.WorkerTransportFault)
					require.NoError(t, fault.Payload.UnmarshalTo(captured))
					found := false
					for _, call := range captured.GetCalls() {
						if call.GetMethod() == workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName && call.GetStage() == uint32(act.WorkerRPCCompleted) {
							request := new(workflowservice.RecordActivityTaskHeartbeatRequest)
							require.NoError(t, call.GetRequest().UnmarshalTo(request))
							require.True(t, proto.Equal(heartbeat, request))
							require.True(t, call.GetAnswered())
							found = true
						}
					}
					require.True(t, found, "actual mutated heartbeat missing from fault evidence")
					require.ErrorContains(t, closer.Close(), "actual heartbeat request differs")
					require.Empty(t, service.canceled)
				} else {
					canceled := continuationReceive(ctx, t, service.canceled)
					require.Equal(t, "namespace", canceled.GetNamespace())
					require.Equal(t, []byte("composed-token"), canceled.GetTaskToken())
					continuationReceive(ctx, t, terminalReturned)
					require.NoError(t, exchange.Close())
					require.NoError(t, closer.Close())
					require.Empty(t, service.heartbeats)
					require.Empty(t, service.canceled)
				}
			})
		}
	}
}
