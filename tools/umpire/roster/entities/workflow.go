package entities

import (
	"context"
	"fmt"
	"time"

	"github.com/looplab/fsm"
	"go.temporal.io/server/tools/umpire/roster"
	rostertypes "go.temporal.io/server/tools/umpire/roster/types"
	"go.temporal.io/server/tools/umpire/scorebook/moves"
	scorebooktypes "go.temporal.io/server/tools/umpire/scorebook/types"
)

var _ roster.Entity = (*Workflow)(nil)


// Workflow represents a workflow execution entity.
// Tracks whether a workflow has been started and whether task completion was received.
// Locking is handled by the registry - individual methods should not lock.
type Workflow struct {
	WorkflowID  string
	NamespaceID string
	FSM         *fsm.FSM
	StartedAt   time.Time // When workflow was started
	CompletedAt time.Time // When workflow completion response was received
	LastSeenAt  time.Time // Last time any event was received for this workflow
}

// NewWorkflow creates a new Workflow entity.
func NewWorkflow() *Workflow {
	wf := &Workflow{}
	wf.FSM = fsm.NewFSM(
		"created",
		fsm.Events{
			{Name: "start", Src: []string{"created"}, Dst: "started"},
			{Name: "complete", Src: []string{"started"}, Dst: "completed"},
		},
		fsm.Callbacks{},
	)
	return wf
}

func (wf *Workflow) Type() rostertypes.EntityType {
	return rostertypes.WorkflowType
}

func (wf *Workflow) OnEvent(_ *rostertypes.Identity, iter scorebooktypes.MoveIterator) error {
	iter(func(ev scorebooktypes.Move) bool {
		switch e := ev.(type) {
		case *moves.StartWorkflow:
			if wf.WorkflowID == "" && e.Request != nil && e.Request.StartRequest != nil {
				wf.WorkflowID = e.Request.StartRequest.WorkflowId
				wf.NamespaceID = e.Request.NamespaceId
			}
			// Update on first start event
			if wf.FSM.Can("start") {
				_ = wf.FSM.Event(context.Background(), "start")
				wf.StartedAt = e.Timestamp
			}
			wf.LastSeenAt = e.Timestamp

		case *moves.RespondWorkflowTaskCompleted:
			// Mark workflow as completed when we see a task completion response
			if wf.FSM.Can("complete") {
				_ = wf.FSM.Event(context.Background(), "complete")
				wf.CompletedAt = e.Timestamp
			}
			wf.LastSeenAt = e.Timestamp
		}
		return true
	})

	return nil
}

func (wf *Workflow) String() string {
	return fmt.Sprintf("Workflow{workflowID=%s, state=%s}",
		wf.WorkflowID, wf.FSM.Current())
}
