package wasi

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
)

func decodeInput(data []byte, target any, keys ...string) error {
	if err := validateJSON(data); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := strictjson.Decode(data, &fields); err != nil {
		return invalid("operation fields")
	}
	return decodeFields(data, fields, target, keys...)
}

func decodeFields(data []byte, fields map[string]json.RawMessage, target any, keys ...string) error {
	if err := refuseUnknownFields(fields, keys); err != nil {
		return err
	}
	if fields == nil || len(fields) != len(keys) {
		return invalid("operation fields")
	}
	for _, key := range keys {
		value, ok := fields[key]
		if !ok || bytes.Equal(value, []byte("null")) {
			return invalid("missing operation field " + key)
		}
	}
	if err := strictjson.Decode(data, target); err != nil {
		if strings.HasPrefix(err.Error(), "json: unknown field ") {
			return &BoundaryError{Kind: "unsupported", Dimension: err.Error()}
		}
		return invalid("operation JSON types")
	}
	return nil
}

func refuseUnknownFields(fields map[string]json.RawMessage, keys []string) error {
	unknown, found := "", false
	for key := range fields {
		if !slices.Contains(keys, key) && (!found || key < unknown) {
			unknown, found = key, true
		}
	}
	if found {
		return &BoundaryError{Kind: "unsupported", Dimension: "unknown field " + strconv.Quote(unknown)}
	}
	return nil
}

type validatedFrame struct {
	data   []byte
	fields map[string]json.RawMessage
}

type validatedInput struct {
	data    json.RawMessage
	object  map[string]json.RawMessage
	decoded bool
	err     error
}

type validatedCall struct {
	call  Call
	input *validatedInput
}

func newValidatedFrame(data []byte) (*validatedFrame, error) {
	data = bytes.Clone(data)
	if err := validateJSON(data); err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := strictjson.Decode(data, &fields); err != nil {
		return nil, invalid("protocol object")
	}
	return &validatedFrame{data: data, fields: fields}, nil
}

func (frame *validatedFrame) decode(target any, keys ...string) error {
	return decodeFields(frame.data, frame.fields, target, keys...)
}

func (frame *validatedFrame) decodeCall(local bool) (validatedCall, *helperLocalState, error) {
	var call Call
	var state *helperLocalState
	if local {
		var value helperCall
		if err := frame.decode(&value, "type", "id", "op", "input", "local_state"); err != nil {
			return validatedCall{}, nil, err
		}
		call, state = value.Call, &value.LocalState
	} else if err := frame.decode(&call, "type", "id", "op", "input"); err != nil {
		return validatedCall{}, nil, err
	}
	for _, operation := range transcriptOperations {
		if operation.name != call.Op {
			continue
		}
		var fields map[string]json.RawMessage
		if err := strictjson.Decode(call.Input, &fields); err != nil {
			break
		}
		keys := strings.Fields(operation.input)
		for i, field := range keys {
			keys[i], _, _ = strings.Cut(field, ":")
		}
		if err := refuseUnknownFields(fields, keys); err != nil {
			return validatedCall{}, nil, err
		}
		break
	}
	return validatedCall{call: call, input: &validatedInput{data: frame.fields["input"]}}, state, nil
}

func (e *Environment) handleValidated(call validatedCall) (Reply, error) {
	if call.input == nil {
		return Reply{}, invalid("validated callback")
	}
	return e.handle(call.call, call.input)
}

func (input *validatedInput) fields() (map[string]json.RawMessage, error) {
	if !input.decoded {
		input.decoded = true
		if err := strictjson.Decode(input.data, &input.object); err != nil || input.object == nil {
			input.err = invalid("operation input")
		}
	}
	return input.object, input.err
}

func decodeOperationInput(data []byte, input *validatedInput, target any, keys ...string) error {
	if input == nil {
		return decodeInput(data, target, keys...)
	}
	fields, err := input.fields()
	if err != nil {
		return err
	}
	return decodeFields(input.data, fields, target, keys...)
}

func validateJSON(data []byte) error {
	err := strictjson.CheckSyntax(data, 32)
	switch {
	case errors.Is(err, strictjson.ErrNestingLimit):
		return invalid("JSON nesting")
	case errors.Is(err, strictjson.ErrDuplicateName):
		return invalid("duplicate JSON field")
	case errors.Is(err, strictjson.ErrTrailingData):
		return invalid("trailing JSON")
	case err != nil:
		return invalid("malformed JSON")
	default:
		return nil
	}
}
