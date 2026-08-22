package world

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
)

const MaximumRecordingBytes = 2*MaximumSnapshotJSONBytes + (1 << 20) + 40

var recordingMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'W', '2', 0}

func RecordingHeader() [8]byte {
	return recordingMagic
}

type Recording struct {
	Initial  Snapshot
	Final    Snapshot
	Terminal Terminal
}

type TerminalKind string

const (
	TerminalNone             TerminalKind = "none"
	TerminalDelivered        TerminalKind = "delivered"
	TerminalIdle             TerminalKind = "idle"
	TerminalDeadlock         TerminalKind = "deadlock"
	TerminalCapacity         TerminalKind = "capacity"
	TerminalReplayDivergence TerminalKind = "replay-divergence"
	TerminalInvalidInput     TerminalKind = "invalid-input"
)

type Terminal struct {
	Kind   TerminalKind `json:"kind"`
	Detail string       `json:"detail,omitempty"`
}

type Recorder struct {
	world *Model
	state *recordingState
}

type recordingState struct {
	initial Snapshot
	start   int
	limit   uint64
	used    uint64
	pending uint64
	closed  bool
}

func (w *Model) StartRecording(transitionByteLimit uint64) (*Recorder, error) {
	if transitionByteLimit == 0 {
		return nil, fmt.Errorf("%w: transition byte limit must be positive", ErrInvalidConfig)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.recording != nil {
		return nil, fmt.Errorf("World recording is already active")
	}
	state := &recordingState{initial: w.snapshotLocked(), start: len(w.history), limit: transitionByteLimit}
	if _, err := EncodeSnapshot(state.initial); err != nil {
		return nil, fmt.Errorf("validate initial World recording snapshot: %w", err)
	}
	w.recording = state
	return &Recorder{world: w, state: state}, nil
}

func (recorder *Recorder) Finish() (Recording, error) {
	return recorder.finish(Terminal{})
}

func (recorder *Recorder) FinishError(terminalErr error) (Recording, error) {
	if terminalErr == nil {
		return Recording{}, fmt.Errorf("World terminal error is required")
	}
	terminal := Terminal{Detail: terminalErr.Error()}
	switch {
	case errors.Is(terminalErr, ErrCapacity):
		terminal.Kind = TerminalCapacity
	case errors.Is(terminalErr, ErrReplayDivergence):
		terminal.Kind = TerminalReplayDivergence
	case errors.Is(terminalErr, ErrInvalidConfig), errors.Is(terminalErr, ErrInvalidRequest), errors.Is(terminalErr, ErrUnknownRequest), errors.Is(terminalErr, ErrRequestState), errors.Is(terminalErr, ErrTimeRegression), errors.Is(terminalErr, ErrInvalidSnapshot):
		terminal.Kind = TerminalInvalidInput
	default:
		return Recording{}, fmt.Errorf("unsupported World terminal error: %w", terminalErr)
	}
	return recorder.finish(terminal)
}

func (recorder *Recorder) finish(terminal Terminal) (Recording, error) {
	if recorder == nil || recorder.world == nil || recorder.state == nil {
		return Recording{}, fmt.Errorf("World recorder is invalid")
	}
	recorder.world.mu.Lock()
	defer recorder.world.mu.Unlock()
	if recorder.world.recording != recorder.state || recorder.state.closed {
		return Recording{}, fmt.Errorf("World recorder is not active")
	}
	final := recorder.world.snapshotLocked()
	if terminal.Kind == "" {
		terminal = terminalFromTransitions(final.Transitions[recorder.state.start:])
	}
	if err := validateRecordingTerminal(recorder.state.initial, final, terminal); err != nil {
		return Recording{}, err
	}
	transitions, err := EncodeTransitions(final.Transitions[recorder.state.start:])
	if err != nil {
		return Recording{}, err
	}
	if uint64(len(transitions)) != recorder.state.used {
		return Recording{}, fmt.Errorf("World recording byte accounting changed")
	}
	recorder.state.closed = true
	recorder.world.recording = nil
	return Recording{Initial: recorder.state.initial, Final: final, Terminal: terminal}, nil
}

func terminalFromTransitions(transitions []Transition) Terminal {
	if len(transitions) == 0 || transitions[len(transitions)-1].Quiesce == nil {
		return Terminal{Kind: TerminalNone}
	}
	switch transitions[len(transitions)-1].Quiesce.Result.Kind {
	case QuiescenceDelivered:
		return Terminal{Kind: TerminalDelivered}
	case QuiescenceIdle:
		return Terminal{Kind: TerminalIdle}
	case QuiescenceDeadlock:
		return Terminal{Kind: TerminalDeadlock}
	default:
		return Terminal{Kind: TerminalNone}
	}
}

func validateTerminal(terminal Terminal) error {
	switch terminal.Kind {
	case TerminalNone, TerminalDelivered, TerminalIdle, TerminalDeadlock:
		if terminal.Detail != "" {
			return fmt.Errorf("World quiescence terminal has error detail")
		}
	case TerminalCapacity, TerminalReplayDivergence, TerminalInvalidInput:
		if terminal.Detail == "" {
			return fmt.Errorf("World error terminal omitted detail")
		}
	default:
		return fmt.Errorf("invalid World terminal kind %q", terminal.Kind)
	}
	return nil
}

func ValidateRecording(recording Recording) error {
	return validateRecordingTerminal(recording.Initial, recording.Final, recording.Terminal)
}

func validateRecordingTerminal(initial, final Snapshot, terminal Terminal) error {
	if err := validateTerminal(terminal); err != nil {
		return err
	}
	if terminal.Kind == TerminalNone {
		return fmt.Errorf("connected World recording omitted its terminal result")
	}
	if terminal.Kind == TerminalCapacity || terminal.Kind == TerminalReplayDivergence || terminal.Kind == TerminalInvalidInput {
		return nil
	}
	if len(final.Transitions) <= len(initial.Transitions) {
		return fmt.Errorf("World terminal has no matching quiescence transition")
	}
	last := final.Transitions[len(final.Transitions)-1]
	if last.Quiesce == nil || TerminalKind(last.Quiesce.Result.Kind) != terminal.Kind {
		return fmt.Errorf("World terminal does not match the final quiescence transition")
	}
	return nil
}

func (w *Model) checkRecordingTransition(transition Transition) error {
	if w.recording == nil {
		return nil
	}
	encoded, err := EncodeTransitions([]Transition{transition})
	if err != nil {
		return err
	}
	delta := uint64(len(encoded))
	if err := checkCapacity("transition-bytes", w.recording.limit, w.recording.used, delta); err != nil {
		return err
	}
	w.recording.pending = delta
	return nil
}

func EncodeRecording(recording Recording) ([]byte, error) {
	initial, err := EncodeSnapshot(recording.Initial)
	if err != nil {
		return nil, fmt.Errorf("encode initial World snapshot: %w", err)
	}
	final, err := EncodeSnapshot(recording.Final)
	if err != nil {
		return nil, fmt.Errorf("encode final World snapshot: %w", err)
	}
	if recording.Terminal.Kind == "" {
		recording.Terminal = Terminal{Kind: TerminalNone}
	}
	if err := ValidateRecording(recording); err != nil {
		return nil, err
	}
	terminal, err := canonicaljson.CanonicalJSON(recording.Terminal)
	if err != nil {
		return nil, fmt.Errorf("encode World terminal: %w", err)
	}
	result := make([]byte, 0, len(recordingMagic)+24+len(initial)+len(final)+len(terminal))
	result = append(result, recordingMagic[:]...)
	result = binary.BigEndian.AppendUint64(result, uint64(len(initial)))
	result = append(result, initial...)
	result = binary.BigEndian.AppendUint64(result, uint64(len(final)))
	result = append(result, final...)
	result = binary.BigEndian.AppendUint64(result, uint64(len(terminal)))
	result = append(result, terminal...)
	if len(result) > MaximumRecordingBytes {
		return nil, fmt.Errorf("World recording exceeds its size bound")
	}
	return result, nil
}

func DecodeRecording(data []byte) (Recording, error) {
	if len(data) > MaximumRecordingBytes || len(data) < len(recordingMagic)+16 || !bytes.Equal(data[:len(recordingMagic)], recordingMagic[:]) {
		return Recording{}, fmt.Errorf("invalid World recording envelope")
	}
	remaining := data[len(recordingMagic):]
	initialSize := binary.BigEndian.Uint64(remaining[:8])
	remaining = remaining[8:]
	if initialSize > MaximumSnapshotJSONBytes || initialSize > uint64(len(remaining)) {
		return Recording{}, fmt.Errorf("invalid initial World snapshot size")
	}
	initialBytes := remaining[:initialSize]
	remaining = remaining[initialSize:]
	if len(remaining) < 8 {
		return Recording{}, fmt.Errorf("missing final World snapshot size")
	}
	finalSize := binary.BigEndian.Uint64(remaining[:8])
	remaining = remaining[8:]
	if finalSize > MaximumSnapshotJSONBytes || finalSize > uint64(len(remaining)) {
		return Recording{}, fmt.Errorf("invalid final World snapshot size")
	}
	finalBytes := remaining[:finalSize]
	remaining = remaining[finalSize:]
	if len(remaining) < 8 {
		return Recording{}, fmt.Errorf("missing World terminal size")
	}
	terminalSize := binary.BigEndian.Uint64(remaining[:8])
	remaining = remaining[8:]
	if terminalSize > 1<<20 || terminalSize != uint64(len(remaining)) {
		return Recording{}, fmt.Errorf("invalid World terminal size")
	}
	initial, err := DecodeSnapshot(initialBytes)
	if err != nil {
		return Recording{}, err
	}
	final, err := DecodeSnapshot(finalBytes)
	if err != nil {
		return Recording{}, err
	}
	var terminal Terminal
	if err := json.Unmarshal(remaining, &terminal); err != nil {
		return Recording{}, fmt.Errorf("decode World terminal: %w", err)
	}
	canonical, err := canonicaljson.CanonicalJSON(terminal)
	if err != nil || !bytes.Equal(canonical, remaining) {
		return Recording{}, fmt.Errorf("noncanonical World terminal")
	}
	if err := validateTerminal(terminal); err != nil {
		return Recording{}, err
	}
	recording := Recording{Initial: initial, Final: final, Terminal: terminal}
	if err := ValidateRecording(recording); err != nil {
		return Recording{}, err
	}
	return recording, nil
}
