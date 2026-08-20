package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func decodeStrictJSON(reader io.Reader, limit int64, kind string, destination any) error {
	if limit <= 0 {
		return fmt.Errorf("%s decode limit must be positive", kind)
	}
	encoded, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return fmt.Errorf("read %s: %w", kind, err)
	}
	if int64(len(encoded)) > limit {
		return fmt.Errorf("%s exceeds %d-byte decode limit", kind, limit)
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode %s: %w", kind, err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode %s: multiple JSON values", kind)
		}
		return fmt.Errorf("decode %s trailer: %w", kind, err)
	}
	return nil
}
