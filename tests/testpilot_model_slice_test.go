//go:build test_dep && integration

package tests

import (
	"context"
	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"go.temporal.io/server/tools/umpire/temporaldriver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
	"testing"
	"time"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	model "umpire.local/model/host"
	"umpire.local/model/runner"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalmodel "umpire.local/temporal"
)

func modelSliceAny(t *testing.T, message proto.Message) *anypb.Any {
	t.Helper()
	value, err := anypb.New(message)
	require.NoError(t, err)
	return value
}

func TestTestpilotModelSlice(t *testing.T) {
	t.Run("Transport", modelSliceProgrammedTransport)
	t.Run("Reconciliation", modelSliceReconciliation)
	t.Run("Annotations", func(t *testing.T) { modelSliceAnnotationControls(t, temporalmodel.Guest()) })
}

func modelSliceProgrammedTransport(t *testing.T) {
	env := activityEnvironment(t)
	host, err := model.New(t.Context(), temporalmodel.Guest())
	require.NoError(t, err)
	defer func() { require.NoError(t, host.Close(context.Background())) }()
	annotations := &modelSliceAnnotations{host: host}
	intercept := func(ctx context.Context, method string, request, response any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if _, ok := request.(proto.Message); !ok {
			return invoke(ctx, method, request, response, conn, opts...)
		}
		return annotations.intercept(ctx, method, request, response, conn, invoke, opts...)
	}
	conn, err := grpc.NewClient(env.FrontendGRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithChainUnaryInterceptor(intercept))
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close()) }()
	for _, kind := range []string{"raw", "sdk"} {
		for _, seed := range []uint64{1, 3, 19} {
			start := &workflowservice.StartActivityExecutionRequest{
				Namespace: env.Namespace().String(), ActivityId: uuid.NewString(), ActivityType: &commonpb.ActivityType{Name: "ModelSlice"},
				TaskQueue: &taskqueuepb.TaskQueue{Name: "transport-" + uuid.NewString()}, StartToCloseTimeout: durationpb.New(20 * time.Second),
				RetryPolicy: &commonpb.RetryPolicy{MaximumAttempts: 1},
				Input:       &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("application-input")}}},
			}
			world := modelSliceAny(t, start)
			reset, err := host.Reset(t.Context(), &modelv1.ResetRequest{Configuration: world})
			require.NoError(t, err)
			program, err := proto.Marshal(reset.GetProgram())
			require.NoError(t, err)
			factory, err := temporaldriver.NewFactory(act.Options{Address: env.FrontendGRPCAddress(), Namespace: start.Namespace, TaskQueue: start.TaskQueue.Name, ExecutorKind: kind, Program: program,
				Transport: act.Transport{DialOptions: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithChainUnaryInterceptor(intercept)}},
			})
			require.NoError(t, err)
			run, err := runner.New(temporalmodel.Guest(), factory.New).Run(t.Context(), runner.ModeExplore, runner.Options{World: world, Seed: seed, Budget: 7, RunTimeout: time.Minute})
			require.NoError(t, err)
			if run.GetVerdict().GetExplained() == nil {
				data, err := protojson.Marshal(run)
				require.NoError(t, err)
				t.Logf("transport run=%s", data)
			}
			require.NotNil(t, run.GetVerdict().GetExplained(), "kind=%s seed=%d verdict=%v", kind, seed, run.GetVerdict())
			require.NoError(t, factory.Close(t.Context()))
			for _, event := range run.GetEvents() {
				packed := event.GetJudgment().GetRequest().GetObservation()
				if packed == nil {
					continue
				}
				annotations.actual(t, packed)
			}

			_, err = workflowservice.NewWorkflowServiceClient(conn).PollActivityExecution(t.Context(), &workflowservice.PollActivityExecutionRequest{Namespace: start.Namespace, ActivityId: start.ActivityId})
			require.NoError(t, err)
		}
	}
	annotations.assert(t)
}
