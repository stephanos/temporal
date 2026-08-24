package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func TestClaudeAdapterConformance(t *testing.T) {
	backendtest.Run(t, func(t *testing.T, scenario string) agentworkflow.Backend {
		t.Helper()
		backend, err := New(Config{Command: helperCommand(scenario)})
		if err != nil {
			t.Fatal(err)
		}
		return backend
	})
}

func TestClaudeAdapterUsesStructuredOutputAndExplicitResume(t *testing.T) {
	t.Setenv("AGENTWORKFLOW_UNDECLARED_SECRET", "secret")
	backend, err := New(Config{Command: helperCommand("valid"), Qualified: true, Model: "test-model", MaxTurns: 7})
	if err != nil {
		t.Fatal(err)
	}
	invocation := testInvocation(t)
	invocation.Session = "session-1"
	sink := &eventSink{}
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

func TestClaudeModelArgumentPrecedence(t *testing.T) {
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
			backend := &backend{config: Config{Command: []string{"claude"}, Model: test.configuredModel}}
			invocation := testInvocation(t)
			invocation.Model = test.invocationModel

			arguments, err := backend.command(invocation)
			require.NoError(t, err)
			if test.wantModel == "" {
				require.NotContains(t, arguments, "--model")
				return
			}
			require.Equal(t, test.wantModel, argValue(arguments, "--model"))
		})
	}
}

func TestClaudeAdapterRejectsInvalidTerminalPayloads(t *testing.T) {
	for _, scenario := range []string{"malformed", "failure", "missing-session", "missing-structured"} {
		t.Run(scenario, func(t *testing.T) {
			backend, _ := New(Config{Command: helperCommand(scenario)})
			_, err := backend.Execute(context.Background(), testInvocation(t), &eventSink{})
			if !errors.Is(err, agentworkflow.ErrAgent) {
				t.Fatalf("Execute() error = %v, want agent failure", err)
			}
		})
	}
}

func TestClaudeAdapterClassifiesOutputOverflow(t *testing.T) {
	backend, _ := New(Config{Command: helperCommand("overflow")})
	invocation := testInvocation(t)
	invocation.MaxOutputBytes = 128
	_, err := backend.Execute(context.Background(), invocation, &eventSink{})
	if !errors.Is(err, agentworkflow.ErrCapacity) {
		t.Fatalf("Execute() error = %v, want capacity", err)
	}
}

//nolint:errcheck,revive // Writes and exits intentionally model a real provider subprocess.
func TestClaudeCLIHelper(t *testing.T) {
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
		fmt.Fprintln(os.Stdout, "2.1.999 (Claude Code)")
		os.Exit(0)
	}
	if scenario == "cancel" {
		for {
			runtime.Gosched()
		}
	}
	if !hasArg(arguments, "-p") || argValue(arguments, "--output-format") != "json" {
		fmt.Fprintln(os.Stderr, "missing print JSON mode")
		os.Exit(30)
	}
	prompt, err := io.ReadAll(os.Stdin)
	if err != nil || string(prompt) != "return ok" || hasArg(arguments, "return ok") {
		fmt.Fprintln(os.Stderr, "prompt was not isolated on stdin")
		os.Exit(33)
	}
	schema := argValue(arguments, "--json-schema")
	if !json.Valid([]byte(schema)) {
		fmt.Fprintln(os.Stderr, "invalid schema")
		os.Exit(31)
	}
	if hasArg(arguments, "--resume") && argValue(arguments, "--resume") != "session-1" {
		fmt.Fprintln(os.Stderr, "resume did not use explicit session")
		os.Exit(32)
	}
	switch scenario {
	case "malformed":
		fmt.Fprintln(os.Stdout, `{not-json}`)
	case "failure":
		fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"error","is_error":true,"result":"failed","session_id":"session-1"}`)
	case "missing-session":
		fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"success","is_error":false,"result":"done","structured_output":{"ok":true}}`)
	case "missing-structured":
		fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"session-1"}`)
	case "overflow":
		fmt.Fprint(os.Stdout, strings.Repeat("x", 1<<20))
	default:
		fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"session-1","structured_output":{"ok":true},"usage":{"input_tokens":10,"output_tokens":2}}`)
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
	return []string{os.Args[0], "-test.run=TestClaudeCLIHelper", "--", scenario}
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
