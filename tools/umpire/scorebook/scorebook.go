package scorebook

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
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
	return moves.FromRequest(request)
}
