//go:build test_dep && integration

package tests

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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
	"google.golang.org/grpc/mem"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	runv1 "umpire.local/model/gen/umpire/run/v1"
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

func terminalReconciliationSeed(scenario uint64) uint64 {
	for seed := uint64(1); ; seed++ {
		if runner.ProposalSeed(seed, 1)%3 == scenario && runner.ProposalSeed(seed, 2)%2 == 1 {
			return seed
		}
	}
}

func reconciliationObservation(method string, request, response any, rpcErr error, options []grpc.CallOption) (*realizationv1.Observation, error) {
	for _, option := range options {
		codec, ok := option.(grpc.ForceCodecV2CallOption)
		if !ok {
			continue
		}
		requestBytes, err := codec.CodecV2.Marshal(request)
		if err != nil {
			return nil, err
		}
		defer requestBytes.Free()
		name := method[strings.LastIndexByte(method, '/')+1:]
		observation := &realizationv1.Observation{Method: method, Request: &anypb.Any{
			TypeUrl: "type.googleapis.com/temporal.api.workflowservice.v1." + name + "Request", Value: requestBytes.Materialize(),
		}}
		if rpcErr != nil {
			observed := status.Convert(rpcErr)
			observation.Result = &realizationv1.Observation_Error{Error: &realizationv1.Error{Code: int32(observed.Code()), Message: observed.Message()}}
		} else {
			responseBytes, err := codec.CodecV2.Marshal(response)
			if err != nil {
				return nil, err
			}
			defer responseBytes.Free()
			observation.Result = &realizationv1.Observation_Response{Response: &anypb.Any{
				TypeUrl: "type.googleapis.com/temporal.api.workflowservice.v1." + name + "Response", Value: responseBytes.Materialize(),
			}}
		}
		return observation, nil
	}
	return nil, fmt.Errorf("%s did not use the actual Driver wire codec", method)
}

