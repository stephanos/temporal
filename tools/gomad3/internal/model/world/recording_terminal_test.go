package world

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type callbackTerminalError struct {
	calls *int
	kind  string
}

func (e callbackTerminalError) Error() string { *e.calls++; return "external terminal" }
func (e callbackTerminalError) Is(target error) bool {
	*e.calls++
	return e.kind == "is" && target == ErrCapacity
}

type singleTerminalError struct{ callbackTerminalError }

func (e singleTerminalError) Unwrap() error { *e.calls++; return ErrCapacity }

type multiTerminalError struct{ callbackTerminalError }

func (e multiTerminalError) Unwrap() []error { *e.calls++; return []error{ErrCapacity} }

func TestRecorderRejectsExternalCallbacksWithoutMutation(t *testing.T) {
	for _, kind := range []string{"error", "is", "single", "multi", "wrapper", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			w := newTestWorld(t, 1)
			recorder, err := w.StartRecording(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			var input error = callbackTerminalError{&calls, kind}
			switch kind {
			case "single":
				input = singleTerminalError{callbackTerminalError{&calls, kind}}
			case "multi":
				input = multiTerminalError{callbackTerminalError{&calls, kind}}
			case "wrapper":
				input = fmt.Errorf("external: %w", ErrCapacity)
			case "unknown":
				input = errors.New("unknown")
			}
			calls = 0
			before := w.Snapshot()
			_, err = recorder.FinishError(input)
			if err == nil {
				t.Errorf("external terminal accepted")
			}
			if calls != 0 {
				t.Errorf("external callbacks invoked %d times", calls)
			}
			if after := w.Snapshot(); after.StateDigest != before.StateDigest || after.TranscriptDigest != before.TranscriptDigest {
				t.Error("rejected input mutated model")
			}
			if _, err := recorder.FinishError(ErrCapacity); err != nil {
				t.Errorf("rejected input closed recorder: %v", err)
			}
		})
	}
}

