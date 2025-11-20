package umpire

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/tools/catch/roster"
	"go.temporal.io/server/tools/catch/rulebook"
	rulebooktypes "go.temporal.io/server/tools/catch/rulebook/types"
	"go.temporal.io/server/tools/catch/scorebook"
)

// Umpire provides an in-memory telemetry verification engine that can be
// embedded into tests or used as a standalone service. It receives OTLP traces,
// converts them to events, routes them to entities, and runs pluggable verification
// models that query entities and return violations when invariants are violated.
type Umpire struct {
	logger        log.Logger
	registry      *roster.Registry
	importer      *scorebook.Importer
	modelRegistry *rulebook.Registry
}

// Config holds configuration for an Umpire instance.
type Config struct {
	// Logger for umpire output.
	Logger log.Logger
}

// New creates a new Umpire instance with the given configuration.
func New(cfg Config) (*Umpire, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("umpire: logger is required")
	}

	// Initialize registry with in-memory storage.
	registry, err := roster.NewEntityRegistry(cfg.Logger, "")
	if err != nil {
		return nil, fmt.Errorf("umpire: failed to create registry: %w", err)
	}

	// TODO: Register default entities
	// roster.RegisterDefaultEntities(registry)

	// Initialize importer.
	importer := scorebook.NewImporter()

	// Initialize model registry.
	modelRegistry := rulebook.NewRegistry()

	// Initialize all registered models.
	if err := modelRegistry.InitModels(context.Background(), nil, rulebooktypes.Deps{
		Registry: registry,
		Logger:   cfg.Logger,
	}); err != nil {
		return nil, fmt.Errorf("umpire: failed to initialize models: %w", err)
	}

	cfg.Logger.Info("umpire initialized",
		tag.NewInt("numModels", len(modelRegistry.Models())),
	)

	return &Umpire{
		logger:        cfg.Logger,
		registry:      registry,
		importer:      importer,
		modelRegistry: modelRegistry,
	}, nil
}

// Close releases resources held by the umpire.
func (w *Umpire) Close() error {
	// Close models.
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()

	if err := w.modelRegistry.Close(closeCtx); err != nil {
		w.logger.Warn("umpire: error closing models", tag.Error(err))
	}

	// Close registry and database.
	if err := w.registry.Close(); err != nil {
		w.logger.Warn("umpire: error closing registry", tag.Error(err))
	}

	w.logger.Info("umpire closed")
	return nil
}

// AddTraces adds a batch of traces to the umpire. It converts spans to events
// and routes them to entities.
func (w *Umpire) AddTraces(ctx context.Context, td ptrace.Traces) error {
	// Create an iterator that yields all spans in the trace batch.
	spanIter := func(yield func(ptrace.Span) bool) {
		resourceSpans := td.ResourceSpans()
		for i := 0; i < resourceSpans.Len(); i++ {
			rs := resourceSpans.At(i)
			scopeSpans := rs.ScopeSpans()
			for j := 0; j < scopeSpans.Len(); j++ {
				ss := scopeSpans.At(j)
				spans := ss.Spans()
				for k := 0; k < spans.Len(); k++ {
					span := spans.At(k)
					w.logger.Debug("umpire: processing span",
						tag.NewStringTag("spanName", span.Name()),
						tag.NewStringTag("spanKind", span.Kind().String()),
						tag.NewStringTag("traceID", span.TraceID().String()),
						tag.NewStringTag("spanID", span.SpanID().String()),
					)
					if !yield(span) {
						return
					}
				}
			}
		}
	}

	// Import spans to events.
	events := w.importer.ImportSpans(spanIter)
	w.logger.Debug("umpire: imported events", tag.NewInt("numEvents", len(events)))

	// Route events to entities.
	if err := w.registry.RouteEvents(ctx, events); err != nil {
		w.logger.Warn("umpire: failed to route events", tag.Error(err))
	}

	return nil
}

// Check runs all model Check() methods and returns detected violations.
// This should be called explicitly from tests to verify invariants.
func (w *Umpire) Check(ctx context.Context) []rulebook.Violation {
	return w.modelRegistry.Check(ctx)
}
