package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestTransportInputValidationPrecedenceAndMutation(t *testing.T) {
	for _, test := range []struct {
		name, op, input, transportError, publicError string
		id, next, calls, transcript                  uint64
	}{
		{name: "write", op: "fd_write", input: `{"fd":1,"data_base64":"eA=="}`},
		{name: "empty-operation", op: "", input: `{}`},
		{name: "empty-operation-sequence-first", id: 2, op: "", input: `{}`, transportError: "WASI invalid: callback sequence"},
		{name: "empty-operation-calls-first", id: 2, next: 2, calls: 1, op: "", input: `{}`, transportError: "WASI capacity: callbacks"},
		{name: "empty-operation-reservation-first", transcript: 1, op: "", input: `{}`, transportError: "WASI capacity: transcript-bytes"},
		{name: "empty-operation-malformed-input", op: "", input: `{"x":}`, transportError: "WASI invalid: malformed JSON", publicError: "WASI invalid: callback JSON"},
		{name: "empty-operation-malformed-before-sequence", id: 2, op: "", input: `{"x":}`, transportError: "WASI invalid: malformed JSON", publicError: "WASI invalid: callback sequence"},
		{name: "escaped-known-key", op: "fd_write", input: `{"f\u0064":1,"data_base64":"eA=="}`},
		{name: "reordered-whitespace", op: "fd_write", input: ` { "data_base64" : "eA==", "fd" : 1 } `},
		{name: "missing", op: "fd_write", input: `{"data_base64":"eA=="}`, transportError: "WASI invalid: operation fields"},
		{name: "missing-same-count", op: "fd_write", input: `{"other":1,"data_base64":"eA=="}`, transportError: `WASI unsupported: unknown field "other"`, publicError: "WASI invalid: missing operation field fd"},
		{name: "null", op: "fd_write", input: `{"fd":null,"data_base64":"eA=="}`, transportError: "WASI invalid: missing operation field fd"},
		{name: "type", op: "fd_write", input: `{"fd":"1","data_base64":"eA=="}`, transportError: "WASI invalid: operation JSON types"},
		{name: "integer-overflow", op: "fd_write", input: `{"fd":4294967296,"data_base64":"eA=="}`, transportError: "WASI invalid: operation JSON types"},
		{name: "unknown-before-type", op: "fd_write", input: `{"fd":"1","data_base64":"eA==","other":0}`, transportError: `WASI unsupported: unknown field "other"`, publicError: "WASI invalid: operation fields"},
		{name: "escaped-duplicate", op: "fd_write", input: `{"fd":1,"f\u0064":2,"data_base64":"eA=="}`, transportError: "WASI invalid: duplicate JSON field"},
		{name: "nested-duplicate-before-fields", op: "fd_write", input: `{"fd":1,"data_base64":"eA==","other":{"x":0,"\u0078":1}}`, transportError: "WASI invalid: duplicate JSON field"},
		{name: "length-before-fields", op: "random_get", input: `{"length":-1,"other":0}`, transportError: `WASI unsupported: unknown field "other"`, publicError: "WASI invalid: operation length"},
		{name: "buffer-before-fields", op: "random_get", input: `{"length":4194305,"other":0}`, transportError: `WASI unsupported: unknown field "other"`, publicError: "WASI capacity: buffer-bytes"},
		{name: "null-length", op: "random_get", input: `{"length":null}`, transportError: "WASI invalid: missing operation field length"},
		{name: "array-input", op: "fd_write", input: `[]`, transportError: "WASI invalid: operation input"},
		{name: "null-input", op: "fd_write", input: `null`, transportError: "WASI invalid: missing operation field input", publicError: "WASI invalid: operation input"},
		{name: "sequence-before-operation-schema", id: 2, op: "fd_write", input: `{}`, transportError: "WASI invalid: callback sequence"},
		{name: "frame-duplicate-before-sequence", id: 2, op: "fd_write", input: `{"fd":1,"fd":2}`, transportError: "WASI invalid: duplicate JSON field", publicError: "WASI invalid: callback sequence"},
		{name: "calls-before-length", id: 2, next: 2, calls: 1, op: "random_get", input: `{"length":-1}`, transportError: "WASI capacity: callbacks"},
		{name: "reservation-before-schema", transcript: 1, op: "fd_write", input: `{}`, transportError: "WASI capacity: transcript-bytes"},
		{name: "full-frame-depth-edge", op: "unsupported", input: `{"x":` + strings.Repeat("[", 30) + "0" + strings.Repeat("]", 30) + `}`},
		{name: "full-frame-empty-container-depth-edge", op: "unsupported", input: `{"x":` + strings.Repeat("[", 31) + strings.Repeat("]", 31) + `}`},
		{name: "full-frame-depth-overflow", op: "unsupported", input: `{"x":` + strings.Repeat("[", 31) + "0" + strings.Repeat("]", 31) + `}`, transportError: "WASI invalid: JSON nesting", publicError: "accepted"},
		{name: "full-frame-malformed-depth-overflow", op: "unsupported", input: `{"x":` + strings.Repeat("[", 31) + "|" + strings.Repeat("]", 31) + `}`, transportError: "WASI invalid: JSON nesting", publicError: "WASI invalid: callback JSON"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig()
			if test.calls != 0 {
				config.Limits.Calls = test.calls
			}
			if test.transcript != 0 {
				config.Limits.TranscriptBytes = test.transcript
			}
			id, next := test.id, test.next
			if id == 0 {
				id = 1
			}
			if next == 0 {
				next = 1
			}
			transport := newTestEnvironment(t, config)
			transport.nextCall = next
			sink := frameSink{environment: transport, started: &Started{}, input: io.Discard}
			frame := []byte(fmt.Sprintf(`{"type":"call","id":%d,"op":%q,"input":%s}`, id, test.op, test.input))
			err := sink.frame(frame)
			assertInputOutcome(t, transport, next, err, test.transportError)
			public := newTestEnvironment(t, config)
			public.nextCall = next
			_, err = public.Handle(Call{Type: "call", ID: id, Op: test.op, Input: json.RawMessage(test.input)})
			want := test.publicError
			if want == "" {
				want = test.transportError
			}
			if want == "accepted" {
				want = ""
			}
			assertInputOutcome(t, public, next, err, want)
			if test.publicError == "" && test.transportError == "" && (!bytes.Equal(transport.Transcript(), public.Transcript()) || !bytes.Equal(transport.stdout, public.stdout)) {
				t.Fatal("transport changed canonical trace/output from public reference")
			}
			if test.name == "write" {
				const trace = "GMWT\x01\x00\x00\x00\x08\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x0d\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00\x00x\x01\x00\x00\x00"
				if string(transport.Transcript()) != trace || string(transport.stdout) != "x" {
					t.Fatalf("literal canonical write = %q stdout=%q", transport.Transcript(), transport.stdout)
				}
			}
		})
	}
}

