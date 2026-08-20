package tests

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexus"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/adminservice/v1"
	"go.temporal.io/server/chasm"
	chasmnexus "go.temporal.io/server/chasm/lib/nexusoperation"
	"go.temporal.io/server/common/dynamicconfig"
	commonnexus "go.temporal.io/server/common/nexus"
	"go.temporal.io/server/common/nexus/nexusrpc"
	"go.temporal.io/server/common/nexus/nexustest"
	"go.temporal.io/server/common/payload"
	"go.temporal.io/server/common/payloads"
	"go.temporal.io/server/common/testing/testhooks"
	"go.temporal.io/server/components/callbacks"
	"go.temporal.io/server/components/nexusoperations"
	"go.temporal.io/server/tests/testcore"
	"go.temporal.io/server/tests/umpire3/participant"
	"go.temporal.io/server/tests/umpire3/protocol"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

type umpire3WorkflowTaskFencer struct {
	env *testcore.TestEnv
}

type umpire3NexusLifecycleObserver struct {
	env *NexusTestEnv
	t   *testing.T
}

type umpire3NexusLifecycleExecution struct {
	endpoint  string
	operation string
	runID     string
	taskQueue string
	task      *workflowservice.PollNexusTaskQueueResponse
	callback  chan umpire3NexusLifecycleCallback
}

type umpire3NexusLifecycleCallback struct {
	url   string
	token string
}

func newUmpire3NexusLifecycleObserver(t *testing.T) *umpire3NexusLifecycleObserver {
	t.Helper()
	return &umpire3NexusLifecycleObserver{t: t, env: newNexusTestEnv(t, true,
		testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true),
		testcore.WithDynamicConfig(dynamicconfig.EnableCHASMCallbacks, true),
		testcore.WithDynamicConfig(chasmnexus.EnableChasmWorkflowOperations, true),
		testcore.WithDynamicConfig(chasmnexus.Enabled, true),
		testcore.WithDynamicConfig(chasmnexus.RetryPolicy, chasmnexus.RetryPolicyConfig{
			InitialInterval: 5 * time.Second,
			MaxInterval:     5 * time.Second,
		}),
	)}
}

func (o *umpire3NexusLifecycleObserver) Observe(ctx context.Context, edge string) error {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Minute)
		defer cancel()
	}
	parts := strings.Split(edge, "/")
	if len(parts) != 4 || parts[0] != "nexus-operation" {
		return fmt.Errorf("invalid generated Nexus lifecycle edge %q", edge)
	}
	from, action, to := parts[1], parts[2], parts[3]
	if from == "unspecified" && action == "reject" {
		return o.reject(ctx, to)
	}
	execution, cleanup, err := o.prepare(ctx, from, action)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := o.apply(ctx, execution, from, action); err != nil {
		return err
	}
	return o.requireState(ctx, execution, to)
}

func (o *umpire3NexusLifecycleObserver) reject(ctx context.Context, to string) error {
	operation := "umpire3-explore-reject-" + uuid.NewString()
	_, err := o.env.FrontendClient().StartNexusOperationExecution(ctx,
		&workflowservice.StartNexusOperationExecutionRequest{
			Namespace: o.env.Namespace().String(), OperationId: operation, RequestId: uuid.NewString(),
			Endpoint: "umpire3-explore-missing-endpoint", Service: "service", Operation: "operation",
			ScheduleToCloseTimeout: durationpb.New(30 * time.Second),
		})
	if err == nil || serviceerror.ToStatus(err).Code() != codes.NotFound {
		return fmt.Errorf("Nexus rejection evidence is unavailable: %w", err)
	}
	if to != "rejected" {
		return fmt.Errorf("Nexus rejection produced rejected, generated edge expects %q", to)
	}
	return nil
}

func (o *umpire3NexusLifecycleObserver) prepare(
	ctx context.Context,
	from string,
	action string,
) (*umpire3NexusLifecycleExecution, func(), error) {
	taskQueue := "umpire3-explore-" + uuid.NewString()
	execution := &umpire3NexusLifecycleExecution{
		operation: "umpire3-explore-" + uuid.NewString(), taskQueue: taskQueue,
	}
	if from == "started" {
		execution.callback = make(chan umpire3NexusLifecycleCallback, 1)
		execution.endpoint = o.env.createRandomExternalNexusServer(ctx, o.t, nexustest.Handler{
			OnStartOperation: func(
				_ context.Context,
				_ string,
				_ string,
				_ *nexus.LazyValue,
				options nexus.StartOperationOptions,
			) (nexus.HandlerStartOperationResult[any], error) {
				execution.callback <- umpire3NexusLifecycleCallback{
					url:   options.CallbackURL,
					token: options.CallbackHeader.Get(commonnexus.CallbackTokenHeader),
				}
				return &nexus.HandlerStartOperationResultAsync{OperationToken: "umpire3-lifecycle-token"}, nil
			},
		})
	} else {
		endpoint := o.env.createNexusEndpoint(ctx, o.t, testcore.RandomizedNexusEndpoint(o.t.Name()), taskQueue)
		execution.endpoint = endpoint.GetSpec().GetName()
	}
	hookCleanup := func() {}
	if action == "timeout" {
		switch from {
		case "scheduled":
			hookCleanup = o.env.InjectHook(testhooks.NewHook(testhooks.NexusOperationForceTimeout,
				testhooks.NexusForceTimeoutFromScheduled))
		case "backing-off":
			hookCleanup = o.env.InjectHook(testhooks.NewHook(testhooks.NexusOperationForceTimeout,
				testhooks.NexusForceTimeoutFromBackingOff))
		}
	}
	startToClose := 2 * time.Minute
	if from == "started" && action == "timeout" {
		startToClose = time.Second
	}
	started, err := o.env.FrontendClient().StartNexusOperationExecution(ctx,
		&workflowservice.StartNexusOperationExecutionRequest{
			Namespace: o.env.Namespace().String(), OperationId: execution.operation, RequestId: uuid.NewString(),
			Endpoint: execution.endpoint, Service: "service", Operation: "operation",
			ScheduleToCloseTimeout: durationpb.New(2 * time.Minute),
			StartToCloseTimeout:    durationpb.New(startToClose),
		})
	if err != nil {
		hookCleanup()
		return nil, func() {}, fmt.Errorf("schedule standalone Nexus operation: %w", err)
	}
	execution.runID = started.GetRunId()
	cleanup := func() {
		hookCleanup()
		describe, describeErr := o.describe(context.Background(), execution)
		if describeErr == nil && describe.GetInfo().GetStatus() == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_RUNNING {
			_, _ = o.env.FrontendClient().TerminateNexusOperationExecution(context.Background(),
				&workflowservice.TerminateNexusOperationExecutionRequest{
					Namespace: o.env.Namespace().String(), OperationId: execution.operation,
					RunId: execution.runID, Reason: "Umpire3 lifecycle observation cleanup",
				})
		}
	}
	if from != "unspecified" && (from != "scheduled" || action != "timeout") {
		if err := o.requireState(ctx, execution, "scheduled"); err != nil {
			cleanup()
			return nil, func() {}, err
		}
	}
	switch from {
	case "unspecified", "scheduled":
	case "backing-off":
		if err := o.pollStart(ctx, execution); err != nil {
			cleanup()
			return nil, func() {}, err
		}
		if err := o.respondRetryable(ctx, execution); err != nil {
			cleanup()
			return nil, func() {}, err
		}
		if err := o.requireState(ctx, execution, from); err != nil {
			cleanup()
			return nil, func() {}, err
		}
	case "started":
		_, err := o.env.FrontendClient().PollNexusOperationExecution(ctx,
			&workflowservice.PollNexusOperationExecutionRequest{
				Namespace: o.env.Namespace().String(), OperationId: execution.operation,
				RunId: execution.runID, WaitStage: enumspb.NEXUS_OPERATION_WAIT_STAGE_STARTED,
			})
		if err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("wait for generated asynchronous Nexus start: %w", err)
		}
		if err := o.requireState(ctx, execution, from); err != nil {
			cleanup()
			return nil, func() {}, err
		}
	default:
		cleanup()
		return nil, func() {}, fmt.Errorf("unsupported generated Nexus lifecycle source %q", from)
	}
	return execution, cleanup, nil
}

func (o *umpire3NexusLifecycleObserver) apply(
	ctx context.Context,
	execution *umpire3NexusLifecycleExecution,
	from string,
	action string,
) error {
	switch action {
	case "schedule":
		if from == "backing-off" {
			return o.pollStart(ctx, execution)
		}
		return nil
	case "attempt-failed":
		if err := o.pollStart(ctx, execution); err != nil {
			return err
		}
		return o.respondRetryable(ctx, execution)
	case "start":
		if err := o.pollStart(ctx, execution); err != nil {
			return err
		}
		return o.respondAsync(ctx, execution)
	case "succeed", "fail", "cancel":
		if from == "scheduled" {
			if err := o.pollStart(ctx, execution); err != nil {
				return err
			}
			return o.respondTerminal(ctx, execution, action)
		}
		return o.completeAsync(ctx, execution, action)
	case "terminate":
		_, err := o.env.FrontendClient().TerminateNexusOperationExecution(ctx,
			&workflowservice.TerminateNexusOperationExecutionRequest{
				Namespace: o.env.Namespace().String(), OperationId: execution.operation,
				RunId: execution.runID, Reason: "Umpire3 generated lifecycle termination",
			})
		return err
	case "timeout":
		_, err := o.env.FrontendClient().PollNexusOperationExecution(ctx,
			&workflowservice.PollNexusOperationExecutionRequest{
				Namespace: o.env.Namespace().String(), OperationId: execution.operation,
				RunId: execution.runID, WaitStage: enumspb.NEXUS_OPERATION_WAIT_STAGE_CLOSED,
			})
		return err
	default:
		return fmt.Errorf("unsupported generated Nexus lifecycle action %q", action)
	}
}

