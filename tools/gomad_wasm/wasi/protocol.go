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
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil || len(fields) != len(keys) {
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
