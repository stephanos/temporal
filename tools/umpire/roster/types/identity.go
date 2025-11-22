package types

// TypedEntity represents any entity that has a type.
// This interface is satisfied by the entity.Entity interface.
type TypedEntity interface {
	Type() EntityType
}

// EntityType is a strongly-typed entity type identifier.
// Each entity defines its type as a constant.
type EntityType string

// Common entity type constants
// These are defined here to avoid import cycles between moves and entities.
const (
	ActivityTaskType     EntityType = "ActivityTask"
	TaskQueueType        EntityType = "TaskQueue"
	WorkflowType         EntityType = "Workflow"
	WorkflowTaskType     EntityType = "WorkflowTask"
	WorkflowExecutionType EntityType = "WorkflowExecution"
	NamespaceType        EntityType = "Namespace"
)

// EntityID uniquely identifies an entity within its type.
type EntityID struct {
	// Type is the entity type
	Type EntityType
	// ID is the unique identifier within the type
	ID string
}

// NewEntityID creates an EntityID from a typed entity instance.
// This ensures type-safety by deriving the type from the entity itself.
func NewEntityID(e TypedEntity, id string) EntityID {
	return EntityID{
		Type: e.Type(),
		ID:   id,
	}
}

// NewEntityIDFromType creates an EntityID from an entity type and ID string.
// Use this when you know the entity type constant (e.g., entities.WorkflowType).
// Example: NewEntityIDFromType(entities.WorkflowType, "workflow-123")
func NewEntityIDFromType(entityType EntityType, id string) EntityID {
	return EntityID{
		Type: entityType,
		ID:   id,
	}
}

// Identity represents the full identity of an entity, including its parent.
type Identity struct {
	// EntityID is this entity's identifier
	EntityID EntityID
	// ParentID is the parent entity's identifier, or nil for root entities
	ParentID *EntityID
}
