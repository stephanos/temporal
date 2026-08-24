package agentworkflow_test

import (
	"context"
	"encoding/json"
	"fmt"

	"go.temporal.io/server/tools/agentworkflow"
)

type projectBackend struct{}

func (projectBackend) Describe(context.Context) (agentworkflow.BackendInfo, error) {
	return agentworkflow.BackendInfo{
		Name: "project-agent", Version: "v1", ConfigurationDigest: "sha256:project-agent-v1",
		Capabilities: []agentworkflow.Capability{
			agentworkflow.CapabilityReadOnly, agentworkflow.CapabilityWorkspaceWrite,
			agentworkflow.CapabilityStructuredOutput, agentworkflow.CapabilityCancellation,
		},
	}, nil
}

func (projectBackend) Execute(_ context.Context, invocation agentworkflow.Invocation, sink agentworkflow.EventSink) (agentworkflow.InvocationResult, error) {
	if err := sink.Emit(agentworkflow.Event{Kind: agentworkflow.EventInvocationStarted}); err != nil {
		return agentworkflow.InvocationResult{}, err
	}
	if err := sink.Emit(agentworkflow.Event{Kind: agentworkflow.EventSessionIdentified, Session: "project-session"}); err != nil {
		return agentworkflow.InvocationResult{}, err
	}
	if err := sink.Emit(agentworkflow.Event{Kind: agentworkflow.EventInvocationCompleted}); err != nil {
		return agentworkflow.InvocationResult{}, err
	}
	return agentworkflow.InvocationResult{Session: "project-session", Output: json.RawMessage(`{}`)}, nil
}

func ExampleBackend() {
	var backend agentworkflow.Backend = projectBackend{}
	info, _ := backend.Describe(context.Background())
	fmt.Println(info.Name)
	// Output: project-agent
}
