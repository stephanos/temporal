package logstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// EncodeJSON encodes a record payload with the standard library encoder.
func EncodeJSON(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode log record: %w", err)
	}
	return payload, nil
}

// DecodeJSON strictly decodes one record payload: unknown fields, a top-level
// null, and any data after the single JSON value are rejected.
func DecodeJSON(payload []byte, value any) error {
	if bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return errors.New("decode log record: payload is null")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode log record: %w", err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("decode log record: trailing data after payload")
	}
	return nil
}
