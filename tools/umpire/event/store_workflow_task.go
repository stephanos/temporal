package event

import (
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/tools/umpire/identity"
)

// StoreWorkflowTaskEvent represents a workflow task being stored to persistence.
// This happens when a task is removed from matching (e.g., via admin API) and
// persisted to the database. The task is no longer in the matching service
// but still exists in persistence.
type StoreWorkflowTaskEvent struct {
	Timestamp  time.Time
	TaskQueue  string
	Identity   *identity.Identity
	WorkflowID string
	RunID      string
}

func (e *StoreWorkflowTaskEvent) EventType() string {
	return "StoreWorkflowTask"
}

func (e *StoreWorkflowTaskEvent) TargetEntity() *identity.Identity {
	return e.Identity
}

// Parse parses a StoreWorkflowTaskEvent from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *StoreWorkflowTaskEvent) Parse(span ptrace.Span) Event {
	// StoreWorkflowTaskEvent is not directly generated from OTLP spans
	// It would need to be generated from admin API calls or other sources
	// For now, return nil
	return nil
}
