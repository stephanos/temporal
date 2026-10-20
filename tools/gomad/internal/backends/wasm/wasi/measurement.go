package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

type GuestProfileConfig = guestProfileConfig

const MaxCooperativeCalls = maxCooperativeCalls

func ValidateMeasurementJSON(data []byte) error { return validateJSON(data) }

func (result *Result) SetHelperStderr(data []byte) { result.helperStderr = data }

func (request *Request) SetRemoteModel(remote bool) { request.remoteModel = remote }

func (request Request) GuestProfile() *GuestProfileConfig { return request.guestProfile }

func (request *Request) SetGuestProfile(profile *GuestProfileConfig) { request.guestProfile = profile }

func (request Request) DiagnosticConfigurationRevision() string {
	return request.diagnosticConfigurationRevision
}

func (request *Request) SetDiagnosticConfigurationRevision(revision string) {
	request.diagnosticConfigurationRevision = revision
}

// SetStressHostConditions opts a measurement into the distinct stress helper.
// These variables belong to the helper process, never to the guest namespace.
func (request *Request) SetStressHostConditions(padding string, jitter bool) {
	request.diagnosticConfigurationRevision = PinnedEngine().Configuration + "-stress-jitter"
	request.stressHostEnvironment = []string{"GOMAD_WASM_STRESS_LAYOUT=" + padding, "GOMAD_WASM_EVENT_ORDER=1"}
	if jitter {
		request.stressHostEnvironment = append(request.stressHostEnvironment, "GOMAD_WASM_STRESS_JITTER=1")
	}
}

// SetDeterminismFixtureConditions admits the helper containing deliberate
// faults used to verify the divergence report; it cannot qualify a workload.
func (request *Request) SetDeterminismFixtureConditions(fault string) error {
	switch fault {
	case "", "draw", "clock", "handoff", "output-clock":
	default:
		return invalid("determinism fixture fault")
	}
	request.diagnosticConfigurationRevision = PinnedEngine().Configuration + "-determinism-fixture"
	request.stressHostEnvironment = []string{"GOMAD_WASM_DETERMINISM_FAULT=" + fault, "GOMAD_WASM_EVENT_ORDER=1"}
	return nil
}

func (result Result) RuntimeTerminal() [16]byte { return result.runtimeTerminal }

func (result *Result) SetRuntimeTerminal(terminal [16]byte) { result.runtimeTerminal = terminal }

func (result Result) HelperStderr() []byte { return result.helperStderr }

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
