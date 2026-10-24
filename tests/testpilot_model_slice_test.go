//go:build test_dep && integration

package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	sdkactivity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/common/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	model "umpire.local/model/host"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	temporalmodel "umpire.local/temporal"
)

type modelSliceReply struct {
	reply *realizationv1.Reply
	err   error
}

type modelSliceDelivery struct {
	task  *workflowservice.PollActivityTaskQueueResponse
	reply chan modelSliceReply
}

func modelSliceAny(t *testing.T, message proto.Message) *anypb.Any {
	t.Helper()
	value, err := anypb.New(message)
	require.NoError(t, err)
	return value
}

// modelSliceDescribed is one captured Describe invocation's actual request and result.
type modelSliceDescribed struct {
	*workflowservice.DescribeActivityExecutionResponse
	request *workflowservice.DescribeActivityExecutionRequest
}

func modelSliceJudge(ctx context.Context, t *testing.T, host *model.Host, described modelSliceDescribed) *modelv1.JudgeResponse {
	t.Helper()
	return modelSliceJudgeObservation(ctx, t, host, &realizationv1.Observation{
		Method:  workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName,
		Result:  &realizationv1.Observation_Response{Response: modelSliceAny(t, described.DescribeActivityExecutionResponse)},
		Request: modelSliceAny(t, described.request),
	})
}

func modelSliceJudgeObservation(ctx context.Context, t *testing.T, host *model.Host, observation *realizationv1.Observation) *modelv1.JudgeResponse {
	t.Helper()
	request := &modelv1.JudgeRequest{Observation: modelSliceAny(t, observation)}
	judgeStart := time.Now()
	judgment, err := host.Judge(ctx, request)
	t.Logf("Judge latency=%s method=%s", time.Since(judgeStart), observation.GetMethod())
	require.NoError(t, err, "observation method: %s", observation.GetMethod())
	return judgment
}

func modelSliceExplained(t *testing.T, judgment *modelv1.JudgeResponse, response modelSliceDescribed, steps ...string) {
	t.Helper()
	require.NotNil(t, judgment.GetExplained(), "unexplained Describe status: %s", response.GetInfo().GetStatus())
	require.Equal(t, steps, judgment.GetExplained().GetSteps())
	var outcomes []string
	for range steps {
		outcomes = append(outcomes, "Accepted")
	}
	require.Equal(t, outcomes, judgment.GetExplained().GetOutcomes())
}

func modelSliceReturn(reply *realizationv1.Reply, dc converter.DataConverter) ([]byte, error) {
	request := reply.GetRequest()
	if request == nil {
		return nil, errors.New("Model reply has no request")
	}
	switch {
	case request.MessageIs(new(workflowservice.RespondActivityTaskCompletedRequest)):
		completed := new(workflowservice.RespondActivityTaskCompletedRequest)
		if err := request.UnmarshalTo(completed); err != nil {
			return nil, err
		}
		if len(completed.GetResult().GetPayloads()) != 1 {
			return nil, errors.New("Model completion requires one result payload")
		}
		var result []byte
		if err := dc.FromPayloads(completed.Result, &result); err != nil {
			return nil, err
		}
		return result, nil
	case request.MessageIs(new(workflowservice.RespondActivityTaskFailedRequest)):
		failed := new(workflowservice.RespondActivityTaskFailedRequest)
		if err := request.UnmarshalTo(failed); err != nil {
			return nil, err
		}
		if failed.GetFailure().GetApplicationFailureInfo() == nil {
			return nil, errors.New("Model failure requires an application failure")
		}
		return nil, temporal.GetDefaultFailureConverter().FailureToError(failed.Failure)
	default:
		return nil, fmt.Errorf("unsupported Model reply request %q", request.GetTypeUrl())
	}
}

