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
	"go.temporal.io/api/workflowservice/v1"
	sdkactivity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"umpire.local/model"
	modelv1 "umpire.local/model/umpire/model/v1"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

type modelSliceReply struct {
	reply *temporalv1.Reply
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

func modelSliceJudge(ctx context.Context, t *testing.T, host *model.Host, response *workflowservice.DescribeActivityExecutionResponse) *modelv1.JudgeResponse {
	t.Helper()
	observation := modelSliceAny(t, &temporalv1.Observation{
		Method: workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName,
		Result: &temporalv1.Observation_Response{Response: modelSliceAny(t, response)},
	})
	judgment, err := host.Judge(ctx, &modelv1.JudgeRequest{Observation: observation})
	require.NoError(t, err, "Describe observation: %v", response)
	return judgment
}

func modelSliceExplained(t *testing.T, judgment *modelv1.JudgeResponse, response *workflowservice.DescribeActivityExecutionResponse, steps ...string) {
	t.Helper()
	require.NotNil(t, judgment.GetExplained(), "first unexplained Describe observation: %v; judgment: %v", response, judgment)
	require.Equal(t, steps, judgment.GetExplained().GetSteps(), "Describe observation: %v", response)
}

func modelSliceReturn(reply *temporalv1.Reply, dc converter.DataConverter) ([]byte, error) {
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
		if failed.GetFailure().GetApplicationFailureInfo() == nil || !failed.GetFailure().GetApplicationFailureInfo().GetNonRetryable() {
			return nil, errors.New("Model failure requires a non-retryable application failure")
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
	guestPath := os.Getenv("UMPIRE_TEMPORAL_WASM")
	require.NotEmpty(t, guestPath, "UMPIRE_TEMPORAL_WASM must name the built Temporal guest; run make umpire-check-model-slice")
	wasm, err := os.ReadFile(guestPath)
	require.NoError(t, err)
	host, err := model.New(t.Context(), wasm)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Second)
		defer cancel()
		require.NoError(t, host.Close(ctx))
	})
	isolated, err := model.New(t.Context(), wasm)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Second)
		defer cancel()
		require.NoError(t, isolated.Close(ctx))
	})
	env := activityEnvironment(t)
	sdkClient := env.SdkClient()
	dc := converter.GetDefaultDataConverter()

	for _, scenario := range []string{"Complete", "Fail", "HostRefusal", "Cancellation"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			_, err := host.Reset(ctx, &modelv1.ResetRequest{})
			require.NoError(t, err)
			queue := "model-slice-" + uuid.NewString()
			deliveries := make(chan modelSliceDelivery)
			finished := make(chan error, 1)
			var calls atomic.Int32
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
			if scenario == "Cancellation" {
				input = "Complete"
			}
			handle, err := sdkClient.ExecuteActivity(ctx, client.StartActivityOptions{
				ID: uuid.NewString(), TaskQueue: queue, StartToCloseTimeout: 20 * time.Second,
				RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1},
			}, "ModelSlice", []byte(input))
			require.NoError(t, err)
			describe := func() *workflowservice.DescribeActivityExecutionResponse {
				response, err := env.FrontendClient().DescribeActivityExecution(ctx, &workflowservice.DescribeActivityExecutionRequest{
					Namespace: env.Namespace().String(), ActivityId: handle.GetID(), RunId: handle.GetRunID(), IncludeOutcome: true,
				})
				require.NoError(t, err)
				return response
			}
			scheduled := describe()
			require.Equal(t, enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, scheduled.GetInfo().GetStatus())
			require.Equal(t, enumspb.PENDING_ACTIVITY_STATE_SCHEDULED, scheduled.GetInfo().GetRunState())
			modelSliceExplained(t, modelSliceJudge(ctx, t, host, scheduled), scheduled)
			repeated := describe()
			modelSliceExplained(t, modelSliceJudge(ctx, t, host, repeated), repeated)
			require.Zero(t, calls.Load(), "worker must remain stopped until Scheduled observations are judged")
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
			callback := modelSliceAny(t, &temporalv1.Callback{Task: modelSliceAny(t, delivery.task)})
			answerStart := time.Now()
			answer, answerErr := host.Answer(ctx, &modelv1.AnswerRequest{Callback: callback})
			t.Logf("Answer latency=%s scenario=%s", time.Since(answerStart), scenario)
			reply := new(temporalv1.Reply)
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
			bodyErr := waitBody()
			wantStatus := enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED
			wantStep := "Worker::RespondCompleted"
			if scenario == "Fail" {
				wantStatus, wantStep = enumspb.ACTIVITY_EXECUTION_STATUS_FAILED, "Worker::RespondFailed"
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
			var terminalResponse *workflowservice.DescribeActivityExecutionResponse
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
			require.EqualValues(t, 1, terminalResponse.GetInfo().GetAttempt())
			require.EqualValues(t, 1, calls.Load())
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
			impossible := proto.Clone(terminalResponse).(*workflowservice.DescribeActivityExecutionResponse)
			impossible.Info.Status = enumspb.ACTIVITY_EXECUTION_STATUS_FAILED
			if wantStatus == enumspb.ACTIVITY_EXECUTION_STATUS_FAILED {
				impossible.Info.Status = enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED
			}
			frozen := modelSliceJudge(ctx, t, host, impossible)
			require.NotNil(t, frozen.GetUnexplained())
			require.Equal(t, []string{"Client::Cancel"}, frozen.GetUnexplained().GetExpectedActions())
			require.True(t, proto.Equal(frozen, modelSliceJudge(ctx, t, host, terminalResponse)))
			require.True(t, proto.Equal(frozen, modelSliceJudge(ctx, t, host, scheduled)))
			_, err = host.Answer(ctx, &modelv1.AnswerRequest{Callback: callback})
			var refusal *model.RefusalError
			require.ErrorAs(t, err, &refusal)
			require.Equal(t, modelv1.RefusalCode_REFUSAL_CODE_STATE, refusal.Code)
			modelSliceExplained(t, modelSliceJudge(ctx, t, isolated, scheduled), scheduled)
			_, err = host.Reset(ctx, &modelv1.ResetRequest{})
			require.NoError(t, err)
			modelSliceExplained(t, modelSliceJudge(ctx, t, host, scheduled), scheduled)
		})
	}
}
