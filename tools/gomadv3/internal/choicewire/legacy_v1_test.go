package choicewire

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"testing"
)

func TestDecodeLegacyV1TraceRemainsInspectableButNotReplayable(t *testing.T) {
	payload := make([]byte, 48)
	binary.BigEndian.PutUint64(payload[:8], 0)
	payload[8] = byte(KindRunnable)
	payload[9] = byte(FlagDecision)
	binary.BigEndian.PutUint32(payload[12:16], 2)
	binary.BigEndian.PutUint32(payload[16:20], 1)
	binary.BigEndian.PutUint64(payload[24:32], 17)
	terminal := legacyV1Terminal(payload, 1)

	trace, err := DecodeLegacyV1Trace(payload, terminal, 64+48)
	if err != nil {
		t.Fatal(err)
	}
	if trace.Version != Version1 || len(trace.Records) != 1 || trace.Records[0].Selected != 1 || trace.Summary.Branching != 1 {
		t.Fatalf("legacy trace = %#v", trace)
	}
	if _, err := ProjectDecisionTape(trace, ExecutionIdentity{}); !errors.Is(err, ErrReplayUnavailable) {
		t.Fatalf("ProjectDecisionTape() error = %v", err)
	}
}

func TestDecodeStoredLegacyV1TraceProjectsObservationalEvidence(t *testing.T) {
	payload := make([]byte, legacyRecordBytes)
	binary.BigEndian.PutUint64(payload[:8], 0)
	payload[8] = byte(KindRunnable)
	payload[9] = byte(FlagDecision)
	binary.BigEndian.PutUint32(payload[12:16], 2)
	binary.BigEndian.PutUint32(payload[16:20], 1)
	binary.BigEndian.PutUint64(payload[24:32], 17)
	digest := sha256.Sum256(payload)

	trace, err := DecodeStoredTrace(LegacyProfile, payload, TerminalMetadata{State: TerminalComplete, Limit: legacyHeaderBytes + legacyRecordBytes, Records: 1, SHA256: digest})
	if err != nil {
		t.Fatal(err)
	}
	projection, err := ProjectTrace(trace, legacyHeaderBytes+legacyRecordBytes, sha256.Sum256([]byte("target")))
	if err != nil {
		t.Fatal(err)
	}
	if projection.Profile != LegacyProfile || projection.Summary.Branching != 1 || projection.PayloadBytes != legacyRecordBytes {
		t.Fatalf("legacy projection = %#v", projection)
	}
}

func legacyV1Terminal(payload []byte, records uint64) []byte {
	frame := make([]byte, 96)
	copy(frame[:8], []byte{'G', 'O', 'M', 'A', 'D', 'C', 'T', 1})
	binary.BigEndian.PutUint32(frame[8:12], 1)
	frame[12] = 1
	binary.BigEndian.PutUint64(frame[16:24], records)
	binary.BigEndian.PutUint64(frame[24:32], 64+uint64(len(payload)))
	digest := sha256.Sum256(payload)
	copy(frame[32:64], digest[:])
	checksum := sha256.Sum256(frame[:64])
	copy(frame[64:], checksum[:])
	return frame
}
