package backendtest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"go.temporal.io/server/tools/agentworkflow/internal/agentworkflow"
)

func TestRunExercisesPortableBackendContract(t *testing.T) {
	Run(t, func(_ *testing.T, scenario string) agentworkflow.Backend {
		return fakeBackend{scenario: scenario}
	})
}

type fakeBackend struct {
	scenario string
}

func (backend fakeBackend) Describe(context.Context) (agentworkflow.BackendInfo, error) {
	return agentworkflow.BackendInfo{
		Name: "fake", Version: "v1", ConfigurationDigest: "sha256:fake-configuration",
		Capabilities: []agentworkflow.Capability{
			agentworkflow.CapabilityReadOnly, agentworkflow.CapabilityWorkspaceWrite,
			agentworkflow.CapabilityStructuredOutput, agentworkflow.CapabilityResume, agentworkflow.CapabilityCancellation,
		},
	}, nil
}

func (backend fakeBackend) Execute(ctx context.Context, _ agentworkflow.Invocation, sink agentworkflow.EventSink) (agentworkflow.InvocationResult, error) {
	if err := ctx.Err(); err != nil {
		return agentworkflow.InvocationResult{}, err
	}
	if backend.scenario == "failure" {
		return agentworkflow.InvocationResult{}, fmt.Errorf("%w: scripted", agentworkflow.ErrAgent)
	}
	for _, event := range []agentworkflow.Event{
		{Kind: agentworkflow.EventInvocationStarted},
		{Kind: agentworkflow.EventSessionIdentified, Session: "session"},
		{Kind: agentworkflow.EventInvocationCompleted},
	} {
		if err := sink.Emit(event); err != nil {
			return agentworkflow.InvocationResult{}, err
		}
	}
	return agentworkflow.InvocationResult{Session: "session", Output: json.RawMessage(`{"ok":true}`)}, nil
}
