package temporal

import (
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/tools/umpire3/execution/participant"
)

type sdkSemanticState struct {
	cancellationCommitted bool
}

func executeSDKOperation(
	ctx workflow.Context,
	input SDKWorkflowInput,
	operation participant.Operation,
	semanticState *sdkSemanticState,
) (participant.Result, error) {
	result := participant.Result{CommandID: operation.CommandID, Status: responseStatus(operation.Response)}
	if operation.Response == participant.ResponseFailure || operation.SDKOperation == participant.SDKReturnFailure {
		return participant.Result{CommandID: operation.CommandID, Status: "failed"}, nil
	}
	if operation.Response == participant.ResponseBlocking {
		if operation.MaxBlockNanos <= 0 {
			return participant.Result{}, errors.New("blocking SDK operation requires a positive bound")
		}
		if err := workflow.Sleep(ctx, time.Duration(operation.MaxBlockNanos)); err != nil {
			return participant.Result{}, err
		}
	}
	if semanticState.cancellationCommitted &&
		(operation.SemanticAction == "worker-returns-success" || operation.SemanticAction == "persist-success") {
		result.Status = "suppressed"
		return result, nil
	}
	switch operation.SDKOperation {
	case participant.SDKExecuteActivity, participant.SDKRetry:
		activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 2},
		})
		future := workflow.ExecuteActivity(activityCtx, input.ActivityType, operation)
		if detachedResponse(operation.Response) {
			return result, nil
		}
		if err := future.Get(ctx, nil); err != nil {
			return participant.Result{}, err
		}
	case participant.SDKExecuteWorkflow, participant.SDKExecuteChild:
		future := workflow.ExecuteChildWorkflow(ctx, input.ChildType, operation)
		if detachedResponse(operation.Response) {
			return result, nil
		}
		if err := future.Get(ctx, nil); err != nil {
			return participant.Result{}, err
		}
	case participant.SDKCancel:
		if operation.SemanticAction == "request-cancellation" && input.NexusEndpoint != "" &&
			input.NexusService != "" && input.NexusOperation != "" {
			nexusCtx, cancelNexus := workflow.WithCancel(ctx)
			client := workflow.NewNexusClient(input.NexusEndpoint, input.NexusService)
			future := client.ExecuteOperation(nexusCtx, input.NexusOperation, operation, workflow.NexusOperationOptions{})
			var execution workflow.NexusOperationExecution
			if err := future.GetNexusOperationExecution().Get(ctx, &execution); err != nil {
				cancelNexus()
				return participant.Result{}, err
			}
			cancelNexus()
			if execution.OperationToken != "" {
				if err := workflow.Sleep(ctx, time.Millisecond); err != nil {
					return participant.Result{}, err
				}
			}
		}
		childCtx, cancel := workflow.WithCancel(ctx)
		future := workflow.ExecuteChildWorkflow(childCtx, input.ChildType, operation)
		cancel()
		if err := future.Get(ctx, nil); err != nil && !temporal.IsCanceledError(err) {
			return participant.Result{}, err
		}
		if operation.SemanticAction == "commit-cancellation" {
			semanticState.cancellationCommitted = true
		}
	case participant.SDKStartTimer:
		future := workflow.NewTimer(ctx, time.Millisecond)
		if detachedResponse(operation.Response) {
			return result, nil
		}
		if err := future.Get(ctx, nil); err != nil {
			return participant.Result{}, err
		}
	case participant.SDKHandleNexus:
		if input.NexusEndpoint == "" || input.NexusService == "" || input.NexusOperation == "" {
			return participant.Result{}, errors.New("nexus command requires endpoint, service, and operation")
		}
		client := workflow.NewNexusClient(input.NexusEndpoint, input.NexusService)
		options := workflow.NexusOperationOptions{}
		if operation.SemanticAction == "timeout-nexus-operation" {
			options.StartToCloseTimeout = 100 * time.Millisecond
		}
		future := client.ExecuteOperation(ctx, input.NexusOperation, operation, options)
		if detachedResponse(operation.Response) {
			return result, nil
		}
		err := future.Get(ctx, nil)
		if operation.SemanticAction == "timeout-nexus-operation" && temporal.IsTimeoutError(err) {
			result.TerminalState = "timed_out"
			result.TerminalDisposition = participant.TerminalDispositionFailure
			return result, nil
		}
		if operation.SemanticAction == "close-nexus-operation" && err != nil {
			result.Status = "failed"
			result.TerminalState = "failed"
			result.TerminalDisposition = participant.TerminalDispositionFailure
			return result, nil
		}
		if err != nil {
			return participant.Result{}, err
		}
		if operation.SemanticAction == "close-nexus-operation" {
			result.TerminalState = "succeeded"
			result.TerminalDisposition = participant.TerminalDispositionSuccess
		}
		if operation.SemanticAction == "link-nexus-activity" {
			activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute,
			})
			if err := workflow.ExecuteActivity(activityCtx, input.ActivityType, operation).Get(ctx, nil); err != nil {
				return participant.Result{}, err
			}
		}
	case participant.SDKHandleUpdate, participant.SDKHandleSignal, participant.SDKHandleQuery:
	default:
		return participant.Result{}, fmt.Errorf("unsupported SDK operation %q", operation.SDKOperation)
	}
	return result, nil
}
