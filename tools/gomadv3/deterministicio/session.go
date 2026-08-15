package deterministicio

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	iowire "go.temporal.io/server/tools/gomadv3/deterministicio/internal/wire"
)

const MaximumTranscriptBytes = 64 << 20

type SessionSpec struct {
	Limit    uint64
	Replay   bool
	Expected []byte
}

type SessionFiles struct {
	Transcript *os.File
	Terminal   *os.File
	Expected   *os.File
}

type Session struct {
	transcript    *os.File
	expected      *os.File
	terminalRead  *os.File
	terminalWrite *os.File
	limit         uint64
}

type Transcript struct {
	Bytes            []byte
	SHA256           [sha256.Size]byte
	Records          uint64
	Complete         bool
	ReplayDivergence *uint64
}

func WriteCompletion(writer io.Writer, transcript Transcript) error {
	if writer == nil || !transcript.Complete || transcript.SHA256 != sha256.Sum256(transcript.Bytes) {
		return errors.New("invalid completed I/O transcript")
	}
	records, err := TranscriptRecordCount(transcript.Bytes)
	if err != nil {
		return err
	}
	if records != transcript.Records {
		return errors.New("completed I/O transcript record count is inconsistent")
	}
	state := iowire.TerminalComplete
	var divergentOrdinal uint64
	if transcript.ReplayDivergence != nil {
		state = iowire.TerminalReplayDivergence
		divergentOrdinal = *transcript.ReplayDivergence
	}
	frame := iowire.EncodeTerminal(iowire.Terminal{
		State: state, Records: records, MappingBytes: iowire.TranscriptHeaderBytes + uint64(len(transcript.Bytes)),
		PayloadHash: transcript.SHA256, DivergentOrdinal: divergentOrdinal,
	})
	written, err := writer.Write(frame[:])
	if err != nil {
		return err
	}
	if written != len(frame) {
		return io.ErrShortWrite
	}
	return nil
}

func NewSession(spec SessionSpec) (_ *Session, retErr error) {
	if err := ValidateSessionSpec(spec); err != nil {
		return nil, err
	}
	transcript, err := os.CreateTemp("", "gomadv3-io-transcript-")
	if err != nil {
		return nil, fmt.Errorf("create I/O transcript backing: %w", err)
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, transcript.Close())
		}
	}()
	if err := os.Remove(transcript.Name()); err != nil {
		return nil, fmt.Errorf("unlink I/O transcript backing: %w", err)
	}
	if err := transcript.Truncate(int64(spec.Limit)); err != nil {
		return nil, fmt.Errorf("size I/O transcript backing: %w", err)
	}
	header := iowire.EncodeProducedTranscriptHeader(spec.Limit)
	if _, err := transcript.WriteAt(header[:], 0); err != nil {
		return nil, fmt.Errorf("initialize I/O transcript backing: %w", err)
	}
	expected, err := newExpectedTranscriptBacking(spec)
	if err != nil {
		return nil, err
	}
	terminalRead, terminalWrite, err := os.Pipe()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("create I/O terminal pipe: %w", err), expected.Close())
	}
	return &Session{
		transcript: transcript, expected: expected, terminalRead: terminalRead, terminalWrite: terminalWrite, limit: spec.Limit,
	}, nil
}

func ValidateSessionSpec(spec SessionSpec) error {
	if spec.Limit < iowire.TranscriptHeaderBytes || spec.Limit > MaximumTranscriptBytes {
		return fmt.Errorf("invalid I/O transcript limit %d", spec.Limit)
	}
	if !spec.Replay && len(spec.Expected) != 0 {
		return errors.New("expected I/O transcript requires replay mode")
	}
	if len(spec.Expected)%iowire.TranscriptRecordBytes != 0 || uint64(len(spec.Expected)) > spec.Limit-iowire.TranscriptHeaderBytes {
		return fmt.Errorf("invalid expected I/O transcript length %d", len(spec.Expected))
	}
	return nil
}

func ExpectedTranscriptBytes(records uint64) (uint64, error) {
	if records > (MaximumTranscriptBytes-iowire.TranscriptHeaderBytes)/iowire.TranscriptRecordBytes {
		return 0, errors.New("I/O transcript record count exceeds its bound")
	}
	return records * iowire.TranscriptRecordBytes, nil
}

func (session *Session) Files() SessionFiles {
	if session == nil {
		return SessionFiles{}
	}
	return SessionFiles{Transcript: session.transcript, Terminal: session.terminalWrite, Expected: session.expected}
}

func (session *Session) Collect() (Transcript, error) {
	if session == nil || session.terminalRead == nil {
		return Transcript{}, errors.New("I/O transcript did not complete")
	}
	frame, err := io.ReadAll(io.LimitReader(session.terminalRead, iowire.TerminalFrameBytes+1))
	if err != nil {
		return Transcript{}, fmt.Errorf("read I/O terminal frame: %w", err)
	}
	return session.collectFrame(frame)
}

func (session *Session) Close() error {
	if session == nil {
		return nil
	}
	return errors.Join(
		closeSessionFile(&session.transcript),
		closeSessionFile(&session.expected),
		closeSessionFile(&session.terminalRead),
		closeSessionFile(&session.terminalWrite),
	)
}

func (session *Session) collectFrame(frame []byte) (Transcript, error) {
	completed, err := iowire.DecodeTerminal(frame)
	if err != nil {
		return Transcript{}, err
	}
	length := completed.MappingBytes
	if length < iowire.TranscriptHeaderBytes || length > session.limit {
		return Transcript{}, fmt.Errorf("invalid I/O transcript length %d", length)
	}
	if completed.State != iowire.TerminalComplete && completed.State != iowire.TerminalReplayDivergence {
		return Transcript{}, errors.New("I/O transcript did not complete")
	}
	payload := make([]byte, length-iowire.TranscriptHeaderBytes)
	if _, err := session.transcript.ReadAt(payload, iowire.TranscriptHeaderBytes); err != nil && !errors.Is(err, io.EOF) {
		return Transcript{}, fmt.Errorf("read I/O transcript: %w", err)
	}
	digest := sha256.Sum256(payload)
	if digest != completed.PayloadHash {
		return Transcript{}, errors.New("I/O transcript digest mismatch")
	}
	result := Transcript{Bytes: payload, SHA256: digest, Records: completed.Records, Complete: true}
	if completed.State == iowire.TerminalReplayDivergence {
		ordinal := completed.DivergentOrdinal
		result.ReplayDivergence = &ordinal
	}
	return result, nil
}

func newExpectedTranscriptBacking(spec SessionSpec) (_ *os.File, retErr error) {
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
	if err := file.Truncate(int64(spec.Limit)); err != nil {
		return nil, fmt.Errorf("size expected I/O transcript backing: %w", err)
	}
	header, err := iowire.EncodeExpectedTranscriptHeader(spec.Replay, spec.Expected)
	if err != nil {
		return nil, err
	}
	if _, err := file.WriteAt(header[:], 0); err != nil {
		return nil, fmt.Errorf("initialize expected I/O transcript backing: %w", err)
	}
	if len(spec.Expected) != 0 {
		if _, err := file.WriteAt(spec.Expected, iowire.TranscriptHeaderBytes); err != nil {
			return nil, fmt.Errorf("write expected I/O transcript: %w", err)
		}
	}
	return file, nil
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