func (o *umpire3NexusLifecycleObserver) pollStart(
	ctx context.Context,
	execution *umpire3NexusLifecycleExecution,
) error {
	task, err := o.env.FrontendClient().PollNexusTaskQueue(ctx, &workflowservice.PollNexusTaskQueueRequest{
		Namespace: o.env.Namespace().String(), Identity: "umpire3-lifecycle-observer",
		TaskQueue: &taskqueuepb.TaskQueue{Name: execution.taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
	})
	if err != nil {
		return fmt.Errorf("poll generated Nexus start task: %w", err)
	}
	if task.GetRequest().GetStartOperation() == nil {
		return errors.New("generated Nexus task is not a start operation")
	}
	execution.task = task
	return nil
}

func (o *umpire3NexusLifecycleObserver) respondRetryable(
	ctx context.Context,
	execution *umpire3NexusLifecycleExecution,
) error {
	_, err := o.env.FrontendClient().RespondNexusTaskFailed(ctx,
		&workflowservice.RespondNexusTaskFailedRequest{
			Namespace: o.env.Namespace().String(), Identity: "umpire3-lifecycle-observer",
			TaskToken: execution.task.GetTaskToken(), Error: &nexuspb.HandlerError{
				ErrorType: string(nexus.HandlerErrorTypeUnavailable),
				Failure:   &nexuspb.Failure{Message: "Umpire3 generated retryable failure"},
			},
		})
	return err
}

func (o *umpire3NexusLifecycleObserver) respondAsync(
	ctx context.Context,
	execution *umpire3NexusLifecycleExecution,
) error {
	_, err := o.env.FrontendClient().RespondNexusTaskCompleted(ctx,
		&workflowservice.RespondNexusTaskCompletedRequest{
			Namespace: o.env.Namespace().String(), Identity: "umpire3-lifecycle-observer",
			TaskToken: execution.task.GetTaskToken(), Response: &nexuspb.Response{
				Variant: &nexuspb.Response_StartOperation{StartOperation: &nexuspb.StartOperationResponse{
					Variant: &nexuspb.StartOperationResponse_AsyncSuccess{AsyncSuccess: &nexuspb.StartOperationResponse_Async{
						OperationToken: "umpire3-lifecycle-token",
					}},
				}},
			},
		})
	return err
}

func (o *umpire3NexusLifecycleObserver) respondTerminal(
	ctx context.Context,
	execution *umpire3NexusLifecycleExecution,
	action string,
) error {
	response := &nexuspb.StartOperationResponse{}
	switch action {
	case "succeed":
		response.Variant = &nexuspb.StartOperationResponse_SyncSuccess{SyncSuccess: &nexuspb.StartOperationResponse_Sync{
			Payload: payload.EncodeString("umpire3-lifecycle-result"),
		}}
	case "fail", "cancel":
		response.Variant = &nexuspb.StartOperationResponse_OperationError{OperationError: &nexuspb.UnsuccessfulOperationError{
			OperationState: action + "ed", Failure: &nexuspb.Failure{Message: "Umpire3 generated " + action},
		}}
	}
	_, err := o.env.FrontendClient().RespondNexusTaskCompleted(ctx,
		&workflowservice.RespondNexusTaskCompletedRequest{
			Namespace: o.env.Namespace().String(), Identity: "umpire3-lifecycle-observer",
			TaskToken: execution.task.GetTaskToken(), Response: &nexuspb.Response{
				Variant: &nexuspb.Response_StartOperation{StartOperation: response},
			},
		})
	return err
}

func (o *umpire3NexusLifecycleObserver) completeAsync(
	ctx context.Context,
	execution *umpire3NexusLifecycleExecution,
	action string,
) error {
	var callback umpire3NexusLifecycleCallback
	select {
	case callback = <-execution.callback:
	case <-ctx.Done():
		return ctx.Err()
	}
	if callback.url == "" || callback.token == "" {
		return errors.New("generated asynchronous Nexus start lacks callback routing evidence")
	}
	options := nexusrpc.CompleteOperationOptions{
		Header: nexus.Header{commonnexus.CallbackTokenHeader: callback.token},
	}
	switch action {
	case "succeed":
		options.Result = payload.EncodeString("umpire3-lifecycle-result")
	case "fail":
		options.Error = nexus.NewOperationFailedErrorf("Umpire3 generated asynchronous failure")
	case "cancel":
		options.Error = nexus.NewOperationCanceledErrorf("Umpire3 generated asynchronous cancellation")
	}
	client := nexusrpc.NewCompletionHTTPClient(nexusrpc.CompletionHTTPClientOptions{
		Serializer: commonnexus.PayloadSerializer,
	})
	return client.CompleteOperation(ctx, callback.url, options)
}

func (o *umpire3NexusLifecycleObserver) describe(
	ctx context.Context,
	execution *umpire3NexusLifecycleExecution,
) (*workflowservice.DescribeNexusOperationExecutionResponse, error) {
	return o.env.FrontendClient().DescribeNexusOperationExecution(ctx,
		&workflowservice.DescribeNexusOperationExecutionRequest{
			Namespace: o.env.Namespace().String(), OperationId: execution.operation, RunId: execution.runID,
			IncludeOutcome: true,
		})
}

func (o *umpire3NexusLifecycleObserver) requireState(
	ctx context.Context,
	execution *umpire3NexusLifecycleExecution,
	want string,
) error {
	describe, err := o.describe(ctx, execution)
	if err != nil {
		return fmt.Errorf("describe generated Nexus operation: %w", err)
	}
	info := describe.GetInfo()
	if info == nil || info.GetOperationId() != execution.operation || info.GetRunId() != execution.runID {
		return errors.New("generated Nexus observation lacks exact operation identity")
	}
	var observed bool
	switch want {
	case "scheduled":
		observed = info.GetStatus() == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_RUNNING &&
			info.GetState() == enumspb.PENDING_NEXUS_OPERATION_STATE_SCHEDULED
	case "backing-off":
		observed = info.GetStatus() == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_RUNNING &&
			info.GetState() == enumspb.PENDING_NEXUS_OPERATION_STATE_BACKING_OFF
	case "started":
		observed = info.GetStatus() == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_RUNNING &&
			info.GetState() == enumspb.PENDING_NEXUS_OPERATION_STATE_STARTED && info.GetOperationToken() != ""
	case "succeeded":
		observed = info.GetStatus() == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_COMPLETED && describe.GetResult() != nil
	case "failed":
		observed = info.GetStatus() == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_FAILED && describe.GetFailure() != nil
	case "canceled":
		observed = info.GetStatus() == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_CANCELED && describe.GetFailure() != nil
	case "timed-out":
		observed = info.GetStatus() == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_TIMED_OUT && describe.GetFailure() != nil
	case "terminated":
		observed = info.GetStatus() == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_TERMINATED && describe.GetFailure() != nil
	}
	if !observed {
		return fmt.Errorf("generated Nexus edge expected %q, observed status %q state %q",
			want, info.GetStatus(), info.GetState())
	}
	return nil
}

type umpire3NexusBehaviorDriver struct {
	mu sync.Mutex

	env       *testcore.TestEnv
	t         *testing.T
	execution *umpire3NexusBehaviorExecution
}

type umpire3NexusBehaviorExecution struct {
	mode              string
	workflowID        string
	runID             string
	taskQueue         string
	handlerWorkflowID string
	handlerRunID      string
	handlerTaskToken  []byte
	handlerClosed     bool
	endpointID        string
	endpointVersion   int64
	nexusTask         *workflowservice.PollNexusTaskQueueResponse
	dispatched        bool
	workerCompleted   bool
	closed            bool
}

func (d *umpire3NexusBehaviorDriver) ExecuteNexusAction(
	ctx context.Context,
	programID string,
	operation participant.Operation,
) (participant.MechanismReceipt, bool, error) {
	mode := ""
	switch {
	case strings.Contains(programID, "SparseRegressionOrdinaryNexusCompletion"):
		mode = "ordinary"
	case strings.Contains(programID, "SparseRegressionCompletionBeforeStartResponse"):
		mode = "completion-before-start"
	case strings.Contains(programID, "ProbeNexusDegraded"):
		mode = "degraded"
	default:
		return participant.MechanismReceipt{}, false, nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.execution != nil && d.execution.mode != mode {
		return participant.MechanismReceipt{}, true, errors.New("dedicated Nexus behavior mode changed during execution")
	}
	var err error
	if mode == "degraded" {
		if operation.SemanticAction != string(protocol.ActionKindCloseNexusOperation) {
			return participant.MechanismReceipt{}, true,
				fmt.Errorf("degraded Nexus behavior does not support action %q", operation.SemanticAction)
		}
		if err = d.scheduleNexusBehavior(ctx, programID, mode); err == nil {
			err = d.dispatchNexusBehavior(ctx)
		}
		if err == nil {
			err = d.completeNexusBehaviorWorker(ctx)
		}
		if err == nil {
			err = d.persistNexusBehavior(ctx)
		}
	} else {
		switch operation.SemanticAction {
		case string(protocol.ActionKindScheduleOperation):
			err = d.scheduleNexusBehavior(ctx, programID, mode)
		case string(protocol.ActionKindDispatchTask):
			err = d.dispatchNexusBehavior(ctx)
		case string(protocol.ActionKindWorkerReturnsSuccess):
			err = d.completeNexusBehaviorWorker(ctx)
		case string(protocol.ActionKindPersistSuccess):
			err = d.persistNexusBehavior(ctx)
		default:
			return participant.MechanismReceipt{}, true,
				fmt.Errorf("dedicated Nexus behavior does not support action %q", operation.SemanticAction)
		}
	}
	if err != nil {
		return participant.MechanismReceipt{}, true, err
	}
	execution := d.execution
	receipt := participant.MechanismReceipt{
		WorkflowID: execution.workflowID, RunID: execution.runID, Lineage: []string{execution.runID},
		Source: "temporal-nexus-phased-driver",
		Reference: fmt.Sprintf("%s/%s/nexus/%s/%s", execution.workflowID, execution.runID,
			execution.mode, operation.SemanticAction),
	}
	if execution.mode == "degraded" {
		receipt.TerminalState = "failed"
		receipt.TerminalDisposition = participant.TerminalDispositionFailure
	} else if execution.closed {
		receipt.TerminalState = "succeeded"
		receipt.TerminalDisposition = participant.TerminalDispositionSuccess
	}
	return receipt, true, nil
}

func (d *umpire3NexusBehaviorDriver) scheduleNexusBehavior(
	ctx context.Context,
	programID string,
	mode string,
) error {
	if d.execution != nil {
		return errors.New("dedicated Nexus behavior is already scheduled")
	}
	execution := &umpire3NexusBehaviorExecution{mode: mode}
	workflowID := umpire3SDKWorkflowID(programID+"-phased-nexus", mode)
	execution.workflowID = workflowID
	execution.taskQueue = workflowID + "-task-queue"
	execution.handlerWorkflowID = workflowID + "-handler"
	handlerTaskQueue := execution.handlerWorkflowID + "-task-queue"
	handler, err := d.env.FrontendClient().StartWorkflowExecution(ctx,
		&workflowservice.StartWorkflowExecutionRequest{
			Namespace: d.env.Namespace().String(), WorkflowId: execution.handlerWorkflowID,
			RequestId: uuid.NewString(), WorkflowType: &commonpb.WorkflowType{Name: "umpire3-nexus-handler"},
			TaskQueue: &taskqueuepb.TaskQueue{Name: handlerTaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:  "umpire3-nexus-phased-driver", WorkflowRunTimeout: durationpb.New(30 * time.Second),
			WorkflowTaskTimeout: durationpb.New(10 * time.Second),
		})
	if err != nil {
		return fmt.Errorf("start Nexus evidence handler workflow: %w", err)
	}
	execution.handlerRunID = handler.GetRunId()
	if mode == "ordinary" {
		handlerTask, err := d.env.FrontendClient().PollWorkflowTaskQueue(ctx,
			&workflowservice.PollWorkflowTaskQueueRequest{
				Namespace: d.env.Namespace().String(),
				TaskQueue: &taskqueuepb.TaskQueue{Name: handlerTaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
				Identity:  "umpire3-nexus-phased-driver",
			})
		if err != nil {
			return fmt.Errorf("poll Nexus evidence handler workflow: %w", err)
		}
		execution.handlerTaskToken = append([]byte(nil), handlerTask.GetTaskToken()...)
		_, err = d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
			&workflowservice.RespondWorkflowTaskCompletedRequest{
				Namespace: d.env.Namespace().String(), TaskToken: execution.handlerTaskToken,
				Identity: "umpire3-nexus-phased-driver",
				Commands: []*commandpb.Command{{
					CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
					Attributes: &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{
						CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{},
					},
				}},
			})
		if err != nil {
			return fmt.Errorf("complete Nexus evidence handler workflow: %w", err)
		}
		execution.handlerClosed = true
	}
	endpointName := testcore.RandomizedNexusEndpoint(d.t.Name() + "-phased")
	endpoint, err := d.env.OperatorClient().CreateNexusEndpoint(ctx,
		&operatorservice.CreateNexusEndpointRequest{Spec: &nexuspb.EndpointSpec{
			Name: endpointName,
			Target: &nexuspb.EndpointTarget{Variant: &nexuspb.EndpointTarget_Worker_{
				Worker: &nexuspb.EndpointTarget_Worker{
					Namespace: d.env.Namespace().String(), TaskQueue: execution.taskQueue,
				},
			}},
		}})
	if err != nil {
		return fmt.Errorf("create phased Nexus endpoint: %w", err)
	}
	execution.endpointID = endpoint.GetEndpoint().GetId()
	execution.endpointVersion = endpoint.GetEndpoint().GetVersion()
	caller, err := d.env.FrontendClient().StartWorkflowExecution(ctx,
		&workflowservice.StartWorkflowExecutionRequest{
			Namespace: d.env.Namespace().String(), WorkflowId: execution.workflowID, RequestId: uuid.NewString(),
			WorkflowType: &commonpb.WorkflowType{Name: "umpire3-phased-nexus-caller"},
			TaskQueue:    &taskqueuepb.TaskQueue{Name: execution.taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:     "umpire3-nexus-phased-driver", WorkflowRunTimeout: durationpb.New(30 * time.Second),
			WorkflowTaskTimeout: durationpb.New(10 * time.Second),
		})
	if err != nil {
		return fmt.Errorf("start phased Nexus caller: %w", err)
	}
	execution.runID = caller.GetRunId()
	callerTask, err := d.env.FrontendClient().PollWorkflowTaskQueue(ctx,
		&workflowservice.PollWorkflowTaskQueueRequest{
			Namespace: d.env.Namespace().String(),
			TaskQueue: &taskqueuepb.TaskQueue{Name: execution.taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:  "umpire3-nexus-phased-driver",
		})
	if err != nil {
		return fmt.Errorf("poll phased Nexus caller: %w", err)
	}
	_, err = d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			Namespace: d.env.Namespace().String(), TaskToken: callerTask.GetTaskToken(),
			Identity: "umpire3-nexus-phased-driver",
			Commands: []*commandpb.Command{{
				CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
				Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{
					ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
						Endpoint: endpointName, Service: "service", Operation: "operation",
						Input:               testcore.MustToPayload(d.t, mode),
						StartToCloseTimeout: durationpb.New(30 * time.Second),
					},
				},
			}},
		})
	if err != nil {
		return fmt.Errorf("schedule phased Nexus operation: %w", err)
	}
	d.execution = execution
	return nil
}

func (d *umpire3NexusBehaviorDriver) dispatchNexusBehavior(ctx context.Context) error {
	if d.execution == nil || d.execution.dispatched {
		return errors.New("dedicated Nexus behavior is not dispatchable")
	}
	task, err := d.env.FrontendClient().PollNexusTaskQueue(ctx,
		&workflowservice.PollNexusTaskQueueRequest{
			Namespace: d.env.Namespace().String(),
			TaskQueue: &taskqueuepb.TaskQueue{
				Name: d.execution.taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
			},
			Identity: "umpire3-nexus-phased-driver",
		})
	if err != nil {
		return fmt.Errorf("poll phased Nexus task: %w", err)
	}
	if task.GetRequest().GetStartOperation() == nil {
		return errors.New("phased Nexus task is not a start request")
	}
	d.execution.nexusTask = task
	d.execution.dispatched = true
	return nil
}

func (d *umpire3NexusBehaviorDriver) completeNexusBehaviorWorker(ctx context.Context) error {
	execution := d.execution
	if execution == nil || !execution.dispatched || execution.workerCompleted {
		return errors.New("dedicated Nexus worker completion is not available")
	}
	handlerLink := &commonpb.Link_WorkflowEvent{
		Namespace: d.env.Namespace().String(), WorkflowId: execution.handlerWorkflowID, RunId: execution.handlerRunID,
		Reference: &commonpb.Link_WorkflowEvent_EventRef{EventRef: &commonpb.Link_WorkflowEvent_EventReference{
			EventId: 1, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		}},
	}
	nexusHandlerLink := commonnexus.ConvertLinkWorkflowEventToNexusLink(handlerLink)
	switch execution.mode {
	case "ordinary":
		_, err := d.env.FrontendClient().RespondNexusTaskCompleted(ctx,
			&workflowservice.RespondNexusTaskCompletedRequest{
				Namespace: d.env.Namespace().String(), Identity: "umpire3-nexus-phased-driver",
				TaskToken: execution.nexusTask.GetTaskToken(),
				Response: &nexuspb.Response{Variant: &nexuspb.Response_StartOperation{
					StartOperation: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_SyncSuccess{
						SyncSuccess: &nexuspb.StartOperationResponse_Sync{
							Payload: payload.EncodeString("ok"), Links: []*nexuspb.Link{{
								Url: nexusHandlerLink.URL.String(), Type: nexusHandlerLink.Type,
							}},
						},
					}},
				}},
			})
		if err != nil {
			return fmt.Errorf("return ordinary Nexus completion: %w", err)
		}
	case "completion-before-start":
		start := execution.nexusTask.GetRequest().GetStartOperation()
		if start.GetCallback() == "" || start.GetCallbackHeader() == nil {
			return errors.New("completion-before-start task has no callback routing")
		}
		start.CallbackHeader[nexus.HeaderOperationToken] = execution.handlerWorkflowID
		attached, err := d.env.FrontendClient().StartWorkflowExecution(ctx,
			&workflowservice.StartWorkflowExecutionRequest{
				Namespace: d.env.Namespace().String(), WorkflowId: execution.handlerWorkflowID,
				RequestId: uuid.NewString(), WorkflowType: &commonpb.WorkflowType{Name: "umpire3-nexus-handler"},
				TaskQueue: &taskqueuepb.TaskQueue{
					Name: execution.handlerWorkflowID + "-task-queue", Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
				},
				Identity: "umpire3-nexus-phased-driver", WorkflowRunTimeout: durationpb.New(30 * time.Second),
				WorkflowTaskTimeout:      durationpb.New(10 * time.Second),
				WorkflowIdConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
				OnConflictOptions: &workflowpb.OnConflictOptions{
					AttachRequestId: true, AttachCompletionCallbacks: true,
				},
				CompletionCallbacks: []*commonpb.Callback{{
					Variant: &commonpb.Callback_Nexus_{Nexus: &commonpb.Callback_Nexus{
						Url: start.GetCallback(), Header: start.GetCallbackHeader(),
					}},
				}},
			})
		if err != nil {
			return fmt.Errorf("attach completion-before-start callback to handler: %w", err)
		}
		if attached.GetStarted() || attached.GetRunId() != execution.handlerRunID {
			return errors.New("completion-before-start callback did not attach to the handler run")
		}
		handlerTask, err := d.env.FrontendClient().PollWorkflowTaskQueue(ctx,
			&workflowservice.PollWorkflowTaskQueueRequest{
				Namespace: d.env.Namespace().String(),
				TaskQueue: &taskqueuepb.TaskQueue{
					Name: execution.handlerWorkflowID + "-task-queue", Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
				},
				Identity: "umpire3-nexus-phased-driver",
			})
		if err != nil {
			return fmt.Errorf("poll completion-before-start handler workflow: %w", err)
		}
		execution.handlerTaskToken = append([]byte(nil), handlerTask.GetTaskToken()...)
		_, err = d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
			&workflowservice.RespondWorkflowTaskCompletedRequest{
				Namespace: d.env.Namespace().String(), TaskToken: execution.handlerTaskToken,
				Identity: "umpire3-nexus-phased-driver",
				Commands: []*commandpb.Command{{
					CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
					Attributes: &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{
						CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{
							Result: payloads.EncodeString("ok"),
						},
					},
				}},
			})
		if err != nil {
			return fmt.Errorf("complete completion-before-start handler: %w", err)
		}
		execution.handlerClosed = true
	default:
		_, err := d.env.FrontendClient().RespondNexusTaskCompleted(ctx,
			&workflowservice.RespondNexusTaskCompletedRequest{
				Namespace: d.env.Namespace().String(), Identity: "umpire3-nexus-phased-driver",
				TaskToken: execution.nexusTask.GetTaskToken(),
				Response: &nexuspb.Response{Variant: &nexuspb.Response_StartOperation{
					StartOperation: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_OperationError{
						OperationError: &nexuspb.UnsuccessfulOperationError{
							OperationState: "failed",
							Failure:        &nexuspb.Failure{Message: "Umpire3 injected modeled operation failure"},
						},
					}},
				}},
			})
		if err != nil {
			return fmt.Errorf("return degraded Nexus operation failure: %w", err)
		}
	}
	execution.workerCompleted = true
	return nil
}

