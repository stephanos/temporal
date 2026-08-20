package participant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	SDKProgramFormatVersion = "umpire3/sdk-program/v1"
	SDKCommandUpdateName    = "umpire3-command"
	SDKCommandSignalName    = "umpire3-signal"
	SDKStateQueryName       = "umpire3-state"
	SDKFinishSignalName     = "umpire3-finish"
)

type SDKOptions struct {
	Client             sdkclient.Client
	Registry           worker.Registry
	Namespace          string
	TaskQueue          string
	WorkflowID         string
	WorkflowType       string
	ActivityType       string
	ChildType          string
	CleanupTimeout     time.Duration
	NexusEndpoint      string
	NexusService       string
	NexusOperation     string
	WorkflowTaskFencer WorkflowTaskFencer
	CallbackDriver     CallbackDriver
	NexusDriver        NexusDriver
}

type MechanismReceipt struct {
	WorkflowID          string
	RunID               string
	Lineage             []string
	Reference           string
	Source              string
	TerminalState       string
	TerminalDisposition TerminalDisposition
}

type WorkflowTaskFencer interface {
	FenceWorkflowOwner(context.Context, string, string) (MechanismReceipt, error)
}

type CallbackDriver interface {
	RegisterCompletionCallback(context.Context, string, string) (MechanismReceipt, error)
	CompleteCompletionCallback(context.Context, string, string) (MechanismReceipt, error)
	CleanupCompletionCallbacks(context.Context) error
}

type NexusDriver interface {
	ExecuteNexusAction(context.Context, string, Operation) (MechanismReceipt, bool, error)
	CleanupNexus(context.Context) error
}

type SDKWorkflowInput struct {
	FormatVersion  string `json:"formatVersion"`
	Plan           Plan   `json:"plan"`
	ActivityType   string `json:"activityType"`
	ChildType      string `json:"childType"`
	NexusEndpoint  string `json:"nexusEndpoint,omitempty"`
	NexusService   string `json:"nexusService,omitempty"`
	NexusOperation string `json:"nexusOperation,omitempty"`
}

type SDKWorkflowState struct {
	FormatVersion string            `json:"formatVersion"`
	Results       map[string]Result `json:"results"`
}

type SDKRunner struct {
	mu sync.Mutex

	options           SDKOptions
	plan              Plan
	run               sdkclient.WorkflowRun
	started           bool
	stopped           bool
	programExecutions int
	workflowClosed    bool
}

func NewSDKRunner(options SDKOptions) (*SDKRunner, error) {
	if options.Client == nil || options.Registry == nil || options.TaskQueue == "" || options.WorkflowID == "" {
		return nil, errors.New("SDK participant requires client, registry, task queue, and workflow ID")
	}
	if options.CleanupTimeout <= 0 {
		return nil, errors.New("SDK participant requires a positive cleanup timeout")
	}
	nexusValues := 0
	for _, value := range []string{options.NexusEndpoint, options.NexusService, options.NexusOperation} {
		if value != "" {
			nexusValues++
		}
	}
	if nexusValues != 0 && nexusValues != 3 {
		return nil, errors.New("SDK participant Nexus endpoint, service, and operation must be supplied together")
	}
	stem := safeSDKName(options.WorkflowID)
	if options.WorkflowType == "" {
		options.WorkflowType = "umpire3-program-" + stem
	}
	if options.ActivityType == "" {
		options.ActivityType = "umpire3-activity-" + stem
	}
	if options.ChildType == "" {
		options.ChildType = "umpire3-child-" + stem
	}
	return &SDKRunner{options: options}, nil
}

