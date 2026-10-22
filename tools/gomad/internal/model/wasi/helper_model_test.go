package wasi

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestCapturedEntropySharedGoldenVectors(t *testing.T) {
	for _, test := range []struct {
		offset uint64
		length uint32
		hex    string
	}{
		{0, 32, "dd7734737675d01bb34069543218175bc6631c6e1e3a35c89ffa0ed113d266c0"},
		{30, 6, "66c0fb3513f4"},
		{32, 32, "fb3513f4792214027f4d8f57a2f26bb438c986eede20a8866595dfed5ebf0c9a"},
		{64, 0, ""},
	} {
		config := testConfig()
		config.EntropyKey = [32]byte{}
		e := newTestEnvironment(t, config)
		e.entropyOffset = test.offset
		var result bytesOutput
		output(t, call(t, e, "random_get", fmt.Sprintf(`{"length":%d}`, test.length)), &result)
		if hex.EncodeToString(result.Data) != test.hex || e.entropyOffset != test.offset+uint64(test.length) {
			t.Fatalf("entropy at %d = %x (offset %d), want %s", test.offset, result.Data, e.entropyOffset, test.hex)
		}
	}
}
func TestHelperInputCarriageLiteralAndMaximumFragmentation(t *testing.T) {
	e := &Environment{config: Config{Args: []string{"a"}, Environment: []string{"PWD=/"}}}
	wire, err := appendHelperInputs(nil, e)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\"type\":\"model_input\",\"stream\":\"arguments\",\"sequence\":0,\"data_base64\":\"YQA=\"}\n{\"type\":\"model_input\",\"stream\":\"environment\",\"sequence\":0,\"data_base64\":\"UFdEPS8A\"}\n{\"type\":\"model_inputs_end\"}\n"
	if string(wire) != want {
		t.Fatalf("input carriage = %q", wire)
	}
	e.config.Args = []string{strings.Repeat("\x01", bufferLimit-1)}
	e.config.WorkingDirectory = "/" + strings.Repeat("\x02", bufferLimit-6)
	e.config.Environment = []string{"PWD=" + e.config.WorkingDirectory}
	wire, err = appendHelperInputs(nil, e)
	if err != nil {
		t.Fatal(err)
	}
	var joined [2][]byte
	var sequences [2]uint64
	frames := bytes.Split(bytes.TrimSpace(wire), []byte{'\n'})
	if len(frames) != 129 {
		t.Fatalf("frames=%d", len(frames))
	}
	for _, frame := range frames[:len(frames)-1] {
		if len(frame) > 128<<10 {
			t.Fatalf("unbounded input frame: %d", len(frame))
		}
		var chunk struct {
			Type     string `json:"type"`
			Stream   string `json:"stream"`
			Sequence uint64 `json:"sequence"`
			Data     []byte `json:"data_base64"`
		}
		if err := json.Unmarshal(frame, &chunk); err != nil {
			t.Fatal(err)
		}
		index := 0
		if chunk.Stream == "environment" {
			index = 1
		}
		if chunk.Type != "model_input" || chunk.Sequence != sequences[index] || len(chunk.Data) != 64<<10 {
			t.Fatalf("chunk %s/%d/%d", chunk.Stream, chunk.Sequence, len(chunk.Data))
		}
		sequences[index]++
		joined[index] = append(joined[index], chunk.Data...)
	}
	if !bytes.Equal(joined[0], append([]byte(e.config.Args[0]), 0)) || !bytes.Equal(joined[1], append([]byte(e.config.Environment[0]), 0)) {
		t.Fatal("fragmented single string changed")
	}
	for _, values := range [][]string{e.config.Args, e.config.Environment} {
		if got := helperStringDeclaration(values); got != (helperStrings{bufferLimit, 1, 64}) {
			t.Fatalf("declaration %+v", got)
		}
	}
}

type helperCallbackModel interface {
	helperCallbackReply(validatedCall, helperLocalState) (helperReply, error)
}

func helperBridgeModel(t *testing.T, e *Environment) helperCallbackModel {
	t.Helper()
	model, ok := any(e).(helperCallbackModel)
	if !ok {
		t.Fatal("independently pure helper callback calculation is unavailable")
	}
	return model
}