func assertInputOutcome(t *testing.T, e *Environment, next uint64, err error, want string) {
	t.Helper()
	got := ""
	if err != nil {
		got = err.Error()
	}
	if got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
	if want != "" && (e.nextCall != next || e.offeredOutput != 0 || e.entropyOffset != 0 || e.now != 0 || len(e.stdout)+len(e.stderr)+len(e.Transcript()) != 0) {
		t.Fatalf("rejected input mutated model: next=%d offered=%d entropy=%d now=%d", e.nextCall, e.offeredOutput, e.entropyOffset, e.now)
	}
}

func TestPublicCallRevalidatesChangedInput(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	input := json.RawMessage(`{"fd":1,"data_base64":"eA=="}`)
	request := Call{Type: "call", ID: 1, Op: "fd_write", Input: input}
	if _, err := e.Handle(request); err != nil {
		t.Fatal(err)
	}
	input[6] = '2'
	request.ID = 2
	if _, err := e.Handle(request); err != nil {
		t.Fatal(err)
	}
	if string(e.stdout) != "x" || string(e.stderr) != "x" {
		t.Fatal("reused public call decoded stale input")
	}
	request.ID, request.Input = 3, json.RawMessage(`{"fd":1,"f\u0064":2,"data_base64":"eA=="}`)
	before := bytes.Clone(e.Transcript())
	if _, err := e.Handle(request); err == nil || err.Error() != "WASI invalid: duplicate JSON field" {
		t.Fatalf("changed public call skipped validation: %v", err)
	}
	if e.nextCall != 3 || string(e.stdout) != "x" || string(e.stderr) != "x" || !bytes.Equal(before, e.Transcript()) {
		t.Fatal("changed invalid public call mutated model")
	}
}