func (r *SDKRunner) Start(ctx context.Context, plan Plan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return errors.New("SDK participant is already started")
	}
	if plan.FormatVersion != FormatVersion || plan.ProgramID == "" || len(plan.Operations) == 0 {
		return errors.New("complete participant plan is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if missing := missingSDKCapabilities(plan, r.options); len(missing) != 0 {
		return fmt.Errorf("SDK participant lacks capabilities %v", missing)
	}
	r.options.Registry.RegisterWorkflowWithOptions(SDKProgramWorkflow, workflow.RegisterOptions{
		Name: r.options.WorkflowType, DisableAlreadyRegisteredCheck: true,
	})
	r.options.Registry.RegisterWorkflowWithOptions(SDKChildWorkflow, workflow.RegisterOptions{
		Name: r.options.ChildType, DisableAlreadyRegisteredCheck: true,
	})
	r.options.Registry.RegisterActivityWithOptions(SDKActivity, activity.RegisterOptions{
		Name: r.options.ActivityType, DisableAlreadyRegisteredCheck: true,
	})
	r.options.Registry.RegisterWorkflowWithOptions(SDKContinueAsNewWorkflow, workflow.RegisterOptions{
		Name: r.options.WorkflowType + "-continuation", DisableAlreadyRegisteredCheck: true,
	})
	r.options.Registry.RegisterWorkflowWithOptions(SDKImmediateWorkflow, workflow.RegisterOptions{
		Name: r.options.WorkflowType + "-immediate", DisableAlreadyRegisteredCheck: true,
	})
	run, err := r.options.Client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID: r.options.WorkflowID, TaskQueue: r.options.TaskQueue,
	}, r.options.WorkflowType, SDKWorkflowInput{
		FormatVersion: SDKProgramFormatVersion, Plan: plan,
		ActivityType: r.options.ActivityType, ChildType: r.options.ChildType,
		NexusEndpoint: r.options.NexusEndpoint, NexusService: r.options.NexusService,
		NexusOperation: r.options.NexusOperation,
	})
	if err != nil {
		return fmt.Errorf("start SDK participant workflow: %w", err)
	}
	r.plan = plan
	r.run = run
	r.started = true
	return nil
}

func missingSDKCapabilities(plan Plan, options SDKOptions) []string {
	supported := map[string]struct{}{
		"workflow": {}, "activity": {}, "update": {}, "signal": {}, "query": {}, "timer": {},
		"child-workflow": {}, "cancellation": {}, "retry": {}, "failure": {},
		"continuation": {}, "routing": {},
	}
	if options.Namespace != "" {
		supported["reset"] = struct{}{}
	}
	if options.WorkflowTaskFencer != nil {
		supported["ownership-fencing"] = struct{}{}
	}
	if options.CallbackDriver != nil {
		supported["callback"] = struct{}{}
	}
	if options.NexusDriver != nil ||
		options.NexusEndpoint != "" && options.NexusService != "" && options.NexusOperation != "" {
		supported["nexus"] = struct{}{}
	}
	var missing []string
	for _, capability := range plan.Capabilities {
		if _, exists := supported[capability]; !exists {
			missing = append(missing, capability)
		}
	}
	slices.Sort(missing)
	return slices.Compact(missing)
}

