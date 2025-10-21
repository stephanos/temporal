package umpire

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/tools/umpire/event"
)

// SpanIterator is a function that yields spans one at a time.
// The yield function returns false to stop iteration early.
type SpanIterator func(yield func(ptrace.Span) bool)

// Parser interface for events that can parse themselves from OTLP spans.
type Parser interface {
	Parse(span ptrace.Span) event.Event
}

// Importer converts OTLP spans into canonical events.
type Importer struct {
	parsers map[string]Parser
}

// NewImporter creates a new importer with default parsers.
func NewImporter() *Importer {
	imp := &Importer{
		parsers: make(map[string]Parser),
	}

	// Register parsers for each span name
	imp.parsers["temporal.server.api.matchingservice.v1.MatchingService/AddWorkflowTask"] = &event.AddWorkflowTaskEvent{}
	imp.parsers["temporal.server.api.matchingservice.v1.MatchingService/PollWorkflowTaskQueue"] = &event.PollWorkflowTaskEvent{}
	imp.parsers["temporal.server.api.matchingservice.v1.MatchingService/AddActivityTask"] = &event.AddActivityTaskEvent{}
	imp.parsers["temporal.server.api.matchingservice.v1.MatchingService/PollActivityTaskQueue"] = &event.PollActivityTaskEvent{}

	return imp
}

// ImportSpans converts spans to events.
func (imp *Importer) ImportSpans(iter SpanIterator) []event.Event {
	var events []event.Event

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
