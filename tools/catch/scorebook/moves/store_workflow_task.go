package moves

import (
	scorebooktypes "go.temporal.io/server/tools/catch/scorebook/types"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	rostertypes "go.temporal.io/server/tools/catch/roster/types"
)

// StoreWorkflowTask represents a workflow task being stored to persistence.
// This happens when a task is removed from matching (e.g., via admin API) and
// persisted to the database. The task is no longer in the matching service
// but still exists in persistence.
type StoreWorkflowTask struct {
	Timestamp  time.Time
	TaskQueue  string
	Identity   *rostertypes.Identity
	WorkflowID string
	RunID      string
}

func (e *StoreWorkflowTask) MoveType() string {
	return "StoreWorkflowTask"
}

func (e *StoreWorkflowTask) TargetEntity() *rostertypes.Identity {
	return e.Identity
}

// Parse parses a StoreWorkflowTask from an OTLP span.
// Returns nil if the span doesn't contain the required attributes.
func (e *StoreWorkflowTask) Parse(span ptrace.Span) scorebooktypes.Move {
	// StoreWorkflowTask is not directly generated from OTLP spans
	// It would need to be generated from admin API calls or other sources
	// For now, return nil
	return nil
}
