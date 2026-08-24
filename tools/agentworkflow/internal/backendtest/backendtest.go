package backendtest

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"go.temporal.io/server/tools/agentworkflow/internal/agentworkflow"
)

type Factory func(*testing.T, string) agentworkflow.Backend

func Run(t *testing.T, factory Factory) {
	t.Helper()
	t.Run("stable-description", func(t *testing.T) { testStableDescription(t, factory) })
	t.Run("structured-execution", func(t *testing.T) { testStructuredExecution(t, factory) })
	t.Run("workspace-write-and-resume", func(t *testing.T) { testWorkspaceWriteAndResume(t, factory) })
	t.Run("provider-failure", func(t *testing.T) { testProviderFailure(t, factory) })
	t.Run("event-backpressure", func(t *testing.T) { testEventBackpressure(t, factory) })
	t.Run("cancelled", func(t *testing.T) { testCancellation(t, factory) })
}

func testStableDescription(t *testing.T, factory Factory) {
	t.Helper()
	backend := factory(t, "valid")
	first, err := backend.Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := backend.Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	firstData, _ := json.Marshal(first)
	secondData, _ := json.Marshal(second)
	if string(firstData) != string(secondData) || first.Name == "" || first.Version == "" || first.ConfigurationDigest == "" {
		t.Fatalf("backend identity is unstable: %s != %s", firstData, secondData)
	}
	for _, required := range requiredCapabilities() {
		if !hasCapability(first.Capabilities, required) {
			t.Fatalf("backend identity lacks required capability %q: %#v", required, first)
		}
	}
}

func testStructuredExecution(t *testing.T, factory Factory) {
	t.Helper()
	backend := factory(t, "valid")
	events := &sink{}
	result, err := backend.Execute(context.Background(), invocation(t, 1<<20, 100), events)
	if err != nil {
		t.Fatal(err)
	}
	if result.Session == "" || !json.Valid(result.Output) || !validLifecycle(events.events, result.Session) {
		t.Fatalf("invalid backend result: %#v, events=%d", result, len(events.events))
	}
}

func testWorkspaceWriteAndResume(t *testing.T, factory Factory) {
	t.Helper()
	backend := factory(t, "valid")
	info, err := backend.Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	request := invocation(t, 1<<20, 100)
	request.Permission = agentworkflow.PermissionWorkspaceWrite
	request.RetainSession = hasCapability(info.Capabilities, agentworkflow.CapabilityResume)
	first, err := backend.Execute(context.Background(), request, &sink{})
	if err != nil {
		t.Fatal(err)
	}
	if !request.RetainSession {
		return
	}
	request.Session = first.Session
	secondSink := &sink{}
	second, err := backend.Execute(context.Background(), request, secondSink)
	if err != nil || second.Session != first.Session || !validLifecycle(secondSink.events, second.Session) {
		t.Fatalf("resumed result=%#v events=%#v error=%v", second, secondSink.events, err)
	}
}

func testProviderFailure(t *testing.T, factory Factory) {
	t.Helper()
	_, err := factory(t, "failure").Execute(context.Background(), invocation(t, 1<<20, 100), &sink{})
	if err == nil {
		t.Fatal("backend failure was accepted")
	}
}

func testEventBackpressure(t *testing.T, factory Factory) {
	t.Helper()
	_, err := factory(t, "valid").Execute(context.Background(), invocation(t, 1<<20, 1), &rejectingSink{})
	if err == nil {
		t.Fatal("event sink rejection was ignored")
	}
}

func testCancellation(t *testing.T, factory Factory) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := factory(t, "cancel").Execute(ctx, invocation(t, 1<<20, 100), &sink{})
	if err == nil || (!errors.Is(err, context.Canceled) && !errors.Is(err, agentworkflow.ErrAgent)) {
		t.Fatalf("cancelled execution error = %v", err)
	}
}

func requiredCapabilities() []agentworkflow.Capability {
	return []agentworkflow.Capability{
		agentworkflow.CapabilityReadOnly, agentworkflow.CapabilityWorkspaceWrite,
		agentworkflow.CapabilityStructuredOutput, agentworkflow.CapabilityCancellation,
	}
}

func hasCapability(capabilities []agentworkflow.Capability, target agentworkflow.Capability) bool {
	for _, capability := range capabilities {
		if capability == target {
			return true
		}
	}
	return false
}

func validLifecycle(events []agentworkflow.Event, session string) bool {
	if len(events) < 3 || events[0].Kind != agentworkflow.EventInvocationStarted || events[len(events)-1].Kind != agentworkflow.EventInvocationCompleted {
		return false
	}
	started, identified, terminal := 0, 0, 0
	for _, event := range events {
		switch event.Kind {
		case agentworkflow.EventInvocationStarted:
			started++
		case agentworkflow.EventSessionIdentified:
			if event.Session != session {
				return false
			}
			identified++
		case agentworkflow.EventInvocationCompleted, agentworkflow.EventInvocationFailed:
			terminal++
		default:
		}
	}
	return started == 1 && identified == 1 && terminal == 1
}

func invocation(t *testing.T, bytes int64, events int) agentworkflow.Invocation {
	t.Helper()
	return agentworkflow.Invocation{
		ID: "conformance", Phase: "test", Workspace: t.TempDir(), Prompt: "return ok",
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}`),
		Permission:   agentworkflow.PermissionReadOnly, Timeout: time.Minute, MaxOutputBytes: bytes, MaxEvents: events,
	}
}

type sink struct {
	events []agentworkflow.Event
}

func (sink *sink) Emit(event agentworkflow.Event) error {
	sink.events = append(sink.events, event)
	return nil
}

type rejectingSink struct{}

func (*rejectingSink) Emit(agentworkflow.Event) error {
	return os.ErrClosed
}
