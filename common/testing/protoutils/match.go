package protoutils

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MatchesPartialProto checks if the actual proto message matches all fields
// that are set in the partial proto message.
//
// For scalar fields, only fields that differ from their default value in the
// partial message are checked. This allows matching on explicit values.
//
// For message fields, only non-nil fields in the partial message are checked.
func MatchesPartialProto(partial, actual proto.Message) bool {
	// Check if types match
	if partial.ProtoReflect().Descriptor().FullName() != actual.ProtoReflect().Descriptor().FullName() {
		return false
	}

	partialRef := partial.ProtoReflect()
	actualRef := actual.ProtoReflect()

	// Iterate through all fields and check those that differ from defaults in partial
	fields := partialRef.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		partialValue := partialRef.Get(fd)
		actualValue := actualRef.Get(fd)

		// Skip fields that are set to their default value in partial
		// This allows matching on explicit values even if they're the default
		if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
			// For message fields, check if it's set (non-nil)
			if !partialRef.Has(fd) {
				continue
			}
			// Compare message fields
			if !proto.Equal(partialValue.Message().Interface(), actualValue.Message().Interface()) {
				return false
			}
		} else {
			// For scalar fields in proto3 without explicit presence,
			// we check if the value in partial differs from the default
			defaultValue := fd.Default()
			if partialValue.Equal(defaultValue) {
				// Value is default in partial, skip checking this field
				continue
			}
			// Compare scalar fields
			if !partialValue.Equal(actualValue) {
				return false
			}
		}
	}

	return true
}
