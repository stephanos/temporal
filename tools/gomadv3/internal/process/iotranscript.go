package process

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	maximumIOTranscriptBytes = 64 << 20
	ioTranscriptHeaderBytes  = 64
	ioTranscriptRecordBytes  = 128
	ioTerminalBytes          = 104
)

var (
	ioTranscriptMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'T', 'R', 1}
	ioExpectedMagic   = [8]byte{'G', 'O', 'M', 'A', 'D', 'X', 'T', 1}
	ioTerminalMagic   = [8]byte{'G', 'O', 'M', 'A', 'D', 'I', 'T', 1}
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
	header := make([]byte, ioTranscriptHeaderBytes)
	copy(header[:8], ioTranscriptMagic[:])
	binary.BigEndian.PutUint32(header[8:12], 1)
	binary.BigEndian.PutUint64(header[16:24], limit)
	binary.BigEndian.PutUint64(header[24:32], ioTranscriptHeaderBytes)
	if _, err := file.WriteAt(header, 0); err != nil {
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
	header := make([]byte, ioTranscriptHeaderBytes)
	copy(header[:8], ioExpectedMagic[:])
	binary.BigEndian.PutUint32(header[8:12], 1)
	if replay {
		header[12] = 1
	}
	binary.BigEndian.PutUint64(header[16:24], uint64(len(expected)))
	binary.BigEndian.PutUint64(header[24:32], uint64(len(expected))/ioTranscriptRecordBytes)
	digest := sha256.Sum256(expected)
	copy(header[32:64], digest[:])
	if _, err := file.WriteAt(header, 0); err != nil {
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
	if len(terminal) != ioTerminalBytes || !bytes.Equal(terminal[:8], ioTerminalMagic[:]) || binary.BigEndian.Uint32(terminal[8:12]) != 1 {
		return IOTranscript{}, errors.New("invalid I/O terminal frame")
	}
	checksum := sha256.Sum256(terminal[:72])
	if !bytes.Equal(checksum[:], terminal[72:]) {
		return IOTranscript{}, errors.New("I/O terminal frame checksum mismatch")
	}
	records := binary.BigEndian.Uint64(terminal[16:24])
	length := binary.BigEndian.Uint64(terminal[24:32])
	if length < ioTranscriptHeaderBytes || length > backing.limit {
		return IOTranscript{}, fmt.Errorf("invalid I/O transcript length %d", length)
	}
	if terminal[12] != 1 && terminal[12] != 3 {
		return IOTranscript{}, errors.New("I/O transcript did not complete")
	}
	payload := make([]byte, length-ioTranscriptHeaderBytes)
	if _, err := backing.file.ReadAt(payload, ioTranscriptHeaderBytes); err != nil && err != io.EOF {
		return IOTranscript{}, fmt.Errorf("read I/O transcript: %w", err)
	}
	digest := sha256.Sum256(payload)
	if !bytes.Equal(digest[:], terminal[32:64]) {
		return IOTranscript{}, errors.New("I/O transcript digest mismatch")
	}
	result := IOTranscript{Bytes: payload, SHA256: digest, Records: records, Complete: true}
	if terminal[12] == 3 {
		ordinal := binary.BigEndian.Uint64(terminal[64:72])
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