func (d *umpire3NexusBehaviorDriver) persistNexusBehavior(ctx context.Context) error {
	execution := d.execution
	if execution == nil || !execution.workerCompleted || execution.closed {
		return errors.New("dedicated Nexus completion is not persistable")
	}
	if execution.mode == "completion-before-start" {
		handlerLink := commonnexus.ConvertLinkWorkflowEventToNexusLink(&commonpb.Link_WorkflowEvent{
			Namespace: d.env.Namespace().String(), WorkflowId: execution.handlerWorkflowID, RunId: execution.handlerRunID,
			Reference: &commonpb.Link_WorkflowEvent_EventRef{EventRef: &commonpb.Link_WorkflowEvent_EventReference{
				EventId: 1, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			}},
		})
		_, err := d.env.FrontendClient().RespondNexusTaskCompleted(ctx,
			&workflowservice.RespondNexusTaskCompletedRequest{
				Namespace: d.env.Namespace().String(), Identity: "umpire3-nexus-phased-driver",
				TaskToken: execution.nexusTask.GetTaskToken(),
				Response: &nexuspb.Response{Variant: &nexuspb.Response_StartOperation{
					StartOperation: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_AsyncSuccess{
						AsyncSuccess: &nexuspb.StartOperationResponse_Async{
							OperationToken: execution.handlerWorkflowID, Links: []*nexuspb.Link{{
								Url: handlerLink.URL.String(), Type: handlerLink.Type,
							}},
						},
					}},
				}},
			})
		if err != nil {
			return fmt.Errorf("return late Nexus start response: %w", err)
		}
	}
	callerTask, err := d.env.FrontendClient().PollWorkflowTaskQueue(ctx,
		&workflowservice.PollWorkflowTaskQueueRequest{
			Namespace: d.env.Namespace().String(),
			TaskQueue: &taskqueuepb.TaskQueue{Name: execution.taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:  "umpire3-nexus-phased-driver",
		})
	if err != nil {
		return fmt.Errorf("poll phased Nexus completion task: %w", err)
	}
	var scheduledEventID int64
	var startedEventID int64
	var terminalEventID int64
	for _, event := range callerTask.GetHistory().GetEvents() {
		switch event.GetEventType() {
		case enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED:
			scheduledEventID = event.GetEventId()
		case enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED:
			startedEventID = event.GetEventId()
			if event.GetNexusOperationStartedEventAttributes().GetOperationToken() != execution.handlerWorkflowID {
				return errors.New("completion-before-start operation token does not name the handler")
			}
			if !umpire3HistoryEventLinksWorkflow(event.GetLinks(), d.env.Namespace().String(),
				execution.handlerWorkflowID, execution.handlerRunID) {
				return errors.New("completion-before-start event lacks the handler link")
			}
		case enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED:
			terminalEventID = event.GetEventId()
			attributes := event.GetNexusOperationCompletedEventAttributes()
			if !proto.Equal(attributes.GetResult(), payload.EncodeString("ok")) {
				return errors.New("Nexus completion result differs from the handler result")
			}
			if execution.mode == "ordinary" && !umpire3HistoryEventLinksWorkflow(event.GetLinks(),
				d.env.Namespace().String(), execution.handlerWorkflowID, execution.handlerRunID) {
				return errors.New("ordinary Nexus completion lacks the handler link")
			}
		case enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED:
			if execution.mode != "degraded" {
				return errors.New("phased Nexus operation failed unexpectedly")
			}
			terminalEventID = event.GetEventId()
		default:
			continue
		}
	}
	if scheduledEventID == 0 || terminalEventID == 0 {
		return errors.New("phased Nexus history lacks schedule or completion evidence")
	}
	if execution.mode == "completion-before-start" &&
		(startedEventID == 0 || (scheduledEventID >= startedEventID || startedEventID >= terminalEventID)) {
		return errors.New("completion-before-start history ordering is invalid")
	}
	_, err = d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			Namespace: d.env.Namespace().String(), TaskToken: callerTask.GetTaskToken(),
			Identity: "umpire3-nexus-phased-driver",
			Commands: []*commandpb.Command{{
				CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
				Attributes: &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{
					CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{},
				},
			}},
		})
	if err != nil {
		return fmt.Errorf("complete phased Nexus caller: %w", err)
	}
	if execution.mode == "ordinary" {
		description, describeErr := d.env.AdminClient().DescribeMutableState(ctx,
			&adminservice.DescribeMutableStateRequest{
				Namespace: d.env.Namespace().String(),
				Execution: &commonpb.WorkflowExecution{WorkflowId: execution.workflowID, RunId: execution.runID},
				Archetype: chasm.WorkflowArchetype,
			})
		if describeErr != nil {
			return fmt.Errorf("describe ordinary Nexus caller storage: %w", describeErr)
		}
		machines := description.GetDatabaseMutableState().GetExecutionInfo().GetSubStateMachinesByType()[nexusoperations.OperationMachineType].GetMachinesById()
		if _, retained := machines[fmt.Sprint(scheduledEventID)]; retained {
			return errors.New("ordinary completed Nexus operation remains in embedded mutable-state storage")
		}
	}
	execution.closed = true
	return nil
}

