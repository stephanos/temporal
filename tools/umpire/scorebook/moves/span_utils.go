package moves

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// GetAttribute returns the first non-empty attribute value from the list of keys.
func GetAttribute(span ptrace.Span, keys ...string) string {
	attrs := span.Attributes()
	for _, key := range keys {
		if val, ok := attrs.Get(key); ok {
			return val.Str()
		}
	}
	return ""
}

// GetRequestPayload extracts and unmarshals the rpc.request.payload attribute into the provided proto message.
// Returns true if successful, false otherwise.
func GetRequestPayload(span ptrace.Span, msg proto.Message) bool {
	attrs := span.Attributes()
	val, ok := attrs.Get("rpc.request.payload")
	if !ok {
		return false
	}

	jsonData := val.Str()
	if jsonData == "" {
		return false
	}

	if err := protojson.Unmarshal([]byte(jsonData), msg); err != nil {
		return false
	}

	return true
}

// GetResponsePayload extracts and unmarshals the rpc.response.payload attribute into the provided proto message.
// Returns true if successful, false otherwise.
func GetResponsePayload(span ptrace.Span, msg proto.Message) bool {
	attrs := span.Attributes()
	val, ok := attrs.Get("rpc.response.payload")
	if !ok {
		return false
	}

	jsonData := val.Str()
	if jsonData == "" {
		return false
	}

	if err := protojson.Unmarshal([]byte(jsonData), msg); err != nil {
		return false
	}

	return true
}
