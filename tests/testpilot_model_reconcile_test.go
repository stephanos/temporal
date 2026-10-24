//go:build test_dep && integration

package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"go.temporal.io/server/tools/umpire/temporaldriver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	model "umpire.local/model/host"
	"umpire.local/model/runner"
	temporalmodel "umpire.local/temporal"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func reconciliationSeed(scenario uint64) uint64 {
	for seed := uint64(1); ; seed++ {
		if runner.ProposalSeed(seed, 1)%3 == scenario && runner.ProposalSeed(seed, 2)%4 == 0 {
			return seed
		}
	}
}

func modelSliceReconciliation(t *testing.T) {
	env := activityEnvironment(t)
	for _, kind := range []string{"raw", "sdk"} {
		for _, control := range []string{"Complete", "Fail", "wrong-result", "wrong-failure", "complete-instead-of-fail"} {
			t.Run(kind+"/"+control, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				queue := "reconcile-" + uuid.NewString()
				world := modelSliceAny(t, &workflowservice.StartActivityExecutionRequest{
					Namespace: env.Namespace().String(), ActivityId: uuid.NewString(), ActivityType: &commonpb.ActivityType{Name: "ModelSlice"}, TaskQueue: &taskqueuepb.TaskQueue{Name: queue}, StartToCloseTimeout: durationpb.New(20 * time.Second), RetryPolicy: &commonpb.RetryPolicy{MaximumAttempts: 1},
				})
				host, err := model.New(ctx, temporalmodel.Guest())
				require.NoError(t, err)
				reset, err := host.Reset(ctx, &modelv1.ResetRequest{Configuration: world})
				require.NoError(t, err)
				defer func() { require.NoError(t, host.Close(context.WithoutCancel(ctx))) }()
				annotations := &modelSliceAnnotations{host: host}
				program := new(temporalv1.WorkerProgram)
				require.NoError(t, reset.Program.UnmarshalTo(program))
				scenario := uint64(2)
				if control == "Complete" || control == "wrong-result" {
					scenario = 0
				}
				for _, rule := range program.Rules {
					key := new(temporalv1.ActivityTaskKey)
					require.NoError(t, rule.Key.UnmarshalTo(key))
					switch control {
					case "wrong-result":
						if key.Scenario == "Complete" {
							rule.Respond = []*anypb.Any{modelSliceAny(t, &workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Different")}}}})}
						}
					case "wrong-failure":
						if key.Scenario == "Fail" {
							failure := new(workflowservice.RespondActivityTaskFailedRequest)
							require.NoError(t, rule.Respond[0].UnmarshalTo(failure))
							failure.Failure.Message = "Different"
							rule.Respond[0] = modelSliceAny(t, failure)
						}
					case "complete-instead-of-fail":
						if key.Scenario == "Fail" {
							rule.Respond = []*anypb.Any{modelSliceAny(t, &workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("Complete")}}}})}
						}
					}
				}
				bytes, err := proto.Marshal(modelSliceAny(t, program))
				require.NoError(t, err)
				answered := make(chan struct{})
				var once sync.Once
				intercept := func(ctx context.Context, method string, request, response any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
					if method == workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName {
						select {
						case <-answered:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
					err := invoke(ctx, method, request, response, conn, opts...)
					if err == nil && (method == workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName || method == workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName) {
						once.Do(func() { close(answered) })
					}
					return err
				}
				seed := reconciliationSeed(scenario)
				var iteration uint64
				newDriver := func(ctx context.Context) (runner.Driver, error) {
					iteration++
					opts := act.Options{Address: env.FrontendGRPCAddress(), Namespace: env.Namespace().String(), TaskQueue: queue, ExecutorKind: kind, Program: bytes}
					if iteration == seed {
						opts.Transport = act.Transport{DialOptions: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithChainUnaryInterceptor(intercept)}}
					}
					factory, err := temporaldriver.NewFactory(opts)
					if err != nil {
						return nil, err
					}
					return factory.New(ctx)
				}
				var run *runner.Run
				_, err = runner.New(temporalmodel.Guest(), newDriver).Run(ctx, runner.ModeExplore, runner.Options{World: world, Seed: seed, Budget: 2, RunTimeout: 20 * time.Second, Record: func(_ context.Context, recorded *runner.Run) error {
					if recorded.Seed == seed {
						run = recorded
					}
					return nil
				}})
				require.NoError(t, err)
				require.NotNil(t, run)

				requireWorker := false
				for _, event := range run.Events {
					packed := event.GetJudgment().GetRequest().GetObservation()
					if packed == nil || !packed.MessageIs(new(temporalv1.ReconciledObservation)) {
						continue
					}
					requireWorker = true
					annotations.actual(t, packed)
					compound := new(temporalv1.ReconciledObservation)
					require.NoError(t, packed.UnmarshalTo(compound))
					ack, current := new(realizationv1.Observation), new(realizationv1.Observation)
					require.NoError(t, compound.Acknowledged.UnmarshalTo(ack))
					require.NoError(t, compound.Current.UnmarshalTo(current))
					require.Equal(t, int32(5), ack.GetError().Code)
					require.Equal(t, workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName, ack.Method)
					require.Equal(t, workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName, current.Method)
					if control != "Complete" && control != "Fail" {
						require.NotNil(t, event.GetJudgment().GetResponse().GetUnexplained())
						require.Equal(t, event.Sequence, run.GetVerdict().GetUnexplained().GetEvent())
					}
				}
				require.Empty(t, annotations.failures)
				if !requireWorker {
					data, err := protojson.Marshal(run)
					require.NoError(t, err)
					t.Logf("missing reconciliation run=%s", data)
				}
				require.True(t, requireWorker, "actual cancellation and terminal read must both enter the Run")
				if control == "Complete" || control == "Fail" {
					require.NotNil(t, run.GetVerdict().GetExplained(), "%v", run.Verdict)
				} else {
					require.NotNil(t, run.GetVerdict().GetUnexplained(), "%v", run.Verdict)
				}
				replay, err := runner.New(temporalmodel.Guest(), func(context.Context) (runner.Driver, error) { t.Fatal("replay contacted server"); return nil, nil }).Run(ctx, runner.ModeReplay, runner.Options{Recorded: run})
				require.NoError(t, err)
				require.True(t, proto.Equal(run.Verdict, replay.Verdict), "%v", replay.Verdict)
			})
		}
	}
}
