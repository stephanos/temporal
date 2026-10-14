package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"reflect"
	"testing"
)

func TestNamespaceGoFilesystemCounterCountsActualServedErrorsAndPoll(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	call(t, e, "fd_read", `{"fd":99,"length":1}`)
	call(t, e, "path_create_directory", `{"fd":99,"path_base64":"eA=="}`)
	call(t, e, "poll_oneoff", `{"subscriptions":[{"userdata":1,"type":1,"fd":99}]}`)
	call(t, e, "clock_time_get", `{"clock_id":1,"precision":0}`)
	call(t, e, "poll_oneoff", `{"subscriptions":[{"userdata":2,"type":0,"clock_id":1,"timeout":1,"precision":0,"flags":0}]}`)
	want := map[string]uint64{"fd_read": 1, "path_create_directory": 1, "poll_oneoff": 1}
	if got := e.filesystemServed(); !reflect.DeepEqual(got, want) {
		t.Fatalf("served filesystem operations=%v, want %v", got, want)
	}
	got := e.filesystemServed()
	got["fd_read"]++
	if !reflect.DeepEqual(e.filesystemServed(), want) {
		t.Fatal("result counter aliases model state")
	}
}

func TestNamespaceOwnedCallbackRefusedBeforeAdoptingRemoteState(t *testing.T) {
	for _, op := range []struct{ name, input string }{
		{"fd_close", `{"fd":3}`}, {"path_create_directory", `{"fd":3,"path_base64":"eA=="}`},
		{"poll_oneoff", `{"subscriptions":[{"userdata":1,"type":0,"clock_id":1,"timeout":1,"precision":0,"flags":0}]}`},
	} {
		t.Run(op.name, func(t *testing.T) {
			e := newTestEnvironment(t, testConfig())
			e.namespaceOwned = true
			before := namespaceConformanceDigest(e)
			owned := validatedCall{call: Call{Type: "call", ID: 3, Op: op.name, Input: json.RawMessage(op.input)}, input: &validatedInput{data: json.RawMessage(op.input)}}
			_, err := e.helperCallbackReply(owned, helperLocalState{Now: 2, Advances: 1, TranscriptBytes: 8})
			if err == nil || e.nextCall != 1 || e.now != 0 || e.advances != 0 || e.transcriptLength != 0 || before != namespaceConformanceDigest(e) || len(e.filesystemServed()) != 0 {
				t.Fatal("helper-owned filesystem callback committed state or was accepted", err)
			}
		})
	}
}

func TestNamespaceHelperOutputStreamsPreserveMirrorAndAggregateCapacity(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	e.helperOwned, e.namespaceOwned = true, true
	var mirror bytes.Buffer
	e.mirror = &mirror
	sink := &frameSink{environment: e, started: &Started{}, request: Request{Config: testConfig()}}
	sink.request.Config.Limits.OutputBytes = 3
	write := func(stream, data string) error {
		frame, err := newValidatedFrame([]byte(`{"type":"evidence","stream":"` + stream + `","data_base64":"` + data + `"}`))
		if err != nil {
			return err
		}
		return sink.receiveHelperEvidence(frame)
	}
	if err := write("stdout", "YWI="); err != nil || mirror.String() != "ab" {
		t.Fatal("helper stdout/mirror was not retained", err)
	}
	if err := write("stderr", "eA=="); err != nil {
		t.Fatal(err)
	}
	if err := write("stderr", "eQ=="); err == nil || err.Error() != "WASI capacity: offered-output-bytes" {
		t.Fatal("combined output capacity classification changed", err)
	}
	if mirror.String() != "ab" {
		t.Fatal("stderr or refusal reached stdout mirror")
	}
	e.namespaceOwned = false
	if err := write("stdout", "eg=="); err == nil {
		t.Fatal("remote namespace accepted helper output")
	}
}

func TestNamespacePartialEvidenceKeepsRawAbnormalPrefixWithoutInventingCounts(t *testing.T) {
	sink := &frameSink{helperEvidence: helperEvidence{transcript: []byte("GMWT\x01\x00\x00\x00\x08"), choice: []byte("partial choice"), diagnostic: []byte("partial diagnostic")}}
	result := Result{Termination: "infrastructure"}
	sink.retainPartialHelperEvidence(&result)
	if !bytes.Equal(result.PartialHelperTranscript, sink.helperEvidence.transcript) || !bytes.Equal(result.PartialHelperChoice, sink.helperEvidence.choice) || !bytes.Equal(result.PartialHelperDiagnostic, sink.helperEvidence.diagnostic) || result.Calls != 0 || result.RuntimeInterrupted {
		t.Fatal("raw abnormal evidence lost or promoted to completed model evidence")
	}
	result.PartialHelperTranscript[0] = 'x'
	if sink.helperEvidence.transcript[0] != 'G' {
		t.Fatal("partial result aliases transport evidence")
	}
}

