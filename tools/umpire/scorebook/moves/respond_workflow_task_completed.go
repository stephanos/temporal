package moves

import (
	scorebooktypes "go.temporal.io/server/tools/umpire/scorebook/types"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/historyservice/v1"
	rostertypes "go.temporal.io/server/tools/umpire/roster/types"
)

// RespondWorkflowTaskCompleted represents a workflow task completion response.
type RespondWorkflowTaskCompleted struct {
	Request   *historyservice.RespondWorkflowTaskCompletedRequest
	Response  *historyservice.RespondWorkflowTaskCompletedResponse
	Timestamp time.Time
	Identity  *rostertypes.Identity
}

func (e *RespondWorkflowTaskCompleted) MoveType() string {
	return "RespondWorkflowTaskCompleted"
}

func (e *RespondWorkflowTaskCompleted) TargetEntity() *rostertypes.Identity {
	return e.Identity
}

// Parse parses a RespondWorkflowTaskCompleted from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *RespondWorkflowTaskCompleted) Parse(span ptrace.Span) scorebooktypes.Move {
	var req historyservice.RespondWorkflowTaskCompletedRequest
	if !GetRequestPayload(span, &req) {
		return nil
	}

	if req.CompleteRequest == nil {
		return nil
	}

	// Try to get response payload
	var resp historyservice.RespondWorkflowTaskCompletedResponse
	_ = GetResponsePayload(span, &resp)

	// The task completion doesn't directly contain the workflow ID,
	// but the event system will route it to the appropriate Workflow entity
	// based on what entities exist. For now, return nil identity which means
	// this event won't be automatically routed - models will need to handle it.
	var ident *rostertypes.Identity

	return &RespondWorkflowTaskCompleted{
		Request:   &req,
		Response:  &resp,
		Timestamp: span.StartTimestamp().AsTime(),
		Identity:  ident,
	}
}