func (r *SDKRunner) Execute(ctx context.Context, operation Operation) (Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started || r.stopped {
		return Result{}, errors.New("SDK participant is not running")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if operation.Response == ResponseFailure {
		return r.qualifyResult(operation, Result{CommandID: operation.CommandID, Status: "failed"})
	}
	if operation.SDKOperation == SDKHandleNexus && r.options.NexusDriver != nil {
		receipt, handled, err := r.options.NexusDriver.ExecuteNexusAction(ctx, r.plan.ProgramID, operation)
		if err != nil {
			return Result{}, fmt.Errorf("execute dedicated Nexus action %q: %w", operation.CommandID, err)
		}
		if handled {
			if receipt.WorkflowID == "" || receipt.RunID == "" || receipt.Reference == "" || receipt.Source == "" {
				return Result{}, errors.New("dedicated Nexus driver returned incomplete evidence")
			}
			return r.qualifyResult(operation, Result{
				CommandID: operation.CommandID, Status: responseStatus(operation.Response),
				WorkflowID: receipt.WorkflowID, RunID: receipt.RunID,
				Lineage: append([]string(nil), receipt.Lineage...), Reference: receipt.Reference,
				Source: receipt.Source, SourceIdentity: receipt.Source,
				TerminalState: receipt.TerminalState, TerminalDisposition: receipt.TerminalDisposition,
			})
		}
	}
	var result Result
	var err error
	executedInProgram := false
	switch operation.SDKOperation {
	case SDKHandleSignal:
		err = r.options.Client.SignalWorkflow(ctx, r.run.GetID(), r.run.GetRunID(), SDKCommandSignalName, operation)
		if err == nil {
			result, err = r.waitForResult(ctx, operation.CommandID)
		}
	case SDKHandleQuery:
		result, err = r.queryResult(ctx, operation.CommandID)
		if err == nil && result.CommandID == "" {
			result = Result{CommandID: operation.CommandID, Status: "completed"}
		}
	case SDKContinueWorkflow:
		result, err = r.executeContinuation(ctx, operation)
	case SDKResetWorkflow:
		result, err = r.executeReset(ctx, operation)
	case SDKRouteWorkflowTask:
		result, err = r.executeRoutedWorkflow(ctx, operation)
	case SDKFenceWorkflowOwner:
		result, err = r.executeOwnershipFence(ctx, operation)
	case SDKRegisterCallback:
		result, err = r.executeCallback(ctx, operation, false)
	case SDKCompleteCallback:
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
		return Result{}, fmt.Errorf("execute SDK operation %q: %w", operation.CommandID, err)
	}
	qualified, err := r.qualifyResult(operation, result)
	if err != nil {
		return Result{}, err
	}
	if executedInProgram {
		r.programExecutions++
		if r.programExecutions == len(r.plan.Operations) {
			var results []Result
			if err := r.run.Get(ctx, &results); err != nil {
				return Result{}, fmt.Errorf("complete SDK participant program: %w", err)
			}
			r.workflowClosed = true
		}
	}
	return qualified, nil
}

func (r *SDKRunner) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started || r.stopped {
		return nil
	}
	var stopErr error
	if !r.workflowClosed {
		if err := r.options.Client.SignalWorkflow(ctx, r.run.GetID(), r.run.GetRunID(), SDKFinishSignalName, nil); err != nil {
			stopErr = r.terminateAfterStopFailure(err)
		} else {
			var results []Result
			if err := r.run.Get(ctx, &results); err != nil {
				stopErr = r.terminateAfterStopFailure(err)
			} else {
				r.workflowClosed = true
				r.stopped = true
			}
		}
	} else {
		r.stopped = true
	}
	if r.options.CallbackDriver != nil {
		if err := r.options.CallbackDriver.CleanupCompletionCallbacks(ctx); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("cleanup SDK completion callbacks: %w", err))
		}
	}
	if r.options.NexusDriver != nil {
		if err := r.options.NexusDriver.CleanupNexus(ctx); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("cleanup dedicated Nexus driver: %w", err))
		}
	}
	return stopErr
}

func (r *SDKRunner) queryResult(ctx context.Context, commandID string) (Result, error) {
	encoded, err := r.options.Client.QueryWorkflow(ctx, r.run.GetID(), r.run.GetRunID(), SDKStateQueryName)
	if err != nil {
		return Result{}, err
	}
	var state SDKWorkflowState
	if err := encoded.Get(&state); err != nil {
		return Result{}, fmt.Errorf("decode SDK participant state: %w", err)
	}
	return state.Results[commandID], nil
}