func TestRecorderAndModelIgnoreSentinelRebinding(t *testing.T) {
	sentinels := []struct {
		slot     *error
		original error
		kind     TerminalKind
	}{
		{&ErrInvalidConfig, ErrInvalidConfig, TerminalInvalidInput}, {&ErrInvalidRequest, ErrInvalidRequest, TerminalInvalidInput}, {&ErrUnknownRequest, ErrUnknownRequest, TerminalInvalidInput}, {&ErrRequestState, ErrRequestState, TerminalInvalidInput}, {&ErrTimeRegression, ErrTimeRegression, TerminalInvalidInput}, {&ErrInvalidSnapshot, ErrInvalidSnapshot, TerminalInvalidInput}, {&ErrCapacity, ErrCapacity, TerminalCapacity}, {&ErrReplayDivergence, ErrReplayDivergence, TerminalReplayDivergence},
	}
	defer func() {
		for _, sentinel := range sentinels {
			*sentinel.slot = sentinel.original
		}
	}()
	calls := 0
	for _, sentinel := range sentinels {
		*sentinel.slot = callbackTerminalError{&calls, "is"}
	}
	_, err := New(Config{})
	if err == nil || err.Error() != "invalid World config: every limit must be nonzero" {
		t.Errorf("model detail followed rebound sentinel: %v", err)
	}
	if calls != 0 {
		t.Errorf("model invoked rebound sentinel %d times", calls)
	}
	for _, sentinel := range sentinels {
		w := newTestWorld(t, 1)
		recorder, err := w.StartRecording(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := recorder.FinishError(*sentinel.slot); err == nil {
			t.Fatal("rebound sentinel accepted")
		}
		result, err := recorder.FinishError(sentinel.original)
		if err != nil || result.Terminal != (Terminal{Kind: sentinel.kind, Detail: sentinel.original.Error()}) {
			t.Errorf("original sentinel lost recognition: terminal=%+v err=%v", result.Terminal, err)
		}
	}
	if calls != 0 {
		t.Errorf("rebound sentinel callbacks invoked %d times", calls)
	}
}

func TestOwnedContextRetainsTypedErrorFields(t *testing.T) {
	capacity := &CapacityError{Dimension: "requests", Limit: 7, Used: 6, Delta: 2}
	replay := &ReplayDivergenceError{Index: 3, ExpectedKind: "ready", ActualKind: "cancel", Field: "digest", ExpectedDigest: "first", ActualDigest: "second"}
	for _, input := range []error{capacity, replay} {
		wrapped := modelContextError("outer", modelContextError("inner", input))
		if wrapped.Error() != "outer: inner: "+input.Error() || errors.Unwrap(errors.Unwrap(wrapped)) != input {
			t.Fatalf("context or identity changed: %v", wrapped)
		}
		var gotCapacity *CapacityError
		var gotReplay *ReplayDivergenceError
		switch input := input.(type) {
		case *CapacityError:
			if !errors.As(wrapped, &gotCapacity) || gotCapacity != input || !errors.Is(wrapped, ErrCapacity) {
				t.Fatal("capacity fields or identity changed")
			}
		case *ReplayDivergenceError:
			if !errors.As(wrapped, &gotReplay) || gotReplay != input || !errors.Is(wrapped, ErrReplayDivergence) {
				t.Fatal("replay fields or identity changed")
			}
		}
	}
}

func TestOwnedTerminalPreservesAllCategoriesAndRecordingBytes(t *testing.T) {
	preimageHashes := []string{"df29237751ff7335adedb503f646380ce39e2cc422ef4c99fd9a6dedf5de36df", "1ad3885c35b94c0ba76fb6e65c413cd905143c0c8c31d866411b4c353bab8b3f", "02b36103c4ed9bc21f34463d2f06342f7e34ff6178e1448264c21696f6d6845e", "08a9114dbf96e540be90e63701001ed3b11341862035d35cd62f10e80b6c1dea", "9f532a5b12c99874b319b2e1a725e20747bf653865ac9afc88ad8439f9549603", "e734ccba5ba2c45ed8b43280828b7f10108e440a87421365582d4805a1dbae4c", "366d8f73102c13fa0df4f584a14c5abc16b105be37462fbc19155ab11ddb1a7f", "bc5e8463e9b9168643d7ab27bb78218f0c0f7eedf7f60b64cda75ad5b66c041c", "6e413aee89e2d8088fb2caed515c669e3ca56a0032c5f352723be6edf9bbf24c", "e4e7bc86470fdead36616f84dcff4943ae398c5444fb38b98658be883127fe4d"}
	for index, test := range []struct {
		input  error
		kind   TerminalKind
		detail string
	}{
		{ErrInvalidConfig, TerminalInvalidInput, "invalid World config"}, {ErrInvalidRequest, TerminalInvalidInput, "invalid World request"}, {ErrUnknownRequest, TerminalInvalidInput, "unknown World request"}, {ErrRequestState, TerminalInvalidInput, "invalid World request state"}, {ErrTimeRegression, TerminalInvalidInput, "World time regression"}, {ErrInvalidSnapshot, TerminalInvalidInput, "invalid World snapshot"}, {ErrCapacity, TerminalCapacity, "World capacity exhausted"}, {ErrReplayDivergence, TerminalReplayDivergence, "World replay divergence"},
		{&CapacityError{Dimension: "雪\"\n" + strings.Repeat("x", 8192), Limit: 7, Used: 6, Delta: 2}, TerminalCapacity, "World capacity exhausted: 雪\"\n" + strings.Repeat("x", 8192) + " limit=7 used=6 delta=2"},
		{&ReplayDivergenceError{Index: 5, ExpectedKind: "ready", ActualKind: "cancel", Field: "digest", ExpectedDigest: Digest(strings.Repeat("a", 64)), ActualDigest: Digest(strings.Repeat("b", 64))}, TerminalReplayDivergence, "World replay divergence: transition=5 field=digest expected-kind=ready actual-kind=cancel expected-digest=" + strings.Repeat("a", 64) + " actual-digest=" + strings.Repeat("b", 64)},
	} {
		t.Run(test.detail[:min(len(test.detail), 50)], func(t *testing.T) {
			w := newTestWorld(t, 1)
			recorder, err := w.StartRecording(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			before := w.Snapshot()
			actual, err := recorder.FinishError(test.input)
			if err != nil {
				t.Fatal(err)
			}
			want := Recording{Initial: before, Final: before, Terminal: Terminal{Kind: test.kind, Detail: test.detail}}
			if !reflect.DeepEqual(actual, want) {
				t.Fatalf("recording changed: terminal=%+v", actual.Terminal)
			}
			actualBytes, err := EncodeRecording(actual)
			if err != nil {
				t.Fatal(err)
			}
			wantBytes, err := EncodeRecording(want)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actualBytes, wantBytes) {
				t.Fatal("complete recording bytes changed")
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(actualBytes)); got != preimageHashes[index] {
				t.Fatalf("preimage recording hash = %s, want %s", got, preimageHashes[index])
			}
			decoded, err := DecodeRecording(actualBytes)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded, want) {
				t.Fatal("decoded recording changed")
			}
		})
	}
}

