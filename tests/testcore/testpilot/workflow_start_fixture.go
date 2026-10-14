package testpilot

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

const (
	// WorkflowStartFixture is the one Case the workflow-start Model's functional set produces.
	WorkflowStartFixture            = "workflowStartTests-started"
	WorkflowStartNamespaceBindingID = "temporal.worker.namespace"
	WorkflowStartTaskQueueBindingID = "temporal.task-queue.resource"
	WorkflowStartWorkflowType       = "umpire-workflowStartTests-started-workflow"
	WorkflowStartRuleID             = "relation"
	WorkflowStartRuleDefinitionID   = "temporal.workflow.start.property.submittedTypeIsRecorded.relation"
	WorkflowStartProgramID          = "temporal.case.workflowStartTests.started.program"
	workflowStartArtifactNamespace  = "workflow-start-namespace"
	workflowStartArtifactTaskQueue  = "workflow-start-task-queue"
)

type WorkflowStartEnvironment struct {
	Namespace string
	TaskQueue string
}

// WorkflowStartProfile authorizes exactly what the workflow-start Case needs: the two
// WorkflowService methods on one endpoint role, the symbolic worker and task queue its
// StartWorkflowExecution reserves, and nothing else.
func WorkflowStartProfile(catalog *testpilot.Catalog, environment WorkflowStartEnvironment) testpilot.ProfileSpec {
	return caseProfile("workflow-start-profile", catalog,
		[]testpilot.RolePolicy{
			workflowServiceRole(0),
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
		},
		[]testpilot.Opcode{testpilot.InvokeRPC, testpilot.Finish},
		[]testpilot.EnvironmentBinding{
			{ID: WorkflowStartNamespaceBindingID, Value: environment.Namespace},
			{ID: WorkflowStartTaskQueueBindingID, Value: environment.TaskQueue},
		})
}
