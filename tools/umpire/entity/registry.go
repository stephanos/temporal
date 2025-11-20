package entity

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tidwall/buntdb"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/tools/umpire/event"
	"go.temporal.io/server/tools/umpire/identity"
)

// Entity is the interface that all entities must implement.
type Entity interface {
	// Type returns the entity type identifier (e.g., "WorkflowTask")
	Type() string

	// OnEvent receives the entity's identity and a batch of events via an iterator.
	// The entity should update its internal state based on the events.
	OnEvent(identity *identity.Identity, iter event.EventIterator) error

	// Verify checks if the entity is in a valid state.
	// This is called after the entity receives events.
	// It should return an error or use softassert to report issues.
	Verify() error
}

// Factory creates a new entity instance.
type Factory func() Entity

// Registry manages the entity tree and routes events to entities.
// All entities and events are persisted to buntdb.
type Registry struct {
	logger log.Logger
	db     *buntdb.DB
	mu     sync.RWMutex

	// factories maps entity type names to their factory functions
	factories map[string]Factory

	// entities stores all entities by their full identity
	entities map[string]Entity

	// children maps parent entity IDs to their child entity IDs
	children map[string][]string

	// subscriptions maps event types to entity types that subscribe to them
	subscriptions map[string][]string
}

// NewEntityRegistry creates a new entity registry with buntdb storage.
// If dbPath is empty, uses in-memory storage (":memory:").
func NewEntityRegistry(logger log.Logger, dbPath string) (*Registry, error) {
	if dbPath == "" {
		dbPath = ":memory:"
	}

	db, err := buntdb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open buntdb: %w", err)
	}

	r := &Registry{
		logger:        logger,
		db:            db,
		factories:     make(map[string]Factory),
		entities:      make(map[string]Entity),
		children:      make(map[string][]string),
		subscriptions: make(map[string][]string),
	}

	return r, nil
}

// RegisterEntity registers an entity type with its factory using a typed reference.
// The entityRef parameter is used only to get the entity type via its Type() method.
func (r *Registry) RegisterEntity(entityRef Entity, factory Factory, subscribesTo ...event.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entityType := entityRef.Type()
	r.factories[entityType] = factory

	// Register subscriptions
	for _, event := range subscribesTo {
		eventType := event.EventType()
		r.subscriptions[eventType] = append(r.subscriptions[eventType], entityType)
	}
}

// RouteEvents routes a batch of events to the appropriate entities.
// Events are grouped by target entity and delivered as an iterator.
func (r *Registry) RouteEvents(ctx context.Context, events []event.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Group events by target entity
	eventsByEntity := make(map[string][]event.Event)
	changedEntities := make(map[string]Entity)
	entityIdentities := make(map[string]*identity.Identity)

	for _, ev := range events {
		targetIdentity := ev.TargetEntity()

		// Find or create the target entity
		ent, err := r.getOrCreateEntity(targetIdentity)
		if err != nil {
			r.logger.Warn("registry: failed to get or create entity",
				tag.NewStringTag("event", ev.EventType()),
				tag.Error(err))
			continue
		}

		if ent == nil {
			// No target entity, skip this event
			continue
		}

		// Use the event's target identity as the key
		entityKey := identityKey(targetIdentity)
		eventsByEntity[entityKey] = append(eventsByEntity[entityKey], ev)
		changedEntities[entityKey] = ent
		entityIdentities[entityKey] = targetIdentity
	}

	// Deliver events to entities
	for entityKey, entityEvents := range eventsByEntity {
		entity := changedEntities[entityKey]
		entityIdentity := entityIdentities[entityKey]

		// Create iterator
		iter := func(yield func(event.Event) bool) {
			for _, e := range entityEvents {
				if !yield(e) {
					return
				}
			}
		}

		// Deliver events with identity
		if err := entity.OnEvent(entityIdentity, iter); err != nil {
			r.logger.Warn("registry: entity OnEvent error",
				tag.NewStringTag("entityType", entity.Type()),
				tag.Error(err))
		}
	}

	// Verify all changed entities
	for _, entity := range changedEntities {
		if err := entity.Verify(); err != nil {
			r.logger.Warn("registry: entity verify error",
				tag.NewStringTag("entityType", entity.Type()),
				tag.Error(err))
		}
	}

	// Persist all changed entities and events to buntdb
	for entityKey, entity := range changedEntities {
		entityIdentity := entityIdentities[entityKey]
		if err := r.saveEntity(entity, entityIdentity); err != nil {
			r.logger.Warn("registry: failed to save entity",
				tag.NewStringTag("entityType", entity.Type()),
				tag.Error(err))
		}
	}

	// Persist all events
	for _, event := range events {
		if err := r.saveEvent(event); err != nil {
			r.logger.Warn("registry: failed to save event",
				tag.NewStringTag("eventType", event.EventType()),
				tag.Error(err))
		}
	}

	return nil
}

