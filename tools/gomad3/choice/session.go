package choice

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	MinimumTraceBytes      = traceHeaderBytes + traceRecordBytes
	MaximumTraceBytes      = 64 << 20
	MaximumReplayPlanBytes = MaximumTraceBytes - traceHeaderBytes + replayPlanHeaderBytes
)

type SessionSpec struct {
	Limit      uint64
	Mode       Mode
	ReplayPlan *ReplayPlan
}

type SessionFiles struct {
	Trace      *os.File
	Terminal   *os.File
	ReplayPlan *os.File
}

type Session struct {
	trace         *os.File
	expected      *os.File
	replayPlan    *ReplayPlan
	mode          Mode
	terminalRead  *os.File
	terminalWrite *os.File
	limit         uint64
}

type ReplayDivergenceError struct {
	Divergence Divergence
}

func (err *ReplayDivergenceError) Error() string {
	return fmt.Sprintf("%v: ordinal=%d reason=%s", ErrDiverged, err.Divergence.Ordinal, DivergenceReasonName(err.Divergence.Reason))
}

func (err *ReplayDivergenceError) Unwrap() error {
	return ErrDiverged
}

func NewSession(spec SessionSpec) (_ *Session, retErr error) {
	if err := ValidateTraceLimit(spec.Limit); err != nil {
		return nil, err
	}
	if err := ValidateController(spec.Mode, replayPlanBytes(spec.ReplayPlan)); err != nil {
		return nil, err
	}
	trace, err := os.CreateTemp("", "gomad3-choice-trace-")
	if err != nil {
		return nil, fmt.Errorf("create choice trace backing: %w", err)
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, trace.Close())
		}
	}()
	if err := os.Remove(trace.Name()); err != nil {
		return nil, fmt.Errorf("unlink choice trace backing: %w", err)
	}
	if err := trace.Truncate(int64(spec.Limit)); err != nil {
		return nil, fmt.Errorf("size choice trace backing: %w", err)
	}
	header := encodeTraceHeader(spec.Limit)
	if _, err := trace.WriteAt(header[:], 0); err != nil {
		return nil, fmt.Errorf("initialize choice trace backing: %w", err)
	}
	terminalRead, terminalWrite, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create choice terminal pipe: %w", err)
	}
	session := &Session{
		trace: trace, mode: spec.Mode, terminalRead: terminalRead, terminalWrite: terminalWrite, limit: spec.Limit,
	}
	if spec.ReplayPlan != nil {
		validated, validateErr := validateReplayPlan(*spec.ReplayPlan, spec.Mode)
		if validateErr != nil {
			return nil, errors.Join(fmt.Errorf("validate choice decision tape backing: %w", validateErr), terminalRead.Close(), terminalWrite.Close())
		}
		session.replayPlan = &validated
		session.expected, err = newReadOnlyReplayPlanBacking(validated.Bytes)
		if err != nil {
			return nil, errors.Join(err, terminalRead.Close(), terminalWrite.Close())
		}
	}
	return session, nil
}

func ValidateTraceLimit(limit uint64) error {
	if limit < MinimumTraceBytes || limit > MaximumTraceBytes {
		return fmt.Errorf("invalid choice trace limit %d", limit)
	}
	return nil
}

func ValidateController(mode Mode, replayPlanBytes uint64) error {
	if mode != ModeRecord && mode != ModeReplay && mode != ModePrefix {
		return errors.New("choice controller mode and tape are inconsistent")
	}
	if mode == ModeRecord && replayPlanBytes != 0 || mode != ModeRecord && replayPlanBytes < replayPlanHeaderBytes || replayPlanBytes > MaximumReplayPlanBytes {
		return errors.New("choice controller mode and tape are inconsistent")
	}
	return nil
}

func TraceBytes(records uint64) (uint64, error) {
	payload, err := TracePayloadBytes(records)
	if err != nil {
		return 0, err
	}
	return traceHeaderBytes + payload, nil
}

func TracePayloadBytes(records uint64) (uint64, error) {
	if records > (MaximumTraceBytes-traceHeaderBytes)/traceRecordBytes {
		return 0, errors.New("choice trace record count exceeds its bound")
	}
	return records * traceRecordBytes, nil
}

func (session *Session) Files() SessionFiles {
	if session == nil {
		return SessionFiles{}
	}
	return SessionFiles{Trace: session.trace, Terminal: session.terminalWrite, ReplayPlan: session.expected}
}

func (session *Session) Collect() (Trace, error) {
	if session == nil || session.terminalRead == nil {
		return Trace{}, ErrUnterminated
	}
	frame, err := io.ReadAll(io.LimitReader(session.terminalRead, terminalFrameBytes+1))
	if err != nil {
		return Trace{}, errors.Join(ErrMalformed, fmt.Errorf("read choice terminal frame: %w", err))
	}
	return session.collectFrame(frame)
}

