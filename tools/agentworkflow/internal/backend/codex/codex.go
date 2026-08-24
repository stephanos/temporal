package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/server/tools/agentworkflow"
	backendconfig "go.temporal.io/server/tools/agentworkflow/internal/backend/configuration"
	"go.temporal.io/server/tools/agentworkflow/internal/process"
)

type Config struct {
	Command   []string
	Model     string
	Qualified bool
}

type backend struct {
	config Config
}

func New(config Config) (agentworkflow.Backend, error) {
	if len(config.Command) == 0 || strings.TrimSpace(config.Command[0]) == "" {
		return nil, errors.New("codex command is required")
	}
	config.Command = append([]string(nil), config.Command...)
	return &backend{config: config}, nil
}

func (backend *backend) Describe(ctx context.Context) (agentworkflow.BackendInfo, error) {
	directory, err := os.Getwd()
	if err != nil {
		return agentworkflow.BackendInfo{}, fmt.Errorf("resolve Codex probe directory: %w", err)
	}
	command := append([]string(nil), backend.config.Command...)
	command = append(command, "--version")
	result, err := (process.Runner{}).Run(ctx, process.Request{
		Command: command, Directory: directory, Environment: codexEnvironment(), Timeout: 15 * time.Second, MaxOutputBytes: 64 << 10,
	})
	if err != nil {
		return agentworkflow.BackendInfo{}, fmt.Errorf("probe Codex: %w", err)
	}
	if result.ExitCode != 0 {
		return agentworkflow.BackendInfo{}, fmt.Errorf("probe Codex: exit %d: %s", result.ExitCode, boundedMessage(result.Stderr))
	}
	version := strings.TrimSpace(string(result.Stdout))
	if version == "" {
		return agentworkflow.BackendInfo{}, errors.New("probe Codex: empty version")
	}
	capabilities := []agentworkflow.Capability{
		agentworkflow.CapabilityReadOnly,
		agentworkflow.CapabilityWorkspaceWrite,
		agentworkflow.CapabilityStructuredOutput,
		agentworkflow.CapabilityMachineEvents,
		agentworkflow.CapabilityResume,
		agentworkflow.CapabilityCancellation,
	}
	if backend.config.Qualified {
		capabilities = append(capabilities, agentworkflow.CapabilityIsolatedConfig)
	}
	configurationDigest, err := backendconfig.Digest(backend.config.Command, struct {
		Model     string `json:"model,omitempty"`
		Qualified bool   `json:"qualified"`
	}{Model: backend.config.Model, Qualified: backend.config.Qualified})
	if err != nil {
		return agentworkflow.BackendInfo{}, err
	}
	return agentworkflow.BackendInfo{Name: "codex", Version: version, ConfigurationDigest: configurationDigest, Capabilities: capabilities}, nil
}

