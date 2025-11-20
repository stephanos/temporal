package event

import (
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/tools/umpire/identity"
)

// AddWorkflowTaskEvent represents a workflow task being added to matching.
type AddWorkflowTaskEvent struct {
	Request   *matchingservice.AddWorkflowTaskRequest
	Timestamp time.Time
	Identity  *identity.Identity
}

func (e *AddWorkflowTaskEvent) EventType() string {
	return "AddWorkflowTask"
}

func (e *AddWorkflowTaskEvent) TargetEntity() *identity.Identity {
	return e.Identity
}

// Parse parses an AddWorkflowTaskEvent from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *AddWorkflowTaskEvent) Parse(span ptrace.Span) Event {
	var req matchingservice.AddWorkflowTaskRequest
	if !GetRequestPayload(span, &req) {
		return nil
	}

	if req.TaskQueue == nil || req.TaskQueue.Name == "" {
		return nil
	}

	// Compute identity from the request fields
	var ident *identity.Identity
	if req.Execution != nil && req.Execution.WorkflowId != "" && req.Execution.RunId != "" {
		workflowTaskID := identity.NewEntityIDFromType("WorkflowTask",
			req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
		taskQueueID := identity.NewEntityIDFromType("TaskQueue", req.TaskQueue.Name)
		ident = &identity.Identity{
			EntityID: workflowTaskID,
			ParentID: &taskQueueID,
		}
	}

	return &AddWorkflowTaskEvent{
		Request:   &req,
		Timestamp: span.StartTimestamp().AsTime(),
		Identity:  ident,
	}
}
