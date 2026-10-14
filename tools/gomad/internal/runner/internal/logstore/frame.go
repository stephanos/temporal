// Package logstore owns the Runner's private append-only record logs.
//
// A store is a private 0700 directory holding a 0600 single-link log file and
// a 0600 single-link lock file. One exclusive writer holds the lock; read-only
// replay takes no lock and never creates, truncates, or chmods anything.
//
// Each record is one frame and one transaction:
//
//	magic "GMLG" | version u8 | flags u8 | kind u16 | payload length u32 | header CRC32C u32
//	payload (stdlib JSON) | payload CRC32C u32
//
// Integers are little-endian. The header checksum covers the twelve bytes
// before it. The payload checksum covers the complete sixteen-byte header and
// the payload, so it independently frames the payload. A frame is trusted only
// after its header checksum, version, flags, kind, and the kind's payload bound
// all validate; until then its length field is never used.
//
// Damage classification: a final frame shorter than its header, or shorter
// than its validated header declares, is a short torn tail; a complete final
// frame whose payload checksum fails is a payload-checksum torn tail. Every
// other failure (an invalid complete header, an unknown kind, an oversized
// declared length, earlier payload damage, or a valid frame whose consumer
// rejects it) is corruption. The classifier covers interrupted appends and
// these damage controls, not coherent rewrites or checksum collisions.
package logstore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
)

const (
	headerSize  = 16
	trailerSize = 4

	// FrameVersion is the frame format this build writes and the only one
	// it reads.
	FrameVersion = 1

	// MaxPayloadBytes is the largest per-kind payload bound Options accepts.
	MaxPayloadBytes = 1 << 30
)

var (
	frameMagic = [4]byte{'G', 'M', 'L', 'G'}
	castagnoli = crc32.MakeTable(crc32.Castagnoli)
)

// Kind identifies a record's typed payload. Zero is never a valid kind.
type Kind uint16

type header struct {
	kind   Kind
	length uint32
}

func encodeFrame(kind Kind, payload []byte) []byte {
	frame := make([]byte, headerSize+len(payload)+trailerSize)
	copy(frame, frameMagic[:])
	frame[4] = FrameVersion
	binary.LittleEndian.PutUint16(frame[6:], uint16(kind))
	binary.LittleEndian.PutUint32(frame[8:], uint32(len(payload)))
	binary.LittleEndian.PutUint32(frame[12:], crc32.Checksum(frame[:12], castagnoli))
	copy(frame[headerSize:], payload)
	binary.LittleEndian.PutUint32(frame[headerSize+len(payload):], payloadChecksum(frame[:headerSize], payload))
	return frame
}

func payloadChecksum(header, payload []byte) uint32 {
	return crc32.Update(crc32.Checksum(header, castagnoli), castagnoli, payload)
}

func decodeHeader(raw []byte, kinds map[Kind]uint32) (header, error) {
	if len(raw) != headerSize {
		return header{}, errors.New("frame header is incomplete")
	}
	if !bytes.Equal(raw[:4], frameMagic[:]) {
		return header{}, errors.New("frame magic is invalid")
	}
	if binary.LittleEndian.Uint32(raw[12:]) != crc32.Checksum(raw[:12], castagnoli) {
		return header{}, errors.New("frame header checksum mismatch")
	}
	if raw[4] != FrameVersion {
		return header{}, fmt.Errorf("unsupported frame version %d", raw[4])
	}
	if raw[5] != 0 {
		return header{}, fmt.Errorf("frame flags %#x are invalid", raw[5])
	}
	parsed := header{kind: Kind(binary.LittleEndian.Uint16(raw[6:])), length: binary.LittleEndian.Uint32(raw[8:])}
	limit, known := kinds[parsed.kind]
	if !known {
		return header{}, fmt.Errorf("record kind %d is unknown", parsed.kind)
	}
	if parsed.length > limit {
		return header{}, fmt.Errorf("record kind %d payload length %d exceeds its bound %d", parsed.kind, parsed.length, limit)
	}
	return parsed, nil
}

// frameBytes returns the encoded size of a frame whose validated header
// declares length payload bytes. Validation bounds length by MaxPayloadBytes,
// so the sum cannot overflow.
func frameBytes(length uint32) int64 {
	return headerSize + int64(length) + trailerSize
}

var (
	// ErrCorrupt matches every CorruptionError.
	ErrCorrupt = errors.New("log store is corrupt")
	// ErrInUse reports that another writer holds the store lock.
	ErrInUse = errors.New("log store is already in use")
	// ErrPoisoned reports a writer whose earlier append or sync outcome is
	// uncertain; only closing and reopening the store clears it.
	ErrPoisoned = errors.New("log writer is poisoned by an uncertain append")
	// ErrSnapshotChanged reports a log that shrank or was replaced while a
	// read-only replay held its snapshot.
	ErrSnapshotChanged = errors.New("log changed during read-only replay")
	// ErrInvalidStore reports invalid store metadata, containment, or a
	// detected identity change; it is raised before any log mutation.
	ErrInvalidStore = errors.New("log store metadata is invalid")
	// ErrClosed reports use of a closed writer.
	ErrClosed = errors.New("log writer is closed")
)

// CorruptionError reports damage that is not a recognized torn tail. Offset is
// the end of the validated prefix and the start of the rejected frame.
type CorruptionError struct {
	Offset  int64
	Records uint64
	Err     error
}

func (e *CorruptionError) Error() string {
	return fmt.Sprintf("log store is corrupt at offset %d after %d records: %v", e.Offset, e.Records, e.Err)
}

func (e *CorruptionError) Unwrap() error { return e.Err }

func (*CorruptionError) Is(target error) bool { return target == ErrCorrupt }

// Tail classifies the bytes after the validated prefix.
type Tail uint8

const (
	// TailNone means the log ends on a frame boundary.
	TailNone Tail = iota
	// TailShortFrame means the final frame is shorter than its header or than
	// its validated header declares.
	TailShortFrame
	// TailPayloadChecksum means the complete final frame fails its payload
	// checksum.
	TailPayloadChecksum
)

func (tail Tail) String() string {
	switch tail {
	case TailNone:
		return "none"
	case TailShortFrame:
		return "short-frame"
	case TailPayloadChecksum:
		return "payload-checksum"
	default:
		return fmt.Sprintf("tail(%d)", uint8(tail))
	}
}

// Report describes one replay. SnapshotBytes is the log size the replay
// captured when it opened the log. ValidBytes and Records describe the
// validated prefix; a read-only replay observes those frames without claiming
// that each append's sync was acknowledged. ObservedTailBytes counts the torn
// bytes found after that prefix; RepairedBytes counts the bytes a writer-open
// truncated and is always zero for read-only replay. A replay that fails as
// corruption still reports its snapshot and validated prefix.
type Report struct {
	SnapshotBytes     int64
	Records           uint64
	ValidBytes        int64
	ObservedTailBytes int64
	RepairedBytes     int64
	Tail              Tail
}

// Record is one validated frame delivered to a replay consumer. Payload is
// owned by the consumer.
type Record struct {
	Kind    Kind
	Payload []byte
	Offset  int64
}