func umpire3HistoryEventLinksWorkflow(
	links []*commonpb.Link,
	namespace string,
	workflowID string,
	runID string,
) bool {
	for _, link := range links {
		workflowLink := link.GetWorkflowEvent()
		if workflowLink != nil && workflowLink.GetNamespace() == namespace &&
			workflowLink.GetWorkflowId() == workflowID && workflowLink.GetRunId() == runID {
			return true
		}
	}
	return false
}

func (d *umpire3NexusBehaviorDriver) CleanupNexus(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.execution == nil {
		return nil
	}
	var cleanupErr error
	if !d.execution.closed && d.execution.workflowID != "" && d.execution.runID != "" {
		_, err := d.env.FrontendClient().TerminateWorkflowExecution(ctx,
			&workflowservice.TerminateWorkflowExecutionRequest{
				Namespace: d.env.Namespace().String(),
				WorkflowExecution: &commonpb.WorkflowExecution{
					WorkflowId: d.execution.workflowID, RunId: d.execution.runID,
				},
				Reason: "Umpire3 phased Nexus cleanup",
			})
		if err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("terminate phased Nexus caller: %w", err))
		}
	}
	if !d.execution.handlerClosed && d.execution.handlerWorkflowID != "" && d.execution.handlerRunID != "" {
		_, err := d.env.FrontendClient().TerminateWorkflowExecution(ctx,
			&workflowservice.TerminateWorkflowExecutionRequest{
				Namespace: d.env.Namespace().String(),
				WorkflowExecution: &commonpb.WorkflowExecution{
					WorkflowId: d.execution.handlerWorkflowID, RunId: d.execution.handlerRunID,
				},
				Reason: "Umpire3 phased Nexus handler cleanup",
			})
		if err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("terminate phased Nexus handler: %w", err))
		}
	}
	if d.execution.endpointID != "" {
		_, err := d.env.OperatorClient().DeleteNexusEndpoint(ctx,
			&operatorservice.DeleteNexusEndpointRequest{
				Id: d.execution.endpointID, Version: d.execution.endpointVersion,
			})
		if err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("delete phased Nexus endpoint: %w", err))
		}
	}
	d.execution = nil
	return cleanupErr
}

type umpire3NexusActivityLinkDriver struct {
	mu sync.Mutex

	env                    *testcore.TestEnv
	activity               *commonpb.Link_Activity
	expectedNexusOperation *commonpb.Link_NexusOperation
	validated              bool
}