func helperBridgeCall(id uint64, op, input string) validatedCall {
	raw := json.RawMessage(input)
	return validatedCall{call: Call{Type: "call", ID: id, Op: op, Input: raw}, input: &validatedInput{data: raw}}
}

func TestHostileHelperCallbackReplyLeavesEnvironmentByteIdentical(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	e.helperOwned = true
	e.nextCall, e.now, e.advances, e.transcriptLength = 2, 20, 2, 9
	e.transcript, e.stdout, e.stderr = []byte("retained"), []byte("stdout"), []byte("stderr")
	e.runtime = &runtimeSession{cursor: 3, diagnostics: []byte("diagnostics")}
	before := helperEnvironmentBytes(reflect.ValueOf(e))
	var wire bytes.Buffer
	sink := frameSink{environment: e, input: &wire, started: &Started{}}
	err := sink.frame([]byte(`{"type":"call","id":3,"op":"fd_write","input":{"fd":1,"data_base64":"aG9zdGlsZQ=="},"local_state":{"now":18446744073709551615,"advances":99,"transcript_bytes":99}}`))
	var boundary *BoundaryError
	if !errors.As(err, &boundary) || boundary.Kind != "invalid" || boundary.Dimension != "helper callback state" {
		t.Fatalf("hostile callback reached reply application: %v", err)
	}
	if after := helperEnvironmentBytes(reflect.ValueOf(e)); !bytes.Equal(before, after) {
		t.Fatalf("hostile callback reply mutated environment\nbefore: %s\nafter: %s", before, after)
	}
	if wire.Len() != 0 {
		t.Fatalf("hostile callback emitted reply: %s", wire.Bytes())
	}
}

func helperEnvironmentBytes(value reflect.Value) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "%s{", value.Type())
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		if value.IsNil() {
			out.WriteString("nil")
		} else {
			out.Write(helperEnvironmentBytes(value.Elem()))
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			out.WriteString(value.Type().Field(i).Name)
			out.Write(helperEnvironmentBytes(value.Field(i)))
		}
	case reflect.Map:
		keys := value.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			return bytes.Compare(helperEnvironmentBytes(a), helperEnvironmentBytes(b))
		})
		fmt.Fprintf(&out, "nil=%t;", value.IsNil())
		for _, key := range keys {
			out.Write(helperEnvironmentBytes(key))
			out.Write(helperEnvironmentBytes(value.MapIndex(key)))
		}
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice {
			fmt.Fprintf(&out, "nil=%t;cap=%d;", value.IsNil(), value.Cap())
		}
		for i := 0; i < value.Len(); i++ {
			out.Write(helperEnvironmentBytes(value.Index(i)))
		}
	default:
		fmt.Fprintf(&out, "%#v", value)
	}
	out.WriteByte('}')
	return out.Bytes()
}

