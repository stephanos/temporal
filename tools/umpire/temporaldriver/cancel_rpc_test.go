package temporaldriver

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	runv1 "umpire.local/model/gen/umpire/run/v1"
	"umpire.local/model/host"
	"umpire.local/model/runner"
	temporalmodel "umpire.local/temporal"
	rawworker "umpire.local/temporal/reference/worker/raw"
	sdkworker "umpire.local/temporal/reference/worker/sdk"
)

type cancelRPCServer struct {
	workflowservice.UnimplementedWorkflowServiceServer
	starts  atomic.Int32
	cancels atomic.Int32
}

func (s *cancelRPCServer) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{}, nil
}
func (s *cancelRPCServer) DescribeNamespace(context.Context, *workflowservice.DescribeNamespaceRequest) (*workflowservice.DescribeNamespaceResponse, error) {
	return &workflowservice.DescribeNamespaceResponse{NamespaceInfo: &namespacepb.NamespaceInfo{State: enumspb.NAMESPACE_STATE_REGISTERED}}, nil
}
func (s *cancelRPCServer) StartActivityExecution(context.Context, *workflowservice.StartActivityExecutionRequest) (*workflowservice.StartActivityExecutionResponse, error) {
	s.starts.Add(1)
	return &workflowservice.StartActivityExecutionResponse{Started: true, RunId: "11111111-1111-4111-8111-111111111111"}, nil
}
func (s *cancelRPCServer) RequestCancelActivityExecution(context.Context, *workflowservice.RequestCancelActivityExecutionRequest) (*workflowservice.RequestCancelActivityExecutionResponse, error) {
	s.cancels.Add(1)
	return nil, serviceerror.NewNotFound("no cancel issued inside this admitted slice")
}
func (s *cancelRPCServer) DescribeActivityExecution(context.Context, *workflowservice.DescribeActivityExecutionRequest) (*workflowservice.DescribeActivityExecutionResponse, error) {
	return &workflowservice.DescribeActivityExecutionResponse{Info: &activitypb.ActivityExecutionInfo{ActivityId: "activity", RunId: "11111111-1111-4111-8111-111111111111", Status: enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, RunState: enumspb.PENDING_ACTIVITY_STATE_SCHEDULED, Attempt: 1}}, nil
}
func (s *cancelRPCServer) PollActivityTaskQueue(ctx context.Context, _ *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s *cancelRPCServer) TerminateActivityExecution(context.Context, *workflowservice.TerminateActivityExecutionRequest) (*workflowservice.TerminateActivityExecutionResponse, error) {
	return nil, serviceerror.NewNotFound("already removed")
}

func TestUnheldCancellationRefusedBeforeCancelRPC(t *testing.T) {
	for _, executor := range []struct{ kind, version string }{{rawworker.Kind, rawworker.Version}, {sdkworker.Kind, sdkworker.Version}} {
		t.Run(executor.kind, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			service := new(cancelRPCServer)
			server := grpc.NewServer()
			workflowservice.RegisterWorkflowServiceServer(server, service)
			served := make(chan error, 1)
			go func() { served <- server.Serve(listener) }()
			defer func() { server.Stop(); require.NoError(t, <-served) }()
			world, err := anypb.New(&workflowservice.StartActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity", ActivityType: &commonpb.ActivityType{Name: "ModelSlice"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"}, StartToCloseTimeout: durationpb.New(20 * time.Second), RetryPolicy: &commonpb.RetryPolicy{InitialInterval: durationpb.New(time.Second), MaximumInterval: durationpb.New(time.Second), MaximumAttempts: 2}, Input: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("application-input")}}}, IdReusePolicy: enumspb.ACTIVITY_ID_REUSE_POLICY_ALLOW_DUPLICATE})
			require.NoError(t, err)
			guest, err := host.New(ctx, temporalmodel.Guest())
			require.NoError(t, err)
			reset, err := guest.Reset(ctx, &modelv1.ResetRequest{Configuration: world})
			require.NoError(t, err)
			require.NoError(t, guest.Close(ctx))
			program, err := proto.Marshal(reset.GetProgram())
			require.NoError(t, err)
			factory, err := NewFactory(act.Options{Address: listener.Addr().String(), Namespace: "namespace", TaskQueue: "queue", ExecutorKind: executor.kind, Program: program, MaxReadTimeout: 10 * time.Millisecond})
			require.NoError(t, err)
			defer func() { require.NoError(t, factory.Close(context.Background())) }()
			var target *runner.Run
			var starts, cancels int32
			_, err = runner.New(temporalmodel.Guest(), func(ctx context.Context) (runner.Driver, error) {
				service.starts.Store(0)
				service.cancels.Store(0)
				return factory.New(ctx)
			}).Run(ctx, runner.ModeExplore, runner.Options{World: world, Seed: 18, Budget: 2, EventTimeout: time.Second, RunTimeout: time.Second, Record: func(_ context.Context, run *runner.Run) error {
				if run.GetSeed() == 18 {
					target = run
					starts, cancels = service.starts.Load(), service.cancels.Load()
				}
				return nil
			}})
			require.NoError(t, err)
			require.NotNil(t, target)
			require.Equal(t, int32(1), starts, "bootstrap must reach the public server once")
			require.Equal(t, int32(0), cancels, "unheld Cancel must be refused before a public Cancel RPC")
			require.True(t, proto.Equal(&runv1.Worker{Kind: executor.kind, Version: executor.version, Namespace: "namespace", TaskQueue: "queue"}, target.GetWorker()), "worker=%v", target.GetWorker())
			require.NotNil(t, target.GetVerdict().GetRefused(), "verdict=%v", target.GetVerdict())
			event := target.GetEvents()[target.GetVerdict().GetRefused().GetEvent()].GetRefusal()
			require.NotNil(t, event)
			require.Equal(t, modelv1.Method_METHOD_PROPOSE, event.GetMethod())
			require.Equal(t, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED, event.GetRefusal().GetCode())
			require.Equal(t, "unheld cancellation of an open activity is outside the atomic cancellation Model", event.GetRefusal().GetMessage())
		})
	}
}
