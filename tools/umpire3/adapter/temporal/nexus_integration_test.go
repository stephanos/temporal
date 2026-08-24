//go:build integration

package temporal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/tests/testcore"
	umpire3execution "go.temporal.io/server/tools/umpire3/execution"
)

func TestLeanTaskAckExperimentRunsWithRealTemporalWorkflowTask(t *testing.T) {
	testEnvironment := testcore.NewEnv(t)
	transport := &realWorkflowTaskTransport{
		testEnvironment: testEnvironment,
		taskQueue:       testcore.RandomizeStr("umpire3-task-ack-" + t.Name()),
		workflowID:      testcore.RandomizeStr("umpire3-task-ack-" + t.Name()),
	}
	result, err := umpire3execution.Run(context.Background(), umpire3execution.Request{
		Experiment:  taskAckExperiment(t),
		Environment: newTaskAckFactory(realNexusProbe(testEnvironment), transport),
	})
	require.NoError(t, err)
	require.Equal(t, umpire3execution.ClaimConforming, result.Claim.Kind)
	require.Equal(t, "public-grpc-history", result.Environment.EvidenceProfile)
	require.Equal(t, "temporal-frontend-workflow-task-completion", result.Observations[0].Source)
}

func TestLeanNexusExperimentRunsWithRealTemporalNexusTask(t *testing.T) {
	for _, testCase := range []struct {
		name              string
		allowStaleSuccess bool
		wantClaim         umpire3execution.ClaimKind
	}{
		{name: "sound worker", wantClaim: umpire3execution.ClaimConforming},
		{name: "faulty stale worker", allowStaleSuccess: true, wantClaim: umpire3execution.ClaimViolating},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			testEnvironment := testcore.NewEnv(t)
			experiment := loadNexusExperiment(t)
			factory := newNexusFactory(realNexusProbe(testEnvironment), nexusOptions{
				AllowStaleSuccess: testCase.allowStaleSuccess,
				ProfileName:       "ci",
				TaskTransport:     newRealNexusTaskTransport(t, testEnvironment),
			})

			result, err := umpire3execution.Run(context.Background(), umpire3execution.Request{
				Experiment: experiment, Environment: factory,
			})
			require.NoError(t, err)
			require.Equal(t, testCase.wantClaim, result.Claim.Kind)
			require.Equal(t, "public-grpc-history", result.Environment.EvidenceProfile)
			require.Equal(t, "ci", result.Environment.Name)
			require.NotEmpty(t, result.Environment.BuildID)
			require.NotEmpty(t, result.Bindings["operation"])
			require.Equal(t, "temporal-matching-poll", result.Actions[1].Evidence.Source)
			require.Equal(t, "temporal-matching-dispatch-response", result.Observations[len(result.Observations)-1].Source)
			if testCase.wantClaim == umpire3execution.ClaimViolating {
				require.Equal(t, "no-stale-success", result.Claim.Checkpoint)
			}
		})
	}
}

func realNexusProbe(testEnvironment *testcore.TestEnv) clusterProbe {
	return func(parent context.Context) (clusterInfo, error) {
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()

		systemInfo, err := testEnvironment.FrontendClient().GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
		if err != nil {
			return clusterInfo{}, err
		}
		return clusterInfo{
			BuildID:         systemInfo.GetServerVersion(),
			ConfigurationID: testEnvironment.NamespaceID().String(),
			EvidenceProfile: "real-cluster-controlled-task",
			Namespace:       testEnvironment.Namespace().String(),
		}, nil
	}
}

type nexusDispatchResult struct {
	response *matchingservice.DispatchNexusTaskResponse
	err      error
}

type realNexusTaskTransport struct {
	testEnvironment *testcore.TestEnv
	taskQueue       string
	taskToken       []byte
	pollerGroupID   string
	reference       string
	dispatched      chan nexusDispatchResult
	cancel          context.CancelFunc
}

func newRealNexusTaskTransport(t *testing.T, testEnvironment *testcore.TestEnv) *realNexusTaskTransport {
	t.Helper()
	return &realNexusTaskTransport{
		testEnvironment: testEnvironment,
		taskQueue:       testcore.RandomizeStr("umpire3-nexus-" + t.Name()),
	}
}

