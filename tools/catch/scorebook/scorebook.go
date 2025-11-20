package scorebook

import (
	"strings"
	"time"

	"go.temporal.io/server/tools/catch/scorebook/types"
	"go.opentelemetry.io/collector/pdata/ptrace"
	moves "go.temporal.io/server/tools/catch/scorebook/moves"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/api/historyservice/v1"
	rostertypes "go.temporal.io/server/tools/catch/roster/types"
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
// The method parameter should be the full gRPC method name (e.g., "/temporal.api.matchingservice.v1.MatchingService/AddWorkflowTask").
// Returns nil if no parser is registered for this method.
func (imp *Importer) ImportRequest(method string, request any) types.Move {
	// Convert gRPC method format to span name format
	// "/temporal.api.matchingservice.v1.MatchingService/AddWorkflowTask"
	// -> "temporal.server.api.matchingservice.v1.MatchingService/AddWorkflowTask"
	spanName := convertGRPCMethodToSpanName(method)

	// Create move directly from the request
	return createMoveFromRequest(spanName, request)
}

// convertGRPCMethodToSpanName converts a gRPC method name to the span name format.
// gRPC format: "/temporal.server.api.matchingservice.v1.MatchingService/AddWorkflowTask"
// Span format: "temporal.server.api.matchingservice.v1.MatchingService/AddWorkflowTask"
func convertGRPCMethodToSpanName(method string) string {
	// Remove leading "/"
	return strings.TrimPrefix(method, "/")
}

// createMoveFromRequest creates a move from a gRPC request.
// This is a type switch based on the request type.
func createMoveFromRequest(spanName string, request any) types.Move {
	now := time.Now()

	switch spanName {
	case "temporal.server.api.matchingservice.v1.MatchingService/AddWorkflowTask":
		if req, ok := request.(*matchingservice.AddWorkflowTaskRequest); ok {
			return createAddWorkflowTaskMove(req, now)
		}

	case "temporal.server.api.matchingservice.v1.MatchingService/PollWorkflowTaskQueue":
		if req, ok := request.(*matchingservice.PollWorkflowTaskQueueRequest); ok {
			return createPollWorkflowTaskMove(req, now)
		}

	case "temporal.server.api.matchingservice.v1.MatchingService/AddActivityTask":
		if req, ok := request.(*matchingservice.AddActivityTaskRequest); ok {
			return createAddActivityTaskMove(req, now)
		}

	case "temporal.server.api.matchingservice.v1.MatchingService/PollActivityTaskQueue":
		if req, ok := request.(*matchingservice.PollActivityTaskQueueRequest); ok {
			return createPollActivityTaskMove(req, now)
		}

	case "temporal.server.api.historyservice.v1.HistoryService/StartWorkflowExecution":
		if req, ok := request.(*historyservice.StartWorkflowExecutionRequest); ok {
			return createStartWorkflowMove(req, now)
		}

	case "temporal.server.api.historyservice.v1.HistoryService/RespondWorkflowTaskCompleted":
		if req, ok := request.(*historyservice.RespondWorkflowTaskCompletedRequest); ok {
			return createRespondWorkflowTaskCompletedMove(req, now)
		}
	}

	return nil
}

// Helper functions to create moves from requests

func createAddWorkflowTaskMove(req *matchingservice.AddWorkflowTaskRequest, timestamp time.Time) types.Move {
	if req.TaskQueue == nil || req.TaskQueue.Name == "" {
		return nil
	}

	var ident *rostertypes.Identity
	if req.Execution != nil && req.Execution.WorkflowId != "" && req.Execution.RunId != "" {
		workflowTaskID := rostertypes.NewEntityIDFromType("WorkflowTask",
			req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
		taskQueueID := rostertypes.NewEntityIDFromType("TaskQueue", req.TaskQueue.Name)
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
	taskQueueID := rostertypes.NewEntityIDFromType("TaskQueue", taskQueue)
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
		activityTaskID := rostertypes.NewEntityIDFromType("ActivityTask",
			req.TaskQueue.Name+":"+req.Execution.WorkflowId+":"+req.Execution.RunId)
		taskQueueID := rostertypes.NewEntityIDFromType("TaskQueue", req.TaskQueue.Name)
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
	taskQueueID := rostertypes.NewEntityIDFromType("TaskQueue", taskQueue)
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

	workflowID := rostertypes.NewEntityIDFromType("Workflow", req.StartRequest.WorkflowId)
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
