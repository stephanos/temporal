package moves

import (
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/matchingservice/v1"
	rostertypes "go.temporal.io/server/tools/umpire/roster/types"
	scorebooktypes "go.temporal.io/server/tools/umpire/scorebook/types"
)

// PollActivityTask represents an activity task being polled.
type PollActivityTask struct {
	Request      *matchingservice.PollActivityTaskQueueRequest
	Response     *matchingservice.PollActivityTaskQueueResponse
	Timestamp    time.Time
	Identity     *rostertypes.Identity
	TaskReturned bool
}

func (e *PollActivityTask) MoveType() string {
	return "PollActivityTask"
}

func (e *PollActivityTask) TargetEntity() *rostertypes.Identity {
	return e.Identity
}

// Parse parses a PollActivityTask from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *PollActivityTask) Parse(span ptrace.Span) scorebooktypes.Move {
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
	var ident *rostertypes.Identity
	if taskReturned && resp.ActivityId != "" {
		// Use ActivityId from response
		activityTaskID := rostertypes.NewEntityIDFromType(rostertypes.ActivityTaskType, taskQueue+":"+resp.ActivityId)
		taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, taskQueue)
		ident = &rostertypes.Identity{
			EntityID: activityTaskID,
			ParentID: &taskQueueID,
		}
	} else if taskReturned {
		// Empty poll or no ActivityId - target is the TaskQueue
		taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, taskQueue)
		ident = &rostertypes.Identity{
			EntityID: taskQueueID,
			ParentID: nil,
		}
	}

	return &PollActivityTask{
		Request:      &req,
		Response:     &resp,
		Timestamp:    span.StartTimestamp().AsTime(),
		Identity:     ident,
		TaskReturned: taskReturned,
	}
}
