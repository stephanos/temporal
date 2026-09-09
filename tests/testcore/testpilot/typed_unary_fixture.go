package testpilot

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
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
	return caseProfile("typed-unary-profile", catalog, source,
		[]testpilot.RolePolicy{
			workflowServiceRole(0),
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
		},
		[]testpilot.Capability{testpilot.InvokeRPC, testpilot.Finish},
		[]testpilot.EnvironmentBinding{
			{ID: TypedUnaryNamespaceBindingID, Value: environment.Namespace},
			{ID: TypedUnaryTaskQueueBindingID, Value: environment.TaskQueue},
		})
}