func TestHelperBridgeModelLiteralPayloadAndState(t *testing.T) {
	const writeTrace = "\x08\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00\x0d\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00\x00x\x01\x00\x00\x00"
	for _, test := range []struct {
		name, op, input, trace, wire, stdout string
		now                                  uint64
		runtime                              bool
	}{
		{"write", "fd_write", `{"fd":1,"data_base64":"eA=="}`, writeTrace,
			`{"type":"reply","id":3,"errno":0,"output":{"written":1},"local_state":{"now":20,"advances":2},"transcript_base64":"CAAAAAMAAAAAAAAADQAAAAEAAAABAAAAeAEAAAA="}`, "x", 20, false},
		{"clock-state-after-handle", "clock_time_get", `{"clock_id":1,"precision":0}`,
			"\x21\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x14\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00",
			`{"type":"reply","id":3,"errno":0,"output":{"timestamp":20},"local_state":{"now":1020,"advances":2},"transcript_base64":"IQAAAAMAAAAAAAAAHAAAAAEAAAAAAAAAAAAAABQAAAAAAAAAAQAAAAAAAAA="}`, "", 1020, false},
		{"runtime-control-excluded", "runtime_config", `{"data_base64":"AAAAAAAAAAAAAAAAAAAAAA=="}`, "",
			`{"type":"reply","id":3,"errno":0,"output":{"data_base64":"AAAAAQAAAAAAAAAAAAAAAA=="},"local_state":{"now":20,"advances":2},"transcript_base64":""}`, "", 20, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig()
			if test.runtime {
				config.Profile = CooperativeProfile
			}
			e := newTestEnvironment(t, config)
			e.helperOwned = true
			if test.runtime {
				e.runtime = &runtimeSession{control: RuntimeControl{Choice: &choice.Request{Mode: choice.ModeRecord}}}
			}
			payload, err := helperBridgeModel(t, e).helperCallbackReply(helperBridgeCall(3, test.op, test.input), helperLocalState{Now: 20, Advances: 2, TranscriptBytes: 9})
			if err != nil {
				t.Fatal(err)
			}
			wire, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if string(wire) != test.wire || string(payload.Transcript) != test.trace || string(e.stdout) != test.stdout ||
				e.nextCall != 4 || e.now != test.now || e.advances != 2 || e.transcriptLength != 9+uint64(len(test.trace)) || len(e.transcript) != 0 {
				t.Fatalf("model payload/state changed: wire=%s payload=%q next=%d now=%d advances=%d bytes=%d retained=%q stdout=%q", wire, payload.Transcript, e.nextCall, e.now, e.advances, e.transcriptLength, e.transcript, e.stdout)
			}
		})
	}
}

func TestHelperBridgeModelRejectsStateBeforeMutationAndKeepsOperationError(t *testing.T) {
	for _, test := range []struct {
		name                          string
		id, now, advances, transcript uint64
	}{
		{"call-backwards", 1, 20, 2, 9}, {"call-over-limit", 10001, 20, 2, 9},
		{"clock-backwards", 3, 19, 2, 9}, {"clock-overflow", 3, math.MaxInt64, 2, 9},
		{"advance-backwards", 3, 20, 1, 9}, {"transcript-backwards", 3, 20, 2, 8},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := newTestEnvironment(t, testConfig())
			e.nextCall, e.now, e.advances, e.transcriptLength = 2, 20, 2, 9
			e.transcript = []byte("retained")
			_, err := helperBridgeModel(t, e).helperCallbackReply(helperBridgeCall(test.id, "fd_write", `{"fd":1,"data_base64":"eA=="}`), helperLocalState{Now: test.now, Advances: test.advances, TranscriptBytes: test.transcript})
			boundary, ok := err.(*BoundaryError)
			if !ok || boundary.Kind != "invalid" || boundary.Dimension != "helper callback state" ||
				e.nextCall != 2 || e.now != 20 || e.advances != 2 || e.transcriptLength != 9 || string(e.transcript) != "retained" || len(e.stdout) != 0 {
				t.Fatalf("rejected state mutated model or lost precedence: err=%v state=%+v", err, e)
			}
		})
	}
	t.Run("operation-error-after-state-adoption", func(t *testing.T) {
		e := newTestEnvironment(t, testConfig())
		e.helperOwned = true
		_, err := helperBridgeModel(t, e).helperCallbackReply(helperBridgeCall(3, "fd_write", `{"fd":"bad","data_base64":"eA=="}`), helperLocalState{Now: 20, Advances: 2, TranscriptBytes: 9})
		boundary, ok := err.(*BoundaryError)
		if !ok || boundary.Kind != "invalid" || boundary.Dimension != "operation JSON types" || e.nextCall != 3 || e.now != 20 || e.advances != 2 || e.transcriptLength != 9 || len(e.transcript)+len(e.stdout) != 0 {
			t.Fatalf("operation failure lost exact model error/adopted state: err=%v state=%+v", err, e)
		}
	})
}

