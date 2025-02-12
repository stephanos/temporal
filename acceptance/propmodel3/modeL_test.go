package propmodel

import (
	"testing"

	. "go.temporal.io/api/workflowservice/v1"
)

func TestExample(t *testing.T) {
	env := NewEnv(t)
	RegisterModel[WorkflowRun](env)
	RegisterModel[WorkflowUpdate](env)
	env.Send(Request[*UpdateWorkflowExecutionRequest]{})
}
