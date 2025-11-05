package event

import (
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/tools/umpire/identity"
)

// StartWorkflowEvent represents a workflow being started.
type StartWorkflowEvent struct {
	Request   *historyservice.StartWorkflowExecutionRequest
	Timestamp time.Time
	Identity  *identity.Identity
}

func (e *StartWorkflowEvent) EventType() string {
	return "StartWorkflow"
}

func (e *StartWorkflowEvent) TargetEntity() *identity.Identity {
	return e.Identity
}

// Parse parses a StartWorkflowEvent from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *StartWorkflowEvent) Parse(span ptrace.Span) Event {
	var req historyservice.StartWorkflowExecutionRequest
	if !GetRequestPayload(span, &req) {
		return nil
	}

	if req.StartRequest == nil || req.StartRequest.WorkflowId == "" {
		return nil
	}

	// Extract workflow ID from the request
	workflowID := req.StartRequest.WorkflowId

	// Create identity for this workflow
	workflowEntityID := identity.NewEntityIDFromType("Workflow",
		workflowID)
	ident := &identity.Identity{
		EntityID: workflowEntityID,
		ParentID: nil,
	}

	return &StartWorkflowEvent{
		Request:   &req,
		Timestamp: span.StartTimestamp().AsTime(),
		Identity:  ident,
	}
}