func (backend *backend) Execute(ctx context.Context, invocation agentworkflow.Invocation, sink agentworkflow.EventSink) (_ agentworkflow.InvocationResult, returnedErr error) {
	if err := validateInvocation(invocation, sink); err != nil {
		return agentworkflow.InvocationResult{}, err
	}
	temporary, err := os.MkdirTemp("", "agentworkflow-codex-*")
	if err != nil {
		return agentworkflow.InvocationResult{}, fmt.Errorf("create Codex invocation directory: %w", err)
	}
	defer func() { returnedErr = errors.Join(returnedErr, os.RemoveAll(temporary)) }()
	schemaPath := filepath.Join(temporary, "schema.json")
	if err := os.WriteFile(schemaPath, invocation.OutputSchema, 0o600); err != nil {
		return agentworkflow.InvocationResult{}, fmt.Errorf("write Codex output schema: %w", err)
	}
	outputPath := filepath.Join(temporary, "output.json")
	command := append([]string(nil), backend.config.Command...)
	if invocation.Session == "" {
		command = append(command, "exec")
		command = append(command, backend.commonArguments(invocation, schemaPath, outputPath)...)
		if !invocation.RetainSession {
			command = append(command, "--ephemeral")
		}
		command = append(command, "-")
	} else {
		command = append(command, "exec", "resume")
		command = append(command, backend.resumeArguments(schemaPath, outputPath)...)
		command = append(command, invocation.Session, "-")
	}
	result, processErr := (process.Runner{}).Run(ctx, process.Request{
		Command: command, Directory: invocation.Workspace, Stdin: invocation.Prompt,
		Environment: codexEnvironment(), Timeout: invocation.Timeout, MaxOutputBytes: invocation.MaxOutputBytes,
	})
	if len(result.Stderr) > 0 {
		raw, _ := json.Marshal(map[string]string{"stderr": string(result.Stderr)})
		if err := sink.Emit(agentworkflow.Event{Kind: agentworkflow.EventDiagnostic, Message: boundedMessage(result.Stderr), Raw: raw}); err != nil {
			return agentworkflow.InvocationResult{}, fmt.Errorf("%w: retain Codex diagnostics: %v", agentworkflow.ErrCapacity, err)
		}
	}
	parsed, parseErr := parseEvents(result.Stdout, invocation.MaxEvents, sink)
	if processErr != nil {
		if len(result.Stdout) > 0 {
			raw, _ := json.Marshal(map[string]string{"partial_stdout": string(result.Stdout)})
			_ = sink.Emit(agentworkflow.Event{Kind: agentworkflow.EventDiagnostic, Message: "Codex stdout ended before successful completion", Raw: raw})
		}
		switch {
		case errors.Is(processErr, process.ErrOutputLimit):
			return agentworkflow.InvocationResult{}, errors.Join(fmt.Errorf("%w: Codex output", agentworkflow.ErrCapacity), parseErr)
		case errors.Is(processErr, process.ErrTimeout):
			return agentworkflow.InvocationResult{}, errors.Join(fmt.Errorf("Codex invocation: %w", context.DeadlineExceeded), parseErr)
		case errors.Is(processErr, process.ErrCancelled):
			return agentworkflow.InvocationResult{}, errors.Join(fmt.Errorf("Codex invocation: %w", context.Canceled), parseErr)
		default:
			return agentworkflow.InvocationResult{}, errors.Join(fmt.Errorf("%w: execute Codex: %v", agentworkflow.ErrAgent, processErr), parseErr)
		}
	}
	if parseErr != nil {
		return agentworkflow.InvocationResult{}, parseErr
	}
	if result.ExitCode != 0 {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Codex exit %d: %s", agentworkflow.ErrAgent, result.ExitCode, boundedMessage(result.Stderr))
	}
	if parsed.failed != "" {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Codex terminal failure: %s", agentworkflow.ErrAgent, parsed.failed)
	}
	if !parsed.completed || !parsed.started || parsed.session == "" {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Codex stream lacks a complete terminal lifecycle", agentworkflow.ErrAgent)
	}
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: read Codex final output: %v", agentworkflow.ErrAgent, err)
	}
	if int64(len(output)) > invocation.MaxOutputBytes {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Codex final output", agentworkflow.ErrCapacity)
	}
	if !json.Valid(output) {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: Codex final output is not valid JSON", agentworkflow.ErrAgent)
	}
	return agentworkflow.InvocationResult{Session: parsed.session, Output: append(json.RawMessage(nil), output...), Usage: parsed.usage}, nil
}

func codexEnvironment() []string {
	return backendconfig.Environment("OPENAI_API_KEY", "CODEX_API_KEY", "CODEX_HOME")
}

func (backend *backend) commonArguments(invocation agentworkflow.Invocation, schemaPath, outputPath string) []string {
	arguments := []string{"--json", "--sandbox", string(invocation.Permission), "--skip-git-repo-check"}
	if backend.config.Qualified {
		arguments = append(arguments, "--ignore-user-config", "--ignore-rules", "--strict-config")
	}
	if backend.config.Model != "" {
		arguments = append(arguments, "--model", backend.config.Model)
	}
	return append(arguments, "--output-schema", schemaPath, "--output-last-message", outputPath)
}

func (backend *backend) resumeArguments(schemaPath, outputPath string) []string {
	arguments := []string{"--json", "--skip-git-repo-check"}
	if backend.config.Qualified {
		arguments = append(arguments, "--ignore-user-config", "--ignore-rules", "--strict-config")
	}
	if backend.config.Model != "" {
		arguments = append(arguments, "--model", backend.config.Model)
	}
	return append(arguments, "--output-schema", schemaPath, "--output-last-message", outputPath)
}

type parsedEvents struct {
	session   string
	started   bool
	completed bool
	failed    string
	usage     agentworkflow.Usage
}

type codexEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Message  string `json:"message"`
	Error    any    `json:"error"`
	Item     struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Command string `json:"command"`
	} `json:"item"`
	Usage struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
}

type codexEventParser struct {
	sink           agentworkflow.EventSink
	result         parsedEvents
	terminal       bool
	pendingSession *agentworkflow.Event
}

func parseEvents(data []byte, maxEvents int, sink agentworkflow.EventSink) (parsedEvents, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64<<10), len(data)+1)
	parser := codexEventParser{sink: sink}
	events := 0
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		events++
		if events > maxEvents {
			return parsedEvents{}, fmt.Errorf("%w: Codex event count", agentworkflow.ErrCapacity)
		}
		if err := parser.accept(line, events); err != nil {
			return parsedEvents{}, err
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return parsedEvents{}, fmt.Errorf("%w: read Codex events: %v", agentworkflow.ErrAgent, err)
	}
	return parser.result, nil
}

