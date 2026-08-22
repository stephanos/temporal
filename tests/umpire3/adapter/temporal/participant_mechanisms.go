package temporal

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/server/tests/umpire3/execution/participant"
)

func (r *SDKParticipantAdapter) executeContinuation(ctx context.Context, operation participant.Operation) (participant.Result, error) {
	workflowID := r.mechanismWorkflowID(operation.CommandID, "continuation")
	run, err := r.options.Client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID: workflowID, TaskQueue: r.options.TaskQueue,
	}, r.options.WorkflowType+"-continuation", false)
	if err != nil {
		return participant.Result{}, fmt.Errorf("start continuation workflow: %w", err)
	}
	firstRunID := run.GetRunID()
	if err := run.Get(ctx, nil); err != nil {
		return participant.Result{}, fmt.Errorf("complete continuation workflow: %w", err)
	}
	description, err := r.options.Client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return participant.Result{}, fmt.Errorf("describe continuation workflow: %w", err)
	}
	successorRunID := description.GetWorkflowExecutionInfo().GetExecution().GetRunId()
	if firstRunID == "" || successorRunID == "" || firstRunID == successorRunID {
		return participant.Result{}, errors.New("continuation workflow did not produce distinct predecessor and successor runs")
	}
	return participant.Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: workflowID, RunID: successorRunID, Lineage: []string{firstRunID, successorRunID},
		Reference: workflowID + "/" + successorRunID + "/continued-from/" + firstRunID,
		Source:    "temporal-sdk-continuation", SourceIdentity: r.options.WorkflowType + "-continuation",
	}, nil
}

func (r *SDKParticipantAdapter) executeReset(ctx context.Context, operation participant.Operation) (participant.Result, error) {
	if r.options.Namespace == "" {
		return participant.Result{}, errors.New("reset workflow requires a namespace")
	}
	workflowID := r.mechanismWorkflowID(operation.CommandID, "reset")
	run, err := r.options.Client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID: workflowID, TaskQueue: r.options.TaskQueue,
	}, r.options.WorkflowType+"-immediate")
	if err != nil {
		return participant.Result{}, fmt.Errorf("start reset base workflow: %w", err)
	}
	baseRunID := run.GetRunID()
	if err := run.Get(ctx, nil); err != nil {
		return participant.Result{}, fmt.Errorf("complete reset base workflow: %w", err)
	}
	response, err := r.options.Client.ResetWorkflowExecution(ctx, &workflowservice.ResetWorkflowExecutionRequest{
		Namespace:         r.options.Namespace,
		WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: baseRunID},
		Reason:            "Umpire3 reset lineage action", WorkflowTaskFinishEventId: 4,
		RequestId: uuid.NewSHA1(uuid.Nil, []byte(r.plan.ProgramID+"\x00"+operation.CommandID)).String(),
	})
	if err != nil {
		return participant.Result{}, fmt.Errorf("reset workflow execution: %w", err)
	}
	resetRunID := response.GetRunId()
	if resetRunID == "" || resetRunID == baseRunID {
		return participant.Result{}, errors.New("reset workflow did not produce a distinct successor run")
	}
	if err := r.options.Client.GetWorkflow(ctx, workflowID, resetRunID).Get(ctx, nil); err != nil {
		return participant.Result{}, fmt.Errorf("complete reset successor workflow: %w", err)
	}
	return participant.Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: workflowID, RunID: resetRunID, Lineage: []string{baseRunID, resetRunID},
		Reference: workflowID + "/" + resetRunID + "/reset-from/" + baseRunID,
		Source:    "temporal-sdk-reset", SourceIdentity: r.options.WorkflowType + "-immediate",
	}, nil
}

func (r *SDKParticipantAdapter) executeRoutedWorkflow(ctx context.Context, operation participant.Operation) (participant.Result, error) {
	workflowID := r.mechanismWorkflowID(operation.CommandID, "routing")
	run, err := r.options.Client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID: workflowID, TaskQueue: r.options.TaskQueue,
	}, r.options.WorkflowType+"-immediate")
	if err != nil {
		return participant.Result{}, fmt.Errorf("start routed workflow: %w", err)
	}
	if err := run.Get(ctx, nil); err != nil {
		return participant.Result{}, fmt.Errorf("complete routed workflow: %w", err)
	}
	return participant.Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: workflowID, RunID: run.GetRunID(), Lineage: []string{run.GetRunID()},
		Reference: workflowID + "/" + run.GetRunID() + "/task-queue/" + r.options.TaskQueue,
		Source:    "temporal-sdk-routing", SourceIdentity: r.options.TaskQueue,
	}, nil
}

func (r *SDKParticipantAdapter) executeOwnershipFence(ctx context.Context, operation participant.Operation) (participant.Result, error) {
	if r.options.WorkflowTaskFencer == nil {
		return participant.Result{}, errors.New("workflow task ownership fencing is unavailable")
	}
	receipt, err := r.options.WorkflowTaskFencer.FenceWorkflowOwner(ctx, r.plan.ProgramID, operation.CommandID)
	if err != nil {
		return participant.Result{}, err
	}
	if receipt.WorkflowID == "" || receipt.RunID == "" || receipt.Reference == "" || receipt.Source == "" {
		return participant.Result{}, errors.New("workflow task ownership fencer returned incomplete evidence")
	}
	return participant.Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: receipt.WorkflowID, RunID: receipt.RunID,
		Lineage: append([]string(nil), receipt.Lineage...), Reference: receipt.Reference,
		Source: receipt.Source, SourceIdentity: receipt.Source,
	}, nil
}

func (r *SDKParticipantAdapter) executeCallback(ctx context.Context, operation participant.Operation, complete bool) (participant.Result, error) {
	if r.options.CallbackDriver == nil {
		return participant.Result{}, errors.New("completion callback driver is unavailable")
	}
	var receipt MechanismReceipt
	var err error
	if complete {
		receipt, err = r.options.CallbackDriver.CompleteCompletionCallback(ctx, r.plan.ProgramID, operation.CommandID)
	} else {
		receipt, err = r.options.CallbackDriver.RegisterCompletionCallback(ctx, r.plan.ProgramID, operation.CommandID)
	}
	if err != nil {
		return participant.Result{}, err
	}
	if receipt.WorkflowID == "" || receipt.RunID == "" || receipt.Reference == "" || receipt.Source == "" {
		return participant.Result{}, errors.New("completion callback driver returned incomplete evidence")
	}
	return participant.Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: receipt.WorkflowID, RunID: receipt.RunID,
		Lineage: append([]string(nil), receipt.Lineage...), Reference: receipt.Reference,
		Source: receipt.Source, SourceIdentity: receipt.Source,
	}, nil
}

func (r *SDKParticipantAdapter) mechanismWorkflowID(commandID, mechanism string) string {
	return safeSDKName(r.options.WorkflowID + "-" + mechanism + "-" + commandID)
}

func (r *SDKParticipantAdapter) terminateAfterStopFailure(stopErr error) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), r.options.CleanupTimeout)
	defer cancel()
	terminateErr := r.options.Client.TerminateWorkflow(
		cleanupCtx, r.run.GetID(), r.run.GetRunID(), "umpire3 participant cleanup after graceful stop failure")
	if terminateErr != nil {
		return errors.Join(fmt.Errorf("stop SDK participant: %w", stopErr),
			fmt.Errorf("terminate SDK participant: %w", terminateErr))
	}
	r.stopped = true
	return fmt.Errorf("stop SDK participant: %w", stopErr)
}
