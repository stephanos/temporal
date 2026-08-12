package process

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tools/gomadv3/internal/iowire"
)

const (
	maximumIOTranscriptBytes = 64 << 20
	ioTranscriptHeaderBytes  = iowire.TranscriptHeaderBytes
	ioTranscriptRecordBytes  = iowire.TranscriptRecordBytes
	ioTerminalBytes          = iowire.TerminalFrameBytes
)

type ioTranscriptBacking struct {
	file          *os.File
	expected      *os.File
	terminalRead  *os.File
	terminalWrite *os.File
	limit         uint64
}

func newIOTranscriptBacking(limit uint64, replay bool, expected []byte) (_ *ioTranscriptBacking, retErr error) {
	if limit < ioTranscriptHeaderBytes || limit > maximumIOTranscriptBytes {
		return nil, fmt.Errorf("invalid I/O transcript limit %d", limit)
	}
	file, err := os.CreateTemp("", "gomadv3-io-transcript-")
	if err != nil {
		return nil, fmt.Errorf("create I/O transcript backing: %w", err)
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, file.Close())
		}
	}()
	name := file.Name()
	if err := os.Remove(name); err != nil {
		return nil, fmt.Errorf("unlink I/O transcript backing: %w", err)
	}
	if err := file.Truncate(int64(limit)); err != nil {
		return nil, fmt.Errorf("size I/O transcript backing: %w", err)
	}
	header := iowire.EncodeProducedTranscriptHeader(limit)
	if _, err := file.WriteAt(header[:], 0); err != nil {
		return nil, fmt.Errorf("initialize I/O transcript backing: %w", err)
	}
	expectedFile, err := newExpectedIOTranscriptBacking(limit, replay, expected)
	if err != nil {
		return nil, err
	}
	terminalRead, terminalWrite, err := os.Pipe()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("create I/O terminal pipe: %w", err), expectedFile.Close())
	}
	return &ioTranscriptBacking{file: file, expected: expectedFile, terminalRead: terminalRead, terminalWrite: terminalWrite, limit: limit}, nil
}

func newExpectedIOTranscriptBacking(limit uint64, replay bool, expected []byte) (_ *os.File, retErr error) {
	file, err := os.CreateTemp("", "gomadv3-io-expected-")
	if err != nil {
		return nil, fmt.Errorf("create expected I/O transcript backing: %w", err)
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, file.Close())
		}
	}()
	if err := os.Remove(file.Name()); err != nil {
		return nil, fmt.Errorf("unlink expected I/O transcript backing: %w", err)
	}
	if err := file.Truncate(int64(limit)); err != nil {
		return nil, fmt.Errorf("size expected I/O transcript backing: %w", err)
	}
	header, err := iowire.EncodeExpectedTranscriptHeader(replay, expected)
	if err != nil {
		return nil, err
	}
	if _, err := file.WriteAt(header[:], 0); err != nil {
		return nil, fmt.Errorf("initialize expected I/O transcript backing: %w", err)
	}
	if len(expected) != 0 {
		if _, err := file.WriteAt(expected, ioTranscriptHeaderBytes); err != nil {
			return nil, fmt.Errorf("write expected I/O transcript: %w", err)
		}
	}
	return file, nil
}

func (backing *ioTranscriptBacking) close() error {
	if backing == nil {
		return nil
	}
	return errors.Join(closeFile(&backing.file), closeFile(&backing.expected), closeFile(&backing.terminalRead), closeFile(&backing.terminalWrite))
}

func (backing *ioTranscriptBacking) result(terminal []byte) (IOTranscript, error) {
	decoded, err := iowire.DecodeTerminal(terminal)
	if err != nil {
		return IOTranscript{}, err
	}
	records := decoded.Records
	length := decoded.MappingBytes
	if length < ioTranscriptHeaderBytes || length > backing.limit {
		return IOTranscript{}, fmt.Errorf("invalid I/O transcript length %d", length)
	}
	if decoded.State != iowire.TerminalComplete && decoded.State != iowire.TerminalReplayDivergence {
		return IOTranscript{}, errors.New("I/O transcript did not complete")
	}
	payload := make([]byte, length-ioTranscriptHeaderBytes)
	if _, err := backing.file.ReadAt(payload, ioTranscriptHeaderBytes); err != nil && err != io.EOF {
		return IOTranscript{}, fmt.Errorf("read I/O transcript: %w", err)
	}
	digest := sha256.Sum256(payload)
	if digest != decoded.PayloadHash {
		return IOTranscript{}, errors.New("I/O transcript digest mismatch")
	}
	result := IOTranscript{Bytes: payload, SHA256: digest, Records: records, Complete: true}
	if decoded.State == iowire.TerminalReplayDivergence {
		ordinal := decoded.DivergentOrdinal
		result.ReplayDivergence = &ordinal
	}
	return result, nil
}

func closeFile(file **os.File) error {
	if file == nil || *file == nil {
		return nil
	}
	err := (*file).Close()
	*file = nil
	return err
}
