package event

import (
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/tools/umpire/identity"
)

// PollActivityTaskEvent represents an activity task being polled.
type PollActivityTaskEvent struct {
	Request      *matchingservice.PollActivityTaskQueueRequest
	Response     *matchingservice.PollActivityTaskQueueResponse
	Timestamp    time.Time
	Identity     *identity.Identity
	TaskReturned bool
}

func (e *PollActivityTaskEvent) EventType() string {
	return "PollActivityTask"
}

func (e *PollActivityTaskEvent) TargetEntity() *identity.Identity {
	return e.Identity
}

// Parse parses a PollActivityTaskEvent from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *PollActivityTaskEvent) Parse(span ptrace.Span) Event {
	var req matchingservice.PollActivityTaskQueueRequest
	if !GetRequestPayload(span, &req) {
		return nil
	}

	if req.PollRequest == nil || req.PollRequest.TaskQueue == nil || req.PollRequest.TaskQueue.Name == "" {
		return nil
	}

	taskQueue := req.PollRequest.TaskQueue.Name

	// Parse response to check if poll returned a task
	var resp matchingservice.PollActivityTaskQueueResponse
	taskReturned := GetResponsePayload(span, &resp) &&
		span.Status().Code() != ptrace.StatusCodeError &&
		len(resp.TaskToken) > 0

	// Compute identity
	var ident *identity.Identity
	if taskReturned && resp.ActivityId != "" {
		// Use ActivityId from response
		activityTaskID := identity.NewEntityIDFromType("ActivityTask", taskQueue+":"+resp.ActivityId)
		taskQueueID := identity.NewEntityIDFromType("TaskQueue", taskQueue)
		ident = &identity.Identity{
			EntityID: activityTaskID,
			ParentID: &taskQueueID,
		}
	} else if taskReturned {
		// Empty poll or no ActivityId - target is the TaskQueue
		taskQueueID := identity.NewEntityIDFromType("TaskQueue", taskQueue)
		ident = &identity.Identity{
			EntityID: taskQueueID,
			ParentID: nil,
		}
	}

	return &PollActivityTaskEvent{
		Request:      &req,
		Response:     &resp,
		Timestamp:    span.StartTimestamp().AsTime(),
		Identity:     ident,
		TaskReturned: taskReturned,
	}
}
