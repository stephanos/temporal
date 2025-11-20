package types

// TypedEntity represents any entity that has a type.
// This interface is satisfied by the entity.Entity interface.
type TypedEntity interface {
	Type() string
}

// EntityType represents an entity type with a name.
type EntityType struct {
	// Name is the entity type name (e.g., "WorkflowTask", "ActivityTask")
	Name string
}

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
		Type: EntityType{Name: e.Type()},
		ID:   id,
	}
}

// NewEntityIDFromType creates an EntityID from a type name.
// Use this only when you don't have an entity instance available (e.g., in event routing).
// Prefer NewEntityID when possible for better type-safety.
func NewEntityIDFromType(typeName string, id string) EntityID {
	return EntityID{
		Type: EntityType{Name: typeName},
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
