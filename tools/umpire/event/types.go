package event

import (
	"go.temporal.io/server/tools/umpire/identity"
)

// EventIterator is a function that yields events one at a time.
// The yield function returns false to stop iteration early.
type EventIterator func(yield func(Event) bool)

// Event is the interface that all events must implement.
type Event interface {
	// EventType returns the event type name (e.g., "AddWorkflowTask")
	EventType() string

	// TargetEntity returns the identity of the entity this event targets.
	// Returns nil if the event doesn't target a specific entity.
	TargetEntity() *identity.Identity
}
