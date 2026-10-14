package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func warmPreparationLegacyTranscript(data []byte, config Config) ([]byte, error) {
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
	for i := 0; i < len(rows); i += 2 {
		if len(rows[i]) > frameLimit || len(rows[i+1]) > frameLimit {
			return nil, fmt.Errorf("historical frame bound")
		}
		var call Call
		if err := decodeInput(rows[i], &call, "type", "id", "op", "input"); err != nil {
			return nil, err
		}
		frame, err := newValidatedFrame(rows[i+1])
		if err != nil {
			return nil, err
		}
		keys := []string{"type", "id", "errno", "output"}
		for _, optional := range []string{"unsupported", "message"} {
			if _, ok := frame.fields[optional]; ok {
				keys = append(keys, optional)
			}
		}
		var reply Reply
		if err := frame.decode(&reply, keys...); err != nil {
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
			if err := decodeInput(reply.Output, &struct{}{}); err != nil {
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
			decoded := reflect.New(reflect.TypeOf(output))
			if err := decodeInput(reply.Output, decoded.Interface(), keys...); err != nil {
				return nil, err
			}
			if !reflect.DeepEqual(output, decoded.Elem().Interface()) {
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
			if err := decodeInput(call.Input, &clock, "clock_id", "precision"); err != nil {
				return nil, err
			}
			if err := decodeInput(reply.Output, &output, "timestamp"); err != nil {
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
	if _, err := DecodeTranscript(converted); err != nil {
		return nil, err
	}
	return converted, nil
}

func warmPreparationBehavior(result Result, config Config, historical bool) (string, error) {
	if historical {
		transcript, err := warmPreparationLegacyTranscript(result.Transcript, config)
		if err != nil {
			return "", err
		}
		result.Transcript = transcript
	}
	evidence, err := projectVariantEvidence(result, config, "")
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(struct {
		Evidence variantEvidence
		Finish   [16]byte
	}{evidence, result.runtimeTerminal})
	if err != nil {
		return "", err
	}
	return digest(data), nil
}

func TestWarmPreparationHistoricalClockCountsAndPositions(t *testing.T) {
	config := testConfig()
	config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
	legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":7}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"timestamp\":1}}\n{\"type\":\"call\",\"id\":2,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":7}}\n{\"type\":\"reply\",\"id\":2,\"errno\":0,\"output\":{\"timestamp\":1}}\n")
	converted, err := warmPreparationLegacyTranscript(legacy, config)
	if err != nil {
		t.Fatal(err)
	}
	records, err := DecodeTranscript(converted)
	if err != nil || len(records) != 1 || records[0].ID != 1 || records[0].Count != 2 || records[0].Now != 1 || string(records[0].Input) != `{"clock_id":1,"precision":7}` {
		t.Fatal("clock run lost original positions, IDs or counts", err, records)
	}
	for _, data := range [][]byte{
		legacy[:bytes.Index(legacy, []byte(`{"type":"call","id":2`))],
		bytes.ReplaceAll(legacy, []byte(`"timestamp":1`), []byte(`"timestamp":2`)),
		bytes.ReplaceAll(legacy, []byte(`"clock_id":1`), []byte(`"clock_id":2`)),
		bytes.TrimSuffix(legacy, []byte{'\n'}),
	} {
		changed, err := warmPreparationLegacyTranscript(data, config)
		if err == nil && bytes.Equal(converted, changed) {
			t.Fatal("clock count/position/domain or framing change was hidden")
		}
	}
}

func TestWarmPreparationCrossEncodingBehaviorControls(t *testing.T) {
	config := testConfig()
	config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
	environment := newTestEnvironment(t, config)
	var legacy []byte
	for i, operation := range []struct{ op, input string }{
		{"args_sizes_get", `{}`}, {"args_get", `{}`}, {"environ_sizes_get", `{}`}, {"environ_get", `{}`},
		{"random_get", `{"length":2}`}, {"random_get", `{"length":3}`},
		{"clock_time_get", `{"clock_id":1,"precision":7}`}, {"clock_time_get", `{"clock_id":1,"precision":7}`},
		{"clock_time_get", `{"clock_id":0,"precision":7}`},
		{"fd_write", `{"fd":1,"data_base64":"YWI="}`},
		{"fd_read", `{"fd":99,"length":1}`}, {"unknown", `{"value":1}`},
	} {
		request := Call{Type: "call", ID: uint64(i + 1), Op: operation.op, Input: json.RawMessage(operation.input)}
		reply, err := environment.Handle(request)
		if err != nil {
			t.Fatal(err)
		}
		requestData, _ := json.Marshal(request)
		replyData, _ := json.Marshal(reply)
		legacy = append(append(append(append(legacy, requestData...), '\n'), replyData...), '\n')
	}
	code := uint32(0)
	current := Result{Transcript: environment.Transcript(), Stdout: []byte("ab"), Termination: "exit", ExitCode: &code, runtimeTerminal: [16]byte{1}}
	historical := current
	historical.Transcript = legacy
	want, err := warmPreparationBehavior(current, config, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := warmPreparationBehavior(historical, config, true)
	if err != nil || got != want {
		t.Fatal("same real owner behavior changed solely because of transcript encoding", err)
	}
	for _, test := range []struct{ name, old, replacement string }{
		{"clock-position", `"timestamp":1`, `"timestamp":2`},
		{"clock-precision", `"precision":7`, `"precision":8`},
		{"operation-id", `"call","id":12`, `"call","id":13`},
		{"operation-input", `"value":1`, `"value":2`},
		{"error", `"errno":8`, `"errno":9`},
		{"output", `"written":2`, `"written":1`},
		{"random-reply", `"data_base64":`, `"extra":1,"data_base64":`},
		{"unsupported-message", `operation outside stock WASI profile: unknown`, `operation outside stock WASI profile: other`},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := historical
			changed.Transcript = bytes.Replace(legacy, []byte(test.old), []byte(test.replacement), 1)
			if bytes.Equal(changed.Transcript, legacy) {
				t.Fatal("control did not mutate its real input")
			}
			got, err := warmPreparationBehavior(changed, config, true)
			if err == nil && got == want {
				t.Fatal("cross-encoding projection hid changed behavior")
			}
		})
	}
	for _, field := range []string{"stdout", "stderr", "network", "choices", "finish", "virtual-time", "message", "termination", "exit"} {
		t.Run(field, func(t *testing.T) {
			changed := historical
			switch field {
			case "stdout":
				changed.Stdout = []byte("other")
			case "stderr":
				changed.Stderr = []byte("other")
			case "network":
				changed.NetworkTranscript = []byte("other")
			case "choices":
				changed.ChoiceTrace.Bytes = []byte("other")
			case "finish":
				changed.runtimeTerminal[0]++
			case "virtual-time":
				changed.VirtualTimeElapsedNanos++
			case "message":
				changed.Message = "other"
			case "termination":
				changed.Termination = "other"
			case "exit":
				code := uint32(1)
				changed.ExitCode = &code
			}
			got, err := warmPreparationBehavior(changed, config, true)
			if err != nil || got == want {
				t.Fatal("non-WASI evidence lost", err)
			}
		})
	}
}

func TestWarmPreparationHistoricalTranscriptOwnerProjection(t *testing.T) {
	config := testConfig()
	config.Args = []string{"guest"}
	environment := newTestEnvironment(t, config)
	legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"args_get\",\"input\":{}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"values\":[\"Z3Vlc3Q=\"]}}\n")
	call(t, environment, "args_get", `{}`)
	converted, err := warmPreparationLegacyTranscript(legacy, config)
	if err != nil || !bytes.Equal(converted, environment.Transcript()) {
		t.Fatal("historical derived arguments did not match the typed owner", err)
	}
	if _, err := DecodeTranscript(legacy); err == nil {
		t.Fatal("production decoder must not accept historical JSONL")
	}
}

func TestWarmPreparationCollectionRetainsBinaryArtifactName(t *testing.T) {
	root := t.TempDir()
	series := warmPreparationFixture()
	result := Result{Transcript: bytes.Clone(transcriptHeader)}
	persistDiagnosticRawResult(t, root, "untimed-evidence", result)
	if failed := warmPreparationPublishCollection(t, root, "candidate-collection.json", &series, result, "", "unqualified control"); failed != "unqualified control" {
		t.Fatal(failed)
	}
	var collection warmPreparationCollection
	data, err := os.ReadFile(series.CollectionPath)
	if err != nil || json.Unmarshal(data, &collection) != nil {
		t.Fatal("missing collection marker", err)
	}
	if collection.CanonicalPath != filepath.Join(root, "untimed-evidence-canonical.bin") || series.Inputs[collection.CanonicalPath] != digest(result.Transcript) {
		t.Fatal("collection did not bind the binary owner's actual artifact")
	}
}

func TestWarmPreparationHistoricalTranscriptRefusesLostEvidence(t *testing.T) {
	config := testConfig()
	config.Args = []string{"guest"}
	legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"args_get\",\"input\":{}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"values\":[\"Z3Vlc3Q=\"]}}\n")
	for _, test := range []struct{ name, old, replacement string }{
		{"derived-argument", "Z3Vlc3Q=", "b3RoZXI="},
		{"reply-id", `"reply","id":1`, `"reply","id":2`},
		{"missing-reply", string(legacy[bytes.IndexByte(legacy, '\n')+1:]), ""},
		{"duplicate-field", `"op":"args_get"`, `"op":"args_get","op":"args_get"`},
		{"unknown-envelope-field", `"errno":0`, `"errno":0,"extra":true`},
		{"hidden-message", `"errno":0`, `"errno":0,"message":"changed"`},
		{"noncanonical-runtime", "args_get", "runtime_choice"},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := bytes.Replace(legacy, []byte(test.old), []byte(test.replacement), 1)
			if _, err := warmPreparationLegacyTranscript(changed, config); err == nil {
				t.Fatal("historical evidence change was discarded")
			}
		})
	}
	for _, replacement := range []Config{func() Config { c := config; c.Args = []string{"other"}; return c }(), func() Config { c := config; c.Environment = []string{"OTHER=value"}; return c }()} {
		op := "args_get"
		if replacement.Environment[0] == "OTHER=value" {
			op = "environ_get"
		}
		environment := newTestEnvironment(t, config)
		request := Call{Type: "call", ID: 1, Op: op, Input: json.RawMessage(`{}`)}
		reply, err := environment.Handle(request)
		if err != nil {
			t.Fatal(err)
		}
		requestData, _ := json.Marshal(request)
		replyData, _ := json.Marshal(reply)
		data := append(append(append(requestData, '\n'), replyData...), '\n')
		if _, err := warmPreparationLegacyTranscript(data, replacement); err == nil {
			t.Fatal("derived captured configuration change was discarded", op)
		}
	}
}