func (d *umpire3NexusActivityLinkDriver) Start(
	ctx context.Context,
	operation participant.Operation,
	options nexus.StartOperationOptions,
) (nexus.HandlerStartOperationResult[any], error) {
	d.mu.Lock()
	if d.activity != nil {
		d.mu.Unlock()
		return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeInternal,
			"Umpire3 Nexus Activity link operation already exists")
	}
	d.mu.Unlock()

	var caller *commonpb.Link_WorkflowEvent
	for _, link := range options.Links {
		workflowLink, err := commonnexus.ConvertNexusLinkToLinkWorkflowEvent(link)
		if err == nil {
			caller = workflowLink
			break
		}
	}
	if caller == nil || caller.GetNamespace() != d.env.Namespace().String() ||
		caller.GetWorkflowId() == "" || caller.GetRunId() == "" {
		return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeInternal,
			"Temporal did not provide a complete caller Workflow link")
	}
	nexusOperation := &commonpb.Link_NexusOperation{
		Namespace: caller.GetNamespace(), OperationId: caller.GetWorkflowId(), RunId: caller.GetRunId(),
	}
	activityID := "umpire3-link-" + uuid.NewString()
	taskQueue := activityID + "-task-queue"
	started, err := d.env.FrontendClient().StartActivityExecution(ctx,
		&workflowservice.StartActivityExecutionRequest{
			Namespace: d.env.Namespace().String(), ActivityId: activityID, RequestId: uuid.NewString(),
			ActivityType:        &commonpb.ActivityType{Name: "umpire3-linked-activity"},
			TaskQueue:           &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:            "umpire3-nexus-activity-link-driver",
			Input:               payloads.EncodeString(operation.CommandID),
			StartToCloseTimeout: durationpb.New(30 * time.Second),
			Links: []*commonpb.Link{{
				Variant: &commonpb.Link_NexusOperation_{NexusOperation: nexusOperation},
			}},
		})
	if err != nil {
		return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeInternal,
			"start linked standalone Activity: %v", err)
	}
	task, err := d.env.FrontendClient().PollActivityTaskQueue(ctx,
		&workflowservice.PollActivityTaskQueueRequest{
			Namespace: d.env.Namespace().String(),
			TaskQueue: &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:  "umpire3-nexus-activity-link-driver",
		})
	if err != nil {
		return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeInternal,
			"poll linked standalone Activity: %v", err)
	}
	if len(task.GetTaskToken()) == 0 {
		return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeInternal,
			"linked standalone Activity returned an empty task token")
	}
	_, err = d.env.FrontendClient().RespondActivityTaskCompleted(ctx,
		&workflowservice.RespondActivityTaskCompletedRequest{
			Namespace: d.env.Namespace().String(), TaskToken: task.GetTaskToken(),
			Identity: "umpire3-nexus-activity-link-driver", Result: payloads.EncodeString("umpire3-ok"),
		})
	if err != nil {
		return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeInternal,
			"complete linked standalone Activity: %v", err)
	}
	activity := started.GetLink().GetActivity()
	if activity == nil || activity.GetNamespace() != d.env.Namespace().String() ||
		activity.GetActivityId() != activityID || activity.GetRunId() == "" {
		return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeInternal,
			"Temporal returned an incomplete standalone Activity link")
	}
	nexus.AddHandlerLinks(ctx, commonnexus.ConvertLinkActivityToNexusLink(activity))
	d.mu.Lock()
	d.activity = activity
	d.expectedNexusOperation = nexusOperation
	d.mu.Unlock()
	return &nexus.HandlerStartOperationResultSync[any]{Value: "umpire3-ok"}, nil
}

func (d *umpire3NexusActivityLinkDriver) Validate(ctx context.Context, parentIdentity string) (string, error) {
	d.mu.Lock()
	activity := d.activity
	expectedNexusOperation := d.expectedNexusOperation
	d.mu.Unlock()
	if activity == nil || expectedNexusOperation == nil {
		return "", errors.New("Nexus Activity link evidence is unavailable")
	}
	parent := strings.Split(parentIdentity, "/")
	if len(parent) != 2 || parent[0] != expectedNexusOperation.GetOperationId() ||
		parent[1] != expectedNexusOperation.GetRunId() {
		return "", errors.New("Nexus Activity link parent identity is inconsistent")
	}
	history := d.env.GetHistory(d.env.Namespace().String(), &commonpb.WorkflowExecution{
		WorkflowId: parent[0], RunId: parent[1],
	})
	forwardLinkFound := false
	for _, event := range history {
		if event.GetEventType() != enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED {
			continue
		}
		for _, link := range event.GetLinks() {
			candidate := link.GetActivity()
			if candidate != nil && candidate.GetNamespace() == activity.GetNamespace() &&
				candidate.GetActivityId() == activity.GetActivityId() && candidate.GetRunId() == activity.GetRunId() {
				forwardLinkFound = true
				break
			}
		}
	}
	if !forwardLinkFound {
		return "", errors.New("caller Nexus completion does not contain the standalone Activity link")
	}
	description, err := d.env.FrontendClient().DescribeActivityExecution(ctx,
		&workflowservice.DescribeActivityExecutionRequest{
			Namespace: activity.GetNamespace(), ActivityId: activity.GetActivityId(), RunId: activity.GetRunId(),
		})
	if err != nil {
		return "", fmt.Errorf("describe linked standalone Activity: %w", err)
	}
	backLinkFound := false
	for _, link := range description.GetInfo().GetLinks() {
		candidate := link.GetNexusOperation()
		if candidate != nil && candidate.GetNamespace() == expectedNexusOperation.GetNamespace() &&
			candidate.GetOperationId() == expectedNexusOperation.GetOperationId() &&
			candidate.GetRunId() == expectedNexusOperation.GetRunId() {
			backLinkFound = true
			break
		}
	}
	if !backLinkFound {
		return "", errors.New("standalone Activity does not contain the caller Nexus-operation link")
	}
	reference := fmt.Sprintf("%s/%s/nexus-activity/%s/%s", parent[0], parent[1],
		activity.GetActivityId(), activity.GetRunId())
	d.mu.Lock()
	d.validated = true
	d.mu.Unlock()
	return reference, nil
}

func (d *umpire3NexusActivityLinkDriver) Validated() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.validated
}

type umpire3CallbackExecution struct {
	programID          string
	workflowID         string
	runID              string
	taskToken          []byte
	callbackURL        string
	callbackToken      string
	kind               string
	referenceValidated bool
	responseValidated  bool
	workflowClosed     bool
	completed          bool
}

type umpire3CallbackDriver struct {
	mu sync.Mutex

	env          *testcore.TestEnv
	nexusEnv     *NexusTestEnv
	t            *testing.T
	variant      string
	server       *httptest.Server
	completionCh chan *nexusrpc.CompletionRequest
	execution    *umpire3CallbackExecution
	shared       *umpire3SharedHandlerExecution
}

type umpire3SharedHandlerCaller struct {
	workflowID string
	runID      string
	taskQueue  string
	nexusTask  *workflowservice.PollNexusTaskQueueResponse
}

type umpire3SharedHandlerExecution struct {
	handlerWorkflowID string
	handlerRunID      string
	handlerTaskQueue  string
	callers           []umpire3SharedHandlerCaller
}

func newUmpire3CallbackDriver(
	t *testing.T,
	env *testcore.TestEnv,
	nexusEnv *NexusTestEnv,
	variant string,
) *umpire3CallbackDriver {
	t.Helper()
	driver := &umpire3CallbackDriver{
		env: env, nexusEnv: nexusEnv, t: t, variant: variant,
		completionCh: make(chan *nexusrpc.CompletionRequest, 1),
	}
	driver.server = httptest.NewServer(nexusrpc.NewCompletionHTTPHandler(
		nexusrpc.CompletionHandlerOptions{Handler: driver}))
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = driver.CleanupCompletionCallbacks(cleanupCtx)
	})
	return driver
}

func (d *umpire3CallbackDriver) CompleteOperation(
	ctx context.Context,
	request *nexusrpc.CompletionRequest,
) error {
	select {
	case d.completionCh <- request:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *umpire3CallbackDriver) RegisterCompletionCallback(
	ctx context.Context,
	programID string,
	commandID string,
) (participant.MechanismReceipt, error) {
	d.mu.Lock()
	if d.execution != nil {
		d.mu.Unlock()
		return participant.MechanismReceipt{}, errors.New("completion callback is already registered")
	}
	d.mu.Unlock()
	if strings.Contains(programID, "SparseRegressionSharedHandlerWorkflow") {
		return d.registerSharedHandlerCallbacks(ctx, programID, commandID)
	}
	if strings.Contains(programID, "SparseRegressionCallbackAfterCallerCompletion") {
		return d.registerNexusCallbackAfterCaller(ctx, programID, commandID)
	}
	workflowID := umpire3SDKWorkflowID(programID+"-callback", commandID)
	taskQueue := workflowID + "-task-queue"
	callbackURL := d.server.URL + "/completion"
	started, err := d.env.FrontendClient().StartWorkflowExecution(ctx,
		&workflowservice.StartWorkflowExecutionRequest{
			Namespace: d.env.Namespace().String(), WorkflowId: workflowID, RequestId: uuid.NewString(),
			WorkflowType:       &commonpb.WorkflowType{Name: "umpire3-callback-workflow"},
			TaskQueue:          &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			WorkflowRunTimeout: durationpb.New(30 * time.Second), WorkflowTaskTimeout: durationpb.New(10 * time.Second),
			Identity: "umpire3-completion-callback-driver",
			CompletionCallbacks: []*commonpb.Callback{{
				Variant: &commonpb.Callback_Nexus_{Nexus: &commonpb.Callback_Nexus{Url: callbackURL}},
			}},
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("start completion callback workflow: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_, _ = d.env.FrontendClient().TerminateWorkflowExecution(context.Background(),
				&workflowservice.TerminateWorkflowExecutionRequest{
					Namespace:         d.env.Namespace().String(),
					WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: started.GetRunId()},
					Reason:            "Umpire3 completion callback registration cleanup",
				})
		}
	}()
	task, err := d.env.FrontendClient().PollWorkflowTaskQueue(ctx, &workflowservice.PollWorkflowTaskQueueRequest{
		Namespace: d.env.Namespace().String(),
		TaskQueue: &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		Identity:  "umpire3-completion-callback-driver",
	})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("poll completion callback workflow task: %w", err)
	}
	if len(task.GetTaskToken()) == 0 {
		return participant.MechanismReceipt{}, errors.New("completion callback workflow returned an empty task token")
	}
	description, err := d.env.SdkClient().DescribeWorkflowExecution(ctx, workflowID, started.GetRunId())
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("describe registered completion callback: %w", err)
	}
	if len(description.GetCallbacks()) != 1 ||
		description.GetCallbacks()[0].GetState() != enumspb.CALLBACK_STATE_STANDBY ||
		description.GetCallbacks()[0].GetCallback().GetNexus().GetUrl() != callbackURL {
		return participant.MechanismReceipt{}, errors.New("Temporal did not expose the registered completion callback")
	}
	if err := d.validateStorageVariant(ctx, workflowID, started.GetRunId()); err != nil {
		return participant.MechanismReceipt{}, err
	}
	d.mu.Lock()
	d.execution = &umpire3CallbackExecution{
		programID: programID, workflowID: workflowID, runID: started.GetRunId(),
		taskToken: append([]byte(nil), task.GetTaskToken()...), referenceValidated: true,
	}
	d.mu.Unlock()
	cleanup = false
	return participant.MechanismReceipt{
		WorkflowID: workflowID, RunID: started.GetRunId(), Lineage: []string{started.GetRunId()},
		Source:    "temporal-completion-callback-registration",
		Reference: fmt.Sprintf("%s/%s/callback/%s/registered", workflowID, started.GetRunId(), callbackURL),
	}, nil
}

