package moves

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/matchingservice/v1"
	rostertypes "go.temporal.io/server/tools/umpire/roster/types"
	scorebooktypes "go.temporal.io/server/tools/umpire/scorebook/types"
)

// AddActivityTask represents an activity task being added to matching.
type AddActivityTask struct {
	Request   *matchingservice.AddActivityTaskRequest
	Timestamp time.Time
	Identity  *rostertypes.Identity
}

func (e *AddActivityTask) MoveType() string {
	return "AddActivityTask"
}

func (e *AddActivityTask) TargetEntity() *rostertypes.Identity {
	return e.Identity
}

// Parse parses an AddActivityTask from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *AddActivityTask) Parse(span ptrace.Span) scorebooktypes.Move {
	var req matchingservice.AddActivityTaskRequest
	if !GetRequestPayload(span, &req) {
		return nil
	}

	if req.TaskQueue == nil || req.TaskQueue.Name == "" {
		return nil
	}

	// Compute identity - use scheduledEventID as activityID since ActivityID isn't in the request
	activityID := fmt.Sprintf("%d", req.ScheduledEventId)
	var ident *rostertypes.Identity
	if activityID != "" {
		activityTaskID := rostertypes.NewEntityIDFromType(rostertypes.ActivityTaskType, req.TaskQueue.Name+":"+activityID)
		ident = &rostertypes.Identity{
			EntityID: activityTaskID,
			ParentID: nil,
		}
	}

	return &AddActivityTask{
		Request:   &req,
		Timestamp: span.StartTimestamp().AsTime(),
		Identity:  ident,
	}
}
