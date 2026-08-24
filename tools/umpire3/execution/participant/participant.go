package participant

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexecution "go.temporal.io/server/tools/umpire3/protocol/execution"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

const FormatVersion = "umpire3/participant-program/v1"

type CommandKind string

const (
	CommandWorkflow         CommandKind = "workflow"
	CommandActivity         CommandKind = "activity"
	CommandNexus            CommandKind = "nexus"
	CommandUpdate           CommandKind = "update"
	CommandSignal           CommandKind = "signal"
	CommandQuery            CommandKind = "query"
	CommandTimer            CommandKind = "timer"
	CommandChild            CommandKind = "child-workflow"
	CommandCancellation     CommandKind = "cancellation"
	CommandRetry            CommandKind = "retry"
	CommandFailure          CommandKind = "failure"
	CommandContinuation     CommandKind = "continuation"
	CommandReset            CommandKind = "reset"
	CommandRouting          CommandKind = "routing"
	CommandOwnership        CommandKind = "ownership-fencing"
	CommandCallbackRegister CommandKind = "callback-register"
	CommandCallbackComplete CommandKind = "callback-complete"
)

type ResponseMode = protocolexperiment.ResponseMode
type TerminalDisposition = protocolexecution.TerminalDisposition

const (
	ResponseSynchronous         = protocolexperiment.ResponseSynchronous
	ResponseAsynchronous        = protocolexperiment.ResponseAsynchronous
	ResponseDeferred            = protocolexperiment.ResponseDeferred
	ResponseBlocking            = protocolexperiment.ResponseBlocking
	ResponseFailure             = protocolexperiment.ResponseFailure
	TerminalDispositionSuccess  = protocolexecution.TerminalDispositionSuccess
	TerminalDispositionFailure  = protocolexecution.TerminalDispositionFailure
	TerminalDispositionUntagged = protocolexecution.TerminalDispositionUntagged
)

type Command struct {
	Identifier     string                          `json:"identifier"`
	Kind           CommandKind                     `json:"kind"`
	SemanticAction string                          `json:"semanticAction,omitempty"`
	Response       ResponseMode                    `json:"response"`
	Arguments      []protocolexperiment.NamedValue `json:"arguments"`
	MaxBlockNanos  int64                           `json:"maxBlockNanos,omitempty"`
}

type Program struct {
	FormatVersion string    `json:"formatVersion"`
	Identifier    string    `json:"identifier"`
	Commands      []Command `json:"commands"`
}

type SDKOperation string

const (
	SDKExecuteWorkflow    SDKOperation = "sdk.execute-workflow"
	SDKExecuteActivity    SDKOperation = "sdk.execute-activity"
	SDKHandleNexus        SDKOperation = "sdk.handle-nexus"
	SDKHandleUpdate       SDKOperation = "sdk.handle-update"
	SDKHandleSignal       SDKOperation = "sdk.handle-signal"
	SDKHandleQuery        SDKOperation = "sdk.handle-query"
	SDKStartTimer         SDKOperation = "sdk.start-timer"
	SDKExecuteChild       SDKOperation = "sdk.execute-child-workflow"
	SDKCancel             SDKOperation = "sdk.cancel"
	SDKRetry              SDKOperation = "sdk.retry"
	SDKReturnFailure      SDKOperation = "sdk.return-failure"
	SDKContinueWorkflow   SDKOperation = "sdk.continue-workflow"
	SDKResetWorkflow      SDKOperation = "sdk.reset-workflow"
	SDKRouteWorkflowTask  SDKOperation = "sdk.route-workflow-task"
	SDKFenceWorkflowOwner SDKOperation = "sdk.fence-workflow-owner"
	SDKRegisterCallback   SDKOperation = "sdk.register-callback"
	SDKCompleteCallback   SDKOperation = "sdk.complete-callback"
)

