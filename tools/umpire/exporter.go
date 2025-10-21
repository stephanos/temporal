package umpire

import (
	"context"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

// SpanExporter implements sdktrace.SpanExporter and forwards exported spans
// to a Watchdog instance for verification. This allows seamless integration
// with existing OTEL instrumentation in tests.
type SpanExporter struct {
	watchdog         *Umpire
	client           *watchdogExporterClient
	internalExporter sdktrace.SpanExporter
}

type watchdogExporterClient struct {
	watchdog    *Umpire
	unmarshaler ptrace.Unmarshaler
}

var _ sdktrace.SpanExporter = (*SpanExporter)(nil)

// NewSpanExporter creates a new SpanExporter that forwards spans to the given Umpire.
func NewSpanExporter(w *Umpire) *SpanExporter {
	client := &watchdogExporterClient{
		watchdog:    w,
		unmarshaler: &ptrace.ProtoUnmarshaler{},
	}
	return &SpanExporter{
		watchdog:         w,
		client:           client,
		internalExporter: otlptrace.NewUnstarted(client),
	}
}

// ExportSpans implements sdktrace.SpanExporter.
// It converts spans to OTLP format and forwards them to the umpire.
func (e *SpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return e.internalExporter.ExportSpans(ctx, spans)
}

// Shutdown implements sdktrace.SpanExporter.
func (e *SpanExporter) Shutdown(ctx context.Context) error {
	return e.client.Stop(ctx)
}

// Start implements the otlptrace.Client interface (required by otlptrace.NewUnstarted).
func (c *watchdogExporterClient) Start(ctx context.Context) error {
	return nil
}

// Stop implements the otlptrace.Client interface.
func (c *watchdogExporterClient) Stop(ctx context.Context) error {
	return nil
}

// UploadTraces implements the otlptrace.Client interface.
// It converts the proto spans to pdata.Traces and forwards them to the umpire.
func (c *watchdogExporterClient) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	if len(protoSpans) == 0 {
		return nil
	}

	// Convert proto format to pdata.Traces for watchdog consumption.
	// This matches the approach used in the main.go traceReceiver.
	for _, rs := range protoSpans {
		// Marshal each ResourceSpan into an ExportTraceServiceRequest.
		bytes, err := proto.Marshal(&tracepb.TracesData{
			ResourceSpans: []*tracepb.ResourceSpans{rs},
		})
		if err != nil {
			continue // Best-effort; skip malformed batches.
		}

		// Unmarshal to pdata.Traces.
		traces, err := c.unmarshaler.UnmarshalTraces(bytes)
		if err != nil {
			continue
		}

		// Forward to umpire.
		_ = c.watchdog.AddTraces(ctx, traces)
	}

	return nil
}
