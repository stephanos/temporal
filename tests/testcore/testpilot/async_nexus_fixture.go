package testpilot

import (
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

func AsyncNexusProfile(catalog *testpilot.Catalog, source *testpilotspb.Case, environment AsyncNexusEnvironment) testpilot.ProfileSpec {
	return caseProfile("async-nexus-profile", catalog, source,
		[]testpilot.RolePolicy{
			workflowServiceRole(1),
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		nexusCapabilities(),
		[]testpilot.EnvironmentBinding{
			{ID: AsyncNexusWorkerNamespaceBindingID, Value: environment.Namespace},
			{ID: AsyncNexusTaskQueueBindingID, Value: environment.TaskQueue},
			{ID: AsyncNexusEndpointBindingID, Value: environment.NexusEndpoint},
		})
}
