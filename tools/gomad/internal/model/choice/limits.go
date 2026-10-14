package choice

import (
	"errors"
	"fmt"
)

func ValidateTraceLimit(limit uint64) error {
	if limit < MinimumTraceBytes || limit > MaximumTraceBytes {
		return fmt.Errorf("invalid choice trace limit %d", limit)
	}
	return nil
}

func TraceBytes(records uint64) (uint64, error) {
	payload, err := TracePayloadBytes(records)
	if err != nil {
		return 0, err
	}
	return traceHeaderBytes + payload, nil
}

func TracePayloadBytes(records uint64) (uint64, error) {
	if records > (MaximumTraceBytes-traceHeaderBytes)/traceRecordBytes {
		return 0, errors.New("choice trace record count exceeds its bound")
	}
	return records * traceRecordBytes, nil
}
