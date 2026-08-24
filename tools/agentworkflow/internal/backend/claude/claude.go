package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.temporal.io/server/tools/agentworkflow"
	backendconfig "go.temporal.io/server/tools/agentworkflow/internal/backend/configuration"
	"go.temporal.io/server/tools/agentworkflow/internal/process"
)

type Config struct {
	Command   []string
	Model     string
	MaxTurns  int
	Qualified bool
}

type backend struct {
	config Config
}

func New(config Config) (agentworkflow.Backend, error) {
	if len(config.Command) == 0 || strings.TrimSpace(config.Command[0]) == "" {
		return nil, errors.New("Claude command is required")
	}
	if config.MaxTurns < 0 {
		return nil, errors.New("Claude max turns cannot be negative")
	}
	config.Command = append([]string(nil), config.Command...)
	return &backend{config: config}, nil
}

func (backend *backend) Describe(ctx context.Context) (agentworkflow.BackendInfo, error) {
	directory, err := os.Getwd()
	if err != nil {
		return agentworkflow.BackendInfo{}, fmt.Errorf("resolve Claude probe directory: %w", err)
	}
	command := append([]string(nil), backend.config.Command...)
	command = append(command, "--version")
	result, err := (process.Runner{}).Run(ctx, process.Request{
		Command: command, Directory: directory, Environment: claudeEnvironment(), Timeout: 15 * time.Second, MaxOutputBytes: 64 << 10,
	})
	if err != nil {
		return agentworkflow.BackendInfo{}, fmt.Errorf("probe Claude: %w", err)
	}
	if result.ExitCode != 0 {
		return agentworkflow.BackendInfo{}, fmt.Errorf("probe Claude: exit %d: %s", result.ExitCode, boundedMessage(result.Stderr))
	}
	version := strings.TrimSpace(string(result.Stdout))
	if version == "" {
		return agentworkflow.BackendInfo{}, errors.New("probe Claude: empty version")
	}
	capabilities := []agentworkflow.Capability{
		agentworkflow.CapabilityReadOnly,
		agentworkflow.CapabilityWorkspaceWrite,
		agentworkflow.CapabilityStructuredOutput,
		agentworkflow.CapabilityResume,
		agentworkflow.CapabilityCancellation,
	}
	if backend.config.Qualified {
		capabilities = append(capabilities, agentworkflow.CapabilityIsolatedConfig)
	}
	configurationDigest, err := backendconfig.Digest(backend.config.Command, struct {
		Model     string `json:"model,omitempty"`
		MaxTurns  int    `json:"max_turns,omitempty"`
		Qualified bool   `json:"qualified"`
	}{Model: backend.config.Model, MaxTurns: backend.config.MaxTurns, Qualified: backend.config.Qualified})
	if err != nil {
		return agentworkflow.BackendInfo{}, err
	}
	return agentworkflow.BackendInfo{Name: "claude", Version: version, ConfigurationDigest: configurationDigest, Capabilities: capabilities}, nil
}

func (backend *backend) Execute(ctx context.Context, invocation agentworkflow.Invocation, sink agentworkflow.EventSink) (agentworkflow.InvocationResult, error) {
	if err := validateInvocation(invocation, sink); err != nil {
		return agentworkflow.InvocationResult{}, err
	}
	command, err := backend.command(invocation)
	if err != nil {
		return agentworkflow.InvocationResult{}, err
	}
	if err := sink.Emit(agentworkflow.Event{Kind: agentworkflow.EventInvocationStarted}); err != nil {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: retain Claude start event: %v", agentworkflow.ErrCapacity, err)
	}
	result, processErr := (process.Runner{}).Run(ctx, process.Request{
		Command: command, Directory: invocation.Workspace, Environment: claudeEnvironment(), Stdin: invocation.Prompt, Timeout: invocation.Timeout,
		MaxOutputBytes: invocation.MaxOutputBytes,
	})
	if err := emitDiagnostics(sink, result.Stderr); err != nil {
		return agentworkflow.InvocationResult{}, err
	}
	if processErr != nil {
		return agentworkflow.InvocationResult{}, classifyProcessError(processErr)
	}
	if result.ExitCode != 0 {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Claude exit %d: %s", agentworkflow.ErrAgent, result.ExitCode, boundedMessage(result.Stderr))
	}
	return decodeResult(result.Stdout, invocation.MaxOutputBytes, invocation.MaxEvents, sink)
}

func (backend *backend) command(invocation agentworkflow.Invocation) ([]string, error) {
	command := append([]string(nil), backend.config.Command...)
	if backend.config.Qualified {
		command = append(command, "--bare")
	}
	command = append(command, "-p", "--output-format", "json", "--json-schema", string(invocation.OutputSchema))
	if model := backend.model(invocation); model != "" {
		command = append(command, "--model", model)
	}
	if backend.config.MaxTurns > 0 {
		command = append(command, "--max-turns", fmt.Sprintf("%d", backend.config.MaxTurns))
	}
	if invocation.Session != "" {
		command = append(command, "--resume", invocation.Session)
	} else if !invocation.RetainSession {
		command = append(command, "--no-session-persistence")
	}
	switch invocation.Permission {
	case agentworkflow.PermissionReadOnly:
		command = append(command, "--permission-mode", "dontAsk", "--allowedTools", "Read,Glob,Grep")
	case agentworkflow.PermissionWorkspaceWrite:
		command = append(command, "--permission-mode", "acceptEdits", "--allowedTools", "Bash,Read,Edit,Write,Glob,Grep")
	default:
		return nil, fmt.Errorf("Claude permission %q is invalid", invocation.Permission)
	}
	return command, nil
}

