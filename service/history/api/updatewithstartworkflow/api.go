package updatewithstartworkflow

import (
	"context"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/common"
	"go.temporal.io/server/common/definition"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/internal/effect"
	"go.temporal.io/server/service/history/api"
	"go.temporal.io/server/service/history/shard"
	"go.temporal.io/server/service/history/workflow"
	"go.temporal.io/server/service/history/workflow/update"
)

func Invoke(
	ctx context.Context,
	req *historyservice.UpdateWithStartWorkflowExecutionRequest,
	shard shard.Context,
	workflowConsistencyChecker api.WorkflowConsistencyChecker,
) (_ *historyservice.UpdateWithStartWorkflowExecutionResponse, retError error) {
	namespaceEntry, err := api.GetActiveNamespace(shard, namespace.ID(req.GetNamespaceId()))
	if err != nil {
		return nil, err
	}

	var currentWorkflowContext api.WorkflowContext
	wfKey := definition.NewWorkflowKey(namespaceEntry.ID().String(), req.Request.StartRequest.WorkflowId, "")
	currentWorkflowContext, err = workflowConsistencyChecker.GetWorkflowContext(
		ctx, nil, api.BypassMutableStateConsistencyPredicate, wfKey, workflow.LockPriorityHigh)
	switch err.(type) {
	case nil:
		defer func() { currentWorkflowContext.GetReleaseFn()(retError) }()
	case *serviceerror.NotFound:
		currentWorkflowContext = nil
	default:
		return nil, err
	}

	newWorkflowContext, err := getOrCreateWorkflow(ctx, req, shard, namespaceEntry, currentWorkflowContext)
	if err != nil {
		return nil, err
	}

	updateState, err := updateWorkflow(ctx, shard, req, newWorkflowContext)
	if err != nil {
		return nil, err
	}

	serverTimeout := shard.GetConfig().LongPollExpirationInterval(namespaceEntry.Name().String())
	waitStage := req.GetRequest().GetWaitPolicy().GetLifecycleStage()
	// If the long-poll times out due to serverTimeout then return a non-error empty response.
	status, err := updateState.WaitLifecycleStage(ctx, waitStage, serverTimeout)
	if err != nil {
		return nil, err
	}

	return &historyservice.UpdateWithStartWorkflowExecutionResponse{
		Response: &workflowservice.UpdateWithStartWorkflowExecutionResponse{
			UpdateResponse: &workflowservice.UpdateWorkflowExecutionResponse{
				UpdateRef: &updatepb.UpdateRef{
					WorkflowExecution: &commonpb.WorkflowExecution{
						WorkflowId: wfKey.WorkflowID,
						RunId:      wfKey.RunID,
					},
					UpdateId: req.GetRequest().GetUpdate().GetMeta().GetUpdateId(),
				},
				Outcome: status.Outcome,
				Stage:   status.Stage,
			},
		},
	}, nil
}

func getOrCreateWorkflow(
	ctx context.Context,
	req *historyservice.UpdateWithStartWorkflowExecutionRequest,
	shard shard.Context,
	namespaceEntry *namespace.Namespace,
	currentWorkflowContext api.WorkflowContext,
) (api.WorkflowContext, error) {
	startReq := common.CreateHistoryStartWorkflowRequest(
		namespaceEntry.ID().String(),
		&workflowservice.StartWorkflowExecutionRequest{
			Namespace:                req.Request.StartRequest.GetNamespace(),
			WorkflowId:               req.Request.StartRequest.GetWorkflowId(),
			WorkflowType:             req.Request.StartRequest.GetWorkflowType(),
			TaskQueue:                req.Request.StartRequest.GetTaskQueue(),
			Input:                    req.Request.StartRequest.GetInput(),
			WorkflowExecutionTimeout: req.Request.StartRequest.GetWorkflowExecutionTimeout(),
			WorkflowRunTimeout:       req.Request.StartRequest.GetWorkflowRunTimeout(),
			WorkflowTaskTimeout:      req.Request.StartRequest.GetWorkflowTaskTimeout(),
			Identity:                 req.Request.StartRequest.GetIdentity(),
			RequestId:                req.Request.StartRequest.GetRequestId(),
			WorkflowIdReusePolicy:    req.Request.StartRequest.GetWorkflowIdReusePolicy(),
			RetryPolicy:              req.Request.StartRequest.GetRetryPolicy(),
			CronSchedule:             req.Request.StartRequest.GetCronSchedule(),
			Memo:                     req.Request.StartRequest.GetMemo(),
			SearchAttributes:         req.Request.StartRequest.GetSearchAttributes(),
			Header:                   req.Request.StartRequest.GetHeader(),
			WorkflowStartDelay:       req.Request.StartRequest.GetWorkflowStartDelay(),
		},
		nil,
		shard.GetTimeSource().Now())

	// TODO OverrideStartWorkflowExecutionRequest?

	err := api.ValidateStartWorkflowExecutionRequest(
		ctx, startReq.StartRequest, shard, namespaceEntry, "UpdateWithStartWorkflow")
	if err != nil {
		return nil, err
	}

	if currentWorkflowContext != nil &&
		currentWorkflowContext.GetMutableState().IsWorkflowExecutionRunning() &&
		req.Request.StartRequest.WorkflowIdReusePolicy != enumspb.WORKFLOW_ID_REUSE_POLICY_TERMINATE_IF_RUNNING {
		return currentWorkflowContext, nil
	}

	runID := uuid.New().String()
	newWorkflowContext, err := api.NewWorkflowWithSignal(
		ctx, shard, namespaceEntry, req.Request.StartRequest.WorkflowId, runID, startReq, nil)
	if err != nil {
		return nil, err
	}

	return newWorkflowContext, nil
}

func updateWorkflow(
	ctx context.Context,
	shard shard.Context,
	req *historyservice.UpdateWithStartWorkflowExecutionRequest,
	currentWorkflowContext api.WorkflowContext,
) (*update.Update, error) {
	// TODO: ApplyWorkflowIDReusePolicy()
	// TODO: CreateWorkflowCASPredicate

	var updateState *update.Update
	err := api.UpdateWorkflowWithNew(
		shard, ctx, currentWorkflowContext,
		func(inner api.WorkflowContext) (*api.UpdateWorkflowAction, error) {
			// TODO: this is incomplete
			ms := currentWorkflowContext.GetMutableState()
			updateReq := req.GetRequest().GetUpdate()
			updateID := updateReq.GetMeta().GetUpdateId()

			updateReg := currentWorkflowContext.GetUpdateRegistry(ctx)

			var err error
			if updateState, _, err = updateReg.FindOrCreate(ctx, updateID); err != nil {
				return nil, err
			}
			if err := updateState.OnMessage(ctx, updateReq, workflow.WithEffects(effect.Immediate(ctx), ms)); err != nil {
				return nil, err
			}

			return &api.UpdateWorkflowAction{
				Noop:               false,
				CreateWorkflowTask: true,
			}, nil
		},
		func() (workflow.Context, workflow.MutableState, error) {
			return currentWorkflowContext.GetContext(), currentWorkflowContext.GetMutableState(), nil
		},
	)

	return updateState, err
}