func (r *SDKRunner) waitForResult(ctx context.Context, commandID string) (Result, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		result, err := r.queryResult(ctx, commandID)
		if err != nil || result.CommandID != "" {
			return result, err
		}
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *SDKRunner) qualifyResult(operation Operation, result Result) (Result, error) {
	if result.CommandID != operation.CommandID || result.Status == "" {
		return Result{}, errors.New("SDK participant returned incomplete command result")
	}
	if result.Source == "" {
		switch operation.SemanticAction {
		case "create-speculative-workflow-task", "commit-speculative-workflow-task":
			result.Source = "temporal-sdk-speculative-update"
		case "dispatch-assurance-workflow-task", "progress-entity":
			result.Source = "temporal-sdk-workflow-progress"
		default:
			result.Source = "temporal-sdk-participant"
		}
	}
	if result.SourceIdentity == "" {
		result.SourceIdentity = r.options.WorkflowType
	}
	if result.WorkflowID == "" {
		result.WorkflowID = r.run.GetID()
	}
	if result.RunID == "" {
		result.RunID = r.run.GetRunID()
	}
	if result.Reference == "" {
		switch result.Source {
		case "temporal-sdk-speculative-update":
			result.Reference = result.WorkflowID + "/" + result.RunID +
				"/speculative-update/" + operation.CommandID
		case "temporal-sdk-workflow-progress":
			result.Reference = result.WorkflowID + "/" + result.RunID +
				"/workflow-progress/" + operation.CommandID
		default:
			result.Reference = result.WorkflowID + "/" + result.RunID + "/" + operation.CommandID
		}
	}
	lineage := []string{r.plan.ProgramID, result.WorkflowID}
	if len(result.Lineage) == 0 {
		lineage = append(lineage, result.RunID)
	} else {
		lineage = append(lineage, result.Lineage...)
	}
	result.Lineage = slices.Compact(lineage)
	encoded, err := json.Marshal(operation)
	if err != nil {
		return Result{}, fmt.Errorf("encode SDK operation receipt: %w", err)
	}
	digest := sha256.Sum256(encoded)
	result.PayloadDigest = "sha256:" + hex.EncodeToString(digest[:])
	return result, nil
}

func (r *SDKRunner) executeContinuation(ctx context.Context, operation Operation) (Result, error) {
	workflowID := r.mechanismWorkflowID(operation.CommandID, "continuation")
	run, err := r.options.Client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID: workflowID, TaskQueue: r.options.TaskQueue,
	}, r.options.WorkflowType+"-continuation", false)
	if err != nil {
		return Result{}, fmt.Errorf("start continuation workflow: %w", err)
	}
	firstRunID := run.GetRunID()
	if err := run.Get(ctx, nil); err != nil {
		return Result{}, fmt.Errorf("complete continuation workflow: %w", err)
	}
	description, err := r.options.Client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return Result{}, fmt.Errorf("describe continuation workflow: %w", err)
	}
	successorRunID := description.GetWorkflowExecutionInfo().GetExecution().GetRunId()
	if firstRunID == "" || successorRunID == "" || firstRunID == successorRunID {
		return Result{}, errors.New("continuation workflow did not produce distinct predecessor and successor runs")
	}
	return Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: workflowID, RunID: successorRunID, Lineage: []string{firstRunID, successorRunID},
		Reference: workflowID + "/" + successorRunID + "/continued-from/" + firstRunID,
		Source:    "temporal-sdk-continuation", SourceIdentity: r.options.WorkflowType + "-continuation",
	}, nil
}

func (r *SDKRunner) executeReset(ctx context.Context, operation Operation) (Result, error) {
	if r.options.Namespace == "" {
		return Result{}, errors.New("reset workflow requires a namespace")
	}
	workflowID := r.mechanismWorkflowID(operation.CommandID, "reset")
	run, err := r.options.Client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID: workflowID, TaskQueue: r.options.TaskQueue,
	}, r.options.WorkflowType+"-immediate")
	if err != nil {
		return Result{}, fmt.Errorf("start reset base workflow: %w", err)
	}
	baseRunID := run.GetRunID()
	if err := run.Get(ctx, nil); err != nil {
		return Result{}, fmt.Errorf("complete reset base workflow: %w", err)
	}
	response, err := r.options.Client.ResetWorkflowExecution(ctx, &workflowservice.ResetWorkflowExecutionRequest{
		Namespace:         r.options.Namespace,
		WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: baseRunID},
		Reason:            "Umpire3 reset lineage action", WorkflowTaskFinishEventId: 4,
		RequestId: uuid.NewSHA1(uuid.Nil, []byte(r.plan.ProgramID+"\x00"+operation.CommandID)).String(),
	})
	if err != nil {
		return Result{}, fmt.Errorf("reset workflow execution: %w", err)
	}
	resetRunID := response.GetRunId()
	if resetRunID == "" || resetRunID == baseRunID {
		return Result{}, errors.New("reset workflow did not produce a distinct successor run")
	}
	if err := r.options.Client.GetWorkflow(ctx, workflowID, resetRunID).Get(ctx, nil); err != nil {
		return Result{}, fmt.Errorf("complete reset successor workflow: %w", err)
	}
	return Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: workflowID, RunID: resetRunID, Lineage: []string{baseRunID, resetRunID},
		Reference: workflowID + "/" + resetRunID + "/reset-from/" + baseRunID,
		Source:    "temporal-sdk-reset", SourceIdentity: r.options.WorkflowType + "-immediate",
	}, nil
}

