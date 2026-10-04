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
	preimageHashes := []string{"5b169739c3c40beb11173accf9c5ca6a27bd64f68ab850175cdf948805e0e583", "f669d9a18a2b41fa9f46b9061b48fefd09dfe01760b8f8b493bbdaf9229b5f0c", "81cc89fd434aceafa491248f4028677e41b7e7493a64de8c55eb6cafedf6e714", "3166ba75214f1f5212cbc10569d88e1606504f70c1857c90dd8514c812eaf31c", "081cde7f25471550e617e5f3eb66380b91b37e16ab3107f36ae8ab8c2cb0ce33", "c8c680dbd0de71adf624aca04ac91789515f20019dc9a1a924d214475ce54ef0", "d966338258e23acfeb24e29c41b190a3514914fe03420765d6c5e74e9751119a", "cbad6f165742d43d6ff0207e3ccba46210857f84ac28bf1308144ba5f16c4018", "34dba116537ca7e08d35ccef95d69f9e913415350f150d4fca5bba0954977677", "306d1508f12014c6b04a4f38da4d1c05d48b47775e14bd294f19ed442e3af3ac"}
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
