package moves

import (
	scorebooktypes "go.temporal.io/server/tools/catch/scorebook/types"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/matchingservice/v1"
	rostertypes "go.temporal.io/server/tools/catch/roster/types"
)

// PollWorkflowTask represents a workflow task being polled.
type PollWorkflowTask struct {
	Request      *matchingservice.PollWorkflowTaskQueueRequest
	Response     *matchingservice.PollWorkflowTaskQueueResponse
	Timestamp    time.Time
	Identity     *rostertypes.Identity
	TaskReturned bool
}

func (e *PollWorkflowTask) MoveType() string {
	return "PollWorkflowTask"
}

func (e *PollWorkflowTask) TargetEntity() *rostertypes.Identity {
	return e.Identity
}

// Parse parses a PollWorkflowTask from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *PollWorkflowTask) Parse(span ptrace.Span) scorebooktypes.Move {
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
	var ident *rostertypes.Identity
	if taskReturned && resp.WorkflowExecution != nil &&
		resp.WorkflowExecution.WorkflowId != "" && resp.WorkflowExecution.RunId != "" {
		// Poll returned a task - target is the WorkflowTask
		workflowTaskID := rostertypes.NewEntityIDFromType("WorkflowTask",
			taskQueue+":"+resp.WorkflowExecution.WorkflowId+":"+resp.WorkflowExecution.RunId)
		taskQueueID := rostertypes.NewEntityIDFromType("TaskQueue", taskQueue)
		ident = &rostertypes.Identity{
			EntityID: workflowTaskID,
			ParentID: &taskQueueID,
		}
	} else {
		// Empty poll - target is the TaskQueue
		taskQueueID := rostertypes.NewEntityIDFromType("TaskQueue", taskQueue)
		ident = &rostertypes.Identity{
			EntityID: taskQueueID,
			ParentID: nil,
		}
	}

	return &PollWorkflowTask{
		Request:      &req,
		Response:     &resp,
		Timestamp:    span.StartTimestamp().AsTime(),
		Identity:     ident,
		TaskReturned: taskReturned,
	}
}
