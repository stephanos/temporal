package temporal

import (
	"errors"
	"fmt"

	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/tests/umpire3/execution/participant"
)

func SDKProgramWorkflow(ctx workflow.Context, input SDKWorkflowInput) ([]participant.Result, error) {
	if input.FormatVersion != SDKProgramFormatVersion || input.Plan.FormatVersion != participant.FormatVersion ||
		input.ActivityType == "" || input.ChildType == "" {
		return nil, errors.New("complete SDK participant workflow input is required")
	}
	results := make(map[string]participant.Result, len(input.Plan.Operations))
	semanticState := sdkSemanticState{}
	completionRequested := false
	completionChannel := workflow.NewBufferedChannel(ctx, 1)
	if err := workflow.SetQueryHandler(ctx, SDKStateQueryName, func() (SDKWorkflowState, error) {
		return SDKWorkflowState{FormatVersion: SDKProgramFormatVersion, Results: cloneResults(results)}, nil
	}); err != nil {
		return nil, fmt.Errorf("register SDK participant query: %w", err)
	}
	execute := func(updateCtx workflow.Context, operation participant.Operation) (participant.Result, error) {
		if _, duplicate := results[operation.CommandID]; duplicate {
			return participant.Result{}, fmt.Errorf("participant command %q already executed", operation.CommandID)
		}
		result, err := executeSDKOperation(updateCtx, input, operation, &semanticState)
		if err != nil {
			return participant.Result{}, err
		}
		results[operation.CommandID] = result
		if len(results) == len(input.Plan.Operations) && !completionRequested {
			completionRequested = true
			completionChannel.SendAsync(true)
		}
		return result, nil
	}
	if err := workflow.SetUpdateHandler(ctx, SDKCommandUpdateName, execute); err != nil {
		return nil, fmt.Errorf("register SDK participant update: %w", err)
	}
	commandSignal := workflow.GetSignalChannel(ctx, SDKCommandSignalName)
	finishSignal := workflow.GetSignalChannel(ctx, SDKFinishSignalName)
	for {
		selector := workflow.NewSelector(ctx)
		finished := false
		selector.AddReceive(finishSignal, func(workflow.ReceiveChannel, bool) { finished = true })
		selector.AddReceive(completionChannel, func(channel workflow.ReceiveChannel, _ bool) {
			var complete bool
			channel.Receive(ctx, &complete)
			finished = complete
		})
		selector.AddReceive(commandSignal, func(channel workflow.ReceiveChannel, _ bool) {
			var operation participant.Operation
			channel.Receive(ctx, &operation)
			result, err := execute(ctx, operation)
			if err != nil {
				results[operation.CommandID] = participant.Result{CommandID: operation.CommandID, Status: "failed"}
				return
			}
			results[operation.CommandID] = result
		})
		selector.Select(ctx)
		if finished {
			ordered := make([]participant.Result, 0, len(results))
			for _, operation := range input.Plan.Operations {
				if result, exists := results[operation.CommandID]; exists {
					ordered = append(ordered, result)
				}
			}
			return ordered, nil
		}
	}
}
