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
	choiceTerminalBytes     = choicewire.TerminalFrameBytes
)

const (
	MinimumChoiceTraceBytes = minimumChoiceTraceBytes
	MaximumChoiceTraceBytes = maximumChoiceTraceBytes
)

var (
	ErrChoiceTraceMalformed    = errors.New("choice trace malformed")
	ErrChoiceTraceOverflow     = errors.New("choice trace overflow")
	ErrChoiceTraceUnterminated = errors.New("choice trace unterminated")
)

type choiceTraceBacking struct {
	file          *os.File
	terminalRead  *os.File
	terminalWrite *os.File
	limit         uint64
}

func newChoiceTraceBacking(limit uint64) (_ *choiceTraceBacking, retErr error) {
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
	return &choiceTraceBacking{file: file, terminalRead: terminalRead, terminalWrite: terminalWrite, limit: limit}, nil
}

func (backing *choiceTraceBacking) close() error {
	if backing == nil {
		return nil
	}
	return errors.Join(closeFile(&backing.file), closeFile(&backing.terminalRead), closeFile(&backing.terminalWrite))
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
	if err != nil {
		return choicewire.Trace{}, errors.Join(ErrChoiceTraceMalformed, err)
	}
	return trace, nil
}
