package choice

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"
)

func TestCreationMetadataCodec(t *testing.T) {
	t.Parallel()
	var encoded [96]byte
	encoded[8] = 4
	binary.BigEndian.PutUint32(encoded[12:16], 2)
	binary.BigEndian.PutUint64(encoded[24:32], 123)
	encoded[32] = 1
	record, err := decodeRecord(encoded[:])
	if err != nil {
		t.Fatal(err)
	}
	if record.SiteOffset != 123 || record.SelectedIdentity[0] != 1 || record.Flags != 0 {
		t.Fatalf("creation metadata = %+v", record)
	}
	actual, err := encodeRecord(record)
	if err != nil || actual != encoded {
		t.Fatalf("creation roundtrip = %x, %v", actual, err)
	}
}

func TestCreationMetadataRequiresCompleteAlternativeSet(t *testing.T) {
	t.Parallel()
	metadata := Record{Ordinal: 0, Kind: KindGoroutineCreation, Alternatives: 2, SiteOffset: 123, SelectedIdentity: [32]byte{1}}
	if _, err := BuildTrace([]Record{metadata}, TerminalComplete); err == nil {
		t.Fatal("orphan creation metadata accepted")
	}
}

func TestPreviousVersionTraceRemainsReadable(t *testing.T) {
	t.Parallel()
	payload := encodeRecords(t, []Record{runnableDecision(t, 0).Record()})
	frame := encodeTerminal(terminal{State: TerminalComplete, Records: 1, MappingBytes: traceHeaderBytes + uint64(len(payload)), PayloadHash: sha256.Sum256(payload)})
	frame[7] = 3
	binary.BigEndian.PutUint32(frame[8:12], 3)
	checksum := sha256.Sum256(frame[:280])
	copy(frame[280:], checksum[:])
	trace, err := DecodeTrace(payload, frame[:], traceHeaderBytes+uint64(len(payload)))
	if err != nil || trace.Version != 3 || string(trace.Bytes) != string(payload) {
		t.Fatalf("previous trace = %+v, %v", trace, err)
	}
}
