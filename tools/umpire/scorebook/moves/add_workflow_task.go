package moves

import (
	scorebooktypes "go.temporal.io/server/tools/catch/scorebook/types"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/matchingservice/v1"
	rostertypes "go.temporal.io/server/tools/catch/roster/types"
)

// AddWorkflowTask represents a workflow task being added to matching.
type AddWorkflowTask struct {
	Request   *matchingservice.AddWorkflowTaskRequest
	Timestamp time.Time
	Identity  *rostertypes.Identity
}

func (e *AddWorkflowTask) MoveType() string {
	return "AddWorkflowTask"
}

func (e *AddWorkflowTask) TargetEntity() *rostertypes.Identity {
	return e.Identity
}

// Parse parses an AddWorkflowTask from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *AddWorkflowTask) Parse(span ptrace.Span) scorebooktypes.Move {
	var req matchingservice.AddWorkflowTaskRequest
	if !GetRequestPayload(span, &req) {
		return nil
	}

	if req.TaskQueue == nil || req.TaskQueue.Name == "" {
		return nil
	}

	// Compute identity from the request fields
	var ident *rostertypes.Identity
	if req.Execution != nil && req.Execution.WorkflowId != "" && req.Execution.RunId != "" {
		workflowTaskID := rostertypes.NewEntityIDFromType("WorkflowTask",
			req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
		taskQueueID := rostertypes.NewEntityIDFromType("TaskQueue", req.TaskQueue.Name)
		ident = &rostertypes.Identity{
			EntityID: workflowTaskID,
			ParentID: &taskQueueID,
		}
	}

	return &AddWorkflowTask{
		Request:   &req,
		Timestamp: span.StartTimestamp().AsTime(),
		Identity:  ident,
	}
}
