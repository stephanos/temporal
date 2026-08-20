package evidence

import (
	"context"
	"errors"
	"fmt"
)

type SourceKind string

const (
	SourcePublicAPI      SourceKind = "public-api"
	SourceHistory        SourceKind = "history"
	SourceTaskProtocol   SourceKind = "task-protocol"
	SourceOpenTelemetry  SourceKind = "opentelemetry"
	SourceInProcessHooks SourceKind = "in-process-hooks"
)

type SourceAdapter interface {
	Kind() SourceKind
	Read(context.Context) ([]Fact, error)
}

func Ingest(ctx context.Context, builder *Builder, adapter SourceAdapter) error {
	if builder == nil || adapter == nil {
		return errors.New("evidence builder and source adapter are required")
	}
	switch adapter.Kind() {
	case SourcePublicAPI, SourceHistory, SourceTaskProtocol, SourceOpenTelemetry, SourceInProcessHooks:
	default:
		return fmt.Errorf("unknown evidence source kind %q", adapter.Kind())
	}
	facts, err := adapter.Read(ctx)
	if err != nil {
		return fmt.Errorf("read %s evidence: %w", adapter.Kind(), err)
	}
	for _, fact := range facts {
		if err := builder.AddFact(fact); err != nil {
			return fmt.Errorf("normalize %s evidence: %w", adapter.Kind(), err)
		}
	}
	return nil
}
