package event

import (
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/tools/umpire/identity"
)

// PollWorkflowTaskEvent represents a workflow task being polled.
type PollWorkflowTaskEvent struct {
	Request      *matchingservice.PollWorkflowTaskQueueRequest
	Response     *matchingservice.PollWorkflowTaskQueueResponse
	Timestamp    time.Time
	Identity     *identity.Identity
	TaskReturned bool
}

func (e *PollWorkflowTaskEvent) EventType() string {
	return "PollWorkflowTask"
}

func (e *PollWorkflowTaskEvent) TargetEntity() *identity.Identity {
	return e.Identity
}

// Parse parses a PollWorkflowTaskEvent from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *PollWorkflowTaskEvent) Parse(span ptrace.Span) Event {
	var req matchingservice.PollWorkflowTaskQueueRequest
	if !GetRequestPayload(span, &req) {
		return nil
	}

	if req.PollRequest == nil || req.PollRequest.TaskQueue == nil || req.PollRequest.TaskQueue.Name == "" {
		return nil
	}

	taskQueue := req.PollRequest.TaskQueue.Name

	// Parse response to check if poll returned a task
	var resp matchingservice.PollWorkflowTaskQueueResponse
	taskReturned := GetResponsePayload(span, &resp) &&
		span.Status().Code() != ptrace.StatusCodeError &&
		len(resp.TaskToken) > 0

	// Compute identity
	var ident *identity.Identity
	if taskReturned && resp.WorkflowExecution != nil &&
		resp.WorkflowExecution.WorkflowId != "" && resp.WorkflowExecution.RunId != "" {
		// Poll returned a task - target is the WorkflowTask
		workflowTaskID := identity.NewEntityIDFromType("WorkflowTask",
			taskQueue+":"+resp.WorkflowExecution.WorkflowId+":"+resp.WorkflowExecution.RunId)
		taskQueueID := identity.NewEntityIDFromType("TaskQueue", taskQueue)
		ident = &identity.Identity{
			EntityID: workflowTaskID,
			ParentID: &taskQueueID,
		}
	} else {
		// Empty poll - target is the TaskQueue
		taskQueueID := identity.NewEntityIDFromType("TaskQueue", taskQueue)
		ident = &identity.Identity{
			EntityID: taskQueueID,
			ParentID: nil,
		}
	}

	return &PollWorkflowTaskEvent{
		Request:      &req,
		Response:     &resp,
		Timestamp:    span.StartTimestamp().AsTime(),
		Identity:     ident,
		TaskReturned: taskReturned,
	}
}
