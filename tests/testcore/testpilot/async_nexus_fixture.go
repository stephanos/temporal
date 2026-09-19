package testpilot

import (
	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

const (
	AsyncNexusWorkerNamespaceBindingID = "temporal.worker.namespace"
	AsyncNexusTaskQueueBindingID       = "temporal.task-queue.resource"
	AsyncNexusEndpointBindingID        = "temporal.nexus-endpoint.resource"
)

type AsyncNexusEnvironment struct {
	Namespace     string
	TaskQueue     string
	NexusEndpoint string
}

func AsyncNexusProfile(catalog *testpilot.Catalog, environment AsyncNexusEnvironment) testpilot.ProfileSpec {
	profile := caseProfile("async-nexus-profile", catalog,
		[]testpilot.RolePolicy{
			workflowServiceRole(1),
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		realizedNexusCapabilities(),
		[]testpilot.EnvironmentBinding{
			{ID: AsyncNexusWorkerNamespaceBindingID, Value: environment.Namespace},
			{ID: AsyncNexusTaskQueueBindingID, Value: environment.TaskQueue},
			{ID: AsyncNexusEndpointBindingID, Value: environment.NexusEndpoint},
		})
	// The realization schedules the operation through a workflow command, which the Profile
	// admits per command type.
	profile.CommandTypes = []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}
	return profile
}