func (session *Session) Close() error {
	if session == nil {
		return nil
	}
	return errors.Join(
		closeSessionFile(&session.trace),
		closeSessionFile(&session.expected),
		closeSessionFile(&session.terminalRead),
		closeSessionFile(&session.terminalWrite),
	)
}

func (session *Session) collectFrame(frame []byte) (Trace, error) {
	if len(frame) == 0 {
		return Trace{}, ErrUnterminated
	}
	completed, err := decodeTerminal(frame)
	if err != nil {
		return Trace{}, errors.Join(ErrMalformed, err)
	}
	if completed.MappingBytes > session.limit || completed.MappingBytes < traceHeaderBytes {
		return Trace{}, errors.Join(ErrMalformed, errors.New("choice trace mapping bytes exceed backing"))
	}
	headerBytes := make([]byte, traceHeaderBytes)
	if _, err := session.trace.ReadAt(headerBytes, 0); err != nil {
		return Trace{}, errors.Join(ErrMalformed, fmt.Errorf("read choice trace header: %w", err))
	}
	capacity, nextOffset, records, err := decodeTraceHeader(headerBytes)
	if err != nil || capacity != session.limit || nextOffset != completed.MappingBytes || records != completed.Records {
		return Trace{}, errors.Join(ErrMalformed, errors.New("choice trace header does not match terminal"), err)
	}
	payload := make([]byte, completed.MappingBytes-traceHeaderBytes)
	if _, err := session.trace.ReadAt(payload, traceHeaderBytes); err != nil && !errors.Is(err, io.EOF) {
		return Trace{}, errors.Join(ErrMalformed, fmt.Errorf("read choice trace payload: %w", err))
	}
	trace, err := DecodeTrace(payload, frame, session.limit)
	if errors.Is(err, ErrOverflow) {
		return trace, ErrOverflow
	}
	if errors.Is(err, ErrUnterminated) {
		return Trace{}, ErrUnterminated
	}
	if errors.Is(err, ErrDiverged) {
		var divergence Divergence
		var divergenceErr error
		if session.replayPlan == nil {
			divergence, divergenceErr = divergenceFromTerminal(completed)
		} else {
			divergence, divergenceErr = validateDivergenceTerminal(*session.replayPlan, session.mode, completed)
		}
		if divergenceErr != nil {
			return Trace{}, errors.Join(ErrMalformed, divergenceErr)
		}
		return trace, &ReplayDivergenceError{Divergence: divergence}
	}
	if err != nil {
		return Trace{}, errors.Join(ErrMalformed, err)
	}
	return trace, nil
}

func validateReplayPlan(plan ReplayPlan, mode Mode) (ReplayPlan, error) {
	if mode == ModePrefix {
		return ValidatePrefixReplayPlan(plan, plan.Identity)
	}
	return ValidateReplayPlan(plan, plan.Identity)
}

func replayPlanBytes(plan *ReplayPlan) uint64 {
	if plan == nil {
		return 0
	}
	return uint64(len(plan.Bytes))
}

func newReadOnlyReplayPlanBacking(contents []byte) (_ *os.File, retErr error) {
	writable, err := os.CreateTemp("", "gomad3-choice-tape-")
	if err != nil {
		return nil, fmt.Errorf("create choice tape backing: %w", err)
	}
	path := writable.Name()
	defer func() {
		if path != "" {
			retErr = errors.Join(retErr, os.Remove(path))
		}
		if writable != nil {
			retErr = errors.Join(retErr, writable.Close())
		}
	}()
	if err := writable.Chmod(0o600); err != nil {
		return nil, fmt.Errorf("make choice tape backing private: %w", err)
	}
	if _, err := writable.Write(contents); err != nil {
		return nil, fmt.Errorf("write choice tape backing: %w", err)
	}
	if err := writable.Sync(); err != nil {
		return nil, fmt.Errorf("sync choice tape backing: %w", err)
	}
	writableInfo, err := writable.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat choice tape backing: %w", err)
	}
	readOnly, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reopen choice tape backing read-only: %w", err)
	}
	readOnlyInfo, err := readOnly.Stat()
	if err != nil || !os.SameFile(writableInfo, readOnlyInfo) {
		return nil, errors.Join(errors.New("choice tape backing changed while reopening"), err, readOnly.Close())
	}
	if err := os.Remove(path); err != nil {
		return nil, errors.Join(fmt.Errorf("unlink choice tape backing: %w", err), readOnly.Close())
	}
	path = ""
	if err := writable.Close(); err != nil {
		return nil, errors.Join(fmt.Errorf("close writable choice tape backing: %w", err), readOnly.Close())
	}
	writable = nil
	return readOnly, nil
}

func closeSessionFile(file **os.File) error {
	if file == nil || *file == nil {
		return nil
	}
	err := (*file).Close()
	*file = nil
	if errors.Is(err, os.ErrClosed) {
		return nil
	}
	return err
}
