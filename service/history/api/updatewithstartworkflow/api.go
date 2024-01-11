package updatewithstartworkflow

import (
	"context"
	"errors"

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
	"go.temporal.io/server/common/persistence"
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

	// @SB find current WorkflowContext, if there is already one,
	// by using Workflow ID and Namespace ID in database query
	var currentWorkflowContext api.WorkflowContext
	wfID := req.Request.StartRequest.WorkflowId
	wfKey := definition.NewWorkflowKey(string(namespaceEntry.ID()), wfID, "")
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

	// @SB this is dumb; but that's one way to get the context into the Workflow Cache
	//ctx, err = workflowConsistencyChecker.GetWorkflowContext(
	//	ctx,
	//	nil,
	//	api.BypassMutableStateConsistencyPredicate,
	//	newWorkflowContext.GetWorkflowKey(),
	//	workflow.LockPriorityHigh,
	//)

	// TODO: ApplyWorkflowIDReusePolicy()
	// TODO: CreateWorkflowCASPredicate

	var updateState *update.Update
	if err = api.GetAndUpdateWorkflowWithNew(
		ctx,
		nil,
		api.BypassMutableStateConsistencyPredicate,
		newWorkflowContext.GetWorkflowKey(),
		func(inner api.WorkflowContext) (*api.UpdateWorkflowAction, error) {
			// TODO: this is incomplete; eg no Speculative WorkflowTask

			ms := inner.GetMutableState()
			updateReq := req.GetRequest().GetUpdate()
			updateID := updateReq.GetMeta().GetUpdateId()
			updateReg := inner.GetUpdateRegistry(ctx)

			var err error
			if updateState, _, err = updateReg.FindOrCreate(ctx, updateID); err != nil {
				return nil, err
			}

			if err = updateState.OnMessage(ctx, updateReq, workflow.WithEffects(effect.Immediate(ctx), ms)); err != nil {
				return nil, err
			}

			return &api.UpdateWorkflowAction{
				Noop:               false,
				CreateWorkflowTask: true,
			}, nil
		},
		nil,
		shard,
		workflowConsistencyChecker,
	); err != nil {
		return nil, err
	}

	// @SB copied from bottom of UpdateWorkflow
	serverTimeout := shard.GetConfig().LongPollExpirationInterval(namespaceEntry.Name().String())
	waitStage := req.GetRequest().GetWaitPolicy().GetLifecycleStage()
	status, err := updateState.WaitLifecycleStage(ctx, waitStage, serverTimeout)
	if err != nil {
		return nil, err
	}

	wfKey = newWorkflowContext.GetMutableState().GetWorkflowKey()
	return &historyservice.UpdateWithStartWorkflowExecutionResponse{
		Response: &workflowservice.UpdateWithStartWorkflowExecutionResponse{
			UpdateResponse: &workflowservice.UpdateWorkflowExecutionResponse{
				UpdateRef: &updatepb.UpdateRef{
					WorkflowExecution: &commonpb.WorkflowExecution{
						WorkflowId: wfKey.GetWorkflowID(),
						RunId:      wfKey.GetRunID(),
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
	// @SB creates a StartWorkflowExecutionRequest (for History)
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
	newWorkflowContext, err := api.NewWorkflowWithSignal( // @SB TODO I don't like that name
		ctx, shard, namespaceEntry, req.Request.StartRequest.WorkflowId, runID, startReq, nil)
	if err != nil {
		return nil, err
	}

	wfSnapshot, newWorkflowEventsSeq, err := newWorkflowContext.GetMutableState().CloseTransactionAsSnapshot(
		workflow.TransactionPolicyActive,
	)
	if err != nil {
		return nil, err
	}
	if len(newWorkflowEventsSeq) != 1 {
		return nil, serviceerror.NewInternal("unable to create 1st event batch")
	}

	// TODO: NewWorkflowVersionCheck? casPredicate?
	// (see startAndSignalWithoutCurrentWorkflow)

	createMode := persistence.CreateWorkflowModeBrandNew
	prevRunID := ""
	err = newWorkflowContext.GetContext().CreateWorkflowExecution(
		ctx,
		shard,
		createMode,
		prevRunID,
		int64(0),
		newWorkflowContext.GetMutableState(),
		wfSnapshot,
		newWorkflowEventsSeq,
	)

	var failedErr *persistence.CurrentWorkflowConditionFailedError
	switch {
	case errors.As(err, &failedErr):
		// @SB TODO is this needed?
		//if failedErr.RequestID == requestID {
		//	return failedErr.RunID, nil
		//}
		return nil, failedErr
	default:
		return newWorkflowContext, err
	}
}
