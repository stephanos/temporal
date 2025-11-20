// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package scout

import (
	"context"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

// TraceHandler receives traces reported by the scout.
// Scout doesn't know who implements this - could be scorebook, umpire, etc.
type TraceHandler interface {
	AddTraces(ctx context.Context, traces ptrace.Traces) error
}

// SpanExporter implements sdktrace.SpanExporter and reports observed spans
// to a TraceHandler. Scout observes and reports, but doesn't know the recipient.
type SpanExporter struct {
	handler          TraceHandler
	client           *reportClient
	internalExporter sdktrace.SpanExporter
}

// reportClient implements otlptrace.Client to convert and forward traces
type reportClient struct {
	handler     TraceHandler
	unmarshaler ptrace.Unmarshaler
}

var _ sdktrace.SpanExporter = (*SpanExporter)(nil)

// NewSpanExporter creates a new SpanExporter that reports spans to the given handler.
func NewSpanExporter(handler TraceHandler) *SpanExporter {
	client := &reportClient{
		handler:     handler,
		unmarshaler: &ptrace.ProtoUnmarshaler{},
	}
	return &SpanExporter{
		handler:          handler,
		client:           client,
		internalExporter: otlptrace.NewUnstarted(client),
	}
}

// ExportSpans implements sdktrace.SpanExporter.
// It converts spans to OTLP format and reports them via the handler.
func (e *SpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return e.internalExporter.ExportSpans(ctx, spans)
}

// Shutdown implements sdktrace.SpanExporter.
func (e *SpanExporter) Shutdown(ctx context.Context) error {
	return e.client.Stop(ctx)
}

// Start implements the otlptrace.Client interface (required by otlptrace.NewUnstarted).
func (c *reportClient) Start(ctx context.Context) error {
	return nil
}

// Stop implements the otlptrace.Client interface.
func (c *reportClient) Stop(ctx context.Context) error {
	return nil
}

// UploadTraces implements the otlptrace.Client interface.
// It converts the proto spans to pdata.Traces and reports them via the handler.
func (c *reportClient) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	if len(protoSpans) == 0 {
		return nil
	}

	// Convert proto format to pdata.Traces.
	for _, rs := range protoSpans {
		// Marshal each ResourceSpan into TracesData.
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

		// Report to handler (scout doesn't know who receives it).
		_ = c.handler.AddTraces(ctx, traces)
	}

	return nil
}