type Operation struct {
	CommandID      string                          `json:"commandID"`
	SemanticAction string                          `json:"semanticAction,omitempty"`
	SDKOperation   SDKOperation                    `json:"sdkOperation"`
	Response       ResponseMode                    `json:"response"`
	Arguments      []protocolexperiment.NamedValue `json:"arguments"`
	MaxBlockNanos  int64                           `json:"maxBlockNanos,omitempty"`
}

type Plan struct {
	FormatVersion string      `json:"formatVersion"`
	ProgramID     string      `json:"programID"`
	Capabilities  []string    `json:"capabilities"`
	Operations    []Operation `json:"operations"`
}

type Result struct {
	CommandID           string              `json:"commandID"`
	Status              string              `json:"status"`
	Reference           string              `json:"reference,omitempty"`
	Source              string              `json:"source,omitempty"`
	SourceIdentity      string              `json:"sourceIdentity,omitempty"`
	WorkflowID          string              `json:"workflowID,omitempty"`
	RunID               string              `json:"runID,omitempty"`
	Lineage             []string            `json:"lineage,omitempty"`
	PayloadDigest       string              `json:"payloadDigest,omitempty"`
	TerminalState       string              `json:"terminalState,omitempty"`
	TerminalDisposition TerminalDisposition `json:"terminalDisposition,omitempty"`
}

type Runner interface {
	Start(context.Context, Plan) error
	Execute(context.Context, Operation) (Result, error)
	Stop(context.Context) error
}

type Session struct {
	mu         sync.Mutex
	runner     Runner
	operations map[string]Operation
	cleaned    bool
}

func AllCommandKinds() []CommandKind {
	return []CommandKind{
		CommandWorkflow, CommandActivity, CommandNexus, CommandUpdate, CommandSignal, CommandQuery,
		CommandTimer, CommandChild, CommandCancellation, CommandRetry, CommandFailure,
		CommandContinuation, CommandReset, CommandRouting, CommandOwnership,
		CommandCallbackRegister, CommandCallbackComplete,
	}
}

func Compile(program Program) (Plan, error) {
	if program.FormatVersion != FormatVersion || program.Identifier == "" || len(program.Commands) == 0 {
		return Plan{}, errors.New("complete versioned participant program is required")
	}
	identifiers := make(map[string]struct{}, len(program.Commands))
	capabilities := make(map[string]struct{})
	operations := make([]Operation, len(program.Commands))
	for index, command := range program.Commands {
		if command.Identifier == "" {
			return Plan{}, errors.New("participant command identifier is required")
		}
		if _, duplicate := identifiers[command.Identifier]; duplicate {
			return Plan{}, fmt.Errorf("duplicate participant command %q", command.Identifier)
		}
		identifiers[command.Identifier] = struct{}{}
		operation, capability, err := compileCommand(command)
		if err != nil {
			return Plan{}, fmt.Errorf("command %q: %w", command.Identifier, err)
		}
		operations[index] = operation
		capabilities[capability] = struct{}{}
	}
	capabilityValues := make([]string, 0, len(capabilities))
	for capability := range capabilities {
		capabilityValues = append(capabilityValues, capability)
	}
	slices.Sort(capabilityValues)
	return Plan{
		FormatVersion: FormatVersion, ProgramID: program.Identifier,
		Capabilities: capabilityValues, Operations: operations,
	}, nil
}

func CompileExperiment(experiment protocolexperiment.Experiment) (Program, Plan, error) {
	if err := experiment.Validate(); err != nil {
		return Program{}, Plan{}, fmt.Errorf("validate experiment: %w", err)
	}
	commands := make([]Command, len(experiment.Actions))
	for index, action := range experiment.Actions {
		kind, known := actionCommandKind(protocolcatalog.ActionKind(action.Kind))
		if !known {
			return Program{}, Plan{}, fmt.Errorf("action %q has no participant command mapping", action.Kind)
		}
		commands[index] = Command{
			Identifier: action.Identifier, Kind: kind, SemanticAction: action.Kind, Response: action.EffectiveResponseMode(),
			Arguments:     append([]protocolexperiment.NamedValue(nil), action.Arguments...),
			MaxBlockNanos: action.MaxBlockNanos,
		}
	}
	program := Program{FormatVersion: FormatVersion, Identifier: experiment.ExperimentID, Commands: commands}
	plan, err := Compile(program)
	if err != nil {
		return Program{}, Plan{}, err
	}
	return program, plan, nil
}

