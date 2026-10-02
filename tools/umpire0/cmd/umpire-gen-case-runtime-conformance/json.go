package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/tools/umpire/internal/casefile"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

const persistedIndent = casefile.Indent

func marshalExpected(expected expectedResult) ([]byte, error) {
	encoded, err := json.MarshalIndent(expected, "", persistedIndent)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// persistedForm is how every generated Testpilot JSON artifact is stored: two-space indentation and
// exactly one trailing newline, so a Case or a conformance corpus change reads as a line diff.
//
// The renderer's own encoding policy is untouched -- it still emits compact canonical ProtoJSON, and
// indentation is presentation of the stored file. Compacting before indenting makes the form
// idempotent whatever whitespace the input carried; both passes preserve key order and string
// escapes exactly.
func persistedForm(encoded []byte) ([]byte, error) {
	return casefile.Persisted(encoded)
}

// requirePersistedForm rejects a staged artifact that is valid JSON but not stored the way the
// generator writes it, naming the file. Without it a hand-edited or compact fixture would surface
// only as an unexplained `diff -ru` hunk.
func requirePersistedForm(path string, encoded []byte) error {
	expected, err := persistedForm(encoded)
	if err != nil {
		return fmt.Errorf("artifact %q is not valid JSON: %w", path, err)
	}
	if !bytes.Equal(expected, encoded) {
		return fmt.Errorf("artifact %q is not in persisted form: indent with %d spaces and end in exactly one newline",
			path, len(persistedIndent))
	}
	return nil
}

// requireDeclarationOrder rejects a staged artifact whose message objects do not list their fields
// in declaration order, naming the file and the JSON path of the first object that does not.
// `Testpilot.ProtoJSON` writes that order, and the protocol declares identity first, so a fixture
// in any other order was not written by the renderer.
func requireDeclarationOrder(path string, encoded []byte, message protoreflect.MessageDescriptor) error {
	if err := checkDeclarationOrder(encoded, message, "$"); err != nil {
		return fmt.Errorf("artifact %q is not in declaration order: %w", path, err)
	}
	return nil
}

// requireCorrelatedDeclarationOrder checks each correlated corpus entry's Cases and events.
func requireCorrelatedDeclarationOrder(path string, encoded []byte) error {
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &entries); err != nil {
		return fmt.Errorf("artifact %q is not a list of correlated entries: %w", path, err)
	}
	evidenceMessage := (&testpilotspb.CorrelatedEvidence{}).ProtoReflect().Descriptor()
	for index, entry := range entries {
		location := fmt.Sprintf("$[%d]", index)
		for _, key := range []string{"case", "runnableCase"} {
			if err := checkDeclarationOrder(entry[key], caseDescriptor, location+"."+key); err != nil {
				return fmt.Errorf("artifact %q is not in declaration order: %w", path, err)
			}
		}
		var events []json.RawMessage
		if err := json.Unmarshal(entry["events"], &events); err != nil {
			return fmt.Errorf("artifact %q: %s.events: %w", path, location, err)
		}
		for event, raw := range events {
			if err := checkDeclarationOrder(raw, evidenceMessage, fmt.Sprintf("%s.events[%d]", location, event)); err != nil {
				return fmt.Errorf("artifact %q is not in declaration order: %w", path, err)
			}
		}
	}
	return nil
}

// customJSON names the well-known types whose ProtoJSON form is not an object of their fields; it is
// the list Testpilot.ProtoJSON writes the same way.
var customJSON = map[protoreflect.FullName]bool{
	"google.protobuf.Timestamp": true, "google.protobuf.Duration": true, "google.protobuf.FieldMask": true,
	"google.protobuf.Struct": true, "google.protobuf.Value": true, "google.protobuf.ListValue": true,
	"google.protobuf.DoubleValue": true, "google.protobuf.FloatValue": true, "google.protobuf.Int64Value": true,
	"google.protobuf.UInt64Value": true, "google.protobuf.Int32Value": true, "google.protobuf.UInt32Value": true,
	"google.protobuf.BoolValue": true, "google.protobuf.StringValue": true, "google.protobuf.BytesValue": true,
}

// checkDeclarationOrder checks one ProtoJSON value of message located at location. Well-known types
// with their own JSON forms are not objects of their fields; an Any lists @type before its payload.
func checkDeclarationOrder(encoded []byte, message protoreflect.MessageDescriptor, location string) error {
	if customJSON[message.FullName()] {
		return nil
	}
	keys, values, err := objectMembers(encoded)
	if err != nil {
		return fmt.Errorf("%s: %w", location, err)
	}
	if message.FullName() == "google.protobuf.Any" && len(keys) > 0 {
		if keys[0] != "@type" {
			return fmt.Errorf("%s: an Any lists %q before @type", location, keys[0])
		}
		var url string
		if err := json.Unmarshal(values[0], &url); err != nil {
			return fmt.Errorf("%s.@type: %w", location, err)
		}
		payload, err := protoregistry.GlobalTypes.FindMessageByURL(url)
		if err != nil {
			return fmt.Errorf("%s.@type: %w", location, err)
		}
		if customJSON[payload.Descriptor().FullName()] {
			return nil
		}
		message, keys, values = payload.Descriptor(), keys[1:], values[1:]
	}
	previous := -1
	for index, key := range keys {
		field := message.Fields().ByJSONName(key)
		if field == nil {
			return fmt.Errorf("%s: %s declares no field %q", location, message.FullName(), key)
		}
		if field.Index() <= previous {
			return fmt.Errorf("%s: field %q follows %q, which %s declares after it", location, key, keys[index-1], message.FullName())
		}
		previous = field.Index()
		if err := checkFieldOrder(values[index], field, location+"."+key); err != nil {
			return err
		}
	}
	return nil
}

func checkFieldOrder(encoded []byte, field protoreflect.FieldDescriptor, location string) error {
	switch {
	case field.IsMap():
		if field.MapValue().Message() == nil {
			return nil
		}
		keys, values, err := objectMembers(encoded)
		if err != nil {
			return fmt.Errorf("%s: %w", location, err)
		}
		for index, key := range keys {
			if err := checkDeclarationOrder(values[index], field.MapValue().Message(), location+"."+key); err != nil {
				return err
			}
		}
	case field.Message() == nil:
	case field.IsList():
		var elements []json.RawMessage
		if err := json.Unmarshal(encoded, &elements); err != nil {
			return fmt.Errorf("%s: %w", location, err)
		}
		for index, element := range elements {
			if err := checkDeclarationOrder(element, field.Message(), fmt.Sprintf("%s[%d]", location, index)); err != nil {
				return err
			}
		}
	default:
		return checkDeclarationOrder(encoded, field.Message(), location)
	}
	return nil
}

// objectMembers reads a JSON object's keys and values in the order the object lists them.
func objectMembers(encoded []byte) ([]string, []json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil, nil, errors.New("not a JSON object")
	}
	var keys []string
	var values []json.RawMessage
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, nil, err
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, nil, err
		}
		keys = append(keys, token.(string))
		values = append(values, value)
	}
	return keys, values, nil
}
