package wasi

import (
	"bytes"
	"encoding/json"
	"io"
)

func decodeInput(data []byte, target any, keys ...string) error {
	if err := validateJSON(data); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return invalid("operation fields")
	}
	return decodeFields(data, fields, target, keys...)
}

func decodeFields(data []byte, fields map[string]json.RawMessage, target any, keys ...string) error {
	if fields == nil || len(fields) != len(keys) {
		return invalid("operation fields")
	}
	for _, key := range keys {
		value, ok := fields[key]
		if !ok || bytes.Equal(value, []byte("null")) {
			return invalid("missing operation field " + key)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalid("operation JSON types")
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
	if err := json.Unmarshal(data, &fields); err != nil {
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
		if err := json.Unmarshal(input.data, &input.object); err != nil || input.object == nil {
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
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := validateValue(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return invalid("trailing JSON")
	}
	return nil
}
func validateValue(decoder *json.Decoder, depth int) error {
	if depth > 32 {
		return invalid("JSON nesting")
	}
	token, err := decoder.Token()
	if err != nil {
		return invalid("malformed JSON")
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return invalid("JSON key")
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return invalid("duplicate JSON field")
			}
			seen[name] = true
			if err := validateValue(decoder, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := validateValue(decoder, depth+1); err != nil {
				return err
			}
		}
	default:
		return invalid("JSON delimiter")
	}
	if _, err := decoder.Token(); err != nil {
		return invalid("JSON close")
	}
	return nil
}