func TestTestpilotModelSlice(t *testing.T) {
	liveStart := time.Now()
	t.Cleanup(func() {
		t.Logf("live slice total=%s toolchain=%s platform=%s/%s CGO_ENABLED=%s", time.Since(liveStart), runtime.Version(), runtime.GOOS, runtime.GOARCH, os.Getenv("CGO_ENABLED"))
	})
	wasm := temporalmodel.Guest()
	t.Run("Annotations", func(t *testing.T) { modelSliceAnnotationControls(t, wasm) })
	host, err := model.New(t.Context(), wasm)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Second)
		defer cancel()
		require.NoError(t, host.Close(ctx))
	})
	env := activityEnvironment(t)
	annotations := &modelSliceAnnotations{host: host}
	t.Cleanup(func() { annotations.assert(t) })
	dc := converter.GetDefaultDataConverter()

	for _, scenario := range []string{"Complete", "Fail", "HostRefusal", "Cancellation", "CancelAfterCompletion", "CancelBeforeWorkerStart", "RetryThenComplete"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			host, err := model.New(ctx, wasm)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, host.Close(context.Background())) })
			queue := "model-slice-" + uuid.NewString()
			deliveries := make(chan modelSliceDelivery)
			finished := make(chan error, 1)
			var calls atomic.Int32
			failedAttempt := make(chan struct{}, 1)
			pollPermission := make(chan struct{}, 1)
			pollPermission <- struct{}{}
			retryTransport := func(callCtx context.Context, method string, req, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				gatedPoll := scenario == "RetryThenComplete" && method == workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName
				if gatedPoll {
					select {
					case <-pollPermission:
					case <-callCtx.Done():
						return callCtx.Err()
					}
				}
				err := invoke(callCtx, method, req, reply, conn, opts...)
				if gatedPoll {
					task, ok := reply.(*workflowservice.PollActivityTaskQueueResponse)
					if err != nil || !ok || len(task.GetTaskToken()) == 0 {
						pollPermission <- struct{}{}
					}
				}
				if scenario == "RetryThenComplete" && method == workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName && err == nil {
					select {
					case failedAttempt <- struct{}{}:
					default:
					}
				}
				return err
			}
			sdkClient, err := client.Dial(client.Options{
				HostPort: env.FrontendGRPCAddress(), Namespace: env.Namespace().String(), Logger: log.NewSdkLogger(env.Logger),
				ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{grpc.WithChainUnaryInterceptor(annotations.intercept, retryTransport)}},
			})
			require.NoError(t, err)
			t.Cleanup(sdkClient.Close)
			w := worker.New(sdkClient, queue, worker.Options{
				DisableWorkflowWorker: true, WorkerStopTimeout: time.Second,
			})
			w.RegisterActivityWithOptions(func(activityCtx context.Context, input []byte) (result []byte, bodyErr error) {
				calls.Add(1)
				defer func() {
					select {
					case finished <- bodyErr:
					default:
					}
				}()
				info := sdkactivity.GetInfo(activityCtx)
				payloads, err := dc.ToPayloads(input)
				if err != nil {
					return nil, err
				}
				delivery := modelSliceDelivery{
					task: &workflowservice.PollActivityTaskQueueResponse{
						TaskToken: info.TaskToken, ActivityId: info.ActivityID,
						WorkflowNamespace: info.Namespace, ActivityRunId: info.ActivityRunID,
						ActivityType: &commonpb.ActivityType{Name: info.ActivityType.Name},
						Input:        payloads, Attempt: info.Attempt,
						ScheduledTime: timestamppb.New(info.ScheduledTime), StartedTime: timestamppb.New(info.StartedTime),
					},
					reply: make(chan modelSliceReply, 1),
				}
				select {
				case deliveries <- delivery:
				case <-activityCtx.Done():
					return nil, activityCtx.Err()
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				select {
				case answer := <-delivery.reply:
					if answer.err != nil {
						return nil, answer.err
					}
					return modelSliceReturn(answer.reply, dc)
				case <-activityCtx.Done():
					return nil, activityCtx.Err()
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}, sdkactivity.RegisterOptions{Name: "ModelSlice"})
			started := false
			defer func() {
				cancel()
				if started {
					w.Stop()
				}
			}()
			input := scenario
			if scenario == "Cancellation" || scenario == "CancelAfterCompletion" || scenario == "CancelBeforeWorkerStart" {
				input = "Complete"
			}
			maximumAttempts := int32(1)
			retryPolicy := &temporal.RetryPolicy{MaximumAttempts: maximumAttempts}
			if scenario == "RetryThenComplete" {
				maximumAttempts = 2
				retryPolicy = &temporal.RetryPolicy{MaximumAttempts: maximumAttempts, InitialInterval: 2 * time.Second, MaximumInterval: 2 * time.Second}
			}
			startCtx, starts := modelSliceCapturing(ctx)
			handle, err := sdkClient.ExecuteActivity(startCtx, client.StartActivityOptions{
				ID: uuid.NewString(), TaskQueue: queue, StartToCloseTimeout: 20 * time.Second,
				RetryPolicy: retryPolicy,
			}, "ModelSlice", []byte(input))
			require.NoError(t, err)
			startCapture := starts.single(t)
			require.NoError(t, startCapture.rpcErr)
			require.NoError(t, startCapture.localErr)
			startRequest := startCapture.request.(*workflowservice.StartActivityExecutionRequest)
			startResponse := startCapture.response.(*workflowservice.StartActivityExecutionResponse)
			require.Equal(t, env.Namespace().String(), startRequest.GetNamespace())
			require.Equal(t, handle.GetID(), startRequest.GetActivityId())
			require.NotEmpty(t, startRequest.GetRequestId())
			require.Equal(t, maximumAttempts, startRequest.GetRetryPolicy().GetMaximumAttempts())
			require.Equal(t, handle.GetRunID(), startResponse.GetRunId())
			require.True(t, startResponse.GetStarted())
			startObservation := startCapture.observation(t)
			require.Equal(t, workflowservice.WorkflowService_StartActivityExecution_FullMethodName, startObservation.GetMethod())
			require.True(t, startObservation.GetRequest().MessageIs(startRequest))
			require.True(t, startObservation.GetResponse().MessageIs(startResponse))
			admission := modelSliceJudgeObservation(ctx, t, host, startObservation)
			require.NotNil(t, admission.GetExplained())
			require.Empty(t, admission.GetExplained().GetSteps())
			t.Logf("Start admitted namespace/activity/request/run identities; maximum_attempts=%d timeout recorded=%s", maximumAttempts, startRequest.GetStartToCloseTimeout())
			replayCtx, replays := modelSliceCapturing(ctx)
			_, err = sdkClient.WorkflowService().StartActivityExecution(replayCtx, proto.Clone(startRequest).(*workflowservice.StartActivityExecutionRequest))
			require.NoError(t, err)
			replayCapture := replays.single(t)
			require.NoError(t, replayCapture.rpcErr)
			require.NoError(t, replayCapture.localErr)
			replayResponse := replayCapture.response.(*workflowservice.StartActivityExecutionResponse)
			require.False(t, replayResponse.GetStarted())
			require.Equal(t, handle.GetRunID(), replayResponse.GetRunId())
			replayObservation := replayCapture.observation(t)
			require.NotNil(t, modelSliceJudgeObservation(ctx, t, host, replayObservation).GetExplained())
			describe := func() modelSliceDescribed {
				describeCtx, captures := modelSliceCapturing(ctx)
				_, err := sdkClient.WorkflowService().DescribeActivityExecution(describeCtx, &workflowservice.DescribeActivityExecutionRequest{
					Namespace: env.Namespace().String(), ActivityId: handle.GetID(), RunId: handle.GetRunID(), IncludeOutcome: true,
				})
				require.NoError(t, err)
				capture := captures.single(t)
				return modelSliceDescribed{
					DescribeActivityExecutionResponse: capture.response.(*workflowservice.DescribeActivityExecutionResponse),
					request:                           capture.request.(*workflowservice.DescribeActivityExecutionRequest),
				}
			}
			cancelActivity := func() (*realizationv1.Observation, error) {
				cancelCtx, captures := modelSliceCapturing(ctx)
				_, err := sdkClient.WorkflowService().RequestCancelActivityExecution(cancelCtx, &workflowservice.RequestCancelActivityExecutionRequest{
					Namespace: env.Namespace().String(), ActivityId: handle.GetID(), RunId: handle.GetRunID(),
					Identity: "model-slice", RequestId: uuid.NewString(), Reason: scenario,
				})
				capture := captures.single(t)
				require.NoError(t, capture.localErr)
				observation := capture.observation(t)
				if err != nil {
					require.Equal(t, int32(serviceerror.ToStatus(err).Code()), observation.GetError().GetCode())
				}
				t.Logf("actual cancel observation method=%s code=%d", observation.GetMethod(), observation.GetError().GetCode())
				return observation, err
			}
			scheduled := describe()
			require.Equal(t, enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, scheduled.GetInfo().GetStatus())
			require.Equal(t, enumspb.PENDING_ACTIVITY_STATE_SCHEDULED, scheduled.GetInfo().GetRunState())
			modelSliceExplained(t, modelSliceJudge(ctx, t, host, scheduled), scheduled)
			repeated := describe()
			modelSliceExplained(t, modelSliceJudge(ctx, t, host, repeated), repeated)
			require.Zero(t, calls.Load(), "worker must remain stopped until Scheduled observations are judged")
			if scenario == "CancelBeforeWorkerStart" {
				observation, err := cancelActivity()
				require.NoError(t, err)
				judgment := modelSliceJudgeObservation(ctx, t, host, observation)
				require.NotNil(t, judgment.GetExplained(), "cancel judgment: %v", judgment)
				require.Equal(t, []string{"Client::Cancel"}, judgment.GetExplained().GetSteps())
				require.Equal(t, []string{"Accepted"}, judgment.GetExplained().GetOutcomes())
				canceled := describe()
				require.Equal(t, enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED, canceled.GetInfo().GetStatus())
				require.Equal(t, enumspb.PENDING_ACTIVITY_STATE_UNSPECIFIED, canceled.GetInfo().GetRunState())
				modelSliceExplained(t, modelSliceJudge(ctx, t, host, canceled), canceled)
				repeated = describe()
				modelSliceExplained(t, modelSliceJudge(ctx, t, host, repeated), repeated)
				require.False(t, started)
				require.Zero(t, calls.Load(), "cancel before worker startup must not execute the activity body")
				return
			}
			require.NoError(t, w.Start())
			started = true
			var delivery modelSliceDelivery
			select {
			case delivery = <-deliveries:
			case err := <-finished:
				t.Fatalf("SDK body exited before callback: %v", err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			require.EqualValues(t, 1, delivery.task.GetAttempt())
			require.Equal(t, handle.GetID(), delivery.task.GetActivityId())
			require.NotEmpty(t, delivery.task.GetTaskToken())
			waitBody := func() error {
				waitCtx, waitCancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer waitCancel()
				select {
				case err := <-finished:
					return err
				case <-waitCtx.Done():
					t.Fatal("SDK body did not release its callback reply waiter")
					return waitCtx.Err()
				}
			}
			if scenario == "Cancellation" {
				cancel()
				require.ErrorIs(t, waitBody(), context.Canceled)
				require.EqualValues(t, 1, calls.Load())
				return
			}
			callback := modelSliceAny(t, &realizationv1.Callback{Task: modelSliceAny(t, delivery.task)})
			answerStart := time.Now()
			answer, answerErr := host.Answer(ctx, &modelv1.AnswerRequest{Callback: callback})
			t.Logf("Answer latency=%s scenario=%s", time.Since(answerStart), scenario)
			reply := new(realizationv1.Reply)
			if answerErr == nil {
				answerErr = answer.GetReply().UnmarshalTo(reply)
			}
			delivery.reply <- modelSliceReply{reply: reply, err: answerErr}
			if scenario == "HostRefusal" {
				var refusal *model.RefusalError
				require.ErrorAs(t, answerErr, &refusal)
				require.Equal(t, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED, refusal.Code)
				require.ErrorIs(t, waitBody(), answerErr)
				modelSliceExplained(t, modelSliceJudge(ctx, t, host, scheduled), scheduled)
				require.EqualValues(t, 1, calls.Load())
				return
			}
			require.NoError(t, answerErr)
			for _, evidence := range []*realizationv1.Observation{startObservation, replayObservation} {
				replay := modelSliceJudgeObservation(ctx, t, host, evidence)
				require.NotNil(t, replay.GetExplained())
				require.Empty(t, replay.GetExplained().GetSteps())
			}
			for _, attempt := range []int32{2, 3} {
				mismatched := proto.Clone(delivery.task).(*workflowservice.PollActivityTaskQueueResponse)
				mismatched.Attempt = attempt
				_, mismatchErr := host.Answer(ctx, &modelv1.AnswerRequest{Callback: modelSliceAny(t, &realizationv1.Callback{Task: modelSliceAny(t, mismatched)})})
				var refusal *model.RefusalError
				require.ErrorAs(t, mismatchErr, &refusal)
				require.Equal(t, modelv1.RefusalCode_REFUSAL_CODE_STATE, refusal.Code)
			}
			bodyErr := waitBody()
			if scenario == "RetryThenComplete" {
				var failure *temporal.ApplicationError
				require.ErrorAs(t, bodyErr, &failure)
				require.False(t, failure.NonRetryable())
				require.Equal(t, "Retry", failure.Message())
				select {
				case <-failedAttempt:
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				delayedDescribe := time.NewTimer(retryPolicy.MaximumInterval + 10*time.Millisecond)
				defer delayedDescribe.Stop()
				select {
				case <-delayedDescribe.C:
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				t.Logf("retry Describe delayed beyond maximum retry interval=%s after actual failure RPC success", retryPolicy.MaximumInterval)
				retryPoll := time.NewTicker(10 * time.Millisecond)
				defer retryPoll.Stop()
				for {
					response := describe()
					t.Logf("retry Describe attempt=%d status=%s run_state=%s", response.GetInfo().GetAttempt(), response.GetInfo().GetStatus(), response.GetInfo().GetRunState())
					judgment := modelSliceJudge(ctx, t, host, response)
					if response.GetInfo().GetAttempt() == 2 {
						require.Equal(t, enumspb.PENDING_ACTIVITY_STATE_SCHEDULED, response.GetInfo().GetRunState())
						modelSliceExplained(t, judgment, response, "Worker::RespondFailed { retryable: true }")
						break
					}
					modelSliceExplained(t, judgment, response)
					select {
					case <-retryPoll.C:
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				}
				require.EqualValues(t, 1, calls.Load(), "retry Poll must remain blocked through delayed Describe")
				pollPermission <- struct{}{}
				select {
				case delivery = <-deliveries:
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				require.EqualValues(t, 2, delivery.task.GetAttempt())
				require.Equal(t, handle.GetRunID(), delivery.task.GetActivityRunId())
				callback = modelSliceAny(t, &realizationv1.Callback{Task: modelSliceAny(t, delivery.task)})
				answer, err := host.Answer(ctx, &modelv1.AnswerRequest{Callback: callback})
				require.NoError(t, err)
				reply := new(realizationv1.Reply)
				require.NoError(t, answer.GetReply().UnmarshalTo(reply))
				current := describe()
				require.EqualValues(t, 2, current.GetInfo().GetAttempt())
				require.Equal(t, enumspb.PENDING_ACTIVITY_STATE_STARTED, current.GetInfo().GetRunState())
				modelSliceExplained(t, modelSliceJudge(ctx, t, host, current), current)
				delivery.reply <- modelSliceReply{reply: reply}
				bodyErr = waitBody()
			}
			wantStatus := enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED
			wantStep := "Worker::RespondCompleted"
			if scenario == "Fail" {
				wantStatus, wantStep = enumspb.ACTIVITY_EXECUTION_STATUS_FAILED, "Worker::RespondFailed { retryable: false }"
				var failure *temporal.ApplicationError
				require.ErrorAs(t, bodyErr, &failure)
				require.True(t, failure.NonRetryable())
				require.Equal(t, "MiniActivityFailure", failure.Type())
				require.Equal(t, "Fail", failure.Message())
			} else {
				require.NoError(t, bodyErr)
			}
			poll := time.NewTicker(10 * time.Millisecond)
			defer poll.Stop()
			var terminalResponse modelSliceDescribed
			for {
				response := describe()
				judgment := modelSliceJudge(ctx, t, host, response)
				if response.GetInfo().GetStatus() != enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING {
					modelSliceExplained(t, judgment, response, wantStep)
					require.Equal(t, wantStatus, response.GetInfo().GetStatus())
					terminalResponse = response
					break
				}
				modelSliceExplained(t, judgment, response)
				select {
				case <-poll.C:
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			repeated = describe()
			modelSliceExplained(t, modelSliceJudge(ctx, t, host, repeated), repeated)
			require.Equal(t, maximumAttempts, terminalResponse.GetInfo().GetAttempt())
			require.Equal(t, maximumAttempts, calls.Load())
			var result []byte
			resultErr := handle.Get(ctx, &result)
			if scenario == "Fail" {
				var failure *temporal.ApplicationError
				require.ErrorAs(t, resultErr, &failure)
				require.True(t, failure.NonRetryable())
				require.Equal(t, "MiniActivityFailure", failure.Type())
				require.Equal(t, "Fail", failure.Message())
			} else {
				require.NoError(t, resultErr)
				require.Equal(t, []byte("Complete"), result)
			}
			var frozen *modelv1.JudgeResponse
			if scenario == "CancelAfterCompletion" {
				observation, err := cancelActivity()
				require.Error(t, err)
				require.Equal(t, codes.NotFound, serviceerror.ToStatus(err).Code())
				require.Equal(t, int32(codes.NotFound), observation.GetError().GetCode())
				require.NotEmpty(t, observation.GetError().GetMessage())
				judgment := modelSliceJudgeObservation(ctx, t, host, observation)
				require.NotNil(t, judgment.GetExplained(), "cancel judgment: %v", judgment)
				require.Equal(t, []string{"Client::Cancel"}, judgment.GetExplained().GetSteps())
				require.Equal(t, []string{"Rejected(NotFound)"}, judgment.GetExplained().GetOutcomes())
				repeated = describe()
				require.Equal(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, repeated.GetInfo().GetStatus())
				modelSliceExplained(t, modelSliceJudge(ctx, t, host, repeated), repeated)
				mutated := proto.Clone(observation).(*realizationv1.Observation)
				mutated.GetError().Code = int32(codes.FailedPrecondition)
				frozen = modelSliceJudgeObservation(ctx, t, host, mutated)
				require.Equal(t, int32(codes.NotFound), observation.GetError().GetCode())
				require.True(t, proto.Equal(frozen, modelSliceJudgeObservation(ctx, t, host, observation)))
				require.True(t, proto.Equal(frozen, modelSliceJudge(ctx, t, host, describe())))
			} else {
				impossible := modelSliceDescribed{proto.Clone(terminalResponse.DescribeActivityExecutionResponse).(*workflowservice.DescribeActivityExecutionResponse), terminalResponse.request}
				impossible.Info.Status = enumspb.ACTIVITY_EXECUTION_STATUS_FAILED
				if wantStatus == enumspb.ACTIVITY_EXECUTION_STATUS_FAILED {
					impossible.Info.Status = enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED
				}
				frozen = modelSliceJudge(ctx, t, host, impossible)
			}
			require.NotNil(t, frozen.GetUnexplained())
			require.Equal(t, []string{"Client::Cancel"}, frozen.GetUnexplained().GetExpectedActions())
			require.True(t, proto.Equal(frozen, modelSliceJudge(ctx, t, host, terminalResponse)))
			if scenario != "RetryThenComplete" {
				require.True(t, proto.Equal(frozen, modelSliceJudge(ctx, t, host, scheduled)))
			}
			_, err = host.Answer(ctx, &modelv1.AnswerRequest{Callback: callback})
			var refusal *model.RefusalError
			require.ErrorAs(t, err, &refusal)
			require.Equal(t, modelv1.RefusalCode_REFUSAL_CODE_STATE, refusal.Code)
			isolated, err := model.New(ctx, wasm)
			require.NoError(t, err)
			defer func() { require.NoError(t, isolated.Close(context.Background())) }()
			require.NotNil(t, modelSliceJudgeObservation(ctx, t, isolated, startObservation).GetExplained())
			modelSliceExplained(t, modelSliceJudge(ctx, t, isolated, scheduled), scheduled)
			_, err = host.Reset(ctx, &modelv1.ResetRequest{})
			require.NoError(t, err)
			for _, evidence := range []*realizationv1.Observation{startObservation, replayObservation} {
				replay := modelSliceJudgeObservation(ctx, t, host, evidence)
				require.NotNil(t, replay.GetExplained())
				require.Empty(t, replay.GetExplained().GetSteps())
			}
			modelSliceExplained(t, modelSliceJudge(ctx, t, host, scheduled), scheduled)
		})
	}
}