func TestHelperBridgeSplitOutputStateAdmission(t *testing.T) {
	for _, test := range []struct {
		name                string
		offered             uint64
		split, owned, valid bool
	}{
		{"adopts-local-rejected-write", 2, true, true, true},
		{"regressing", 0, true, true, false},
		{"over-limit", 1025, true, true, false},
		{"without-owned-namespace", 2, true, false, false},
		{"outside-split", 2, false, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig()
			config.Limits.OutputBytes = 1024
			e := newTestEnvironment(t, config)
			e.splitOwned, e.namespaceOwned, e.offeredOutput = test.split, test.owned, 1
			_, err := e.helperCallbackReply(helperBridgeCall(1, "fd_write", `{"fd":2,"data_base64":"eA=="}`), helperLocalState{Now: e.now, OfferedOutputBytes: test.offered})
			if test.valid {
				if err != nil || e.offeredOutput != 3 || string(e.stderr) != "x" {
					t.Fatalf("adoption failed: %v offered=%d stderr=%q", err, e.offeredOutput, e.stderr)
				}
			} else {
				boundary, ok := err.(*BoundaryError)
				if !ok || boundary.Kind != "invalid" || e.offeredOutput != 1 || e.nextCall != 1 || len(e.stderr) != 0 {
					t.Fatalf("invalid state mutated environment: %v offered=%d", err, e.offeredOutput)
				}
			}
		})
	}
}

func TestHelperBridgeModelPreservesConcreteStorageCrashAndTransportInterrupt(t *testing.T) {
	for _, transport := range []bool{false, true} {
		t.Run(fmt.Sprint(transport), func(t *testing.T) {
			config := storageConfig()
			config.Storage.Faults = []StorageFault{{Name: "before-sync", Resource: "/workspace/db", Operation: "sync", Occurrence: 1, Action: "crash-before"}}
			e := newTestEnvironment(t, config)
			fd := openPath(t, e, 4, "db", 1, rightsAll)
			e.helperOwned = true
			owned := helperBridgeCall(e.nextCall, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd))
			state := helperLocalState{Now: e.now, Advances: e.advances, TranscriptBytes: e.transcriptLength}
			if !transport {
				_, err := helperBridgeModel(t, e).helperCallbackReply(owned, state)
				crash, ok := err.(*StorageCrash)
				if !ok || crash.Fault != "before-sync" {
					t.Fatalf("model crash became transport error: %#v", err)
				}
				return
			}
			var wire bytes.Buffer
			sink := frameSink{environment: e, input: &wire}
			if err := sink.helperCallback(owned, state); err != nil {
				t.Fatal(err)
			}
			if wire.String() != "{\"type\":\"interrupt\",\"termination\":\"storage_crash\",\"message\":\"modeled storage crash: before-sync\"}\n" || !sink.storageInterrupted || sink.interruption != "storage_crash" {
				t.Fatalf("storage interruption wire/state changed: %q %+v", wire.String(), sink)
			}
		})
	}
}

func TestHelperBridgeTransportChunkOrderAndWriteErrorPrecedence(t *testing.T) {
	for _, failAt := range []int{0, 1, 2, 3} {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			e := newTestEnvironment(t, testConfig())
			e.helperOwned = true
			data := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 90000)))
			input := `{"fd":1,"data_base64":"` + data + `"}`
			wantTrace := "\x08\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x9c\x5f\x01\x00\x01\x00\x00\x00\x90\x5f\x01\x00" + strings.Repeat("x", 90000) + "\x90\x5f\x01\x00"
			failure := errors.New("transport failed")
			var frames [][]byte
			sink := frameSink{environment: e, input: transportWriteFunc(func(frame []byte) (int, error) {
				frames = append(frames, bytes.Clone(frame))
				if len(frames) == failAt {
					return len(frame) - 1, failure
				}
				return len(frame), nil
			})}
			err := sink.helperCallback(helperBridgeCall(1, "fd_write", input), helperLocalState{Now: 20, Advances: 2, TranscriptBytes: 9})
			wantCount := 3
			if failAt != 0 {
				wantCount = failAt
			}
			if len(frames) != wantCount || failAt == 0 && err != nil || failAt != 0 && err != failure {
				t.Fatalf("write error/order precedence: err=%v frames=%d", err, len(frames))
			}
			var joined []byte
			for index, frame := range frames {
				if index == 2 {
					if string(frame) != "{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"written\":90000},\"local_state\":{\"now\":20,\"advances\":2},\"transcript_base64\":\"\"}\n" {
						t.Fatalf("terminal reply bytes changed: %q", frame)
					}
					continue
				}
				var chunk struct {
					Type string `json:"type"`
					Data []byte `json:"data_base64"`
				}
				if err := json.Unmarshal(frame, &chunk); err != nil {
					t.Fatal(err)
				}
				if chunk.Type != "remote_transcript" || index == 0 && len(chunk.Data) != 65536 || index == 1 && len(chunk.Data) != len(wantTrace)-65536 {
					t.Fatalf("transcript chunk/order changed: %s %d", chunk.Type, len(chunk.Data))
				}
				joined = append(joined, chunk.Data...)
			}
			if !bytes.Equal(joined, []byte(wantTrace[:len(joined)])) || failAt == 0 && string(joined) != wantTrace || e.nextCall != 2 || e.transcriptLength != 9+uint64(len(wantTrace)) || len(e.transcript) != 0 {
				t.Fatal("transport changed canonical transcript or calculated model state")
			}
		})
	}
}