func (d *umpire3CallbackDriver) registerSharedHandlerCallbacks(
	ctx context.Context,
	programID string,
	commandID string,
) (participant.MechanismReceipt, error) {
	if d.nexusEnv == nil {
		return participant.MechanismReceipt{}, errors.New("shared handler callback driver requires a Nexus environment")
	}
	handlerWorkflowID := umpire3SDKWorkflowID(programID+"-shared-handler", commandID)
	handlerTaskQueue := handlerWorkflowID + "-task-queue"
	shared := &umpire3SharedHandlerExecution{
		handlerWorkflowID: handlerWorkflowID, handlerTaskQueue: handlerTaskQueue,
	}
	cleanup := true
	defer func() {
		if cleanup {
			d.cleanupSharedHandler(context.Background(), shared)
		}
	}()
	for index := 0; index < 2; index++ {
		callerID := umpire3SDKWorkflowID(programID+fmt.Sprintf("-shared-caller-%d", index), commandID)
		callerTaskQueue := callerID + "-task-queue"
		endpointName := testcore.RandomizedNexusEndpoint(fmt.Sprintf("%s-%d", d.t.Name(), index))
		endpoint, err := d.env.OperatorClient().CreateNexusEndpoint(ctx,
			&operatorservice.CreateNexusEndpointRequest{Spec: &nexuspb.EndpointSpec{
				Name: endpointName,
				Target: &nexuspb.EndpointTarget{Variant: &nexuspb.EndpointTarget_Worker_{
					Worker: &nexuspb.EndpointTarget_Worker{
						Namespace: d.env.Namespace().String(), TaskQueue: callerTaskQueue,
					},
				}},
			}})
		if err != nil {
			return participant.MechanismReceipt{}, fmt.Errorf("create shared handler Nexus endpoint: %w", err)
		}
		d.t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = d.env.OperatorClient().DeleteNexusEndpoint(cleanupCtx,
				&operatorservice.DeleteNexusEndpointRequest{
					Id: endpoint.GetEndpoint().GetId(), Version: endpoint.GetEndpoint().GetVersion(),
				})
		})
		started, err := d.env.FrontendClient().StartWorkflowExecution(ctx,
			&workflowservice.StartWorkflowExecutionRequest{
				Namespace: d.env.Namespace().String(), WorkflowId: callerID, RequestId: uuid.NewString(),
				WorkflowType: &commonpb.WorkflowType{Name: "umpire3-shared-handler-caller"},
				TaskQueue: &taskqueuepb.TaskQueue{
					Name: callerTaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
				},
				Identity:            "umpire3-shared-handler-driver",
				WorkflowRunTimeout:  durationpb.New(30 * time.Second),
				WorkflowTaskTimeout: durationpb.New(10 * time.Second),
			})
		if err != nil {
			return participant.MechanismReceipt{}, fmt.Errorf("start shared handler caller: %w", err)
		}
		caller := umpire3SharedHandlerCaller{
			workflowID: callerID, runID: started.GetRunId(), taskQueue: callerTaskQueue,
		}
		shared.callers = append(shared.callers, caller)
		workflowTask, err := d.env.FrontendClient().PollWorkflowTaskQueue(ctx,
			&workflowservice.PollWorkflowTaskQueueRequest{
				Namespace: d.env.Namespace().String(),
				TaskQueue: &taskqueuepb.TaskQueue{
					Name: callerTaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
				},
				Identity: "umpire3-shared-handler-driver",
			})
		if err != nil {
			return participant.MechanismReceipt{}, fmt.Errorf("poll shared handler caller task: %w", err)
		}
		_, err = d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
			&workflowservice.RespondWorkflowTaskCompletedRequest{
				Namespace: d.env.Namespace().String(), TaskToken: workflowTask.GetTaskToken(),
				Identity: "umpire3-shared-handler-driver",
				Commands: []*commandpb.Command{{
					CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
					Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{
						ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
							Endpoint: endpointName, Service: "service", Operation: "operation",
							Input: testcore.MustToPayload(d.t, commandID),
						},
					},
				}},
			})
		if err != nil {
			return participant.MechanismReceipt{}, fmt.Errorf("schedule shared handler Nexus operation: %w", err)
		}
		nexusTask, err := d.env.FrontendClient().PollNexusTaskQueue(ctx,
			&workflowservice.PollNexusTaskQueueRequest{
				Namespace: d.env.Namespace().String(),
				TaskQueue: &taskqueuepb.TaskQueue{
					Name: callerTaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
				},
				Identity: "umpire3-shared-handler-driver",
			})
		if err != nil {
			return participant.MechanismReceipt{}, fmt.Errorf("poll shared handler Nexus task: %w", err)
		}
		start := nexusTask.GetRequest().GetStartOperation()
		if start == nil || start.GetCallback() == "" || start.GetCallbackHeader() == nil {
			return participant.MechanismReceipt{}, errors.New("shared handler Nexus task has no completion callback")
		}
		start.CallbackHeader[nexus.HeaderOperationToken] = handlerWorkflowID
		handlerRequestID := uuid.NewString()
		handler, err := d.env.FrontendClient().StartWorkflowExecution(ctx,
			&workflowservice.StartWorkflowExecutionRequest{
				Namespace: d.env.Namespace().String(), WorkflowId: handlerWorkflowID, RequestId: handlerRequestID,
				WorkflowType: &commonpb.WorkflowType{Name: "umpire3-shared-handler"},
				TaskQueue: &taskqueuepb.TaskQueue{
					Name: handlerTaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
				},
				Identity:                 "umpire3-shared-handler-driver",
				WorkflowRunTimeout:       durationpb.New(30 * time.Second),
				WorkflowTaskTimeout:      durationpb.New(10 * time.Second),
				WorkflowIdConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
				OnConflictOptions: &workflowpb.OnConflictOptions{
					AttachRequestId: true, AttachCompletionCallbacks: true,
				},
				CompletionCallbacks: []*commonpb.Callback{{
					Variant: &commonpb.Callback_Nexus_{Nexus: &commonpb.Callback_Nexus{
						Url: start.GetCallback(), Header: start.GetCallbackHeader(),
					}},
				}},
			})
		if err != nil {
			return participant.MechanismReceipt{}, fmt.Errorf("start or attach shared handler callback: %w", err)
		}
		if shared.handlerRunID == "" {
			if !handler.GetStarted() {
				return participant.MechanismReceipt{}, errors.New("first shared handler request did not start the handler")
			}
			shared.handlerRunID = handler.GetRunId()
		} else if handler.GetStarted() || handler.GetRunId() != shared.handlerRunID {
			return participant.MechanismReceipt{}, errors.New("second shared handler request did not attach to the existing run")
		}
		shared.callers[index].nexusTask = nexusTask
	}
	description, err := d.env.SdkClient().DescribeWorkflowExecution(ctx, shared.handlerWorkflowID, shared.handlerRunID)
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("describe shared handler callbacks: %w", err)
	}
	if len(description.GetCallbacks()) != 2 {
		return participant.MechanismReceipt{}, fmt.Errorf("shared handler has %d callbacks, want 2", len(description.GetCallbacks()))
	}
	for _, callback := range description.GetCallbacks() {
		if callback.GetState() != enumspb.CALLBACK_STATE_STANDBY || callback.GetCallback().GetNexus().GetUrl() == "" {
			return participant.MechanismReceipt{}, errors.New("shared handler callback is not retained in standby")
		}
	}
	d.mu.Lock()
	d.shared = shared
	d.execution = &umpire3CallbackExecution{
		programID: programID, workflowID: shared.callers[0].workflowID, runID: shared.callers[0].runID,
		kind: "shared-handler", referenceValidated: true,
	}
	d.mu.Unlock()
	cleanup = false
	return participant.MechanismReceipt{
		WorkflowID: shared.callers[0].workflowID, RunID: shared.callers[0].runID,
		Lineage: []string{shared.callers[0].runID}, Source: "temporal-shared-handler-registration",
		Reference: fmt.Sprintf("%s/%s/shared-handler/%s/%s/callbacks/2",
			shared.callers[0].workflowID, shared.callers[0].runID,
			shared.handlerWorkflowID, shared.handlerRunID),
	}, nil
}