func (parser *codexEventParser) accept(line []byte, number int) error {
	if parser.terminal {
		return fmt.Errorf("%w: Codex emitted an event after its terminal event", agentworkflow.ErrAgent)
	}
	var event codexEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return fmt.Errorf("%w: decode Codex event %d: %v", agentworkflow.ErrAgent, number, err)
	}
	normalized, retain, err := parser.normalize(event, line)
	if err != nil || !retain {
		return err
	}
	if err := parser.sink.Emit(normalized); err != nil {
		return fmt.Errorf("%w: retain Codex event %d: %v", agentworkflow.ErrCapacity, number, err)
	}
	if normalized.Kind == agentworkflow.EventInvocationStarted && parser.pendingSession != nil {
		if err := parser.sink.Emit(*parser.pendingSession); err != nil {
			return fmt.Errorf("%w: retain Codex session event: %v", agentworkflow.ErrCapacity, err)
		}
		parser.pendingSession = nil
	}
	return nil
}

func (parser *codexEventParser) normalize(event codexEvent, line []byte) (agentworkflow.Event, bool, error) {
	normalized := agentworkflow.Event{Kind: agentworkflow.EventProgress, Raw: append(json.RawMessage(nil), line...)}
	switch event.Type {
	case "thread.started":
		if parser.result.session != "" || strings.TrimSpace(event.ThreadID) == "" {
			return agentworkflow.Event{}, false, fmt.Errorf("%w: invalid duplicate or empty Codex session event", agentworkflow.ErrAgent)
		}
		parser.result.session = event.ThreadID
		normalized.Kind = agentworkflow.EventSessionIdentified
		normalized.Session = event.ThreadID
		parser.pendingSession = &normalized
		return agentworkflow.Event{}, false, nil
	case "turn.started":
		if parser.result.started {
			return agentworkflow.Event{}, false, fmt.Errorf("%w: duplicate Codex turn start", agentworkflow.ErrAgent)
		}
		parser.result.started = true
		normalized.Kind = agentworkflow.EventInvocationStarted
	case "turn.completed":
		parser.result.completed = true
		parser.result.usage = agentworkflow.Usage{InputTokens: event.Usage.InputTokens, OutputTokens: event.Usage.OutputTokens}
		normalized.Kind = agentworkflow.EventInvocationCompleted
		normalized.Usage = parser.result.usage
		parser.terminal = true
	case "turn.failed", "error":
		parser.result.failed = jsonValue(event.Error, event.Message)
		normalized.Kind = agentworkflow.EventInvocationFailed
		normalized.Message = parser.result.failed
		parser.terminal = true
	case "item.started", "item.updated", "item.completed":
		normalizeCodexItem(&normalized, event)
	default:
	}
	return normalized, true, nil
}

func normalizeCodexItem(normalized *agentworkflow.Event, event codexEvent) {
	switch event.Item.Type {
	case "agent_message":
		normalized.Kind = agentworkflow.EventAgentMessage
		normalized.Message = event.Item.Text
	case "command_execution":
		normalized.Kind = agentworkflow.EventCommand
		normalized.Message = event.Item.Command
	case "file_change":
		normalized.Kind = agentworkflow.EventFileChange
	default:
		normalized.Kind = agentworkflow.EventTool
	}
}

func validateInvocation(invocation agentworkflow.Invocation, sink agentworkflow.EventSink) error {
	if sink == nil {
		return errors.New("Codex event sink is required")
	}
	if strings.TrimSpace(invocation.ID) == "" || strings.TrimSpace(invocation.Phase) == "" || strings.TrimSpace(invocation.Prompt) == "" {
		return errors.New("Codex invocation identity, phase, and prompt are required")
	}
	if invocation.Permission != agentworkflow.PermissionReadOnly && invocation.Permission != agentworkflow.PermissionWorkspaceWrite {
		return fmt.Errorf("Codex permission %q is invalid", invocation.Permission)
	}
	if !json.Valid(invocation.OutputSchema) {
		return errors.New("Codex output schema is not valid JSON")
	}
	if invocation.Timeout <= 0 || invocation.MaxOutputBytes <= 0 || invocation.MaxEvents <= 0 {
		return errors.New("Codex invocation bounds must be positive")
	}
	info, err := os.Stat(invocation.Workspace)
	if err != nil || !info.IsDir() {
		return errors.Join(errors.New("Codex workspace is not a directory"), err)
	}
	return nil
}

func jsonValue(value any, fallback string) string {
	if value == nil {
		if fallback == "" {
			return "provider reported failure"
		}
		return fallback
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fallback
	}
	return string(encoded)
}

func boundedMessage(data []byte) string {
	const limit = 4 << 10
	data = bytes.TrimSpace(data)
	if len(data) > limit {
		return string(data[:limit]) + "…"
	}
	return string(data)
}