func TestDetachedTerminalRejectsInvalidDataAndTypedNilWithoutClosing(t *testing.T) {
	w := newTestWorld(t, 1)
	recorder, err := w.StartRecording(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, terminal := range []Terminal{{}, {Kind: TerminalNone}, {Kind: TerminalDelivered}, {Kind: TerminalCapacity}, {Kind: "unknown", Detail: "detail"}} {
		if _, err := recorder.FinishTerminal(terminal); err == nil {
			t.Fatalf("accepted invalid terminal %+v", terminal)
		}
	}
	var capacity *CapacityError
	var replay *ReplayDivergenceError
	for _, input := range []error{capacity, replay} {
		if _, err := recorder.FinishError(input); err == nil {
			t.Fatal("accepted typed nil terminal")
		}
	}
	want := Terminal{Kind: TerminalInvalidInput, Detail: "雪\"\n" + strings.Repeat("x", 8192)}
	actual, err := recorder.FinishTerminal(want)
	if err != nil || actual.Terminal != want {
		t.Fatalf("detached terminal changed: %+v %v", actual.Terminal, err)
	}
}

func TestOwnedModelErrorPreservesMessagesAndCauseIdentity(t *testing.T) {
	_, configErr := New(Config{})
	w := newTestWorld(t, 1)
	_, requestErr := w.Register(Request{})
	_, unknownErr := w.Cancel(0)
	_, recordErr := w.StartRecording(0)
	bad := w.Snapshot()
	bad.SchemaVersion++
	_, snapshotErr := Restore(bad, nil)
	_, contextErr := EncodeRecording(Recording{Initial: bad})
	for _, test := range []struct {
		input, target error
		detail        string
	}{
		{configErr, ErrInvalidConfig, "invalid World config: every limit must be nonzero"}, {requestErr, ErrInvalidRequest, "invalid World request: invalid request.kind"}, {unknownErr, ErrUnknownRequest, "unknown World request: requestId"}, {recordErr, ErrInvalidConfig, "invalid World config: transition byte limit must be positive"}, {snapshotErr, ErrInvalidSnapshot, "invalid World snapshot: schemaVersion"}, {contextErr, ErrInvalidSnapshot, "encode initial World snapshot: invalid World snapshot: schemaVersion"},
	} {
		if test.input == nil || test.input.Error() != test.detail || !errors.Is(test.input, test.target) {
			t.Fatalf("owned error changed: %v, want %q", test.input, test.detail)
		}
		w := newTestWorld(t, 1)
		recorder, err := w.StartRecording(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		result, err := recorder.FinishError(test.input)
		if err != nil || result.Terminal != (Terminal{Kind: TerminalInvalidInput, Detail: test.detail}) {
			t.Fatalf("owned terminal changed: %+v %v", result.Terminal, err)
		}
	}
}

func TestModelGeneratedTransitionContextRemainsRecordable(t *testing.T) {
	_, input := EncodeTransitions([]Transition{{}})
	if input == nil || input.Error() != "transition 0: invalid World snapshot: transition.shape" || !errors.Is(input, ErrInvalidSnapshot) {
		t.Fatalf("transition context changed: %v", input)
	}
	w := newTestWorld(t, 1)
	recorder, err := w.StartRecording(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := recorder.FinishError(input)
	if err != nil || actual.Terminal != (Terminal{Kind: TerminalInvalidInput, Detail: "transition 0: invalid World snapshot: transition.shape"}) {
		t.Fatalf("owned transition context rejected: %+v %v", actual.Terminal, err)
	}
}
