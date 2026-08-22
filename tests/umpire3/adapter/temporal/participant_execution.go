package temporal

import (
	"context"
	"errors"
	"fmt"

	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/server/tests/umpire3/execution/participant"
)

func (r *SDKParticipantAdapter) Execute(ctx context.Context, operation participant.Operation) (participant.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started || r.stopped {
		return participant.Result{}, errors.New("SDK participant is not running")
	}
	if err := ctx.Err(); err != nil {
		return participant.Result{}, err
	}
	if operation.Response == participant.ResponseFailure {
		return r.qualifyResult(operation, participant.Result{CommandID: operation.CommandID, Status: "failed"})
	}
	if operation.SDKOperation == participant.SDKHandleNexus && r.options.NexusDriver != nil {
		receipt, handled, err := r.options.NexusDriver.ExecuteNexusAction(ctx, r.plan.ProgramID, operation)
		if err != nil {
			return participant.Result{}, fmt.Errorf("execute dedicated Nexus action %q: %w", operation.CommandID, err)
		}
		if handled {
			if receipt.WorkflowID == "" || receipt.RunID == "" || receipt.Reference == "" || receipt.Source == "" {
				return participant.Result{}, errors.New("dedicated Nexus driver returned incomplete evidence")
			}
			return r.qualifyResult(operation, participant.Result{
				CommandID: operation.CommandID, Status: responseStatus(operation.Response),
				WorkflowID: receipt.WorkflowID, RunID: receipt.RunID,
				Lineage: append([]string(nil), receipt.Lineage...), Reference: receipt.Reference,
				Source: receipt.Source, SourceIdentity: receipt.Source,
				TerminalState: receipt.TerminalState, TerminalDisposition: receipt.TerminalDisposition,
			})
		}
	}
	if detachedResponse(operation.Response) && executesInParticipantWorkflow(operation.SDKOperation) {
		if err := r.options.Client.SignalWorkflow(
			ctx, r.run.GetID(), r.run.GetRunID(), SDKCommandSignalName, operation,
		); err != nil {
			return participant.Result{}, fmt.Errorf("dispatch asynchronous SDK operation %q: %w", operation.CommandID, err)
		}
		r.programExecutions++
		return r.qualifyResult(operation, participant.Result{
			CommandID: operation.CommandID,
			Status:    responseStatus(operation.Response),
		})
	}
	var result participant.Result
	var err error
	executedInProgram := false
	switch operation.SDKOperation {
	case participant.SDKHandleSignal:
		err = r.options.Client.SignalWorkflow(ctx, r.run.GetID(), r.run.GetRunID(), SDKCommandSignalName, operation)
		if err == nil {
			result, err = r.waitForResult(ctx, operation.CommandID)
		}
	case participant.SDKHandleQuery:
		result, err = r.queryResult(ctx, operation.CommandID)
		if err == nil && result.CommandID == "" {
			result = participant.Result{CommandID: operation.CommandID, Status: "completed"}
		}
	case participant.SDKContinueWorkflow:
		result, err = r.executeContinuation(ctx, operation)
	case participant.SDKResetWorkflow:
		result, err = r.executeReset(ctx, operation)
	case participant.SDKRouteWorkflowTask:
		result, err = r.executeRoutedWorkflow(ctx, operation)
	case participant.SDKFenceWorkflowOwner:
		result, err = r.executeOwnershipFence(ctx, operation)
	case participant.SDKRegisterCallback:
		result, err = r.executeCallback(ctx, operation, false)
	case participant.SDKCompleteCallback:
		result, err = r.executeCallback(ctx, operation, true)
	default:
		executedInProgram = true
		var handle sdkclient.WorkflowUpdateHandle
		handle, err = r.options.Client.UpdateWorkflow(ctx, sdkclient.UpdateWorkflowOptions{
			UpdateID:   updateID(r.plan.ProgramID, operation.CommandID),
			WorkflowID: r.run.GetID(), RunID: r.run.GetRunID(), UpdateName: SDKCommandUpdateName,
			Args: []any{operation}, WaitForStage: sdkclient.WorkflowUpdateStageCompleted,
		})
		if err == nil {
			err = handle.Get(ctx, &result)
		}
	}
	if err != nil {
		return participant.Result{}, fmt.Errorf("execute SDK operation %q: %w", operation.CommandID, err)
	}
	qualified, err := r.qualifyResult(operation, result)
	if err != nil {
		return participant.Result{}, err
	}
	if executedInProgram {
		r.programExecutions++
		if r.programExecutions == len(r.plan.Operations) {
			var results []participant.Result
			if err := r.run.Get(ctx, &results); err != nil {
				return participant.Result{}, fmt.Errorf("complete SDK participant program: %w", err)
			}
			r.workflowClosed = true
		}
	}
	return qualified, nil
}

func executesInParticipantWorkflow(operation participant.SDKOperation) bool {
	switch operation {
	case participant.SDKContinueWorkflow, participant.SDKResetWorkflow,
		participant.SDKRouteWorkflowTask, participant.SDKFenceWorkflowOwner,
		participant.SDKRegisterCallback, participant.SDKCompleteCallback:
		return false
	default:
		return true
	}
}