func (r *SDKRunner) executeRoutedWorkflow(ctx context.Context, operation Operation) (Result, error) {
	workflowID := r.mechanismWorkflowID(operation.CommandID, "routing")
	run, err := r.options.Client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID: workflowID, TaskQueue: r.options.TaskQueue,
	}, r.options.WorkflowType+"-immediate")
	if err != nil {
		return Result{}, fmt.Errorf("start routed workflow: %w", err)
	}
	if err := run.Get(ctx, nil); err != nil {
		return Result{}, fmt.Errorf("complete routed workflow: %w", err)
	}
	return Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: workflowID, RunID: run.GetRunID(), Lineage: []string{run.GetRunID()},
		Reference: workflowID + "/" + run.GetRunID() + "/task-queue/" + r.options.TaskQueue,
		Source:    "temporal-sdk-routing", SourceIdentity: r.options.TaskQueue,
	}, nil
}

func (r *SDKRunner) executeOwnershipFence(ctx context.Context, operation Operation) (Result, error) {
	if r.options.WorkflowTaskFencer == nil {
		return Result{}, errors.New("workflow task ownership fencing is unavailable")
	}
	receipt, err := r.options.WorkflowTaskFencer.FenceWorkflowOwner(ctx, r.plan.ProgramID, operation.CommandID)
	if err != nil {
		return Result{}, err
	}
	if receipt.WorkflowID == "" || receipt.RunID == "" || receipt.Reference == "" || receipt.Source == "" {
		return Result{}, errors.New("workflow task ownership fencer returned incomplete evidence")
	}
	return Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: receipt.WorkflowID, RunID: receipt.RunID,
		Lineage: append([]string(nil), receipt.Lineage...), Reference: receipt.Reference,
		Source: receipt.Source, SourceIdentity: receipt.Source,
	}, nil
}

func (r *SDKRunner) executeCallback(ctx context.Context, operation Operation, complete bool) (Result, error) {
	if r.options.CallbackDriver == nil {
		return Result{}, errors.New("completion callback driver is unavailable")
	}
	var receipt MechanismReceipt
	var err error
	if complete {
		receipt, err = r.options.CallbackDriver.CompleteCompletionCallback(ctx, r.plan.ProgramID, operation.CommandID)
	} else {
		receipt, err = r.options.CallbackDriver.RegisterCompletionCallback(ctx, r.plan.ProgramID, operation.CommandID)
	}
	if err != nil {
		return Result{}, err
	}
	if receipt.WorkflowID == "" || receipt.RunID == "" || receipt.Reference == "" || receipt.Source == "" {
		return Result{}, errors.New("completion callback driver returned incomplete evidence")
	}
	return Result{
		CommandID: operation.CommandID, Status: responseStatus(operation.Response),
		WorkflowID: receipt.WorkflowID, RunID: receipt.RunID,
		Lineage: append([]string(nil), receipt.Lineage...), Reference: receipt.Reference,
		Source: receipt.Source, SourceIdentity: receipt.Source,
	}, nil
}

func (r *SDKRunner) mechanismWorkflowID(commandID, mechanism string) string {
	return safeSDKName(r.options.WorkflowID + "-" + mechanism + "-" + commandID)
}

