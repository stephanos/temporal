package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestCapturedEntropySharedGoldenVectors(t *testing.T) {
	for _, test := range []struct {
		offset uint64
		length uint32
		hex    string
	}{
		{0, 32, "c0728b9120addf2a1f4090cb916ed459e1acff6210a7ffc7884aa6d4eca74847"},
		{30, 6, "48476a64d720"},
		{32, 32, "6a64d7206e0172d7a257e0d0e7674667f6cd8f1af581460acbae65523c4c108e"},
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
func TestHelperModelCapturedInputBoundaryParity(t *testing.T) {
	imports := []Import{}
	for _, name := range []string{"config", "decision", "observation", "finish", "idle"} {
		imports = append(imports, Import{"gomad_wasm_v1", name, []string{"i32", "i32"}, []string{"i32"}})
	}
	for _, name := range []string{"args_sizes_get", "args_get", "environ_sizes_get", "environ_get", "random_get"} {
		imports = append(imports, Import{"wasi_snapshot_preview1", name, []string{"i32", "i32"}, []string{"i32"}})
	}
	request := fixtureRequest(t, `(module
 (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "decision" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "observation" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "idle" (func (param i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "args_sizes_get" (func $args (param i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "args_get" (func $argsget (param i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "environ_sizes_get" (func $env (param i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "environ_get" (func $envget (param i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "random_get" (func $random (param i32 i32) (result i32)))
 (memory (export "memory") 1) (data (i32.const 16) "\00\00\00\01")
 (func (export "_start") i32.const 0 i32.const 16 call $config drop
 i32.const 32 i32.const 36 call $args drop i32.const 40 i32.const 44 call $env drop
 i32.const 36 i32.load i32.const 100 i32.le_u if i32.const 128 i32.const 256 call $argsget drop end
 i32.const 44 i32.load i32.const 100 i32.le_u if i32.const 512 i32.const 640 call $envget drop end
 i32.const 2048 i32.const 30 call $random drop i32.const 2048 i32.const 6 call $random drop
 i32.const 2048 i32.load i32.const 0x646a4748 i32.ne if unreachable end
 i32.const 2052 i32.load16_u i32.const 0x20d7 i32.ne if unreachable end
 i32.const 16 i32.const 16 call $finish drop))`, imports)
	request.Config.Profile = CooperativeProfile
	request.Config.Clock.ReadStepNanos = 0
	request.Config.EntropyKey = [32]byte{}
	request.Runtime = &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 8192, ExecutionIdentity: choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}}}
	baselineConfig := request.Config
	for _, boundary := range []bool{false, true} {
		if boundary {
			request.Config.Args = []string{strings.Repeat("\x01", bufferLimit-1)}
			request.Config.WorkingDirectory = "/" + strings.Repeat("\x02", bufferLimit-6)
			request.Config.Environment = []string{"PWD=" + request.Config.WorkingDirectory}
			request.Config.WritableDirectories = []string{request.Config.WorkingDirectory}
		}
		for _, digestOnly := range []bool{false, true} {
			request.DigestTranscript = digestOnly
			remote := request
			remote.remoteModel = true
			reference, err := Run(t.Context(), remote)
			if err != nil {
				t.Fatal(err)
			}
			requireExit(t, "Go captured inputs", reference, 0)
			local, err := Run(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			requireExit(t, "helper captured inputs", local, 0)
			if diff := sameEvidence(reference, local); diff != "" || reference.Calls != local.Calls || reference.EnvironmentIdentity != local.EnvironmentIdentity {
				t.Fatalf("boundary=%t digest=%t differs in %s", boundary, digestOnly, diff)
			}
		}
	}
	request.Config = baselineConfig
	for _, limit := range []uint64{99, 100} {
		request.Config.Limits.TranscriptBytes = limit
		for _, digestOnly := range []bool{false, true} {
			request.DigestTranscript = digestOnly
			remote := request
			remote.remoteModel = true
			reference, err := Run(t.Context(), remote)
			if err != nil {
				t.Fatal(err)
			}
			local, err := Run(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			if diff := sameEvidence(reference, local); diff != "" || reference.Calls != local.Calls {
				t.Fatalf("limit=%d digest=%t differs in %s calls=%d/%d: %s %s", limit, digestOnly, diff, reference.Calls, local.Calls, local.Termination, local.Message)
			}
			if digestOnly {
				requireExit(t, "digest capacity parity", local, 0)
			} else {
				wantCalls := uint64(5)
				if limit == 100 {
					wantCalls = 6
				}
				if local.Termination != "capacity" || local.Message != "WASI capacity: transcript-bytes" || local.Calls != wantCalls {
					t.Fatalf("same-call capacity at %d = %s %s calls=%d", limit, local.Termination, local.Message, local.Calls)
				}
			}
		}
	}
}

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
	records, err := DecodeTranscript(local.Transcript)
	if err != nil || len(records) != 1 || records[0].Op != "random_get" || string(records[0].Input) != `{"length":4194304,"offset":0}` {
		t.Fatalf("maximum entropy request was not bound by its derived record: %+v %v", records, err)
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
