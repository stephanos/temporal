package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/agentworkflow/internal/agentworkflow"
	"go.temporal.io/server/tools/agentworkflow/internal/backendtest"
)

func TestCodexAdapterConformance(t *testing.T) {
	backendtest.Run(t, func(t *testing.T, scenario string) agentworkflow.Backend {
		t.Helper()
		backend, err := New(Config{Command: helperCommand(scenario)})
		if err != nil {
			t.Fatal(err)
		}
		return backend
	})
}

func TestCodexAdapterUsesDedicatedOutputAndExplicitResume(t *testing.T) {
	t.Setenv("AGENTWORKFLOW_UNDECLARED_SECRET", "secret")
	backend, err := New(Config{Command: helperCommand("valid"), Qualified: true, Model: "test-model"})
	if err != nil {
		t.Fatal(err)
	}
	sink := &eventSink{}
	invocation := testInvocation(t)
	invocation.Session = "session-1"
	result, err := backend.Execute(context.Background(), invocation, sink)
	if err != nil {
		t.Fatal(err)
	}
	if result.Session != "session-1" || string(result.Output) != `{"ok":true}` {
		t.Fatalf("result = %#v", result)
	}
	if !sink.has(agentworkflow.EventInvocationCompleted) || !sink.has(agentworkflow.EventSessionIdentified) {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestCodexModelArgumentPrecedence(t *testing.T) {
	for _, test := range []struct {
		name            string
		configuredModel string
		invocationModel string
		wantModel       string
	}{
		{name: "CLI override", configuredModel: "cli-model", invocationModel: "stage-model", wantModel: "cli-model"},
		{name: "stage model", invocationModel: "stage-model", wantModel: "stage-model"},
		{name: "provider default"},
	} {
		t.Run(test.name, func(t *testing.T) {
			backend := &backend{config: Config{Model: test.configuredModel}}
			invocation := agentworkflow.Invocation{Permission: agentworkflow.PermissionReadOnly, Model: test.invocationModel}

			for _, arguments := range [][]string{
				backend.commonArguments(invocation, "schema.json", "output.json"),
				backend.resumeArguments(invocation, "schema.json", "output.json"),
			} {
				if test.wantModel == "" {
					require.NotContains(t, arguments, "--model")
					continue
				}
				require.Equal(t, test.wantModel, argValue(arguments, "--model"))
			}
		})
	}
}

func TestCodexAdapterRejectsMalformedAndPostTerminalStreams(t *testing.T) {
	for _, scenario := range []string{"malformed", "after-terminal", "no-terminal", "missing-output"} {
		t.Run(scenario, func(t *testing.T) {
			backend, _ := New(Config{Command: helperCommand(scenario)})
			_, err := backend.Execute(context.Background(), testInvocation(t), &eventSink{})
			if !errors.Is(err, agentworkflow.ErrAgent) {
				t.Fatalf("Execute() error = %v, want agent failure", err)
			}
		})
	}
}

func TestCodexAdapterClassifiesOutputOverflow(t *testing.T) {
	backend, _ := New(Config{Command: helperCommand("overflow")})
	invocation := testInvocation(t)
	invocation.MaxOutputBytes = 128
	_, err := backend.Execute(context.Background(), invocation, &eventSink{})
	if !errors.Is(err, agentworkflow.ErrCapacity) {
		t.Fatalf("Execute() error = %v, want capacity", err)
	}
}

//nolint:errcheck,revive // Writes and exits intentionally model a real provider subprocess.
func TestCodexCLIHelper(t *testing.T) {
	separator := argIndex(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	scenario := os.Args[separator+1]
	arguments := os.Args[separator+2:]
	if os.Getenv("AGENTWORKFLOW_UNDECLARED_SECRET") != "" {
		fmt.Fprintln(os.Stderr, "inherited undeclared environment")
		os.Exit(29)
	}
	if hasArg(arguments, "--version") {
		fmt.Fprintln(os.Stdout, "codex-cli fake-1.0")
		os.Exit(0)
	}
	if scenario == "cancel" {
		for {
			runtime.Gosched()
		}
	}
	if !hasArg(arguments, "--json") {
		fmt.Fprintln(os.Stderr, "missing --json")
		os.Exit(30)
	}
	if scenario == "valid" && hasArg(arguments, "resume") && !hasArg(arguments, "session-1") {
		fmt.Fprintln(os.Stderr, "resume did not use explicit session")
		os.Exit(31)
	}
	outputPath := argValue(arguments, "--output-last-message")
	schemaPath := argValue(arguments, "--output-schema")
	if outputPath == "" || schemaPath == "" {
		fmt.Fprintln(os.Stderr, "missing output paths")
		os.Exit(32)
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil || !json.Valid(schema) {
		fmt.Fprintln(os.Stderr, "invalid schema")
		os.Exit(33)
	}
	if scenario != "missing-output" {
		if err := os.WriteFile(outputPath, []byte(`{"ok":true}`), 0o600); err != nil {
			os.Exit(34)
		}
	}
	session := "session-1"
	fmt.Fprintf(os.Stdout, "{\"type\":\"thread.started\",\"thread_id\":%q}\n", session)
	fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
	switch scenario {
	case "malformed":
		fmt.Fprintln(os.Stdout, `{not-json}`)
	case "failure":
		fmt.Fprintln(os.Stdout, `{"type":"turn.failed","error":{"message":"failed"}}`)
	case "no-terminal", "missing-output":
		fmt.Fprintln(os.Stdout, `{"type":"item.completed","item":{"type":"agent_message","text":"done"}}`)
		if scenario == "missing-output" {
			fmt.Fprintln(os.Stdout, `{"type":"turn.completed","usage":{}}`)
		}
	case "after-terminal":
		fmt.Fprintln(os.Stdout, `{"type":"turn.completed","usage":{}}`)
		fmt.Fprintln(os.Stdout, `{"type":"item.completed","item":{"type":"agent_message","text":"late"}}`)
	case "overflow":
		fmt.Fprint(os.Stdout, strings.Repeat("x", 1<<20))
	default:
		fmt.Fprintln(os.Stdout, `{"type":"item.completed","item":{"type":"agent_message","text":"done"}}`)
		fmt.Fprintln(os.Stdout, `{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":2}}`)
	}
	os.Exit(0)
}

type eventSink struct {
	mu     sync.Mutex
	events []agentworkflow.Event
}

func (sink *eventSink) Emit(event agentworkflow.Event) error {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	sink.events = append(sink.events, event)
	return nil
}

func (sink *eventSink) has(kind agentworkflow.EventKind) bool {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	for _, event := range sink.events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}

func testInvocation(t *testing.T) agentworkflow.Invocation {
	t.Helper()
	return agentworkflow.Invocation{
		ID: "test", Phase: "review", Workspace: t.TempDir(), Prompt: "return ok",
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}`),
		Permission:   agentworkflow.PermissionReadOnly, Timeout: time.Minute, MaxOutputBytes: 1 << 20, MaxEvents: 100,
	}
}

func helperCommand(scenario string) []string {
	return []string{os.Args[0], "-test.run=TestCodexCLIHelper", "--", scenario}
}

func argIndex(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func hasArg(values []string, target string) bool {
	return argIndex(values, target) >= 0
}

func argValue(values []string, name string) string {
	index := argIndex(values, name)
	if index < 0 || index+1 >= len(values) {
		return ""
	}
	return values[index+1]
}