func (r *SDKRunner) terminateAfterStopFailure(stopErr error) error {
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

func SDKProgramWorkflow(ctx workflow.Context, input SDKWorkflowInput) ([]Result, error) {
	if input.FormatVersion != SDKProgramFormatVersion || input.Plan.FormatVersion != FormatVersion ||
		input.ActivityType == "" || input.ChildType == "" {
		return nil, errors.New("complete SDK participant workflow input is required")
	}
	results := make(map[string]Result, len(input.Plan.Operations))
	semanticState := sdkSemanticState{}
	completionRequested := false
	completionChannel := workflow.NewBufferedChannel(ctx, 1)
	if err := workflow.SetQueryHandler(ctx, SDKStateQueryName, func() (SDKWorkflowState, error) {
		return SDKWorkflowState{FormatVersion: SDKProgramFormatVersion, Results: cloneResults(results)}, nil
	}); err != nil {
		return nil, fmt.Errorf("register SDK participant query: %w", err)
	}
	execute := func(updateCtx workflow.Context, operation Operation) (Result, error) {
		if _, duplicate := results[operation.CommandID]; duplicate {
			return Result{}, fmt.Errorf("participant command %q already executed", operation.CommandID)
		}
		result, err := executeSDKOperation(updateCtx, input, operation, &semanticState)
		if err != nil {
			return Result{}, err
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
			var operation Operation
			channel.Receive(ctx, &operation)
			result, err := execute(ctx, operation)
			if err != nil {
				results[operation.CommandID] = Result{CommandID: operation.CommandID, Status: "failed"}
				return
			}
			results[operation.CommandID] = result
		})
		selector.Select(ctx)
		if finished {
			ordered := make([]Result, 0, len(results))
			for _, operation := range input.Plan.Operations {
				if result, exists := results[operation.CommandID]; exists {
					ordered = append(ordered, result)
				}
			}
			return ordered, nil
		}
	}
}

type sdkSemanticState struct {
	cancellationCommitted bool
}

