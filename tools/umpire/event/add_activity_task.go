package event

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/tools/umpire/identity"
)

// AddActivityTaskEvent represents an activity task being added to matching.
type AddActivityTaskEvent struct {
	Request   *matchingservice.AddActivityTaskRequest
	Timestamp time.Time
	Identity  *identity.Identity
}

func (e *AddActivityTaskEvent) EventType() string {
	return "AddActivityTask"
}

func (e *AddActivityTaskEvent) TargetEntity() *identity.Identity {
	return e.Identity
}

// Parse parses an AddActivityTaskEvent from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *AddActivityTaskEvent) Parse(span ptrace.Span) Event {
	var req matchingservice.AddActivityTaskRequest
	if !GetRequestPayload(span, &req) {
		return nil
	}

	if req.TaskQueue == nil || req.TaskQueue.Name == "" {
		return nil
	}

	// Compute identity - use scheduledEventID as activityID since ActivityID isn't in the request
	activityID := fmt.Sprintf("%d", req.ScheduledEventId)
	var ident *identity.Identity
	if activityID != "" {
		activityTaskID := identity.NewEntityIDFromType("ActivityTask", req.TaskQueue.Name+":"+activityID)
		ident = &identity.Identity{
			EntityID: activityTaskID,
			ParentID: nil,
		}
	}

	return &AddActivityTaskEvent{
		Request:   &req,
		Timestamp: span.StartTimestamp().AsTime(),
		Identity:  ident,
	}
}