func ValidateActionMappings() error {
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return err
	}
	for _, action := range catalog.Actions {
		if _, known := actionCommandKind(protocolcatalog.ActionKind(action.Identifier)); !known {
			return fmt.Errorf("catalog action %q has no participant command mapping", action.Identifier)
		}
	}
	return nil
}

func actionCommandKind(kind protocolcatalog.ActionKind) (CommandKind, bool) {
	switch kind {
	case protocolcatalog.ActionKindScheduleOperation, protocolcatalog.ActionKindDispatchTask,
		protocolcatalog.ActionKindWorkerReturnsSuccess, protocolcatalog.ActionKindPersistSuccess,
		protocolcatalog.ActionKindAckTask, protocolcatalog.ActionKindCloseNexusOperation,
		protocolcatalog.ActionKindLinkNexusActivity, protocolcatalog.ActionKindTimeoutNexusOperation:
		return CommandNexus, true
	case protocolcatalog.ActionKindRequestCancellation, protocolcatalog.ActionKindCommitCancellation:
		return CommandCancellation, true
	case protocolcatalog.ActionKindRetryTask, protocolcatalog.ActionKindAcquireOwnership,
		protocolcatalog.ActionKindRecoverOwner:
		return CommandRetry, true
	case protocolcatalog.ActionKindCrashOwner:
		return CommandFailure, true
	case protocolcatalog.ActionKindStartUpdate, protocolcatalog.ActionKindAcceptUpdate,
		protocolcatalog.ActionKindCompleteUpdate:
		return CommandUpdate, true
	case protocolcatalog.ActionKindRecordUpdateHistory, protocolcatalog.ActionKindDispatchWorkflowTask,
		protocolcatalog.ActionKindCompleteWorkflowTask, protocolcatalog.ActionKindEnqueueWorkflowTask,
		protocolcatalog.ActionKindDeliverWorkflowTask, protocolcatalog.ActionKindAcknowledgeWorkflowTask,
		protocolcatalog.ActionKindCreateSpeculativeWorkflowTask, protocolcatalog.ActionKindCommitSpeculativeWorkflowTask,
		protocolcatalog.ActionKindDispatchAssuranceWorkflowTask, protocolcatalog.ActionKindProgressEntity:
		return CommandWorkflow, true
	case protocolcatalog.ActionKindRegisterCallback:
		return CommandCallbackRegister, true
	case protocolcatalog.ActionKindRecordCallbackResponse:
		return CommandCallbackComplete, true
	case protocolcatalog.ActionKindContinueWorkflow:
		return CommandContinuation, true
	case protocolcatalog.ActionKindResetWorkflow:
		return CommandReset, true
	case protocolcatalog.ActionKindRouteWorkflowTask:
		return CommandRouting, true
	case protocolcatalog.ActionKindFenceWorkflowOwner:
		return CommandOwnership, true
	default:
		return "", false
	}
}

