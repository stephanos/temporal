package world

import (
	"errors"
	"testing"
)

func TestWorldRecordingEnforcesTransitionBytesBeforeMutation(t *testing.T) {
	w := newTestWorld(t, 4)
	if _, err := w.StartRecording(1); err != nil {
		t.Fatal(err)
	}
	before := w.Snapshot()
	_, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}})
	var capacityError *CapacityError
	if !errors.As(err, &capacityError) || capacityError.Dimension != "transition-bytes" {
		t.Fatalf("Register() error = %#v", err)
	}
	if after := w.Snapshot(); after.StateDigest != before.StateDigest {
		t.Fatal("transition-byte rejection mutated World")
	}
}

func TestWorldRecordingEnvelopeRoundTripsExecutedTransitions(t *testing.T) {
	w := newTestWorld(t, 5)
	recorder, err := w.StartRecording(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	id, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Ready(Readiness{RequestID: id, At: InitialTime, Kind: "done"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Quiesce(); err != nil {
		t.Fatal(err)
	}
	recording, err := recorder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeRecording(recording)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRecording(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Initial.StateDigest != recording.Initial.StateDigest || decoded.Final.StateDigest != recording.Final.StateDigest || len(decoded.Final.Transitions)-len(decoded.Initial.Transitions) != 3 {
		t.Fatalf("decoded recording = %#v", decoded)
	}
}

func TestWorldRecordingCarriesStructuredTerminalResults(t *testing.T) {
	w := newTestWorld(t, 1)
	recorder, err := w.StartRecording(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}}); err != nil {
		t.Fatal(err)
	}
	if result, err := w.Quiesce(); err != nil || result.Kind != QuiescenceDeadlock {
		t.Fatalf("Quiesce() = %#v, %v", result, err)
	}
	recording, err := recorder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeRecording(recording)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRecording(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Terminal.Kind != TerminalDeadlock || decoded.Terminal.Detail != "" {
		t.Fatalf("terminal = %#v", decoded.Terminal)
	}
}

func TestWorldRecordingCarriesStructuredCapacityError(t *testing.T) {
	limits := testLimits()
	limits.MaxRequests = 1
	w, err := New(Config{Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := w.StartRecording(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}}
	if _, err := w.Register(request); err != nil {
		t.Fatal(err)
	}
	_, capacityErr := w.Register(request)
	recording, err := recorder.FinishError(capacityErr)
	if err != nil {
		t.Fatal(err)
	}
	if recording.Terminal.Kind != TerminalCapacity || recording.Terminal.Detail == "" {
		t.Fatalf("terminal = %#v", recording.Terminal)
	}
}
