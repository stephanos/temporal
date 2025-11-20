package event

import (
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/tools/umpire/identity"
)

// RespondWorkflowTaskCompletedEvent represents a workflow task completion response.
type RespondWorkflowTaskCompletedEvent struct {
	Request   *historyservice.RespondWorkflowTaskCompletedRequest
	Response  *historyservice.RespondWorkflowTaskCompletedResponse
	Timestamp time.Time
	Identity  *identity.Identity
}

func (e *RespondWorkflowTaskCompletedEvent) EventType() string {
	return "RespondWorkflowTaskCompleted"
}

func (e *RespondWorkflowTaskCompletedEvent) TargetEntity() *identity.Identity {
	return e.Identity
}

// Parse parses a RespondWorkflowTaskCompletedEvent from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *RespondWorkflowTaskCompletedEvent) Parse(span ptrace.Span) Event {
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
	var ident *identity.Identity

	return &RespondWorkflowTaskCompletedEvent{
		Request:   &req,
		Response:  &resp,
		Timestamp: span.StartTimestamp().AsTime(),
		Identity:  ident,
	}
}
