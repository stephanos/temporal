package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

const MaxCooperativeCalls = maxCooperativeCalls

func ValidateMeasurementJSON(data []byte) error { return validateJSON(data) }

func MeasurementLegacyTranscript(data []byte, config Config) ([]byte, error) {
	environment, err := NewEnvironment(config)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	if uint64(len(data)) > environment.config.Limits.TranscriptBytes || data[len(data)-1] != '\n' {
		return nil, fmt.Errorf("historical transcript bounds or framing")
	}
	rows := bytes.Split(data[:len(data)-1], []byte{'\n'})
	if len(rows)%2 != 0 {
		return nil, fmt.Errorf("unpaired historical transcript")
	}
	var previous uint64
	i := 0
	for ; i < len(rows); i += 2 {
		if len(rows[i]) > frameLimit || len(rows[i+1]) > frameLimit {
			return nil, fmt.Errorf("historical frame bound")
		}
		var call Call
		err := decodeInput(rows[i], &call, "type", "id", "op", "input")
		if err != nil {
			return nil, err
		}
		frame, err := newValidatedFrame(rows[i+1])
		if err != nil {
			return nil, err
		}
		keys := []string{"type", "id", "errno", "output"}
		for _, optional := range []string{"unsupported", "message"} {
			_, ok := frame.fields[optional]
			if ok {
				keys = append(keys, optional)
			}
		}
		var reply Reply
		err = frame.decode(&reply, keys...)
		if err != nil {
			return nil, err
		}
		if call.Type != "call" || reply.Type != "reply" || call.ID == 0 || call.ID <= previous || call.ID != reply.ID || strings.HasPrefix(call.Op, "runtime_") || !reply.Unsupported && reply.Message != "" {
			return nil, fmt.Errorf("historical call/reply identity or error evidence")
		}
		previous = call.ID
		tag, input, err := transcriptInput(call, environment.entropyOffset)
		if err != nil {
			return nil, err
		}
		if reply.Unsupported || reply.Errno != 0 {
			err := decodeInput(reply.Output, &struct{}{})
			if err != nil {
				return nil, err
			}
		} else if tag >= 3 && tag <= 7 {
			output, errno, message, err := environment.dispatch(call, nil)
			if err != nil {
				return nil, err
			}
			if errno != 0 || message != "" {
				return nil, fmt.Errorf("historical derived output owner refused")
			}
			keys := []string{"values"}
			switch tag {
			case 3, 5:
				keys = []string{"count", "bytes"}
			case 7:
				keys = []string{"data_base64"}
			}
			var decoded any
			switch output.(type) {
			case sizesOutput:
				decoded = &sizesOutput{}
			case valuesOutput:
				decoded = &valuesOutput{}
			case bytesOutput:
				decoded = &bytesOutput{}
			default:
				return nil, fmt.Errorf("historical derived output has no typed owner")
			}
			err = decodeInput(reply.Output, decoded, keys...)
			if err != nil {
				return nil, err
			}
			expectedJSON, err := json.Marshal(output)
			if err != nil {
				return nil, err
			}
			actualJSON, err := json.Marshal(decoded)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(expectedJSON, actualJSON) {
				return nil, fmt.Errorf("historical derived output differs from captured inputs")
			}
		}
		if tag == 1 && reply.Errno == 0 && !reply.Unsupported {
			var clock struct {
				ID        uint32 `json:"clock_id"`
				Precision uint64 `json:"precision"`
			}
			var output struct {
				Timestamp uint64 `json:"timestamp"`
			}
			err := decodeInput(call.Input, &clock, "clock_id", "precision")
			if err != nil {
				return nil, err
			}
			err = decodeInput(reply.Output, &output, "timestamp")
			if err != nil {
				return nil, err
			}
			if clock.ID > 1 || clock.ID == 0 && output.Timestamp < config.Clock.EpochNanos {
				return nil, fmt.Errorf("historical clock domain")
			}
			now := output.Timestamp
			if clock.ID == 0 {
				now -= config.Clock.EpochNanos
			}
			if now > math.MaxInt64 {
				return nil, fmt.Errorf("historical clock overflow")
			}
			environment.appendClockRead(call.ID, clock.ID, clock.Precision, now)
			continue
		}
		record, err := encodeTranscriptRecord(tag, call.ID, input, reply)
		if err != nil {
			return nil, err
		}
		environment.appendTranscript(record)
	}
	converted := environment.Transcript()
	_, err = DecodeTranscript(converted)
	if err != nil {
		return nil, err
	}
	return converted, nil
}