func (d *umpire3CallbackDriver) completeSharedHandlerCallbacks(
	ctx context.Context,
	execution *umpire3CallbackExecution,
	commandID string,
) (participant.MechanismReceipt, error) {
	d.mu.Lock()
	shared := d.shared
	d.mu.Unlock()
	if shared == nil || len(shared.callers) != 2 {
		return participant.MechanismReceipt{}, errors.New("shared handler execution is incomplete")
	}
	handlerTask, err := d.env.FrontendClient().PollWorkflowTaskQueue(ctx,
		&workflowservice.PollWorkflowTaskQueueRequest{
			Namespace: d.env.Namespace().String(),
			TaskQueue: &taskqueuepb.TaskQueue{
				Name: shared.handlerTaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
			},
			Identity: "umpire3-shared-handler-driver",
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("poll shared handler completion task: %w", err)
	}
	_, err = d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			Namespace: d.env.Namespace().String(), TaskToken: handlerTask.GetTaskToken(),
			Identity: "umpire3-shared-handler-driver",
			Commands: []*commandpb.Command{{
				CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
				Attributes: &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{
					CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{
						Result: payloads.EncodeString("umpire3-shared-result"),
					},
				},
			}},
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("complete shared handler workflow: %w", err)
	}
	for _, caller := range shared.callers {
		callerTask, pollErr := d.env.FrontendClient().PollWorkflowTaskQueue(ctx,
			&workflowservice.PollWorkflowTaskQueueRequest{
				Namespace: d.env.Namespace().String(),
				TaskQueue: &taskqueuepb.TaskQueue{Name: caller.taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
				Identity:  "umpire3-shared-handler-driver",
			})
		if pollErr != nil {
			return participant.MechanismReceipt{}, fmt.Errorf("poll completed shared caller: %w", pollErr)
		}
		startedFound := false
		completedFound := false
		for _, event := range callerTask.GetHistory().GetEvents() {
			if event.GetEventType() == enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED {
				if event.GetNexusOperationStartedEventAttributes().GetOperationToken() != shared.handlerWorkflowID {
					return participant.MechanismReceipt{}, errors.New("shared caller operation token does not name the handler")
				}
				if len(event.GetLinks()) != 1 {
					return participant.MechanismReceipt{}, errors.New("shared caller start does not contain exactly one handler link")
				}
				link := event.GetLinks()[0].GetWorkflowEvent()
				if link == nil || link.GetNamespace() != d.env.Namespace().String() ||
					link.GetWorkflowId() != shared.handlerWorkflowID || link.GetRunId() != shared.handlerRunID {
					return participant.MechanismReceipt{}, errors.New("shared caller callback link does not resolve to the shared handler run")
				}
				startedFound = true
			}
			completedFound = completedFound || event.GetEventType() == enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED
		}
		if !startedFound || !completedFound {
			return participant.MechanismReceipt{}, errors.New("shared caller did not record started and completed Nexus events")
		}
		_, lateErr := d.env.FrontendClient().RespondNexusTaskCompleted(ctx,
			&workflowservice.RespondNexusTaskCompletedRequest{
				Namespace: d.env.Namespace().String(), Identity: "umpire3-shared-handler-driver",
				TaskToken: caller.nexusTask.GetTaskToken(),
				Response: &nexuspb.Response{Variant: &nexuspb.Response_StartOperation{
					StartOperation: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_AsyncSuccess{
						AsyncSuccess: &nexuspb.StartOperationResponse_Async{OperationToken: shared.handlerWorkflowID},
					}},
				}},
			})
		if lateErr != nil {
			return participant.MechanismReceipt{}, fmt.Errorf("return late shared handler start response: %w", lateErr)
		}
		_, completeErr := d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
			&workflowservice.RespondWorkflowTaskCompletedRequest{
				Namespace: d.env.Namespace().String(), TaskToken: callerTask.GetTaskToken(),
				Identity: "umpire3-shared-handler-driver",
				Commands: []*commandpb.Command{{
					CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
					Attributes: &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{
						CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{},
					},
				}},
			})
		if completeErr != nil {
			return participant.MechanismReceipt{}, fmt.Errorf("complete shared handler caller: %w", completeErr)
		}
	}
	d.mu.Lock()
	execution.responseValidated = true
	execution.workflowClosed = true
	execution.completed = true
	d.mu.Unlock()
	return participant.MechanismReceipt{
		WorkflowID: execution.workflowID, RunID: execution.runID, Lineage: []string{execution.runID},
		Source: "temporal-shared-handler-completion",
		Reference: fmt.Sprintf("%s/%s/shared-handler/%s/%s/%s/completed",
			execution.workflowID, execution.runID, shared.handlerWorkflowID, shared.handlerRunID, commandID),
	}, nil
}

func (d *umpire3CallbackDriver) cleanupSharedHandler(ctx context.Context, shared *umpire3SharedHandlerExecution) {
	if shared == nil {
		return
	}
	for _, caller := range shared.callers {
		_, _ = d.env.FrontendClient().TerminateWorkflowExecution(ctx,
			&workflowservice.TerminateWorkflowExecutionRequest{
				Namespace:         d.env.Namespace().String(),
				WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: caller.workflowID, RunId: caller.runID},
				Reason:            "Umpire3 shared handler cleanup",
			})
	}
	if shared.handlerRunID != "" {
		_, _ = d.env.FrontendClient().TerminateWorkflowExecution(ctx,
			&workflowservice.TerminateWorkflowExecutionRequest{
				Namespace: d.env.Namespace().String(),
				WorkflowExecution: &commonpb.WorkflowExecution{
					WorkflowId: shared.handlerWorkflowID, RunId: shared.handlerRunID,
				},
				Reason: "Umpire3 shared handler cleanup",
			})
	}
}

type umpire3CapturedNexusCallback struct {
	url   string
	token string
}

func (d *umpire3CallbackDriver) registerNexusCallbackAfterCaller(
	ctx context.Context,
	programID string,
	commandID string,
) (participant.MechanismReceipt, error) {
	if d.nexusEnv == nil {
		return participant.MechanismReceipt{}, errors.New("Nexus callback driver requires a Nexus environment")
	}
	captured := make(chan umpire3CapturedNexusCallback, 1)
	endpoint := d.nexusEnv.createRandomExternalNexusServer(ctx, d.t, nexustest.Handler{
		OnStartOperation: func(
			requestCtx context.Context,
			_ string,
			_ string,
			_ *nexus.LazyValue,
			options nexus.StartOperationOptions,
		) (nexus.HandlerStartOperationResult[any], error) {
			callback := umpire3CapturedNexusCallback{
				url: options.CallbackURL, token: options.CallbackHeader.Get(commonnexus.CallbackTokenHeader),
			}
			if callback.url == "" || callback.token == "" {
				return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeInternal,
					"Temporal supplied an incomplete Nexus completion callback")
			}
			select {
			case captured <- callback:
			case <-requestCtx.Done():
				return nil, requestCtx.Err()
			}
			return &nexus.HandlerStartOperationResultAsync{OperationToken: "umpire3-callback-after-caller"}, nil
		},
	})
	workflowID := umpire3SDKWorkflowID(programID+"-nexus-caller", commandID)
	taskQueue := workflowID + "-task-queue"
	started, err := d.env.FrontendClient().StartWorkflowExecution(ctx,
		&workflowservice.StartWorkflowExecutionRequest{
			Namespace: d.env.Namespace().String(), WorkflowId: workflowID, RequestId: uuid.NewString(),
			WorkflowType:        &commonpb.WorkflowType{Name: "umpire3-callback-after-caller"},
			TaskQueue:           &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:            "umpire3-nexus-callback-driver",
			WorkflowRunTimeout:  durationpb.New(30 * time.Second),
			WorkflowTaskTimeout: durationpb.New(10 * time.Second),
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("start Nexus callback caller: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_, _ = d.env.FrontendClient().TerminateWorkflowExecution(context.Background(),
				&workflowservice.TerminateWorkflowExecutionRequest{
					Namespace:         d.env.Namespace().String(),
					WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: started.GetRunId()},
					Reason:            "Umpire3 Nexus callback registration cleanup",
				})
		}
	}()
	firstTask, err := d.env.FrontendClient().PollWorkflowTaskQueue(ctx,
		&workflowservice.PollWorkflowTaskQueueRequest{
			Namespace: d.env.Namespace().String(),
			TaskQueue: &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:  "umpire3-nexus-callback-driver",
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("poll Nexus callback caller start task: %w", err)
	}
	_, err = d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			Namespace: d.env.Namespace().String(), TaskToken: firstTask.GetTaskToken(),
			Identity: "umpire3-nexus-callback-driver",
			Commands: []*commandpb.Command{{
				CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
				Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{
					ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
						Endpoint: endpoint, Service: "service", Operation: "operation",
						Input:               testcore.MustToPayload(d.t, commandID),
						StartToCloseTimeout: durationpb.New(30 * time.Second),
					},
				},
			}},
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("schedule Nexus callback operation: %w", err)
	}
	var callback umpire3CapturedNexusCallback
	select {
	case callback = <-captured:
	case <-ctx.Done():
		return participant.MechanismReceipt{}, fmt.Errorf("capture Nexus completion callback: %w", ctx.Err())
	}
	terminalTask, err := d.env.FrontendClient().PollWorkflowTaskQueue(ctx,
		&workflowservice.PollWorkflowTaskQueueRequest{
			Namespace: d.env.Namespace().String(),
			TaskQueue: &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:  "umpire3-nexus-callback-driver",
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("poll Nexus started caller task: %w", err)
	}
	startedObserved := false
	for _, event := range terminalTask.GetHistory().GetEvents() {
		if event.GetEventType() == enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED {
			startedObserved = true
			break
		}
	}
	if !startedObserved {
		return participant.MechanismReceipt{}, errors.New("caller history did not record the asynchronous Nexus operation")
	}
	_, err = d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			Namespace: d.env.Namespace().String(), TaskToken: terminalTask.GetTaskToken(),
			Identity: "umpire3-nexus-callback-driver",
			Commands: []*commandpb.Command{{
				CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
				Attributes: &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{
					CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{},
				},
			}},
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("complete Nexus callback caller: %w", err)
	}
	d.mu.Lock()
	d.execution = &umpire3CallbackExecution{
		programID: programID, workflowID: workflowID, runID: started.GetRunId(),
		callbackURL: callback.url, callbackToken: callback.token,
		kind: "nexus-after-caller", referenceValidated: true, workflowClosed: true,
	}
	d.mu.Unlock()
	cleanup = false
	return participant.MechanismReceipt{
		WorkflowID: workflowID, RunID: started.GetRunId(), Lineage: []string{started.GetRunId()},
		Source:    "temporal-nexus-callback-registration",
		Reference: fmt.Sprintf("%s/%s/nexus-callback/registered", workflowID, started.GetRunId()),
	}, nil
}

