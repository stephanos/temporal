package testpilot

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
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
	return testpilot.ProfileSpec{
		Identity: "async-nexus-profile",
		Catalog:  catalog,
		Roles: []testpilot.RolePolicy{
			{
				ID: "temporal.workflow-service", Kind: testpilotspb.ROLE_KIND_ENDPOINT,
				Methods: []string{
					"/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					"/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory",
				},
				ReservationCarriers: []testpilot.ReservationCarrierPolicy{{
					Method: "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					Shapes: []testpilot.ReservationCarrierShape{
						{Context: testpilotspb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 1},
						{Context: testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 1},
					},
				}},
			},
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		Capabilities: []testpilot.Capability{
			testpilot.InvokeRPC, testpilot.AwaitSlot, testpilot.CompleteNexusOperation,
			testpilot.StartNexusOperation, testpilot.Await, testpilot.Finish, testpilot.RespondNexus,
		},
		EnvironmentBindings: []testpilot.EnvironmentBinding{
			{ID: AsyncNexusWorkerNamespaceBindingID, Value: environment.Namespace},
			{ID: AsyncNexusTaskQueueBindingID, Value: environment.TaskQueue},
			{ID: AsyncNexusEndpointBindingID, Value: environment.NexusEndpoint},
		},
		ProgramLimits:  proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(source.GetContract().GetLimits()),
	}
}
