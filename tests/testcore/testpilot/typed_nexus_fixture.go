package testpilot

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

const (
	TypedNexusNamespaceBindingID = "temporal.typed-nexus.namespace"
	TypedNexusTaskQueueBindingID = "temporal.typed-nexus.task-queue"
	TypedNexusEndpointBindingID  = "temporal.typed-nexus.nexus-endpoint"
	TypedNexusService            = "umpire.case.service"
	TypedNexusFirstOperation     = "complete"
	TypedNexusSecondOperation    = "confirm"
)

type TypedNexusEnvironment struct {
	Namespace     string
	TaskQueue     string
	NexusEndpoint string
}

// TypedNexusProfile authorizes exactly what the two-operation Nexus Case needs: the two
// WorkflowService methods on one endpoint role, one workflow and two Nexus handler activations
// reserved by its StartWorkflowExecution, and nothing else.
func TypedNexusProfile(catalog *testpilot.Catalog, source *testpilotspb.Case, environment TypedNexusEnvironment) testpilot.ProfileSpec {
	return caseProfile("typed-nexus-profile", catalog, source,
		[]testpilot.RolePolicy{
			workflowServiceRole(2),
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		nexusCapabilities(),
		[]testpilot.EnvironmentBinding{
			{ID: TypedNexusNamespaceBindingID, Value: environment.Namespace},
			{ID: TypedNexusTaskQueueBindingID, Value: environment.TaskQueue},
			{ID: TypedNexusEndpointBindingID, Value: environment.NexusEndpoint},
		})
}
