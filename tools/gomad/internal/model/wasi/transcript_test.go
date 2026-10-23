package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
)

func TestTranscriptUsesVersionedBinaryRecords(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	call(t, e, "fd_sync", `{"fd":3}`)
	// Header v1, fd_sync tag 13, successful call 1, four-byte fd payload.
	want := []byte{'G', 'M', 'W', 'T', 1, 0, 0, 0,
		13, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0,
		3, 0, 0, 0}
	if got := e.Transcript(); !bytes.Equal(got, want) {
		t.Fatalf("binary transcript = %x, want %x", got, want)
	}
}

func TestTranscriptClockRunFoldsAndPreservesDigestModes(t *testing.T) {
	want := []byte("GMWT\x01\x00\x00\x00\x21\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x01\x00\x00\x00\x07\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00")
	for _, digestOnly := range []bool{false, true} {
		e := newTestEnvironment(t, testConfig())
		e.digestOnly = digestOnly
		e.transcriptHash = sha256.New()
		for range 3 {
			call(t, e, "clock_time_get", `{"clock_id":1,"precision":7}`)
		}
		if !digestOnly && !bytes.Equal(e.Transcript(), want) {
			t.Fatalf("clock reads were not folded: %x, want %x", e.Transcript(), want)
		}
		if digestOnly && len(e.Transcript()) != 0 || e.TranscriptSHA256() != sha256.Sum256(want) {
			t.Fatalf("pending clock run digest/retention mismatch: %x", e.TranscriptSHA256())
		}
		// Inspection of pending evidence must not end the run or charge another record.
		call(t, e, "clock_time_get", `{"clock_id":1,"precision":7}`)
		if !digestOnly {
			records, err := DecodeTranscript(e.Transcript())
			if err != nil || len(records) != 1 || records[0].Count != 4 || records[0].Now != 0 {
				t.Fatalf("inspection ended pending run: %+v %v", records, err)
			}
		}
	}
}

func TestTranscriptClockRunChargesCapacityOnce(t *testing.T) {
	for _, limit := range []uint64{51, 52} {
		config := testConfig()
		config.Limits.TranscriptBytes = limit
		e := newTestEnvironment(t, config)
		_, err := e.Handle(Call{Type: "call", ID: 1, Op: "clock_time_get", Input: []byte(`{"clock_id":1,"precision":0}`)})
		if limit == 51 {
			if err == nil || e.now != 0 || e.nextCall != 1 || len(e.Transcript()) != 0 {
				t.Fatalf("capacity rejection mutated clock: %v %+v", err, e)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		call(t, e, "clock_time_get", `{"clock_id":1,"precision":0}`)
		if len(e.Transcript()) != 52 || e.now != 2000 {
			t.Fatalf("folded read charged another record: bytes=%d now=%d", len(e.Transcript()), e.now)
		}
	}
}

func TestTranscriptClockRunDecoderBoundsCountsAndPositions(t *testing.T) {
	valid := []byte("GMWT\x01\x00\x00\x00\x21\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x01\x00\x00\x00\x07\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00")
	for _, test := range []struct {
		name, kind string
		offset     int
		value      uint64
	}{
		{"zero count", "invalid", 44, 0},
		{"count overflow", "capacity", 44, maxCooperativeCalls + 1},
		{"position overflow", "invalid", 36, 1 << 63},
		{"call id overflow", "invalid", 12, ^uint64(0)},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := bytes.Clone(valid)
			binary.LittleEndian.PutUint64(data[test.offset:], test.value)
			_, err := DecodeTranscript(data)
			var boundary *BoundaryError
			if !errors.As(err, &boundary) || boundary.Kind != test.kind {
				t.Fatalf("invalid clock run = %v", err)
			}
		})
	}
	duplicate := append(bytes.Clone(valid), []byte("\x0d\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x03\x00\x00\x00")...)
	if _, err := DecodeTranscript(duplicate); err == nil {
		t.Fatal("record id inside preceding clock run admitted")
	}
	plainClock := []byte("GMWT\x01\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x14\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xe9\x03\x00\x00\x00\x00\x00\x00")
	if _, err := DecodeTranscript(plainClock); err == nil {
		t.Fatal("successful clock outside canonical run record admitted")
	}
}

func TestTranscriptClockRunDigestBindsFrozenCountsKeysAndPositions(t *testing.T) {
	var reference [32]byte
	for index, input := range []struct {
		count          int
		clockID        uint32
		precision, now uint64
	}{
		{3, 1, 7, 1}, {2, 1, 7, 1}, {3, 0, 7, 1}, {3, 1, 8, 1}, {3, 1, 7, 2},
	} {
		config := testConfig()
		config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
		e := newTestEnvironment(t, config)
		e.now = input.now
		for range input.count {
			call(t, e, "clock_time_get", fmt.Sprintf(`{"clock_id":%d,"precision":%d}`, input.clockID, input.precision))
		}
		digest := e.TranscriptSHA256()
		if index == 0 {
			reference = digest
		} else if digest == reference {
			t.Fatalf("clock variant %+v kept digest", input)
		}
	}
}

