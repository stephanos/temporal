package process

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/world"
)

func terminalSession(t *testing.T) (*Session, string) {
	t.Helper()
	core, err := world.New(world.Config{Limits: world.Limits{MaxRequests: 8, MaxEvents: 8, MaxQueuedEvents: 8, MaxPayloadBytes: 1 << 20, MaxStringBytes: 1 << 16, MaxTransitions: 64}})
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := core.StartRecording(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	output, err := os.CreateTemp(t.TempDir(), "recording")
	if err != nil {
		t.Fatal(err)
	}
	header := world.RecordingHeader()
	if _, err := output.Write(header[:]); err != nil {
		t.Fatal(err)
	}
	return &Session{recorder: recorder, output: output, core: core}, output.Name()
}

type changingTerminalError struct {
	text   string
	visits []error
	match  error
	calls  int
}

func (e *changingTerminalError) Error() string { e.calls++; return e.text }
func (e *changingTerminalError) Is(target error) bool {
	e.visits = append(e.visits, target)
	e.text = "after"
	return target == e.match
}

func TestSessionNormalizesDetailBeforeOrderedClassification(t *testing.T) {
	for _, test := range []struct {
		match  error
		kind   world.TerminalKind
		visits []error
	}{
		{world.ErrCapacity, world.TerminalCapacity, []error{world.ErrCapacity}},
		{world.ErrReplayDivergence, world.TerminalReplayDivergence, []error{world.ErrCapacity, world.ErrReplayDivergence}},
		{world.ErrInvalidSnapshot, world.TerminalInvalidInput, []error{world.ErrCapacity, world.ErrReplayDivergence, world.ErrInvalidConfig, world.ErrInvalidRequest, world.ErrUnknownRequest, world.ErrRequestState, world.ErrTimeRegression, world.ErrInvalidSnapshot}},
	} {
		session, path := terminalSession(t)
		input := &changingTerminalError{text: "before", match: test.match}
		if err := session.FinishError(input); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		recording, err := world.DecodeRecording(data)
		if err != nil {
			t.Fatal(err)
		}
		if recording.Terminal != (world.Terminal{Kind: test.kind, Detail: "before"}) || !reflect.DeepEqual(input.visits, test.visits) {
			t.Fatalf("classification changed: terminal=%+v visits=%v", recording.Terminal, input.visits)
		}
		if session.core != nil || session.recorder != nil || session.output != nil {
			t.Fatal("successful session retained state")
		}
	}
}

func TestSessionRetainsJoinedErrorPrecedenceAndUnknownIdentity(t *testing.T) {
	for _, input := range []error{errors.Join(world.ErrReplayDivergence, world.ErrCapacity), fmt.Errorf("context: %w", errors.Join(world.ErrInvalidRequest, world.ErrCapacity))} {
		session, path := terminalSession(t)
		detail := input.Error()
		if err := session.FinishError(input); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		recording, err := world.DecodeRecording(data)
		if err != nil {
			t.Fatal(err)
		}
		if recording.Terminal != (world.Terminal{Kind: world.TerminalCapacity, Detail: detail}) {
			t.Fatalf("joined terminal changed: %+v", recording.Terminal)
		}
	}
	input := errors.New("unknown")
	session, _ := terminalSession(t)
	if err := session.output.Close(); err != nil {
		t.Fatal(err)
	}
	err := session.FinishError(input)
	if err == nil || !errors.Is(err, input) || !strings.HasPrefix(err.Error(), "unsupported World terminal error: unknown\n") {
		t.Fatalf("unknown/cleanup identity changed: %v", err)
	}
	if session.output == nil || session.recorder == nil || session.core == nil {
		t.Fatal("failed finish cleared state")
	}
}

func TestSessionValidatesBeforeCallingErrors(t *testing.T) {
	var session *Session
	if err := session.FinishError(nil); err == nil || err.Error() != "World terminal error is required" {
		t.Fatalf("nil precedence: %v", err)
	}
	input := &changingTerminalError{text: "unknown"}
	if err := session.FinishError(input); err == nil || err.Error() != "World child session is invalid" {
		t.Fatalf("invalid-session precedence: %v", err)
	}
	if input.calls != 0 || len(input.visits) != 0 {
		t.Fatal("invalid session invoked callback")
	}
	valid, _ := terminalSession(t)
	if err := valid.FinishError(nil); err == nil {
		t.Fatal("nil accepted")
	}
	if valid.output == nil {
		t.Fatal("nil closed session")
	}
	if err := valid.FinishError(world.ErrCapacity); err != nil {
		t.Fatal(err)
	}
}
