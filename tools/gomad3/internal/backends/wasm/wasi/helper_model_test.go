package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	runnerbackend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

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

func TestHelperModelPreservesMaximumRemoteReply(t *testing.T) {
	imports := []Import{}
	for _, name := range []string{"config", "decision", "observation", "finish", "idle"} {
		imports = append(imports, Import{"gomad_wasm_v1", name, []string{"i32", "i32"}, []string{"i32"}})
	}
	imports = append(imports, Import{"wasi_snapshot_preview1", "random_get", []string{"i32", "i32"}, []string{"i32"}})
	request := fixtureRequest(t, `(module
 (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "decision" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "observation" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "idle" (func (param i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "random_get" (func $random (param i32 i32) (result i32)))
 (memory (export "memory") 65)
 (data (i32.const 16) "\00\00\00\01")
 (func (export "_start")
 i32.const 0 i32.const 16 call $config drop
 i32.const 64 i32.const 4194304 call $random drop
 i32.const 16 i32.const 16 call $finish drop))`, imports)
	request.InitialMemoryPages = 65
	request.Config.Profile = CooperativeProfile
	request.Config.Clock.ReadStepNanos = 0
	request.Config.Limits.TranscriptBytes = 64 << 20
	request.Runtime = &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 8192, ExecutionIdentity: choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}}}
	remote := request
	remote.remoteModel = true
	reference, err := Run(t.Context(), remote)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "maximum Go reply", reference, 0)
	local, err := Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if difference := sameEvidence(reference, local); difference != "" {
		t.Fatalf("maximum reply differs in %s: %s %s", difference, local.Termination, local.Message)
	}
	if len(local.Transcript) < bufferLimit {
		t.Fatal("maximum byte reply was not observed")
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
	const writeTrace = "{\"type\":\"call\",\"id\":3,\"op\":\"fd_write\",\"input\":{\"fd\":1,\"data_base64\":\"eA==\"}}\n{\"type\":\"reply\",\"id\":3,\"errno\":0,\"output\":{\"written\":1}}\n"
	for _, test := range []struct {
		name, op, input, trace, wire, stdout string
		now                                  uint64
		runtime                              bool
	}{
		{"write", "fd_write", `{"fd":1,"data_base64":"eA=="}`, writeTrace,
			`{"type":"reply","id":3,"errno":0,"output":{"written":1},"local_state":{"now":20,"advances":2},"transcript_base64":"eyJ0eXBlIjoiY2FsbCIsImlkIjozLCJvcCI6ImZkX3dyaXRlIiwiaW5wdXQiOnsiZmQiOjEsImRhdGFfYmFzZTY0IjoiZUE9PSJ9fQp7InR5cGUiOiJyZXBseSIsImlkIjozLCJlcnJubyI6MCwib3V0cHV0Ijp7IndyaXR0ZW4iOjF9fQo="}`, "x", 20, false},
		{"clock-state-after-handle", "clock_time_get", `{"clock_id":1,"precision":0}`,
			"{\"type\":\"call\",\"id\":3,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":0}}\n{\"type\":\"reply\",\"id\":3,\"errno\":0,\"output\":{\"timestamp\":20}}\n",
			`{"type":"reply","id":3,"errno":0,"output":{"timestamp":20},"local_state":{"now":1020,"advances":2},"transcript_base64":"eyJ0eXBlIjoiY2FsbCIsImlkIjozLCJvcCI6ImNsb2NrX3RpbWVfZ2V0IiwiaW5wdXQiOnsiY2xvY2tfaWQiOjEsInByZWNpc2lvbiI6MH19CnsidHlwZSI6InJlcGx5IiwiaWQiOjMsImVycm5vIjowLCJvdXRwdXQiOnsidGltZXN0YW1wIjoyMH19Cg=="}`, "", 1020, false},
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
				e.runtime = &runtimeSession{}
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
			data := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 50000)))
			input := `{"fd":1,"data_base64":"` + data + `"}`
			wantTrace := `{"type":"call","id":1,"op":"fd_write","input":` + input + "}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"written\":50000}}\n"
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
					if string(frame) != "{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"written\":50000},\"local_state\":{\"now\":20,\"advances\":2},\"transcript_base64\":\"\"}\n" {
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

func TestHelperBridgeTapeChunkBoundary(t *testing.T) {
	for _, count := range []int{512, 513} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			decisions := make([]choice.Decision, count)
			for index := range decisions {
				decisions[index].Ordinal = uint64(index)
			}
			e := newTestEnvironment(t, testConfig())
			e.helperOwned = true
			e.runtime = &runtimeSession{tape: &choice.ReplayPlan{Decisions: decisions}}
			var wire bytes.Buffer
			sink := frameSink{environment: e, input: &wire}
			if err := sink.sendHelperTape(); err != nil {
				t.Fatal(err)
			}
			frames := bytes.Split(bytes.TrimSuffix(wire.Bytes(), []byte{'\n'}), []byte{'\n'})
			wantFrames := 1
			if count == 513 {
				wantFrames = 2
			}
			if len(frames) != wantFrames {
				t.Fatalf("tape chunks=%d want=%d", len(frames), wantFrames)
			}
			for index, frame := range frames {
				var tape struct {
					Type      string            `json:"type"`
					Decisions []choice.Decision `json:"decisions"`
				}
				if err := json.Unmarshal(frame, &tape); err != nil {
					t.Fatal(err)
				}
				wantCount := 512
				if index == 1 {
					wantCount = 1
				}
				if tape.Type != "tape" || len(tape.Decisions) != wantCount || tape.Decisions[0].Ordinal != uint64(index*512) || tape.Decisions[len(tape.Decisions)-1].Ordinal != uint64(index*512+wantCount-1) {
					t.Fatal("tape slice boundary/order changed")
				}
			}
		})
	}
}
