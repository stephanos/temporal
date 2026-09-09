package testpilot

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
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
	return testpilot.ProfileSpec{
		Identity: "typed-nexus-profile",
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
						{Context: testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 2},
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
			{ID: TypedNexusNamespaceBindingID, Value: environment.Namespace},
			{ID: TypedNexusTaskQueueBindingID, Value: environment.TaskQueue},
			{ID: TypedNexusEndpointBindingID, Value: environment.NexusEndpoint},
		},
		ProgramLimits:  proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(source.GetContract().GetLimits()),
	}
}
