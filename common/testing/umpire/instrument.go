package umpire

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/server/tools/catch/roster/types"
)

const (
	// TracerName is the name of the scout tracer for instrumentation
	TracerName = "go.temporal.io/server/testing/scout"
)

// EntityTag creates OTEL attributes from an EntityID.
// This is the primary way to tag observable events with entity identity.
//
// Example:
//
//	entityID := types.NewEntityIDFromType(entities.WorkflowType, workflowID)
//	ctx, span := catch.Instrument(ctx, "workflow.lock.acquire",
//		catch.EntityTag(entityID),
//		attribute.String("lock.type", "exclusive"),
//	)
func EntityTag(entityID types.EntityID) attribute.KeyValue {
	// Use entity type and ID as a single tag for simplicity
	return attribute.String("entity", string(entityID.Type)+":"+entityID.ID)
}

// Instrument creates an OTEL span for an event with the given attributes.
// Use this to make events observable that are important for property validation.
//
// Example usage:
//
//	ctx, span := catch.Instrument(ctx, "workflow.lock.acquire",
//		attribute.String(catch.AttrWorkflowID, workflowID),
//		attribute.String(catch.AttrNamespaceID, namespaceID),
//		attribute.String(catch.AttrLockType, "exclusive"),
//	)
//	defer span.End()
func Instrument(ctx context.Context, eventName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(TracerName)
	ctx, span := tracer.Start(ctx, eventName)
	span.SetAttributes(attrs...)
	return ctx, span
}

// RecordEvent records a point-in-time event with attributes.
// Unlike Instrument which creates a span with duration, this records an instantaneous event.
//
// Example usage:
//
//	catch.RecordEvent(ctx, "workflow.lock.released",
//		attribute.String(catch.AttrWorkflowID, workflowID),
//		attribute.String(catch.AttrNamespaceID, namespaceID),
//	)
func RecordEvent(ctx context.Context, eventName string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		// No active span, create a minimal one
		ctx, span = Instrument(ctx, eventName, attrs...)
		span.End()
		return
	}

	// Add event to current span
	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}

// RecordError records an error in the current span with optional attributes.
//
// Example usage:
//
//	if err := acquireLock(); err != nil {
//		catch.RecordError(ctx, err,
//			attribute.String(catch.AttrWorkflowID, workflowID),
//		)
//		return err
//	}
func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}