func TestValidatedFrameOwnsBytesAndOperationIsLazy(t *testing.T) {
	data := []byte(`{"type":"call","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"eA=="}}`)
	frame, err := newValidatedFrame(data)
	if err != nil {
		t.Fatal(err)
	}
	data[bytes.Index(data, []byte(`"id":1`))+5] = '2'
	data[bytes.Index(data, []byte(`"fd":1`))+5] = '2'
	owned, _, err := frame.decodeCall(false)
	if err != nil {
		t.Fatal(err)
	}
	if owned.input.decoded {
		t.Fatal("operation fields decoded before reservation")
	}
	e := newTestEnvironment(t, testConfig())
	if _, err := e.handleValidated(owned); err != nil {
		t.Fatalf("caller mutation changed validated frame: %v", err)
	}
	if string(e.stdout) != "x" || len(e.stderr) != 0 || !owned.input.decoded {
		t.Fatal("owned frame mixed original fields with caller-mutated bytes")
	}
	bad, err := newValidatedFrame([]byte(`{"type":"call","id":2,"op":"fd_write","input":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	call, _, err := bad.decodeCall(false)
	if err != nil || call.input.decoded {
		t.Fatalf("operation map eagerly rejected before sequence: %v", err)
	}
	_, err = newTestEnvironment(t, testConfig()).handleValidated(call)
	if err == nil || err.Error() != "WASI invalid: callback sequence" || call.input.decoded {
		t.Fatalf("lazy operation changed sequence precedence: %v", err)
	}
}

func TestValidatedFrameRejectsInvalidUTF8(t *testing.T) {
	data := append([]byte(`{"type":"call","id":1,"op":"`), 0xff)
	data = append(data, []byte(`","input":{}}`)...)
	if _, err := newValidatedFrame(data); err == nil {
		t.Fatal("invalid UTF-8 frame was accepted")
	}
}

func TestValidatedNamespaceMatchesPublicReference(t *testing.T) {
	transport, public := newTestEnvironment(t, testConfig()), newTestEnvironment(t, testConfig())
	for _, operation := range []struct{ op, input string }{
		{"args_get", `{}`},
		{"environ_sizes_get", `{}`},
		{"random_get", `{"length":64}`},
		{"fd_prestat_get", `{"fd":3}`},
		{"fd_prestat_dir_name", `{"fd":3,"length":4}`},
		{"path_open", fmt.Sprintf(`{"fd":3,"path_base64":"ZmlsZQ==","dirflags":0,"oflags":1,"rights_base":%d,"rights_inheriting":0,"fdflags":0}`, rightsAll)},
		{"fd_fdstat_get", `{"fd":5}`},
		{"fd_write", `{"fd":5,"data_base64":"YWJj"}`},
		{"fd_pwrite", `{"fd":5,"data_base64":"eA==","offset":1}`},
		{"fd_pread", `{"fd":5,"length":3,"offset":0}`},
		{"fd_seek", `{"fd":5,"offset":0,"whence":0}`},
		{"fd_read", `{"fd":5,"length":3}`},
		{"fd_filestat_set_size", `{"fd":5,"size":2}`},
		{"fd_filestat_get", `{"fd":5}`},
		{"fd_fdstat_set_flags", `{"fd":5,"flags":1}`},
		{"fd_sync", `{"fd":5}`},
		{"fd_readdir", `{"fd":3,"length":128,"cookie":0}`},
		{"path_create_directory", `{"fd":3,"path_base64":"ZGly"}`},
		{"path_filestat_get", `{"fd":3,"path_base64":"ZmlsZQ==","flags":0}`},
		{"path_rename", `{"fd":3,"path_base64":"ZmlsZQ==","new_fd":3,"new_path_base64":"bW92ZWQ="}`},
		{"path_unlink_file", `{"fd":3,"path_base64":"bW92ZWQ="}`},
		{"path_remove_directory", `{"fd":3,"path_base64":"ZGly"}`},
		{"clock_time_get", `{"clock_id":0,"precision":0}`},
		{"poll_oneoff", `{"subscriptions":[{"userdata":1,"type":0,"clock_id":1,"timeout":100,"precision":0,"flags":0}]}`},
		{"fd_close", `{"fd":5}`},
	} {
		t.Run(operation.op, func(t *testing.T) {
			call := Call{Type: "call", ID: public.nextCall, Op: operation.op, Input: json.RawMessage(operation.input)}
			data, err := json.Marshal(call)
			if err != nil {
				t.Fatal(err)
			}
			frame, err := newValidatedFrame(data)
			if err != nil {
				t.Fatal(err)
			}
			owned, _, err := frame.decodeCall(false)
			if err != nil {
				t.Fatal(err)
			}
			got, err := transport.handleValidated(owned)
			if err != nil {
				t.Fatal(err)
			}
			want, err := public.Handle(call)
			if err != nil {
				t.Fatal(err)
			}
			gotBytes, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			wantBytes, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotBytes, wantBytes) || !bytes.Equal(transport.Transcript(), public.Transcript()) || transport.nextCall != public.nextCall || transport.now != public.now || transport.entropyOffset != public.entropyOffset || transport.namespace.bytes != public.namespace.bytes {
				t.Fatalf("private/public branch differs: %s / %s", gotBytes, wantBytes)
			}
		})
	}
}

func BenchmarkValidatedTransportInput(b *testing.B) {
	data := []byte(`{"type":"call","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"eA=="}}`)
	for _, owned := range []bool{false, true} {
		b.Run(map[bool]string{false: "reference", true: "owned"}[owned], func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var input struct {
					FD   uint32 `json:"fd"`
					Data []byte `json:"data_base64"`
				}
				if owned {
					frame, err := newValidatedFrame(data)
					if err != nil {
						b.Fatal(err)
					}
					call, _, err := frame.decodeCall(false)
					if err != nil {
						b.Fatal(err)
					}
					if _, err := call.input.fields(); err != nil {
						b.Fatal(err)
					}
					if err := decodeOperationInput(call.call.Input, call.input, &input, "fd", "data_base64"); err != nil {
						b.Fatal(err)
					}
				} else {
					if err := validateJSON(data); err != nil {
						b.Fatal(err)
					}
					var fields map[string]json.RawMessage
					if err := json.Unmarshal(data, &fields); err != nil {
						b.Fatal(err)
					}
					var call Call
					if err := decodeInput(data, &call, "type", "id", "op", "input"); err != nil {
						b.Fatal(err)
					}
					if err := validateJSON(call.Input); err != nil {
						b.Fatal(err)
					}
					var operationFields map[string]json.RawMessage
					if err := json.Unmarshal(call.Input, &operationFields); err != nil {
						b.Fatal(err)
					}
					if err := decodeInput(call.Input, &input, "fd", "data_base64"); err != nil {
						b.Fatal(err)
					}
				}
				if input.FD != 1 || string(input.Data) != "x" {
					b.Fatal("diagnostic decoder changed the literal operation")
				}
			}
		})
	}
}
