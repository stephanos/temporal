package moves

import (
	"time"

	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/api/matchingservice/v1"
	rostertypes "go.temporal.io/server/tools/umpire/roster/types"
	scorebooktypes "go.temporal.io/server/tools/umpire/scorebook/types"
)

// FromRequest creates a move from a gRPC request.
// Returns nil if the request type is not recognized.
func FromRequest(request any) scorebooktypes.Move {
	now := time.Now()

	switch req := request.(type) {
	case *matchingservice.AddWorkflowTaskRequest:
		return CreateAddWorkflowTaskMove(req, now)

	case *matchingservice.PollWorkflowTaskQueueRequest:
		return CreatePollWorkflowTaskMove(req, now)

	case *matchingservice.AddActivityTaskRequest:
		return CreateAddActivityTaskMove(req, now)

	case *matchingservice.PollActivityTaskQueueRequest:
		return CreatePollActivityTaskMove(req, now)

	case *historyservice.StartWorkflowExecutionRequest:
		return CreateStartWorkflowMove(req, now)

	case *historyservice.RespondWorkflowTaskCompletedRequest:
		return CreateRespondWorkflowTaskCompletedMove(req, now)

	default:
		return nil
	}
}

// Helper functions to create moves from requests

func CreateAddWorkflowTaskMove(req *matchingservice.AddWorkflowTaskRequest, timestamp time.Time) scorebooktypes.Move {
	if req.TaskQueue == nil || req.TaskQueue.Name == "" {
		return nil
	}

	var ident *rostertypes.Identity
	if req.Execution != nil && req.Execution.WorkflowId != "" && req.Execution.RunId != "" {
		workflowTaskID := rostertypes.NewEntityIDFromType(rostertypes.WorkflowTaskType,
			req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
		taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, req.TaskQueue.Name)
		ident = &rostertypes.Identity{
			EntityID: workflowTaskID,
			ParentID: &taskQueueID,
		}
	}

	return &AddWorkflowTask{
		Request:   req,
		Timestamp: timestamp,
		Identity:  ident,
	}
}

func CreatePollWorkflowTaskMove(req *matchingservice.PollWorkflowTaskQueueRequest, timestamp time.Time) scorebooktypes.Move {
	if req.PollRequest == nil || req.PollRequest.TaskQueue == nil || req.PollRequest.TaskQueue.Name == "" {
		return nil
	}

	taskQueue := req.PollRequest.TaskQueue.Name
	taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, taskQueue)
	ident := &rostertypes.Identity{
		EntityID: taskQueueID,
	}

	return &PollWorkflowTask{
		Request:      req,
		Response:     nil, // Response not available in interceptor
		Timestamp:    timestamp,
		Identity:     ident,
		TaskReturned: false, // Not known yet at interceptor time
	}
}

func CreateAddActivityTaskMove(req *matchingservice.AddActivityTaskRequest, timestamp time.Time) scorebooktypes.Move {
	if req.TaskQueue == nil || req.TaskQueue.Name == "" {
		return nil
	}

	var ident *rostertypes.Identity
	if req.Execution != nil && req.Execution.WorkflowId != "" && req.Execution.RunId != "" {
		activityTaskID := rostertypes.NewEntityIDFromType(rostertypes.ActivityTaskType,
			req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
		taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, req.TaskQueue.Name)
		ident = &rostertypes.Identity{
			EntityID: activityTaskID,
			ParentID: &taskQueueID,
		}
	}

	return &AddActivityTask{
		Request:   req,
		Timestamp: timestamp,
		Identity:  ident,
	}
}

func CreatePollActivityTaskMove(req *matchingservice.PollActivityTaskQueueRequest, timestamp time.Time) scorebooktypes.Move {
	if req.PollRequest == nil || req.PollRequest.TaskQueue == nil || req.PollRequest.TaskQueue.Name == "" {
		return nil
	}

	taskQueue := req.PollRequest.TaskQueue.Name
	taskQueueID := rostertypes.NewEntityIDFromType(rostertypes.TaskQueueType, taskQueue)
	ident := &rostertypes.Identity{
		EntityID: taskQueueID,
	}

	return &PollActivityTask{
		Request:      req,
		Response:     nil, // Response not available in interceptor
		Timestamp:    timestamp,
		Identity:     ident,
		TaskReturned: false, // Not known yet at interceptor time
	}
}

func CreateStartWorkflowMove(req *historyservice.StartWorkflowExecutionRequest, timestamp time.Time) scorebooktypes.Move {
	if req.StartRequest == nil || req.StartRequest.WorkflowId == "" {
		return nil
	}

	workflowID := rostertypes.NewEntityIDFromType(rostertypes.WorkflowType, req.StartRequest.WorkflowId)
	ident := &rostertypes.Identity{
		EntityID: workflowID,
	}

	return &StartWorkflow{
		Request:   req,
		Timestamp: timestamp,
		Identity:  ident,
	}
}

func CreateRespondWorkflowTaskCompletedMove(req *historyservice.RespondWorkflowTaskCompletedRequest, timestamp time.Time) scorebooktypes.Move {
	// No specific identity for this move type yet
	return &RespondWorkflowTaskCompleted{
		Request:   req,
		Timestamp: timestamp,
	}
}