func TestNamespaceUnsupportedTranscriptAdmissionBound(t *testing.T) {
	for _, test := range []struct {
		name, input string
		limit       uint64
	}{
		{"fd_fdstat_set_flags", `{"fd":3,"flags":2}`, 30},
		{"path_open", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":0,"rights_base":2,"rights_inheriting":0,"fdflags":2}`, 61},
		{"path_readlink", `{"fd":3,"path_base64":"eA==","length":0}`, 41},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig()
			config.Limits.TranscriptBytes = test.limit
			e := newTestEnvironment(t, config)
			before := namespaceConformanceDigest(e)
			_, err := e.Handle(Call{Type: "call", ID: 1, Op: test.name, Input: json.RawMessage(test.input)})
			if err == nil || err.Error() != "WASI capacity: transcript-bytes" || e.nextCall != 1 || before != namespaceConformanceDigest(e) || len(e.Transcript()) != 0 || len(e.filesystemServed()) != 0 {
				t.Fatalf("unsupported admission=%v next=%d transcript=%d limit=%d", err, e.nextCall, len(e.Transcript()), test.limit)
			}
		})
	}
}

func TestNamespaceUnsupportedTranscriptExactBoundary(t *testing.T) {
	for _, test := range []struct {
		op, input, supported, message string
		limit                         uint64
	}{
		{"fd_fdstat_set_flags", `{"fd":3,"flags":2}`, `{"fd":3,"flags":1}`, "descriptor flags outside volatile profile", 75},
		{"path_open", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":0,"rights_base":2,"rights_inheriting":0,"fdflags":2}`, `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":1,"rights_base":2,"rights_inheriting":0,"fdflags":0}`, "open flags outside volatile profile", 96},
		{"path_readlink", `{"fd":3,"path_base64":"eA==","length":0}`, "", "operation outside stock WASI profile: path_readlink", 92},
	} {
		t.Run(test.op, func(t *testing.T) {
			inputs := []string{test.input}
			if test.supported != "" {
				inputs = append(inputs, test.supported)
			}
			for index, input := range inputs {
				config := testConfig()
				config.Limits.TranscriptBytes = test.limit - 1
				e := newTestEnvironment(t, config)
				before := namespaceConformanceDigest(e)
				_, err := e.Handle(Call{Type: "call", ID: 1, Op: test.op, Input: json.RawMessage(input)})
				if err == nil || err.Error() != "WASI capacity: transcript-bytes" || e.nextCall != 1 || before != namespaceConformanceDigest(e) || len(e.Transcript()) != 0 || len(e.filesystemServed()) != 0 {
					t.Fatal("ceiling-minus-one committed before admission", err)
				}
				config.Limits.TranscriptBytes = test.limit
				e = newTestEnvironment(t, config)
				reply := call(t, e, test.op, input)
				if reply.Errno != 0 || reply.Unsupported != (index == 0) || index == 0 && reply.Message != test.message || e.nextCall != 2 || e.filesystemServed()[test.op] != 1 || uint64(len(e.Transcript())) > test.limit {
					t.Fatal("exact supported/unsupported boundary", reply, len(e.Transcript()))
				}
				records, err := DecodeTranscript(e.Transcript())
				if err != nil || len(records) != 1 || records[0].Message != reply.Message {
					t.Fatal("unsupported message bytes changed", err)
				}
				normal := newTestEnvironment(t, testConfig())
				normalReply := call(t, normal, test.op, input)
				if !reflect.DeepEqual(reply, normalReply) || !bytes.Equal(e.Transcript(), normal.Transcript()) || namespaceConformanceDigest(e) != namespaceConformanceDigest(normal) {
					t.Fatal("normal-limit behavior changed")
				}
				if index == 0 {
					bad := append([]byte(nil), e.Transcript()...)
					bad[len(bad)-1] = 0xff
					if _, err := DecodeTranscript(bad); err == nil {
						t.Fatal("malformed unsupported message accepted")
					}
					config.Limits.TranscriptBytes = 1
					digested := newTestEnvironment(t, config)
					digested.digestOnly = true
					digested.transcriptHash = sha256.New()
					if got := call(t, digested, test.op, input); !reflect.DeepEqual(got, reply) || len(digested.Transcript()) != 0 || digested.TranscriptSHA256() != e.TranscriptSHA256() {
						t.Fatal("digest-only unsupported behavior changed")
					}
				}
			}
		})
	}
}
