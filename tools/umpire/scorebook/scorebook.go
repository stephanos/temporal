package scorebook

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/api/matchingservice/v1"
	rostertypes "go.temporal.io/server/tools/umpire/roster/types"
	"go.temporal.io/server/tools/umpire/scorebook/moves"
	"go.temporal.io/server/tools/umpire/scorebook/types"
)

// SpanIterator is a function that yields spans one at a time.
// The yield function returns false to stop iteration early.
type SpanIterator func(yield func(ptrace.Span) bool)

// Importer converts gRPC requests and OTEL spans into canonical moves.
type Importer struct {
	// Empty for now, may add configuration options later
}

// NewImporter creates a new importer.
func NewImporter() *Importer {
	return &Importer{}
}

// ImportRequest converts a gRPC request directly to a move.
// This is used by the gRPC interceptor to record moves synchronously.
// Returns nil if the request type is not recognized.
func (imp *Importer) ImportRequest(request any) types.Move {
	return FromRequest(request)
}

// ImportSpans converts OTEL spans to events.
// This is primarily for non-RPC OTEL events like "workflow locked/unlocked".
// RPC calls are now recorded directly via the gRPC interceptor.
func (imp *Importer) ImportSpans(iter SpanIterator) []types.Move {
	var events []types.Move

	iter(func(span ptrace.Span) bool {
		// For now, we don't process any OTEL spans as moves.
		// RPC calls are handled by the gRPC interceptor.
		// Other OTEL events (like workflow locked/unlocked) would be handled here
		// if needed in the future.
		return true
	})

	return events
}

// FromRequest creates a move from a gRPC request.
// Returns nil if the request type is not recognized or invalid.
func FromRequest(request any) types.Move {
	switch req := request.(type) {
	case *matchingservice.AddWorkflowTaskRequest:
		move := &moves.AddWorkflowTask{}
		if req.TaskQueue != nil && req.TaskQueue.Name != "" &&
			req.Execution != nil && req.Execution.WorkflowId != "" && req.Execution.RunId != "" {
			workflowTaskID := rostertypes.NewEntityIDFromType(rostertypes.WorkflowTaskType,
				req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
			taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, req.TaskQueue.Name)
			move.Identity = &rostertypes.Identity{
				EntityID: workflowTaskID,
				ParentID: &taskQueueID,
			}
		}
		if err := move.Parse(req); err != nil {
			return nil
		}
		return move

	case *matchingservice.PollWorkflowTaskQueueRequest:
		move := &moves.PollWorkflowTask{}
		if req.PollRequest != nil && req.PollRequest.TaskQueue != nil && req.PollRequest.TaskQueue.Name != "" {
			taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, req.PollRequest.TaskQueue.Name)
			move.Identity = &rostertypes.Identity{
				EntityID: taskQueueID,
			}
		}
		if err := move.Parse(req); err != nil {
			return nil
		}
		return move

	case *matchingservice.AddActivityTaskRequest:
		move := &moves.AddActivityTask{}
		if req.TaskQueue != nil && req.TaskQueue.Name != "" &&
			req.Execution != nil && req.Execution.WorkflowId != "" && req.Execution.RunId != "" {
			activityTaskID := rostertypes.NewEntityIDFromType(rostertypes.ActivityTaskType,
				req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
			taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, req.TaskQueue.Name)
			move.Identity = &rostertypes.Identity{
				EntityID: activityTaskID,
				ParentID: &taskQueueID,
			}
		}
		if err := move.Parse(req); err != nil {
			return nil
		}
		return move

	case *matchingservice.PollActivityTaskQueueRequest:
		move := &moves.PollActivityTask{}
		if req.PollRequest != nil && req.PollRequest.TaskQueue != nil && req.PollRequest.TaskQueue.Name != "" {
			taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, req.PollRequest.TaskQueue.Name)
			move.Identity = &rostertypes.Identity{
				EntityID: taskQueueID,
			}
		}
		if err := move.Parse(req); err != nil {
			return nil
		}
		return move

	case *historyservice.StartWorkflowExecutionRequest:
		move := &moves.StartWorkflow{}
		if req.StartRequest != nil && req.StartRequest.WorkflowId != "" {
			workflowID := rostertypes.NewEntityIDFromType(rostertypes.WorkflowType, req.StartRequest.WorkflowId)
			move.Identity = &rostertypes.Identity{
				EntityID: workflowID,
			}
		}
		if err := move.Parse(req); err != nil {
			return nil
		}
		return move

	case *historyservice.RespondWorkflowTaskCompletedRequest:
		move := &moves.RespondWorkflowTaskCompleted{}
		if err := move.Parse(req); err != nil {
			return nil
		}
		return move

	default:
		return nil
	}
}
