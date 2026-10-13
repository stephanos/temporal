package logstore

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
)

const replayBufferBytes = 64 << 10

// errShortRead marks a read that ended before the snapshot size it was
// promised; the caller decides whether that is a changed snapshot.
var errShortRead = errors.New("log ended before its snapshot size")

// scan validates the first size bytes of log, delivering each complete valid
// frame to consume in order. It stops at the first torn tail or corruption.
// A consumer error is corruption of that frame: one frame is one transaction,
// and the consumer must validate it completely before applying any of it.
func scan(log io.ReaderAt, size int64, kinds map[Kind]uint32, consume func(Record) error) (Report, error) {
	reader := bufio.NewReaderSize(io.NewSectionReader(log, 0, size), replayBufferBytes)
	report := Report{SnapshotBytes: size}
	torn := func(tail Tail) (Report, error) {
		report.Tail = tail
		report.ObservedTailBytes = size - report.ValidBytes
		return report, nil
	}
	corrupt := func(err error) (Report, error) {
		return report, &CorruptionError{Offset: report.ValidBytes, Records: report.Records, Err: err}
	}
	var raw [headerSize]byte
	for report.ValidBytes < size {
		remaining := size - report.ValidBytes
		if remaining < headerSize {
			return torn(TailShortFrame)
		}
		if _, err := io.ReadFull(reader, raw[:]); err != nil {
			return report, readError(err)
		}
		parsed, err := decodeHeader(raw[:], kinds)
		if err != nil {
			return corrupt(err)
		}
		length := frameBytes(parsed.length)
		if length > remaining {
			return torn(TailShortFrame)
		}
		body := make([]byte, int(parsed.length)+trailerSize)
		if _, err := io.ReadFull(reader, body); err != nil {
			return report, readError(err)
		}
		payload := body[:parsed.length]
		if binary.LittleEndian.Uint32(body[parsed.length:]) != payloadChecksum(raw[:], payload) {
			if length == remaining {
				return torn(TailPayloadChecksum)
			}
			return corrupt(errors.New("payload checksum mismatch before the final frame"))
		}
		if consume != nil {
			if err := consume(Record{Kind: parsed.kind, Payload: payload, Offset: report.ValidBytes}); err != nil {
				return corrupt(err)
			}
		}
		report.ValidBytes += length
		report.Records++
	}
	return report, nil
}

func readError(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return errShortRead
	}
	return fmt.Errorf("read log: %w", err)
}

// Replay validates the store at path without the writer lock and delivers each
// validated record to consume. It captures the log size once at open, reports
// a torn tail without repairing it, and fails with ErrSnapshotChanged when the
// log shrinks or is replaced before replay finishes. It creates, chmods,
// locks, and truncates nothing.
func Replay(path string, options Options, consume func(Record) error) (_ Report, retErr error) {
	kinds, err := options.validate()
	if err != nil {
		return Report{}, err
	}
	root, rootInfo, err := pinRoot(path, options.Hook, false)
	if err != nil {
		return Report{}, err
	}
	s := &store{path: path, root: root, rootInfo: rootInfo}
	defer func() { retErr = errors.Join(retErr, s.close()) }()

	lockInfo, err := root.Lstat(LockFile)
	if err != nil {
		return Report{}, invalidStore("%s: %v", LockFile, err)
	}
	if err := validateMemberInfo(LockFile, lockInfo); err != nil {
		return Report{}, err
	}
	s.lock = member{name: LockFile, info: lockInfo}
	if err := options.Hook.observe(PointOpen, LockFile); err != nil {
		return Report{}, err
	}
	log, logInfo, err := hostfs.OpenRootGrowing(root, LogFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Report{}, fmt.Errorf("open log store: %w", err)
		}
		return Report{}, fmt.Errorf("%w: %w", ErrInvalidStore, err)
	}
	s.log = member{name: LogFile, file: log, info: logInfo}
	if err := validateMemberInfo(LogFile, logInfo); err != nil {
		return Report{}, err
	}
	if err := options.Hook.observe(PointOpen, LogFile); err != nil {
		return Report{}, err
	}
	if err := s.verify(); err != nil {
		return Report{}, err
	}
	snapshot := logInfo.Size()
	report, err := scan(log, snapshot, kinds, consume)
	if errors.Is(err, errShortRead) {
		return report, fmt.Errorf("%w: log shrank below %d bytes", ErrSnapshotChanged, snapshot)
	}
	if err != nil {
		return report, err
	}
	after, err := log.Stat()
	if err != nil {
		return report, fmt.Errorf("stat log after replay: %w", err)
	}
	if after.Size() < snapshot {
		return report, fmt.Errorf("%w: log shrank from %d to %d bytes", ErrSnapshotChanged, snapshot, after.Size())
	}
	if err := s.verify(); err != nil {
		return report, fmt.Errorf("%w: %w", ErrSnapshotChanged, err)
	}
	return report, nil
}