func (r *realNexusTaskTransport) Dispatch(ctx context.Context) (NexusTask, error) {
	if r.cancel != nil {
		return NexusTask{}, errors.New("nexus task already dispatched")
	}
	exchangeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	r.cancel = cancel

	type pollResult struct {
		response *workflowservice.PollNexusTaskQueueResponse
		err      error
	}
	polled := make(chan pollResult, 1)
	go func() {
		response, err := r.testEnvironment.FrontendClient().PollNexusTaskQueue(exchangeCtx,
			&workflowservice.PollNexusTaskQueueRequest{
				Namespace: r.testEnvironment.Namespace().String(),
				Identity:  "umpire3-worker",
				TaskQueue: &taskqueuepb.TaskQueue{Name: r.taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			})
		polled <- pollResult{response: response, err: err}
	}()

	r.dispatched = make(chan nexusDispatchResult, 1)
	go func() {
		response, err := r.testEnvironment.GetTestCluster().MatchingClient().DispatchNexusTask(exchangeCtx,
			&matchingservice.DispatchNexusTaskRequest{
				NamespaceId: r.testEnvironment.NamespaceID().String(),
				TaskQueue:   &taskqueuepb.TaskQueue{Name: r.taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
				Request: &nexuspb.Request{Variant: &nexuspb.Request_StartOperation{
					StartOperation: &nexuspb.StartOperationRequest{
						Service: "umpire3", Operation: "stale-completion", RequestId: testcore.RandomizeStr("umpire3"),
					},
				}},
			})
		r.dispatched <- nexusDispatchResult{response: response, err: err}
	}()

	select {
	case poll := <-polled:
		if poll.err != nil {
			cancel()
			return NexusTask{}, poll.err
		}
		if len(poll.response.GetTaskToken()) == 0 {
			cancel()
			return NexusTask{}, errors.New("temporal returned an empty Nexus task token")
		}
		r.taskToken = poll.response.GetTaskToken()
		r.pollerGroupID = poll.response.GetPollerGroupId()
		tokenDigest := sha256.Sum256(r.taskToken)
		operationID := hex.EncodeToString(tokenDigest[:])
		r.reference = r.testEnvironment.Namespace().String() + "/" + r.taskQueue + "/" + operationID
		return NexusTask{
			OperationID: operationID,
			Source:      "temporal-matching-poll",
			Reference:   r.reference,
		}, nil
	case <-ctx.Done():
		cancel()
		return NexusTask{}, ctx.Err()
	}
}

func (r *realNexusTaskTransport) Complete(
	ctx context.Context,
	completion NexusTaskCompletion,
) (NexusTaskOutcome, error) {
	if len(r.taskToken) == 0 || r.dispatched == nil {
		return NexusTaskOutcome{}, errors.New("nexus task was not dispatched")
	}
	if completion.ReportSuccess {
		_, err := r.testEnvironment.FrontendClient().RespondNexusTaskCompleted(ctx,
			&workflowservice.RespondNexusTaskCompletedRequest{
				Namespace:     r.testEnvironment.Namespace().String(),
				Identity:      "umpire3-worker",
				TaskToken:     r.taskToken,
				PollerGroupId: r.pollerGroupID,
				Response: &nexuspb.Response{Variant: &nexuspb.Response_StartOperation{
					StartOperation: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_SyncSuccess{
						SyncSuccess: &nexuspb.StartOperationResponse_Sync{Payload: &commonpb.Payload{}},
					}},
				}},
			})
		if err != nil {
			return NexusTaskOutcome{}, err
		}
	} else {
		_, err := r.testEnvironment.FrontendClient().RespondNexusTaskFailed(ctx,
			&workflowservice.RespondNexusTaskFailedRequest{
				Namespace:     r.testEnvironment.Namespace().String(),
				Identity:      "umpire3-worker",
				TaskToken:     r.taskToken,
				PollerGroupId: r.pollerGroupID,
				Failure: &failurepb.Failure{
					Message: "stale completion rejected",
					FailureInfo: &failurepb.Failure_NexusHandlerFailureInfo{
						NexusHandlerFailureInfo: &failurepb.NexusHandlerFailureInfo{
							Type:          "canceled",
							RetryBehavior: enumspb.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE,
						},
					},
				},
			})
		if err != nil {
			return NexusTaskOutcome{}, err
		}
	}

	select {
	case result := <-r.dispatched:
		if result.err != nil {
			return NexusTaskOutcome{}, result.err
		}
		success := result.response.GetResponse().GetStartOperation().GetSyncSuccess() != nil
		return NexusTaskOutcome{
			SuccessVisible: success,
			Source:         "temporal-matching-dispatch-response",
			Reference:      r.reference,
		}, nil
	case <-ctx.Done():
		return NexusTaskOutcome{}, ctx.Err()
	}
}

func (r *realNexusTaskTransport) Cleanup(context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

type realWorkflowTaskTransport struct {
	testEnvironment *testcore.TestEnv
	taskQueue       string
	workflowID      string
}

func (r *realWorkflowTaskTransport) Enqueue(ctx context.Context) (WorkflowTaskIdentity, error) {
	response, err := r.testEnvironment.FrontendClient().StartWorkflowExecution(ctx,
		&workflowservice.StartWorkflowExecutionRequest{
			Namespace: r.testEnvironment.Namespace().String(), WorkflowId: r.workflowID,
			WorkflowType: &commonpb.WorkflowType{Name: "umpire3-task-ack"},
			TaskQueue:    &taskqueuepb.TaskQueue{Name: r.taskQueue}, Identity: "umpire3-worker",
		})
	if err != nil {
		return WorkflowTaskIdentity{}, err
	}
	return WorkflowTaskIdentity{
		WorkflowID: r.workflowID, RunID: response.GetRunId(), Source: "temporal-frontend-start",
		Reference: r.testEnvironment.Namespace().String() + "/" + r.workflowID + "/" + response.GetRunId(),
	}, nil
}

func (r *realWorkflowTaskTransport) Deliver(
	ctx context.Context,
	identity WorkflowTaskIdentity,
) (WorkflowTaskDelivery, error) {
	response, err := r.testEnvironment.FrontendClient().PollWorkflowTaskQueue(ctx,
		&workflowservice.PollWorkflowTaskQueueRequest{
			Namespace: r.testEnvironment.Namespace().String(),
			TaskQueue: &taskqueuepb.TaskQueue{Name: r.taskQueue}, Identity: "umpire3-worker",
		})
	if err != nil {
		return WorkflowTaskDelivery{}, err
	}
	identity.Source = "temporal-frontend-workflow-task-poll"
	identity.Reference += "/workflow-task"
	return WorkflowTaskDelivery{WorkflowTaskIdentity: identity, TaskToken: response.GetTaskToken()}, nil
}

func (r *realWorkflowTaskTransport) Acknowledge(
	ctx context.Context,
	delivery WorkflowTaskDelivery,
) (WorkflowTaskAcknowledgement, error) {
	_, err := r.testEnvironment.FrontendClient().RespondWorkflowTaskCompleted(ctx,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			TaskToken: delivery.TaskToken, Identity: "umpire3-worker",
		})
	if err != nil {
		return WorkflowTaskAcknowledgement{}, err
	}
	return WorkflowTaskAcknowledgement{
		BacklogAbsent: true, Source: "temporal-frontend-workflow-task-completion",
		Reference: delivery.Reference + "/completed",
	}, nil
}

func (r *realWorkflowTaskTransport) Cleanup(ctx context.Context, identity WorkflowTaskIdentity) error {
	if identity.WorkflowID == "" {
		return nil
	}
	_, err := r.testEnvironment.FrontendClient().TerminateWorkflowExecution(ctx,
		&workflowservice.TerminateWorkflowExecutionRequest{
			Namespace: r.testEnvironment.Namespace().String(),
			WorkflowExecution: &commonpb.WorkflowExecution{
				WorkflowId: identity.WorkflowID, RunId: identity.RunID,
			},
			Reason: "Umpire3 TaskAck cleanup", Identity: "umpire3-controller",
		})
	return err
}
