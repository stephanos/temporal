package testpilot

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

const (
	TypedUnaryNamespaceBindingID = "temporal.typed-unary.namespace"
	TypedUnaryTaskQueueBindingID = "temporal.typed-unary.task-queue"
	TypedUnaryWorkflowType       = "umpire-typed-unary-workflow"
)

type TypedUnaryEnvironment struct {
	Namespace string
	TaskQueue string
}

// TypedUnaryProfile authorizes exactly what the typed unary Case needs: the two WorkflowService
// methods on one endpoint role, the symbolic worker and task queue its StartWorkflowExecution
// reserves, and nothing else.
func TypedUnaryProfile(catalog *testpilot.Catalog, source *testpilotspb.Case, environment TypedUnaryEnvironment) testpilot.ProfileSpec {
	return testpilot.ProfileSpec{
		Identity: "typed-unary-profile",
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
					},
				}},
			},
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
		},
		Capabilities: []testpilot.Capability{testpilot.InvokeRPC, testpilot.Finish},
		EnvironmentBindings: []testpilot.EnvironmentBinding{
			{ID: TypedUnaryNamespaceBindingID, Value: environment.Namespace},
			{ID: TypedUnaryTaskQueueBindingID, Value: environment.TaskQueue},
		},
		ProgramLimits:  proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(source.GetContract().GetLimits()),
	}
}
