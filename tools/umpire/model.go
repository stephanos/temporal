package umpire

import (
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/tools/umpire/model"
)

// Deps bundles dependencies provided to models at initialization.
type Deps struct {
	// Registry provides access to the entity registry for querying entities.
	Registry model.EntityRegistry
	// Logger to use for logs and soft assertions.
	Logger log.Logger
}

// GetLogger returns the logger for interface-based dependency injection.
func (d Deps) GetLogger() log.Logger {
	return d.Logger
}

// GetRegistry returns the registry for interface-based dependency injection.
func (d Deps) GetRegistry() model.EntityRegistry {
	return d.Registry
}