func executeSDKOperation(
	ctx workflow.Context,
	input SDKWorkflowInput,
	operation Operation,
	semanticState *sdkSemanticState,
) (Result, error) {
	result := Result{CommandID: operation.CommandID, Status: responseStatus(operation.Response)}
	if operation.Response == ResponseFailure || operation.SDKOperation == SDKReturnFailure {
		return Result{CommandID: operation.CommandID, Status: "failed"}, nil
	}
	if operation.Response == ResponseBlocking {
		if operation.MaxBlockNanos <= 0 {
			return Result{}, errors.New("blocking SDK operation requires a positive bound")
		}
		if err := workflow.Sleep(ctx, time.Duration(operation.MaxBlockNanos)); err != nil {
			return Result{}, err
		}
	}
	if semanticState.cancellationCommitted &&
		(operation.SemanticAction == "worker-returns-success" || operation.SemanticAction == "persist-success") {
		result.Status = "suppressed"
		return result, nil
	}
	switch operation.SDKOperation {
	case SDKExecuteActivity, SDKRetry:
		activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 2},
		})
		future := workflow.ExecuteActivity(activityCtx, input.ActivityType, operation)
		if detachedResponse(operation.Response) {
			return result, nil
		}
		if err := future.Get(ctx, nil); err != nil {
			return Result{}, err
		}
	case SDKExecuteWorkflow, SDKExecuteChild:
		future := workflow.ExecuteChildWorkflow(ctx, input.ChildType, operation)
		if detachedResponse(operation.Response) {
			return result, nil
		}
		if err := future.Get(ctx, nil); err != nil {
			return Result{}, err
		}
	case SDKCancel:
		if operation.SemanticAction == "request-cancellation" && input.NexusEndpoint != "" &&
			input.NexusService != "" && input.NexusOperation != "" {
			nexusCtx, cancelNexus := workflow.WithCancel(ctx)
			client := workflow.NewNexusClient(input.NexusEndpoint, input.NexusService)
			future := client.ExecuteOperation(nexusCtx, input.NexusOperation, operation, workflow.NexusOperationOptions{})
			var execution workflow.NexusOperationExecution
			if err := future.GetNexusOperationExecution().Get(ctx, &execution); err != nil {
				cancelNexus()
				return Result{}, err
			}
			cancelNexus()
			if execution.OperationToken != "" {
				if err := workflow.Sleep(ctx, time.Millisecond); err != nil {
					return Result{}, err
				}
			}
		}
		childCtx, cancel := workflow.WithCancel(ctx)
		future := workflow.ExecuteChildWorkflow(childCtx, input.ChildType, operation)
		cancel()
		if err := future.Get(ctx, nil); err != nil && !temporal.IsCanceledError(err) {
			return Result{}, err
		}
		if operation.SemanticAction == "commit-cancellation" {
			semanticState.cancellationCommitted = true
		}
	case SDKStartTimer:
		future := workflow.NewTimer(ctx, time.Millisecond)
		if detachedResponse(operation.Response) {
			return result, nil
		}
		if err := future.Get(ctx, nil); err != nil {
			return Result{}, err
		}
	case SDKHandleNexus:
		if input.NexusEndpoint == "" || input.NexusService == "" || input.NexusOperation == "" {
			return Result{}, errors.New("nexus command requires endpoint, service, and operation")
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
			result.TerminalDisposition = TerminalDispositionFailure
			return result, nil
		}
		if operation.SemanticAction == "close-nexus-operation" && err != nil {
			result.Status = "failed"
			result.TerminalState = "failed"
			result.TerminalDisposition = TerminalDispositionFailure
			return result, nil
		}
		if err != nil {
			return Result{}, err
		}
		if operation.SemanticAction == "close-nexus-operation" {
			result.TerminalState = "succeeded"
			result.TerminalDisposition = TerminalDispositionSuccess
		}
		if operation.SemanticAction == "link-nexus-activity" {
			activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute,
			})
			if err := workflow.ExecuteActivity(activityCtx, input.ActivityType, operation).Get(ctx, nil); err != nil {
				return Result{}, err
			}
		}
	case SDKHandleUpdate, SDKHandleSignal, SDKHandleQuery:
	default:
		return Result{}, fmt.Errorf("unsupported SDK operation %q", operation.SDKOperation)
	}
	return result, nil
}

func SDKActivity(ctx context.Context, operation Operation) error {
	if operation.SDKOperation == SDKRetry && activity.GetInfo(ctx).Attempt == 1 {
		return temporal.NewApplicationError("umpire3 retry probe", "umpire3-retryable")
	}
	return nil
}

func SDKChildWorkflow(ctx workflow.Context, operation Operation) error {
	if operation.SDKOperation == SDKCancel {
		return workflow.Await(ctx, func() bool { return false })
	}
	return nil
}

func SDKContinueAsNewWorkflow(ctx workflow.Context, continued bool) error {
	if continued {
		return nil
	}
	return workflow.NewContinueAsNewError(ctx, SDKContinueAsNewWorkflow, true)
}

func SDKImmediateWorkflow(workflow.Context) error { return nil }

func responseStatus(mode ResponseMode) string {
	switch mode {
	case ResponseAsynchronous:
		return "accepted"
	case ResponseDeferred:
		return "deferred"
	default:
		return "completed"
	}
}

func detachedResponse(mode ResponseMode) bool {
	return mode == ResponseAsynchronous || mode == ResponseDeferred
}

func cloneResults(source map[string]Result) map[string]Result {
	result := make(map[string]Result, len(source))
	for identifier, value := range source {
		value.Lineage = append([]string(nil), value.Lineage...)
		result[identifier] = value
	}
	return result
}

func updateID(programID, commandID string) string {
	return "umpire3-" + safeSDKName(programID+"-"+commandID)
}

func safeSDKName(value string) string {
	var builder strings.Builder
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || character == '-' {
			builder.WriteRune(character)
		} else {
			builder.WriteByte('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "participant"
	}
	if len(result) > 120 {
		digest := sha256.Sum256([]byte(result))
		return result[:80] + "-" + hex.EncodeToString(digest[:8])
	}
	return result
}
