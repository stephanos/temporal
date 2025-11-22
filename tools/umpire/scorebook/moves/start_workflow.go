package moves

import (
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/historyservice/v1"
	rostertypes "go.temporal.io/server/tools/umpire/roster/types"
	scorebooktypes "go.temporal.io/server/tools/umpire/scorebook/types"
)

// StartWorkflow represents a workflow being started.
type StartWorkflow struct {
	Request   *historyservice.StartWorkflowExecutionRequest
	Timestamp time.Time
	Identity  *rostertypes.Identity
}

func (e *StartWorkflow) MoveType() string {
	return "StartWorkflow"
}

func (e *StartWorkflow) TargetEntity() *rostertypes.Identity {
	return e.Identity
}

// Parse parses a StartWorkflow from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *StartWorkflow) Parse(span ptrace.Span) scorebooktypes.Move {
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
	workflowEntityID := rostertypes.NewEntityIDFromType(rostertypes.WorkflowType, workflowID)
	ident := &rostertypes.Identity{
		EntityID: workflowEntityID,
		ParentID: nil,
	}

	return &StartWorkflow{
		Request:   &req,
		Timestamp: span.StartTimestamp().AsTime(),
		Identity:  ident,
	}
}