func (d *umpire3CallbackDriver) CompleteCompletionCallback(
	ctx context.Context,
	programID string,
	commandID string,
) (participant.MechanismReceipt, error) {
	d.mu.Lock()
	execution := d.execution
	if execution == nil || execution.programID != programID || execution.completed {
		d.mu.Unlock()
		return participant.MechanismReceipt{}, errors.New("completion callback is not deliverable")
	}
	d.mu.Unlock()
	if execution.kind == "shared-handler" {
		return d.completeSharedHandlerCallbacks(ctx, execution, commandID)
	}
	if execution.kind == "nexus-after-caller" {
		return d.completeNexusCallbackAfterCaller(ctx, execution, commandID)
	}
	_, err := d.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			Namespace: d.env.Namespace().String(), TaskToken: execution.taskToken,
			Identity: "umpire3-completion-callback-driver",
			Commands: []*commandpb.Command{{
				CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
				Attributes: &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{
					CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{},
				},
			}},
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("complete callback workflow: %w", err)
	}
	var completion *nexusrpc.CompletionRequest
	select {
	case completion = <-d.completionCh:
	case <-ctx.Done():
		return participant.MechanismReceipt{}, fmt.Errorf("wait for completion callback delivery: %w", ctx.Err())
	}
	if completion == nil || completion.State != nexus.OperationStateSucceeded {
		return participant.MechanismReceipt{}, errors.New("completion callback receiver did not observe a successful terminal state")
	}
	if err := d.waitForSucceededCallback(ctx, execution); err != nil {
		return participant.MechanismReceipt{}, err
	}
	d.mu.Lock()
	execution.responseValidated = true
	execution.completed = true
	d.mu.Unlock()
	return participant.MechanismReceipt{
		WorkflowID: execution.workflowID, RunID: execution.runID, Lineage: []string{execution.runID},
		Source: "temporal-nexus-completion-callback-receiver",
		Reference: fmt.Sprintf("%s/%s/callback/%s/delivered/%s",
			execution.workflowID, execution.runID, commandID, completion.HTTPRequest.URL.Path),
	}, nil
}

func (d *umpire3CallbackDriver) completeNexusCallbackAfterCaller(
	ctx context.Context,
	execution *umpire3CallbackExecution,
	commandID string,
) (participant.MechanismReceipt, error) {
	client := nexusrpc.NewCompletionHTTPClient(nexusrpc.CompletionHTTPClientOptions{
		Serializer: commonnexus.PayloadSerializer,
	})
	err := client.CompleteOperation(ctx, execution.callbackURL, nexusrpc.CompleteOperationOptions{
		Header: nexus.Header{commonnexus.CallbackTokenHeader: execution.callbackToken},
		Result: payload.EncodeString("umpire3-late-success"),
	})
	if err == nil {
		return participant.MechanismReceipt{}, errors.New("Temporal accepted a Nexus callback after its caller completed")
	}
	history := d.env.GetHistory(d.env.Namespace().String(), &commonpb.WorkflowExecution{
		WorkflowId: execution.workflowID, RunId: execution.runID,
	})
	startedObserved := false
	callerCompleted := false
	for _, event := range history {
		startedObserved = startedObserved || event.GetEventType() == enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED
		callerCompleted = callerCompleted || event.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED
	}
	if !startedObserved || !callerCompleted {
		return participant.MechanismReceipt{}, errors.New("late Nexus callback rejection lacks caller history evidence")
	}
	d.mu.Lock()
	execution.responseValidated = true
	execution.completed = true
	d.mu.Unlock()
	return participant.MechanismReceipt{
		WorkflowID: execution.workflowID, RunID: execution.runID, Lineage: []string{execution.runID},
		Source: "temporal-nexus-callback-rejection",
		Reference: fmt.Sprintf("%s/%s/nexus-callback/%s/rejected-after-close",
			execution.workflowID, execution.runID, commandID),
	}, nil
}

func (d *umpire3CallbackDriver) ValidatedObservation(observation string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.execution == nil {
		return false
	}
	switch observation {
	case string(protocol.ObservationIDCallbackReferenceValid):
		return d.execution.referenceValidated
	case string(protocol.ObservationIDCallbackResponseConsistent):
		return d.execution.referenceValidated && d.execution.responseValidated
	default:
		return false
	}
}

func (d *umpire3CallbackDriver) waitForSucceededCallback(
	ctx context.Context,
	execution *umpire3CallbackExecution,
) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		description, err := d.env.SdkClient().DescribeWorkflowExecution(ctx, execution.workflowID, execution.runID)
		if err != nil {
			return fmt.Errorf("describe delivered completion callback: %w", err)
		}
		if len(description.GetCallbacks()) == 1 &&
			description.GetCallbacks()[0].GetState() == enumspb.CALLBACK_STATE_SUCCEEDED &&
			description.GetCallbacks()[0].GetAttempt() > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for succeeded completion callback state: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (d *umpire3CallbackDriver) validateStorageVariant(ctx context.Context, workflowID, runID string) error {
	if d.variant == "" {
		return nil
	}
	description, err := d.env.AdminClient().DescribeMutableState(ctx, &adminservice.DescribeMutableStateRequest{
		Namespace: d.env.Namespace().String(),
		Execution: &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: runID},
		Archetype: chasm.WorkflowArchetype,
	})
	if err != nil {
		return fmt.Errorf("describe completion callback mechanism variant: %w", err)
	}
	inHSM := len(description.GetDatabaseMutableState().GetExecutionInfo().
		GetSubStateMachinesByType()[callbacks.StateMachineType].GetMachinesById()) != 0
	inCHASM := false
	for key := range description.GetDatabaseMutableState().GetChasmNodes() {
		if strings.HasPrefix(key, "Callbacks#") {
			inCHASM = true
			break
		}
	}
	expectCHASM := d.variant == "chasm"
	if inCHASM != expectCHASM || inHSM == expectCHASM {
		return fmt.Errorf("completion callback mechanism variant mismatch: variant=%s hsm=%t chasm=%t",
			d.variant, inHSM, inCHASM)
	}
	return nil
}

func (d *umpire3CallbackDriver) CleanupCompletionCallbacks(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.server != nil {
		defer d.server.Close()
		d.server = nil
	}
	if d.execution == nil || d.execution.completed || d.execution.workflowClosed {
		return nil
	}
	if d.shared != nil {
		d.cleanupSharedHandler(ctx, d.shared)
		d.execution.completed = true
		return nil
	}
	_, err := d.env.FrontendClient().TerminateWorkflowExecution(ctx,
		&workflowservice.TerminateWorkflowExecutionRequest{
			Namespace: d.env.Namespace().String(),
			WorkflowExecution: &commonpb.WorkflowExecution{
				WorkflowId: d.execution.workflowID, RunId: d.execution.runID,
			},
			Reason: "Umpire3 completion callback driver cleanup",
		})
	if err != nil {
		return fmt.Errorf("terminate completion callback workflow: %w", err)
	}
	d.execution.completed = true
	return nil
}

func (f *umpire3WorkflowTaskFencer) FenceWorkflowOwner(
	ctx context.Context,
	programID string,
	commandID string,
) (participant.MechanismReceipt, error) {
	if f.env == nil {
		return participant.MechanismReceipt{}, errors.New("Workflow Task fencer requires a Temporal environment")
	}
	workflowID := umpire3SDKWorkflowID(programID+"-fence", commandID)
	taskQueue := workflowID + "-task-queue"
	identity := "umpire3-workflow-task-fencer"
	started, err := f.env.FrontendClient().StartWorkflowExecution(ctx, &workflowservice.StartWorkflowExecutionRequest{
		Namespace: f.env.Namespace().String(), WorkflowId: workflowID, RequestId: uuid.NewString(),
		WorkflowType:        &commonpb.WorkflowType{Name: "umpire3-fenced-workflow"},
		TaskQueue:           &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		WorkflowRunTimeout:  durationpb.New(30 * time.Second),
		WorkflowTaskTimeout: durationpb.New(10 * time.Second), Identity: identity,
	})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("start fenced workflow: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_, _ = f.env.FrontendClient().TerminateWorkflowExecution(context.Background(),
				&workflowservice.TerminateWorkflowExecutionRequest{
					Namespace:         f.env.Namespace().String(),
					WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: started.GetRunId()},
					Reason:            "Umpire3 Workflow Task fencer cleanup",
				})
		}
	}()
	poll := func(owner string) (*workflowservice.PollWorkflowTaskQueueResponse, error) {
		response, pollErr := f.env.FrontendClient().PollWorkflowTaskQueue(ctx, &workflowservice.PollWorkflowTaskQueueRequest{
			Namespace: f.env.Namespace().String(),
			TaskQueue: &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:  owner,
		})
		if pollErr != nil {
			return nil, pollErr
		}
		if len(response.GetTaskToken()) == 0 {
			return nil, errors.New("Workflow Task poll returned an empty task token")
		}
		return response, nil
	}
	stale, err := poll(identity + "-stale")
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("poll stale Workflow Task owner: %w", err)
	}
	_, err = f.env.FrontendClient().RespondWorkflowTaskFailed(ctx, &workflowservice.RespondWorkflowTaskFailedRequest{
		Namespace: f.env.Namespace().String(), TaskToken: stale.GetTaskToken(),
		Cause:    enumspb.WORKFLOW_TASK_FAILED_CAUSE_WORKFLOW_WORKER_UNHANDLED_FAILURE,
		Identity: identity + "-stale",
	})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("supersede stale Workflow Task owner: %w", err)
	}
	current, err := poll(identity + "-current")
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("poll current Workflow Task owner: %w", err)
	}
	_, staleErr := f.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			Namespace: f.env.Namespace().String(), TaskToken: stale.GetTaskToken(), Identity: identity + "-stale",
		})
	var notFound *serviceerror.NotFound
	if !errors.As(staleErr, &notFound) {
		return participant.MechanismReceipt{}, fmt.Errorf("stale Workflow Task completion was not fenced: %w", staleErr)
	}
	_, err = f.env.FrontendClient().RespondWorkflowTaskCompleted(ctx,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			Namespace: f.env.Namespace().String(), TaskToken: current.GetTaskToken(), Identity: identity + "-current",
			Commands: []*commandpb.Command{{
				CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
				Attributes: &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{
					CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{},
				},
			}},
		})
	if err != nil {
		return participant.MechanismReceipt{}, fmt.Errorf("complete current Workflow Task owner: %w", err)
	}
	cleanup = false
	return participant.MechanismReceipt{
		WorkflowID: workflowID, RunID: started.GetRunId(), Lineage: []string{started.GetRunId()},
		Source: identity,
		Reference: fmt.Sprintf("%s/%s/workflow-task/%d/fenced-before/%d",
			workflowID, started.GetRunId(), stale.GetStartedEventId(), current.GetStartedEventId()),
	}, nil
}
