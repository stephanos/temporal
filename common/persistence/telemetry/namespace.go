package telemetry

import (
	"reflect"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/server/common/telemetry"
)

func annotateSpanWithNamespaceID(span trace.Span, request any) {
	v := reflect.ValueOf(request)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	f := v.FieldByName("NamespaceID")
	if !f.IsValid() || f.Kind() != reflect.String {
		return
	}
	if nsID := f.String(); nsID != "" {
		span.SetAttributes(attribute.Key(telemetry.NamespaceIDKey).String(nsID))
	}
}
