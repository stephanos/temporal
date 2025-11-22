package scorebook

import (
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/tools/umpire/roster/entities"
	rostertypes "go.temporal.io/server/tools/umpire/roster/types"
	moves "go.temporal.io/server/tools/umpire/scorebook/moves"
	"go.temporal.io/server/tools/umpire/scorebook/types"
)

// SpanIterator is a function that yields spans one at a time.
// The yield function returns false to stop iteration early.
type SpanIterator func(yield func(ptrace.Span) bool)

// Parser interface for events that can parse themselves from OTLP spans.
type MoveParser interface {
	Parse(span ptrace.Span) types.Move
}

// Importer converts OTLP spans into canonical moves.
type Importer struct {
	parsers map[string]MoveParser
}

// NewImporter creates a new importer with default parsers.
func NewImporter() *Importer {
	imp := &Importer{
		parsers: make(map[string]MoveParser),
	}

	// Register parsers for each span name
	imp.parsers["temporal.server.api.matchingservice.v1.MatchingService/AddWorkflowTask"] = &moves.AddWorkflowTask{}
	imp.parsers["temporal.server.api.matchingservice.v1.MatchingService/PollWorkflowTaskQueue"] = &moves.PollWorkflowTask{}
	imp.parsers["temporal.server.api.matchingservice.v1.MatchingService/AddActivityTask"] = &moves.AddActivityTask{}
	imp.parsers["temporal.server.api.matchingservice.v1.MatchingService/PollActivityTaskQueue"] = &moves.PollActivityTask{}
	imp.parsers["temporal.server.api.historyservice.v1.HistoryService/StartWorkflowExecution"] = &moves.StartWorkflow{}
	imp.parsers["temporal.server.api.historyservice.v1.HistoryService/RespondWorkflowTaskCompleted"] = &moves.RespondWorkflowTaskCompleted{}

	return imp
}

// ImportSpans converts spans to events.
func (imp *Importer) ImportSpans(iter SpanIterator) []types.Move {
	var events []types.Move

	iter(func(span ptrace.Span) bool {
		// Accept both Server and Client spans for matching service operations
		// Client spans represent inter-service RPC calls from the client perspective
		if span.Kind() != ptrace.SpanKindServer && span.Kind() != ptrace.SpanKindClient {
			return true
		}

		spanName := span.Name()
		parser, exists := imp.parsers[spanName]
		if !exists {
			return true
		}

		ev := parser.Parse(span)
		if ev != nil {
			events = append(events, ev)
		}

		return true
	})

	return events
}

// ImportRequest converts a gRPC request directly to a move.
// This is used by the gRPC interceptor to record moves synchronously.
// Returns nil if the request type is not recognized.
func (imp *Importer) ImportRequest(request any) types.Move {
	now := time.Now()

	switch req := request.(type) {
	case *matchingservice.AddWorkflowTaskRequest:
		return createAddWorkflowTaskMove(req, now)
	case *matchingservice.PollWorkflowTaskQueueRequest:
		return createPollWorkflowTaskMove(req, now)
	case *matchingservice.AddActivityTaskRequest:
		return createAddActivityTaskMove(req, now)
	case *matchingservice.PollActivityTaskQueueRequest:
		return createPollActivityTaskMove(req, now)
	case *historyservice.StartWorkflowExecutionRequest:
		return createStartWorkflowMove(req, now)
	case *historyservice.RespondWorkflowTaskCompletedRequest:
		return createRespondWorkflowTaskCompletedMove(req, now)
	default:
		return nil
	}
}

// Helper functions to create moves from requests

func createAddWorkflowTaskMove(req *matchingservice.AddWorkflowTaskRequest, timestamp time.Time) types.Move {
	if req.TaskQueue == nil || req.TaskQueue.Name == "" {
		return nil
	}

	var ident *rostertypes.Identity
	if req.Execution != nil && req.Execution.WorkflowId != "" && req.Execution.RunId != "" {
		workflowTaskID := rostertypes.NewEntityIDFromType(&entities.WorkflowTask{},
			req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
		taskQueueID := rostertypes.NewEntityIDFromType(&entities.TaskQueue{}, req.TaskQueue.Name)
		ident = &rostertypes.Identity{
			EntityID: workflowTaskID,
			ParentID: &taskQueueID,
		}
	}

	return &moves.AddWorkflowTask{
		Request:   req,
		Timestamp: timestamp,
		Identity:  ident,
	}
}

func createPollWorkflowTaskMove(req *matchingservice.PollWorkflowTaskQueueRequest, timestamp time.Time) types.Move {
	if req.PollRequest == nil || req.PollRequest.TaskQueue == nil || req.PollRequest.TaskQueue.Name == "" {
		return nil
	}

	taskQueue := req.PollRequest.TaskQueue.Name
	taskQueueID := rostertypes.NewEntityIDFromType(&entities.TaskQueue{}, taskQueue)
	ident := &rostertypes.Identity{
		EntityID: taskQueueID,
	}

	return &moves.PollWorkflowTask{
		Request:      req,
		Response:     nil, // Response not available in interceptor
		Timestamp:    timestamp,
		Identity:     ident,
		TaskReturned: false, // Not known yet at interceptor time
	}
}

func createAddActivityTaskMove(req *matchingservice.AddActivityTaskRequest, timestamp time.Time) types.Move {
	if req.TaskQueue == nil || req.TaskQueue.Name == "" {
		return nil
	}

	var ident *rostertypes.Identity
	if req.Execution != nil && req.Execution.WorkflowId != "" && req.Execution.RunId != "" {
		activityTaskID := rostertypes.NewEntityIDFromType(&entities.ActivityTask{},
			req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
		taskQueueID := rostertypes.NewEntityIDFromType(&entities.TaskQueue{}, req.TaskQueue.Name)
		ident = &rostertypes.Identity{
			EntityID: activityTaskID,
			ParentID: &taskQueueID,
		}
	}

	return &moves.AddActivityTask{
		Request:   req,
		Timestamp: timestamp,
		Identity:  ident,
	}
}

func createPollActivityTaskMove(req *matchingservice.PollActivityTaskQueueRequest, timestamp time.Time) types.Move {
	if req.PollRequest == nil || req.PollRequest.TaskQueue == nil || req.PollRequest.TaskQueue.Name == "" {
		return nil
	}

	taskQueue := req.PollRequest.TaskQueue.Name
	taskQueueID := rostertypes.NewEntityIDFromType(&entities.TaskQueue{}, taskQueue)
	ident := &rostertypes.Identity{
		EntityID: taskQueueID,
	}

	return &moves.PollActivityTask{
		Request:      req,
		Response:     nil, // Response not available in interceptor
		Timestamp:    timestamp,
		Identity:     ident,
		TaskReturned: false, // Not known yet at interceptor time
	}
}

func createStartWorkflowMove(req *historyservice.StartWorkflowExecutionRequest, timestamp time.Time) types.Move {
	if req.StartRequest == nil || req.StartRequest.WorkflowId == "" {
		return nil
	}

	workflowID := rostertypes.NewEntityIDFromType(&entities.Workflow{}, req.StartRequest.WorkflowId)
	ident := &rostertypes.Identity{
		EntityID: workflowID,
	}

	return &moves.StartWorkflow{
		Request:   req,
		Timestamp: timestamp,
		Identity:  ident,
	}
}

func createRespondWorkflowTaskCompletedMove(req *historyservice.RespondWorkflowTaskCompletedRequest, timestamp time.Time) types.Move {
	// No specific identity for this move type yet
	return &moves.RespondWorkflowTaskCompleted{
		Request:   req,
		Timestamp: timestamp,
	}
}