func compileCommand(command Command) (Operation, string, error) {
	operation, capability, known := commandMapping(command.Kind)
	if !known {
		return Operation{}, "", fmt.Errorf("unknown command kind %q", command.Kind)
	}
	switch command.Response {
	case ResponseSynchronous, ResponseAsynchronous, ResponseDeferred, ResponseFailure:
		if command.MaxBlockNanos != 0 {
			return Operation{}, "", errors.New("only blocking responses accept maxBlockNanos")
		}
	case ResponseBlocking:
		if command.MaxBlockNanos <= 0 {
			return Operation{}, "", errors.New("blocking response requires a positive maxBlockNanos")
		}
	default:
		return Operation{}, "", fmt.Errorf("unknown response mode %q", command.Response)
	}
	for _, argument := range command.Arguments {
		if argument.Name == "" || sensitiveName(argument.Name) {
			return Operation{}, "", fmt.Errorf("unsafe argument name %q", argument.Name)
		}
	}
	return Operation{
		CommandID: command.Identifier, SemanticAction: command.SemanticAction,
		SDKOperation: operation, Response: command.Response,
		Arguments: append([]protocolexperiment.NamedValue(nil), command.Arguments...), MaxBlockNanos: command.MaxBlockNanos,
	}, capability, nil
}

func commandMapping(kind CommandKind) (SDKOperation, string, bool) {
	switch kind {
	case CommandWorkflow:
		return SDKExecuteWorkflow, "workflow", true
	case CommandActivity:
		return SDKExecuteActivity, "activity", true
	case CommandNexus:
		return SDKHandleNexus, "nexus", true
	case CommandUpdate:
		return SDKHandleUpdate, "update", true
	case CommandSignal:
		return SDKHandleSignal, "signal", true
	case CommandQuery:
		return SDKHandleQuery, "query", true
	case CommandTimer:
		return SDKStartTimer, "timer", true
	case CommandChild:
		return SDKExecuteChild, "child-workflow", true
	case CommandCancellation:
		return SDKCancel, "cancellation", true
	case CommandRetry:
		return SDKRetry, "retry", true
	case CommandFailure:
		return SDKReturnFailure, "failure", true
	case CommandContinuation:
		return SDKContinueWorkflow, "continuation", true
	case CommandReset:
		return SDKResetWorkflow, "reset", true
	case CommandRouting:
		return SDKRouteWorkflowTask, "routing", true
	case CommandOwnership:
		return SDKFenceWorkflowOwner, "ownership-fencing", true
	case CommandCallbackRegister:
		return SDKRegisterCallback, "callback", true
	case CommandCallbackComplete:
		return SDKCompleteCallback, "callback", true
	default:
		return "", "", false
	}
}

func Start(ctx context.Context, program Program, runner Runner) (*Session, error) {
	if runner == nil {
		return nil, errors.New("participant runner is required")
	}
	plan, err := Compile(program)
	if err != nil {
		return nil, fmt.Errorf("compile participant program: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := runner.Start(ctx, plan); err != nil {
		return nil, fmt.Errorf("start participant runner: %w", err)
	}
	operations := make(map[string]Operation, len(plan.Operations))
	for _, operation := range plan.Operations {
		operations[operation.CommandID] = operation
	}
	return &Session{runner: runner, operations: operations}, nil
}

func (s *Session) Execute(ctx context.Context, commandID string) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cleaned {
		return Result{}, errors.New("participant session is cleaned")
	}
	operation, exists := s.operations[commandID]
	if !exists {
		return Result{}, fmt.Errorf("unknown participant command %q", commandID)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	result, err := s.runner.Execute(ctx, operation)
	if err != nil {
		return Result{}, fmt.Errorf("execute participant command %q: %w", commandID, err)
	}
	if result.CommandID != commandID || result.Status == "" {
		return Result{}, errors.New("participant runner returned incomplete result")
	}
	return result, nil
}

func (s *Session) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cleaned {
		return nil
	}
	if err := s.runner.Stop(ctx); err != nil {
		return fmt.Errorf("stop participant runner: %w", err)
	}
	s.cleaned = true
	return nil
}

func sensitiveName(name string) bool {
	normalized := strings.ToLower(name)
	for _, fragment := range []string{"authorization", "credential", "header", "password", "payload", "secret", "token"} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
