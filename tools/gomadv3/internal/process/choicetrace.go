package process

import (
	"errors"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
)

const (
	minimumChoiceTraceBytes = choicewire.HeaderBytes + choicewire.RecordBytes
	maximumChoiceTraceBytes = 64 << 20
	maximumChoiceTapeBytes  = maximumChoiceTraceBytes - choicewire.HeaderBytes + choicewire.TapeHeaderBytes
	choiceTerminalBytes     = choicewire.TerminalFrameBytes
)

const (
	MinimumChoiceTraceBytes = minimumChoiceTraceBytes
	MaximumChoiceTraceBytes = maximumChoiceTraceBytes
	MaximumChoiceTapeBytes  = maximumChoiceTapeBytes
)

var (
	ErrChoiceTraceMalformed    = errors.New("choice trace malformed")
	ErrChoiceTraceOverflow     = errors.New("choice trace overflow")
	ErrChoiceReplayDivergence  = errors.New("choice replay divergence")
	ErrChoiceTraceUnterminated = errors.New("choice trace unterminated")
)

type ChoiceReplayDivergenceError struct {
	Divergence choicewire.Divergence
}

func (err *ChoiceReplayDivergenceError) Error() string {
	return fmt.Sprintf("%v: ordinal=%d reason=%s", ErrChoiceReplayDivergence, err.Divergence.Ordinal, choicewire.DivergenceReasonName(err.Divergence.Reason))
}

func (err *ChoiceReplayDivergenceError) Unwrap() error {
	return ErrChoiceReplayDivergence
}

type choiceTraceBacking struct {
	file          *os.File
	expected      *os.File
	tape          *choicewire.Tape
	mode          choicewire.Mode
	terminalRead  *os.File
	terminalWrite *os.File
	limit         uint64
}

func newChoiceTraceBacking(limit uint64, mode choicewire.Mode, tape *choicewire.Tape) (_ *choiceTraceBacking, retErr error) {
	if limit < minimumChoiceTraceBytes || limit > maximumChoiceTraceBytes {
		return nil, fmt.Errorf("invalid choice trace limit %d", limit)
	}
	file, err := os.CreateTemp("", "gomadv3-choice-trace-")
	if err != nil {
		return nil, fmt.Errorf("create choice trace backing: %w", err)
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, file.Close())
		}
	}()
	if err := os.Remove(file.Name()); err != nil {
		return nil, fmt.Errorf("unlink choice trace backing: %w", err)
	}
	if err := file.Truncate(int64(limit)); err != nil {
		return nil, fmt.Errorf("size choice trace backing: %w", err)
	}
	header := choicewire.EncodeHeader(limit)
	if _, err := file.WriteAt(header[:], 0); err != nil {
		return nil, fmt.Errorf("initialize choice trace backing: %w", err)
	}
	terminalRead, terminalWrite, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create choice terminal pipe: %w", err)
	}
	backing := &choiceTraceBacking{file: file, mode: mode, terminalRead: terminalRead, terminalWrite: terminalWrite, limit: limit}
	if tape != nil {
		if len(tape.Bytes) > maximumChoiceTapeBytes {
			return nil, errors.Join(errors.New("choice decision tape exceeds its bound"), terminalRead.Close(), terminalWrite.Close())
		}
		validated, validateErr := choicewire.ValidateDecisionTape(*tape, tape.Identity)
		if validateErr != nil {
			return nil, errors.Join(fmt.Errorf("validate choice decision tape backing: %w", validateErr), terminalRead.Close(), terminalWrite.Close())
		}
		backing.tape = &validated
		backing.expected, err = newReadOnlyChoiceTapeBacking(validated.Bytes)
		if err != nil {
			return nil, errors.Join(err, terminalRead.Close(), terminalWrite.Close())
		}
	}
	return backing, nil
}

func newReadOnlyChoiceTapeBacking(contents []byte) (_ *os.File, retErr error) {
	writable, err := os.CreateTemp("", "gomadv3-choice-tape-")
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

func (backing *choiceTraceBacking) close() error {
	if backing == nil {
		return nil
	}
	return errors.Join(closeFile(&backing.file), closeFile(&backing.expected), closeFile(&backing.terminalRead), closeFile(&backing.terminalWrite))
}

func (backing *choiceTraceBacking) result(terminalFrame []byte) (choicewire.Trace, error) {
	if len(terminalFrame) == 0 {
		return choicewire.Trace{}, ErrChoiceTraceUnterminated
	}
	terminal, err := choicewire.DecodeTerminal(terminalFrame)
	if err != nil {
		return choicewire.Trace{}, errors.Join(ErrChoiceTraceMalformed, err)
	}
	if terminal.MappingBytes > backing.limit || terminal.MappingBytes < choicewire.HeaderBytes {
		return choicewire.Trace{}, errors.Join(ErrChoiceTraceMalformed, errors.New("choice trace mapping bytes exceed backing"))
	}
	headerBytes := make([]byte, choicewire.HeaderBytes)
	if _, err := backing.file.ReadAt(headerBytes, 0); err != nil {
		return choicewire.Trace{}, errors.Join(ErrChoiceTraceMalformed, fmt.Errorf("read choice trace header: %w", err))
	}
	header, err := choicewire.DecodeHeader(headerBytes)
	if err != nil || header.Capacity != backing.limit || header.NextOffset != terminal.MappingBytes || header.RecordCount != terminal.Records {
		return choicewire.Trace{}, errors.Join(ErrChoiceTraceMalformed, errors.New("choice trace header does not match terminal"), err)
	}
	payload := make([]byte, terminal.MappingBytes-choicewire.HeaderBytes)
	if _, err := backing.file.ReadAt(payload, choicewire.HeaderBytes); err != nil && !errors.Is(err, io.EOF) {
		return choicewire.Trace{}, errors.Join(ErrChoiceTraceMalformed, fmt.Errorf("read choice trace payload: %w", err))
	}
	trace, err := choicewire.DecodeTrace(payload, terminalFrame, backing.limit)
	if errors.Is(err, choicewire.ErrOverflow) {
		return trace, ErrChoiceTraceOverflow
	}
	if errors.Is(err, choicewire.ErrUnterminated) {
		return choicewire.Trace{}, ErrChoiceTraceUnterminated
	}
	if errors.Is(err, choicewire.ErrDiverged) {
		var divergence choicewire.Divergence
		var divergenceErr error
		if backing.tape == nil {
			divergence, divergenceErr = choicewire.DivergenceFromTerminal(terminal)
		} else {
			divergence, divergenceErr = choicewire.ValidateDivergenceTerminal(*backing.tape, backing.mode, terminal)
		}
		if divergenceErr != nil {
			return choicewire.Trace{}, errors.Join(ErrChoiceTraceMalformed, divergenceErr)
		}
		return trace, &ChoiceReplayDivergenceError{Divergence: divergence}
	}
	if err != nil {
		return choicewire.Trace{}, errors.Join(ErrChoiceTraceMalformed, err)
	}
	return trace, nil
}
