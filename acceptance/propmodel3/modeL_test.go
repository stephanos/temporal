package propmodel

import (
	"testing"

	. "go.temporal.io/api/workflowservice/v1"
)

func TestExample(t *testing.T) {
	env := NewEnv()
	RegisterModel(env, WorkflowRun{}.Model)
	RegisterModel(env, WorkflowUpdate{}.Model)
	env.Send(Request[*UpdateWorkflowExecutionRequest]{})
}