func modelSliceReconciliation(t *testing.T) {
	env := activityEnvironment(t)
	controls := []string{"Complete", "Fail", "wrong-result", "wrong-failure", "complete-instead-of-fail"}
	cases := append([]string(nil), controls...)
	for _, control := range append(append([]string(nil), controls...), "response-mutated-history") {
		cases = append(cases, "Terminal/"+control)
	}
	for _, kind := range []string{"raw", "sdk"} {
		for _, name := range cases {
			t.Run(kind+"/"+name, func(t *testing.T) {
				terminal := strings.HasPrefix(name, "Terminal/")
				control := strings.TrimPrefix(name, "Terminal/")
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
				if control == "Complete" || control == "wrong-result" || control == "response-mutated-history" {
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
				var cancelCalls atomic.Int32
				var evidenceMu sync.Mutex
				var actualAck, actualRead *realizationv1.Observation
				intercept := func(ctx context.Context, method string, request, response any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
					describe := method == workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName
					cancelRPC := method == workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName
					if cancelRPC {
						cancelCalls.Add(1)
					}
					if terminal && describe {
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
					if !terminal || (!cancelRPC && !describe) {
						return err
					}
					evidenceMu.Lock()
					defer evidenceMu.Unlock()
					if !cancelRPC && actualAck == nil {
						return err
					}
					captured, captureErr := reconciliationObservation(method, request, response, err, opts)
					if captureErr != nil {
						return captureErr
					}
					if cancelRPC {
						actualAck = captured
					} else {
						actualRead = captured
						if control == "response-mutated-history" && err == nil {
							changed := new(workflowservice.DescribeActivityExecutionResponse)
							if decodeErr := captured.GetResponse().UnmarshalTo(changed); decodeErr != nil {
								return decodeErr
							}
							changed.Outcome.GetResult().Payloads[0].Data = []byte("Contradictory fresh history")
							data, marshalErr := proto.Marshal(changed)
							if marshalErr != nil {
								return marshalErr
							}
							for _, option := range opts {
								if codec, ok := option.(grpc.ForceCodecV2CallOption); ok {
									if decodeErr := codec.CodecV2.Unmarshal(mem.BufferSlice{mem.SliceBuffer(data)}, response); decodeErr != nil {
										return decodeErr
									}
									break
								}
							}
						}
					}
					return err
				}
				seed, budget := reconciliationSeed(scenario), 2
				if terminal {
					seed, budget = terminalReconciliationSeed(scenario), 3
				}
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
				_, err = runner.New(temporalmodel.Guest(), newDriver).Run(ctx, runner.ModeExplore, runner.Options{World: world, Seed: seed, Budget: budget, RunTimeout: 20 * time.Second, Record: func(_ context.Context, recorded *runner.Run) error {
					if recorded.Seed == seed {
						run = recorded
					}
					return nil
				}})
				require.NoError(t, err)
				require.NotNil(t, run)
				if !terminal {
					require.True(t, isUnheldCancelScopeRefusal(run), "pre-sideeffect scope refusal required: %v", run.Verdict)
					require.Equal(t, int32(0), cancelCalls.Load(), "scope refusal must send zero actual cancellation RPCs")
					require.Empty(t, explainedSteps(run), "no worker action is claimed by this scope control")
				} else if control == "wrong-result" || control == "wrong-failure" || control == "complete-instead-of-fail" {
					unexplained := run.GetVerdict().GetUnexplained()
					require.NotNil(t, unexplained, "%v", run.Verdict)
					deciding := run.Events[unexplained.Event]
					require.Equal(t, uint64(2), deciding.GetRequest(), "the first normal READ detects the worker mutant")
					observation := new(realizationv1.Observation)
					require.NoError(t, deciding.GetJudgment().GetRequest().GetObservation().UnmarshalTo(observation))
					require.Equal(t, workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName, observation.Method)
					require.NotNil(t, deciding.GetJudgment().GetResponse().GetUnexplained())
					require.Equal(t, int32(0), cancelCalls.Load(), "normal READ detects the worker mutant before terminal Cancel")
					annotations.actual(t, deciding.GetJudgment().GetRequest().GetObservation())
				} else {
					terminalRead := false
					expected := "Worker::RespondCompleted"
					if control == "Fail" {
						expected = "Worker::RespondFailed { retryable: false }"
					}
					for _, event := range run.Events {
						if event.GetRequest() != 2 || event.GetJudgment() == nil {
							continue
						}
						require.NotNil(t, event.GetJudgment().GetResponse().GetExplained())
						require.Contains(t, event.GetJudgment().GetResponse().GetExplained().GetSteps(), expected)
						annotations.actual(t, event.GetJudgment().GetRequest().GetObservation())
						terminalRead = true
					}
					require.True(t, terminalRead, "an actual normal READ closes the Model before Cancel")
					require.Equal(t, int32(1), cancelCalls.Load(), "one actual terminal cancellation RPC")
					evidenceMu.Lock()
					ack, fresh := proto.CloneOf(actualAck), proto.CloneOf(actualRead)
					evidenceMu.Unlock()
					require.NotNil(t, ack)
					require.NotNil(t, fresh)
					require.Equal(t, int32(5), ack.GetError().GetCode())
					require.Equal(t, workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName, ack.Method)
					require.Equal(t, workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName, fresh.Method)
					var deciding *runv1.Event
					for _, event := range run.Events {
						evidence := event.GetJudgment().GetRequest().GetObservation()
						if evidence == nil || !evidence.MessageIs(new(temporalv1.ReconciledObservation)) {
							continue
						}
						require.Nil(t, deciding, "one compound cancellation judgment")
						deciding = event
						require.Equal(t, uint64(3), event.GetRequest(), "compound evidence belongs to the subsequent terminal Cancel")
						annotations.actual(t, evidence)
						compound := new(temporalv1.ReconciledObservation)
						require.NoError(t, evidence.UnmarshalTo(compound))
						require.True(t, proto.Equal(modelSliceAny(t, ack), compound.Acknowledged), "compound preserves the exact actual cancellation ACK")
						if control == "response-mutated-history" {
							require.False(t, proto.Equal(modelSliceAny(t, fresh), compound.Current), "this is an explicit response mutation after the actual server read")
							original := new(workflowservice.DescribeActivityExecutionResponse)
							require.NoError(t, fresh.GetResponse().UnmarshalTo(original))
							require.Equal(t, "Complete", string(original.GetOutcome().GetResult().GetPayloads()[0].GetData()))
							require.NotNil(t, event.GetJudgment().GetResponse().GetUnexplained())
							require.Equal(t, event.Sequence, run.GetVerdict().GetUnexplained().GetEvent())
						} else {
							require.True(t, proto.Equal(modelSliceAny(t, fresh), compound.Current), "compound preserves the exact actual post-ACK read")
							require.NotNil(t, event.GetJudgment().GetResponse().GetExplained())
						}
					}
					require.NotNil(t, deciding, "actual cancellation ACK and post-ACK read must both enter the Run")
					if control == "response-mutated-history" {
						require.NotNil(t, run.GetVerdict().GetUnexplained())
					} else {
						require.NotNil(t, run.GetVerdict().GetExplained(), "%v", run.Verdict)
					}
				}
				require.Empty(t, annotations.failures)
				replay, err := runner.New(temporalmodel.Guest(), func(context.Context) (runner.Driver, error) { t.Fatal("replay contacted server"); return nil, nil }).Run(ctx, runner.ModeReplay, runner.Options{Recorded: run})
				require.NoError(t, err)
				require.True(t, proto.Equal(run.Verdict, replay.Verdict), "%v", replay.Verdict)
			})
		}
	}
}
