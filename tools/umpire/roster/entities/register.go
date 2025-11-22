package entities

import (
	"go.temporal.io/server/tools/umpire/roster"
	"go.temporal.io/server/tools/umpire/scorebook/moves"
)

// RegisterDefaultEntities registers the default entity types with a registry.
// This is separated from the roster package to avoid import cycles.
func RegisterDefaultEntities(r *roster.Registry) {
	r.RegisterEntity(NewTaskQueue(), func() roster.Entity {
		return NewTaskQueue()
	}, &moves.AddWorkflowTask{}, &moves.PollWorkflowTask{}, &moves.AddActivityTask{}, &moves.PollActivityTask{})

	r.RegisterEntity(NewWorkflowTask(), func() roster.Entity {
		return NewWorkflowTask()
	}, &moves.AddWorkflowTask{}, &moves.PollWorkflowTask{}, &moves.StoreWorkflowTask{})
}
