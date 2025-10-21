package entity

import (
	"time"

	"go.temporal.io/server/tools/umpire/event"
	"go.temporal.io/server/tools/umpire/identity"
)

var _ Entity = (*TaskQueue)(nil)

// TaskQueue represents a task queue entity.
// Locking is handled by the registry - individual methods should not lock.
type TaskQueue struct {
	Name              string
	LastEmptyPollTime time.Time
}

// NewTaskQueue creates a new TaskQueue entity.
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{}
}

func (tq *TaskQueue) Type() string {
	return "TaskQueue"
}

func (tq *TaskQueue) OnEvent(_ *identity.Identity, iter event.EventIterator) error {
	iter(func(ev event.Event) bool {
		switch e := ev.(type) {
		case *event.PollWorkflowTaskEvent:
			if tq.Name == "" && e.Request != nil && e.Request.PollRequest != nil && e.Request.PollRequest.TaskQueue != nil {
				tq.Name = e.Request.PollRequest.TaskQueue.Name
			}
			// Track empty polls (polls that returned nothing)
			if !e.TaskReturned {
				tq.LastEmptyPollTime = e.Timestamp
			}
		}
		return true
	})
	return nil
}

func (tq *TaskQueue) Verify() error {
	return nil
}