// getOrCreateEntity finds an existing entity or creates a new one.
func (r *Registry) getOrCreateEntity(ident *identity.Identity) (Entity, error) {
	if ident == nil {
		return nil, nil
	}

	key := identityKey(ident)

	// Check if entity already exists
	if entity, exists := r.entities[key]; exists {
		return entity, nil
	}

	// If the entity has a parent, ensure the parent exists first
	if ident.ParentID != nil {
		parentIdentity := &identity.Identity{
			EntityID: *ident.ParentID,
			ParentID: nil, // Parents are assumed to be root entities for now
		}
		_, err := r.getOrCreateEntity(parentIdentity)
		if err != nil {
			return nil, fmt.Errorf("failed to create parent entity: %w", err)
		}
	}

	// Create new entity
	factory, exists := r.factories[ident.EntityID.Type.Name]
	if !exists {
		return nil, fmt.Errorf("no factory registered for entity type: %s", ident.EntityID.Type.Name)
	}

	entity := factory()

	// Store the entity
	r.entities[key] = entity

	// Track parent-child relationship
	if ident.ParentID != nil {
		parentKey := entityIDKey(ident.ParentID)
		r.children[parentKey] = append(r.children[parentKey], key)
	}

	return entity, nil
}

// QueryEntities returns all entities of a given type.
// The entityRef parameter is used only to get the entity type via its Type() method.
func (r *Registry) QueryEntities(entityRef interface{ Type() string }) []interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entityType := entityRef.Type()
	var result []interface{}
	for _, e := range r.entities {
		if e.Type() == entityType {
			result = append(result, e)
		}
	}
	return result
}

// identityKey creates a string key from an Identity.
func identityKey(ident *identity.Identity) string {
	if ident == nil {
		return ""
	}
	key := fmt.Sprintf("%s:%s", ident.EntityID.Type.Name, ident.EntityID.ID)
	if ident.ParentID != nil {
		key = fmt.Sprintf("%s@%s:%s", key, ident.ParentID.Type.Name, ident.ParentID.ID)
	}
	return key
}

// entityIDKey creates a string key from an EntityID.
func entityIDKey(id *identity.EntityID) string {
	if id == nil {
		return ""
	}
	return fmt.Sprintf("%s:%s", id.Type.Name, id.ID)
}

// Close closes the registry and its underlying database.
func (r *Registry) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// saveEntity persists an entity to buntdb.
func (r *Registry) saveEntity(e Entity, ident *identity.Identity) error {
	if ident == nil {
		return nil // Skip entities without identity
	}

	key := "entity:" + identityKey(ident)

	// For now, store a minimal representation
	// In a real implementation, entities would need to be serializable
	data := map[string]interface{}{
		"type": e.Type(),
		"id":   ident.EntityID.ID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal entity: %w", err)
	}

	return r.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(jsonData), nil)
		return err
	})
}

// saveEvent persists an event to buntdb, keyed by the entity it targets.
func (r *Registry) saveEvent(ev event.Event) error {
	identity := ev.TargetEntity()
	if identity == nil {
		return nil // Skip events without target
	}

	entityKey := identityKey(identity)
	eventKey := fmt.Sprintf("event:%s:%d", entityKey, r.getEventCounter(entityKey))

	data := map[string]interface{}{
		"type":      ev.EventType(),
		"entityKey": entityKey,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return r.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(eventKey, string(jsonData), nil)
		return err
	})
}

// getEventCounter returns the next event counter for an entity.
func (r *Registry) getEventCounter(entityKey string) int64 {
	var counter int64
	_ = r.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("", func(key, value string) bool {
			// Count existing events for this entity
			prefix := fmt.Sprintf("event:%s:", entityKey)
			if len(key) > len(prefix) && key[:len(prefix)] == prefix {
				counter++
			}
			return true
		})
		return nil
	})
	return counter
}

// RegisterDefaultEntities registers the default entity types with a registry.
func RegisterDefaultEntities(r *Registry) {
	r.RegisterEntity(NewTaskQueue(), func() Entity {
		return NewTaskQueue()
	}, &event.AddWorkflowTaskEvent{}, &event.PollWorkflowTaskEvent{}, &event.AddActivityTaskEvent{}, &event.PollActivityTaskEvent{})

	r.RegisterEntity(NewWorkflowTask(), func() Entity {
		return NewWorkflowTask()
	}, &event.AddWorkflowTaskEvent{}, &event.PollWorkflowTaskEvent{}, &event.StoreWorkflowTaskEvent{})
}