func TestHelperBridgeReplayTraceFrameBoundary(t *testing.T) {
	for _, count := range []int{0, 65536, 65537} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			source := bytes.Repeat([]byte{'\n'}, count)
			e := newTestEnvironment(t, testConfig())
			e.helperOwned = true
			e.runtime = &runtimeSession{control: RuntimeControl{Choice: &choice.Request{Mode: choice.ModeReplay, Limit: 128 << 10, Trace: source}}}
			var wire bytes.Buffer
			sink := frameSink{environment: e, input: &wire}
			if err := sink.sendHelperReplay(); err != nil {
				t.Fatal(err)
			}
			reader := bufio.NewReader(&wire)
			wantFrames := uint64((count + 65535) / 65536)
			marker := func(wantType string) {
				t.Helper()
				line, err := reader.ReadBytes('\n')
				if err != nil {
					t.Fatal(err)
				}
				var declaration struct {
					Type        string `json:"type"`
					Schema      string `json:"schema"`
					TraceBytes  uint64 `json:"trace_bytes"`
					TraceFrames uint64 `json:"trace_frames"`
				}
				if err := json.Unmarshal(line, &declaration); err != nil {
					t.Fatal(err)
				}
				if declaration.Type != wantType || declaration.Schema != "gomad.replay-trace/v1" || declaration.TraceBytes != uint64(count) || declaration.TraceFrames != wantFrames {
					t.Fatalf("replay marker differs: %+v", declaration)
				}
			}
			marker("replay_trace")
			for offset := 0; offset < count; {
				var length [4]byte
				if _, err := io.ReadFull(reader, length[:]); err != nil {
					t.Fatal(err)
				}
				want := min(count-offset, 65536)
				if binary.BigEndian.Uint32(length[:]) != uint32(want) {
					t.Fatalf("frame length=%d want=%d", binary.BigEndian.Uint32(length[:]), want)
				}
				payload := make([]byte, want)
				if _, err := io.ReadFull(reader, payload); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(payload, source[offset:offset+want]) {
					t.Fatal("raw LF-bearing replay payload changed")
				}
				offset += want
			}
			marker("replay_trace_end")
			if _, err := reader.ReadByte(); err != io.EOF {
				t.Fatalf("unexpected trailing replay input: %v", err)
			}
		})
	}
}

type transportWriteFunc func([]byte) (int, error)

func (write transportWriteFunc) Write(data []byte) (int, error) { return write(data) }

func TestHelperModelRejectsRetiredNetworkEvidence(t *testing.T) {
	environment := newTestEnvironment(t, testConfig())
	environment.helperOwned = true
	sink := frameSink{environment: environment, started: &Started{}}
	frame, err := newValidatedFrame([]byte(`{"type":"evidence","stream":"network","data_base64":"AQ=="}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.receiveHelperEvidence(frame); err == nil {
		t.Fatal("retired network v1 helper evidence admitted")
	}
	if len(sink.helperEvidence.choice)+len(sink.helperEvidence.diagnostic)+len(sink.helperEvidence.transcript) != 0 {
		t.Fatal("retired network v1 helper evidence mutated retained streams")
	}
}