func (backend *backend) model(invocation agentworkflow.Invocation) string {
	if backend.config.Model != "" {
		return backend.config.Model
	}
	return invocation.Model
}

func claudeEnvironment() []string {
	return backendconfig.Environment("ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN", "CLAUDE_CONFIG_DIR")
}

func emitDiagnostics(sink agentworkflow.EventSink, stderr []byte) error {
	if len(stderr) == 0 {
		return nil
	}
	raw, _ := json.Marshal(map[string]string{"stderr": string(stderr)})
	if err := sink.Emit(agentworkflow.Event{Kind: agentworkflow.EventDiagnostic, Message: boundedMessage(stderr), Raw: raw}); err != nil {
		return fmt.Errorf("%w: retain Claude diagnostics: %v", agentworkflow.ErrCapacity, err)
	}
	return nil
}

func classifyProcessError(err error) error {
	switch {
	case errors.Is(err, process.ErrOutputLimit):
		return fmt.Errorf("%w: Claude output", agentworkflow.ErrCapacity)
	case errors.Is(err, process.ErrTimeout):
		return fmt.Errorf("Claude invocation: %w", context.DeadlineExceeded)
	case errors.Is(err, process.ErrCancelled):
		return fmt.Errorf("Claude invocation: %w", context.Canceled)
	default:
		return fmt.Errorf("%w: execute Claude: %v", agentworkflow.ErrAgent, err)
	}
}

type terminalPayload struct {
	Type             string          `json:"type"`
	Subtype          string          `json:"subtype"`
	IsError          bool            `json:"is_error"`
	Result           string          `json:"result"`
	SessionID        string          `json:"session_id"`
	StructuredOutput json.RawMessage `json:"structured_output"`
	Usage            struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
}

func decodeResult(stdout []byte, maxOutputBytes int64, maxEvents int, sink agentworkflow.EventSink) (agentworkflow.InvocationResult, error) {
	var payload terminalPayload
	if err := json.Unmarshal(stdout, &payload); err != nil {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: decode Claude result: %v", agentworkflow.ErrAgent, err)
	}
	if payload.Type != "result" || payload.IsError || (payload.Subtype != "" && payload.Subtype != "success") {
		message := payload.Result
		if message == "" {
			message = "invalid terminal result"
		}
		raw := append(json.RawMessage(nil), bytes.TrimSpace(stdout)...)
		terminalErr := fmt.Errorf("%w: Claude terminal failure: %s", agentworkflow.ErrAgent, message)
		if err := sink.Emit(agentworkflow.Event{Kind: agentworkflow.EventInvocationFailed, Message: message, Raw: raw}); err != nil {
			return agentworkflow.InvocationResult{}, errors.Join(terminalErr, fmt.Errorf("%w: retain Claude failure event: %v", agentworkflow.ErrCapacity, err))
		}
		return agentworkflow.InvocationResult{}, terminalErr
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Claude result has no session identity", agentworkflow.ErrAgent)
	}
	if !json.Valid(payload.StructuredOutput) {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Claude structured output is missing or invalid", agentworkflow.ErrAgent)
	}
	if int64(len(payload.StructuredOutput)) > maxOutputBytes {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Claude structured output", agentworkflow.ErrCapacity)
	}
	usage := agentworkflow.Usage{InputTokens: payload.Usage.InputTokens, OutputTokens: payload.Usage.OutputTokens}
	events := []agentworkflow.Event{
		{Kind: agentworkflow.EventSessionIdentified, Session: payload.SessionID},
		{Kind: agentworkflow.EventAgentMessage, Message: payload.Result},
		{Kind: agentworkflow.EventInvocationCompleted, Usage: usage, Raw: append(json.RawMessage(nil), bytes.TrimSpace(stdout)...)},
	}
	if 1+len(events) > maxEvents {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Claude event count", agentworkflow.ErrCapacity)
	}
	for _, event := range events {
		if err := sink.Emit(event); err != nil {
			return agentworkflow.InvocationResult{}, fmt.Errorf("%w: retain Claude event: %v", agentworkflow.ErrCapacity, err)
		}
	}
	return agentworkflow.InvocationResult{Session: payload.SessionID, Output: append(json.RawMessage(nil), payload.StructuredOutput...), Usage: usage}, nil
}

func validateInvocation(invocation agentworkflow.Invocation, sink agentworkflow.EventSink) error {
	if sink == nil {
		return errors.New("Claude event sink is required")
	}
	if strings.TrimSpace(invocation.ID) == "" || strings.TrimSpace(invocation.Phase) == "" || strings.TrimSpace(invocation.Prompt) == "" {
		return errors.New("Claude invocation identity, phase, and prompt are required")
	}
	if invocation.Permission != agentworkflow.PermissionReadOnly && invocation.Permission != agentworkflow.PermissionWorkspaceWrite {
		return fmt.Errorf("Claude permission %q is invalid", invocation.Permission)
	}
	if !json.Valid(invocation.OutputSchema) {
		return errors.New("Claude output schema is not valid JSON")
	}
	if invocation.Timeout <= 0 || invocation.MaxOutputBytes <= 0 || invocation.MaxEvents <= 0 {
		return errors.New("Claude invocation bounds must be positive")
	}
	info, err := os.Stat(invocation.Workspace)
	if err != nil || !info.IsDir() {
		return errors.Join(errors.New("Claude workspace is not a directory"), err)
	}
	return nil
}

func boundedMessage(data []byte) string {
	const limit = 4 << 10
	data = bytes.TrimSpace(data)
	if len(data) > limit {
		return string(data[:limit]) + "…"
	}
	return string(data)
}
