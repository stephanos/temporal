package sdk

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"umpire.local/temporal/reference/worker"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

type namespaceServer struct {
	workflowservice.UnimplementedWorkflowServiceServer
	delivered  atomic.Bool
	completion chan *workflowservice.RespondActivityTaskCompletedRequest
}

func (s *namespaceServer) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{}, nil
}
func (s *namespaceServer) DescribeNamespace(context.Context, *workflowservice.DescribeNamespaceRequest) (*workflowservice.DescribeNamespaceResponse, error) {
	return &workflowservice.DescribeNamespaceResponse{NamespaceInfo: &namespacepb.NamespaceInfo{State: enumspb.NAMESPACE_STATE_REGISTERED}}, nil
}
func (s *namespaceServer) PollActivityTaskQueue(ctx context.Context, request *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	if !s.delivered.Swap(true) {
		return &workflowservice.PollActivityTaskQueueResponse{TaskToken: []byte("token"), WorkflowNamespace: request.GetNamespace(), ScheduledTime: timestamppb.Now(), StartedTime: timestamppb.Now(), StartToCloseTimeout: durationpb.New(time.Hour), ScheduleToCloseTimeout: durationpb.New(time.Hour), ActivityType: &commonpb.ActivityType{Name: "activity"}, Attempt: 1, Header: &commonpb.Header{Fields: map[string]*commonpb.Payload{"umpire-scenario": {Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Complete")}}}}, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s *namespaceServer) RespondActivityTaskCompleted(_ context.Context, r *workflowservice.RespondActivityTaskCompletedRequest) (*workflowservice.RespondActivityTaskCompletedResponse, error) {
	s.completion <- r
	return &workflowservice.RespondActivityTaskCompletedResponse{}, nil
}

func TestFollowBindsProgramNamespaceAndKeepsTransportHold(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bridge := worker.NewBridge(ctx)
	defer func() {
		if err := bridge.Close(); err != nil {
			t.Error(err)
		}
	}()
	capture := &sdkCapture{bridge: bridge, completed: make(chan worker.RPC, 128), run: ctx}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	service := &namespaceServer{completion: make(chan *workflowservice.RespondActivityTaskCompletedRequest, 1)}
	workflowservice.RegisterWorkflowServiceServer(server, service)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	defer func() { server.Stop(); <-served }()
	intercepted := make(chan string, 10)
	release := make(chan struct{})
	conn, err := client.NewLazyClient(client.Options{HostPort: listener.Addr().String(), Namespace: "connection-B", ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, request, response any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if method == workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName {
			select {
			case intercepted <- request.(*workflowservice.PollActivityTaskQueueRequest).GetNamespace():
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return invoke(ctx, method, request, response, cc, opts...)
	}, capture.intercept), grpc.WithStatsHandler(sdkProgressStats{})}}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	key, err := anypb.New(&temporalv1.ActivityTaskKey{Scenario: "Complete", Attempt: 1, ActivityType: "activity"})
	if err != nil {
		t.Fatal(err)
	}
	payload := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("exact")}
	response, err := anypb.New(&workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{payload}}})
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := anypb.New(&temporalv1.WorkerProgram{Namespace: "program-A", TaskQueue: "queue-A", Rules: []*temporalv1.Rule{{Key: key, Respond: []*anypb.Any{response}}}})
	if err != nil {
		t.Fatal(err)
	}
	program, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- Follow(ctx, conn, program, bridge) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	select {
	case namespace := <-intercepted:
		if namespace != "program-A" {
			t.Errorf("poll namespace = %q, want program-A", namespace)
		}
	case err := <-done:
		done <- err
		t.Fatalf("executor ended: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if service.delivered.Load() {
		t.Fatal("poll bypassed original transport hold")
	}
	close(release)
	select {
	case completed := <-service.completion:
		if completed.GetNamespace() != "program-A" || !proto.Equal(payload, completed.GetResult().GetPayloads()[0]) {
			t.Fatalf("completion = %v", completed)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
