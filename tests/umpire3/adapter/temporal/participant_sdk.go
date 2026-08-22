package temporal

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"go.temporal.io/sdk/activity"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/tests/umpire3/execution/participant"
)

const (
	SDKProgramFormatVersion = "umpire3/sdk-program/v1"
	SDKCommandUpdateName    = "umpire3-command"
	SDKCommandSignalName    = "umpire3-signal"
	SDKStateQueryName       = "umpire3-state"
	SDKFinishSignalName     = "umpire3-finish"
)

type SDKParticipantOptions struct {
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
	TerminalDisposition participant.TerminalDisposition
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
	ExecuteNexusAction(context.Context, string, participant.Operation) (MechanismReceipt, bool, error)
	CleanupNexus(context.Context) error
}

type SDKWorkflowInput struct {
	FormatVersion  string           `json:"formatVersion"`
	Plan           participant.Plan `json:"plan"`
	ActivityType   string           `json:"activityType"`
	ChildType      string           `json:"childType"`
	NexusEndpoint  string           `json:"nexusEndpoint,omitempty"`
	NexusService   string           `json:"nexusService,omitempty"`
	NexusOperation string           `json:"nexusOperation,omitempty"`
}

type SDKWorkflowState struct {
	FormatVersion string                        `json:"formatVersion"`
	Results       map[string]participant.Result `json:"results"`
}

type SDKParticipantAdapter struct {
	mu sync.Mutex

	options           SDKParticipantOptions
	plan              participant.Plan
	run               sdkclient.WorkflowRun
	started           bool
	stopped           bool
	programExecutions int
	workflowClosed    bool
}

func NewSDKParticipantAdapter(options SDKParticipantOptions) (*SDKParticipantAdapter, error) {
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
	return &SDKParticipantAdapter{options: options}, nil
}

func (r *SDKParticipantAdapter) Start(ctx context.Context, plan participant.Plan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return errors.New("SDK participant is already started")
	}
	if plan.FormatVersion != participant.FormatVersion || plan.ProgramID == "" || len(plan.Operations) == 0 {
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

func missingSDKCapabilities(plan participant.Plan, options SDKParticipantOptions) []string {
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