func TestTranscriptDecoderRejectsMalformedRecords(t *testing.T) {
	valid := []byte{'G', 'M', 'W', 'T', 1, 0, 0, 0,
		13, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 3, 0, 0, 0}
	records, err := DecodeTranscript(valid)
	if err != nil || len(records) != 1 || records[0].ID != 1 || records[0].Op != "fd_sync" || string(records[0].Input) != `{"fd":3}` || records[0].Count != 1 {
		t.Fatalf("literal record decode = %+v: %v", records, err)
	}
	for length := 1; length < len(valid); length++ {
		if length == 8 {
			continue
		}
		if _, err := DecodeTranscript(valid[:length]); err == nil {
			t.Fatalf("truncated prefix %d admitted", length)
		}
	}
	unknown := bytes.Clone(valid)
	unknown[8] = 255
	flags := bytes.Clone(valid)
	flags[9] = 2
	trailing := append(bytes.Clone(valid), 0)
	overlong := bytes.Clone(valid)
	binary.LittleEndian.PutUint32(overlong[20:], 1<<24)
	oversizedCount := []byte{'G', 'M', 'W', 'T', 1, 0, 0, 0, 32, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 1, 16, 0, 0}
	for _, test := range []struct {
		name, kind string
		data       []byte
	}{
		{"unknown tag", "invalid", unknown},
		{"unknown flags", "invalid", flags},
		{"trailing bytes", "invalid", trailing},
		{"oversized record", "capacity", overlong},
		{"oversized count", "capacity", oversizedCount},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeTranscript(test.data)
			var boundary *BoundaryError
			if !errors.As(err, &boundary) || boundary.Kind != test.kind {
				t.Fatalf("malformed record = %v, want %s refusal", err, test.kind)
			}
		})
	}
}

func TestTranscriptRejectedOperationKeepsExactValidatedInput(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	reply := call(t, e, "x", ` { } `)
	want := []byte("GMWT\x01\x00\x00\x00\x22\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x39\x00\x00\x00\x01\x00\x00\x00x\x05\x00\x00\x00 { } \x27\x00\x00\x00operation outside stock WASI profile: x")
	if !reply.Unsupported || !bytes.Equal(e.Transcript(), want) {
		t.Fatalf("rejected operation = %x, want %x", e.Transcript(), want)
	}
	records, err := DecodeTranscript(want)
	if err != nil || len(records) != 1 || records[0].Op != "x" || string(records[0].Input) != ` { } ` || !records[0].Unsupported {
		t.Fatalf("rejected operation decode = %+v %v", records, err)
	}
	malformed := bytes.Clone(want)
	malformed[35] = '!'
	if _, err := DecodeTranscript(malformed); err == nil {
		t.Fatal("rejected-operation raw malformed JSON admitted")
	}
	known := append([]byte("GMWT\x01\x00\x00\x00\x22\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00"), []byte("\x13\x00\x00\x00\x07\x00\x00\x00fd_sync\x00\x00\x00\x00\x00\x00\x00\x00")...)
	if _, err := DecodeTranscript(known); err == nil {
		t.Fatal("known typed operation admitted through rejection tag")
	}
}
func TestTranscriptEmptyOperationPreservesUnsupportedRecord(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	reply, err := e.Handle(Call{Type: "call", ID: 1, Op: "", Input: []byte(`{}`)})
	if err != nil || !reply.Unsupported || reply.Errno != 0 || reply.Message != "operation outside stock WASI profile: " {
		t.Fatalf("empty operation reply=%+v err=%v", reply, err)
	}
	want := []byte("GMWT\x01\x00\x00\x00\x22\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x34\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00{}\x26\x00\x00\x00operation outside stock WASI profile: ")
	if !bytes.Equal(e.Transcript(), want) {
		t.Errorf("empty operation record=%x want=%x", e.Transcript(), want)
	}
	records, err := DecodeTranscript(want)
	if err != nil || len(records) != 1 || records[0].Op != "" || !records[0].Unsupported || records[0].ID != 1 || string(records[0].Input) != `{}` || records[0].Message != reply.Message {
		t.Fatalf("empty operation decode=%+v err=%v", records, err)
	}
	if roundtrip, err := DecodeTranscript(e.Transcript()); err != nil || len(roundtrip) != 1 || roundtrip[0].Op != "" || !roundtrip[0].Unsupported || roundtrip[0].Message != "operation outside stock WASI profile: " {
		t.Fatalf("empty operation producer/decoder roundtrip=%+v err=%v", roundtrip, err)
	}
}
