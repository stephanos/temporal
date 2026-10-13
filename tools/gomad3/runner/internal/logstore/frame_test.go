package logstore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"strings"
	"testing"
)

func TestFrameRoundTripProtectsHeaderAndPayload(t *testing.T) {
	payload := []byte(`{"sequence":0,"text":"abc"}`)
	frame := encodeFrame(kindEntry, payload)
	if len(frame) != headerSize+len(payload)+trailerSize || frameBytes(uint32(len(payload))) != int64(len(frame)) {
		t.Fatalf("frame length = %d", len(frame))
	}
	parsed, err := decodeHeader(frame[:headerSize], testKinds)
	if err != nil || parsed.kind != kindEntry || parsed.length != uint32(len(payload)) {
		t.Fatalf("decoded header = %#v, %v", parsed, err)
	}
	if got := binary.LittleEndian.Uint32(frame[headerSize+len(payload):]); got != payloadChecksum(frame[:headerSize], payload) {
		t.Fatalf("payload checksum = %#x", got)
	}
	if frameBytes(math.MaxUint32) != headerSize+math.MaxUint32+trailerSize {
		t.Fatal("frame arithmetic overflowed")
	}
}

func TestDecodeHeaderRejectsEveryUntrustedField(t *testing.T) {
	valid := encodeFrame(kindEntry, []byte(`{}`))[:headerSize]
	resum := func(raw []byte) []byte {
		binary.LittleEndian.PutUint32(raw[12:], crc32.Checksum(raw[:12], castagnoli))
		return raw
	}
	mutate := func(edit func([]byte) []byte) []byte {
		return edit(bytes.Clone(valid))
	}
	for _, test := range []struct {
		name, want string
		raw        []byte
	}{
		{"short", "incomplete", valid[:headerSize-1]},
		{"magic", "magic", mutate(func(raw []byte) []byte { raw[0] = 'X'; return resum(raw) })},
		{"header-checksum", "header checksum", mutate(func(raw []byte) []byte { raw[12] ^= 1; return raw })},
		{"unchecksummed-length", "header checksum", mutate(func(raw []byte) []byte { raw[11] = 0xff; return raw })},
		{"version", "version", mutate(func(raw []byte) []byte { raw[4] = 2; return resum(raw) })},
		{"flags", "flags", mutate(func(raw []byte) []byte { raw[5] = 1; return resum(raw) })},
		{"zero-kind", "unknown", mutate(func(raw []byte) []byte { binary.LittleEndian.PutUint16(raw[6:], 0); return resum(raw) })},
		{"unknown-kind", "unknown", mutate(func(raw []byte) []byte { binary.LittleEndian.PutUint16(raw[6:], 9); return resum(raw) })},
		{"oversized", "exceeds", mutate(func(raw []byte) []byte { binary.LittleEndian.PutUint32(raw[8:], 257); return resum(raw) })},
		{"maximum-length", "exceeds", mutate(func(raw []byte) []byte { binary.LittleEndian.PutUint32(raw[8:], math.MaxUint32); return resum(raw) })},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeHeader(test.raw, testKinds); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("decodeHeader error = %v, want %q", err, test.want)
			}
		})
	}
	bounded := mutate(func(raw []byte) []byte { binary.LittleEndian.PutUint32(raw[8:], 256); return resum(raw) })
	if _, err := decodeHeader(bounded, testKinds); err != nil {
		t.Fatalf("payload at its kind bound rejected: %v", err)
	}
}

func TestDecodeJSONIsStrict(t *testing.T) {
	var value entry
	if err := DecodeJSON([]byte(" {\"sequence\":1,\"text\":\"x\"}\n"), &value); err != nil || value != (entry{1, "x"}) {
		t.Fatalf("valid payload = %#v, %v", value, err)
	}
	for _, payload := range []string{`{"sequence":1,"extra":true}`, `{"sequence":1} {}`, `{"sequence":1}x`, `null`, ` null `, `{"sequence":"1"}`, `{`, ``} {
		if err := DecodeJSON([]byte(payload), &entry{}); err == nil {
			t.Errorf("DecodeJSON(%q) accepted", payload)
		}
	}
	if _, err := EncodeJSON(func() {}); err == nil {
		t.Fatal("unencodable value accepted")
	}
}

func TestOptionsRequireBoundedKnownKinds(t *testing.T) {
	for _, kinds := range []map[Kind]uint32{nil, {}, {0: 1}, {1: 0}, {1: MaxPayloadBytes + 1}} {
		if _, err := (Options{Kinds: kinds}).validate(); err == nil {
			t.Errorf("kinds %v accepted", kinds)
		}
		if _, _, err := OpenWriter(t.TempDir(), Options{Kinds: kinds}, nil); err == nil {
			t.Errorf("OpenWriter accepted kinds %v", kinds)
		}
		if _, err := Replay(t.TempDir(), Options{Kinds: kinds}, nil); err == nil {
			t.Errorf("Replay accepted kinds %v", kinds)
		}
	}
	if Tail(9).String() != "tail(9)" || TailPayloadChecksum.String() != "payload-checksum" {
		t.Fatal("tail names changed")
	}
}

// TestEveryTruncationIsATornTailNeverCorruption cuts a valid log at every
// byte: readers report the complete-frame prefix and the observed tail without
// mutation, and writer-open truncates exactly that tail and can append again.
func TestEveryTruncationIsATornTailNeverCorruption(t *testing.T) {
	log, boundaries := entryFrames(t, 3)
	for length := 0; length <= len(log); length++ {
		t.Run(fmt.Sprint(length), func(t *testing.T) {
			complete := 0
			for complete+1 < len(boundaries) && boundaries[complete+1] <= int64(length) {
				complete++
			}
			valid := boundaries[complete]
			wantTail := TailNone
			if valid != int64(length) {
				wantTail = TailShortFrame
			}
			path := rawStore(t, log[:length])
			before := tree(t, path)
			replayed, report := replayState(t, path)
			requireEntries(t, replayed.entries, complete)
			if report != (Report{Records: uint64(complete), ValidBytes: valid, ObservedTailBytes: int64(length) - valid, Tail: wantTail}) {
				t.Fatalf("read-only report = %#v", report)
			}
			assertTree(t, path, before)

			var recovered state
			writer, report, err := OpenWriter(path, testOptions(false, nil), recovered.consume)
			if err != nil {
				t.Fatal(err)
			}
			defer closeWriter(t, writer)
			if report.RepairedBytes != int64(length)-valid || report.ObservedTailBytes != report.RepairedBytes || report.Tail != wantTail {
				t.Fatalf("writer report = %#v", report)
			}
			requireEntries(t, recovered.entries, complete)
			if got := readLog(t, path); !bytes.Equal(got, log[:valid]) {
				t.Fatalf("repair left %d bytes, want %d", len(got), valid)
			}
			if err := writer.AppendJSON(kindEntry, makeEntry(complete)); err != nil {
				t.Fatal(err)
			}
			if writer.Records() != uint64(complete+1) || writer.Offset() != valid+int64(len(frameOf(t, kindEntry, makeEntry(complete)))) {
				t.Fatalf("writer position = %d records, %d bytes", writer.Records(), writer.Offset())
			}
			closeWriter(t, writer)
			replayed, report = replayState(t, path)
			requireEntries(t, replayed.entries, complete+1)
			if report.Tail != TailNone {
				t.Fatalf("appended log report = %#v", report)
			}
		})
	}
}

// TestEveryByteFlipIsClassified damages every byte of a three-frame log. Only
// payload or trailer damage in the complete final frame is a torn tail; all
// other damage, including any header byte, is corruption that neither reader
// nor writer repairs.
func TestEveryByteFlipIsClassified(t *testing.T) {
	log, boundaries := entryFrames(t, 3)
	finalStart := boundaries[2]
	for index := range log {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			damaged := bytes.Clone(log)
			damaged[index] ^= 0xff
			path := rawStore(t, damaged)
			before := tree(t, path)
			var replayed state
			report, readErr := Replay(path, testOptions(false, nil), replayed.consume)
			var recovered state
			writer, writerReport, writeErr := OpenWriter(path, testOptions(false, nil), recovered.consume)
			if int64(index) >= finalStart+headerSize {
				if readErr != nil || report.Tail != TailPayloadChecksum || report.Records != 2 || report.RepairedBytes != 0 {
					t.Fatalf("final payload damage read = %#v, %v", report, readErr)
				}
				if writeErr != nil || writerReport.RepairedBytes != int64(len(log))-finalStart {
					t.Fatalf("final payload damage repair = %#v, %v", writerReport, writeErr)
				}
				closeWriter(t, writer)
				requireEntries(t, recovered.entries, 2)
				return
			}
			frame := 0
			for boundaries[frame+1] <= int64(index) {
				frame++
			}
			for _, err := range []error{readErr, writeErr} {
				var corruption *CorruptionError
				if !errors.As(err, &corruption) || !errors.Is(err, ErrCorrupt) || corruption.Offset != boundaries[frame] || corruption.Records != uint64(frame) {
					t.Fatalf("damage in frame %d = %v", frame, err)
				}
			}
			if writer != nil {
				t.Fatal("writer opened over corruption")
			}
			requireEntries(t, replayed.entries, frame)
			assertTree(t, path, before)
		})
	}
}

// TestCorruptLengthIsNotALargeTornTail proves that an earlier header whose
// length was damaged, or rewritten past its kind bound, fails as corruption
// instead of claiming the rest of the log as one torn final record.
func TestCorruptLengthIsNotALargeTornTail(t *testing.T) {
	log, _ := entryFrames(t, 3)
	for _, test := range []struct {
		name   string
		length uint32
		resum  bool
	}{
		{"unchecksummed-large", 1 << 20, false},
		{"unchecksummed-small", 1, false},
		{"checksummed-over-bound", 257, true},
		{"checksummed-maximum", math.MaxUint32, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			damaged := bytes.Clone(log)
			binary.LittleEndian.PutUint32(damaged[8:], test.length)
			if test.resum {
				binary.LittleEndian.PutUint32(damaged[12:], crc32.Checksum(damaged[:12], castagnoli))
			}
			path := rawStore(t, damaged)
			before := tree(t, path)
			if _, err := Replay(path, testOptions(false, nil), nil); !errors.Is(err, ErrCorrupt) {
				t.Fatalf("Replay error = %v", err)
			}
			if _, _, err := OpenWriter(path, testOptions(false, nil), nil); !errors.Is(err, ErrCorrupt) {
				t.Fatalf("OpenWriter error = %v", err)
			}
			assertTree(t, path, before)
		})
	}
}

// TestValidFramesWithInvalidContentsAreCorruption covers fully framed records
// whose kind, JSON or state is invalid, in both earlier and final position.
func TestValidFramesWithInvalidContentsAreCorruption(t *testing.T) {
	first := frameOf(t, kindEntry, makeEntry(0))
	for _, test := range []struct {
		name  string
		frame []byte
	}{
		{"unknown-field", encodeFrame(kindEntry, []byte(`{"sequence":1,"text":"x","extra":1}`))},
		{"trailing-data", encodeFrame(kindEntry, []byte(`{"sequence":1,"text":"x"} {}`))},
		{"not-json", encodeFrame(kindEntry, []byte(`not json`))},
		{"null", encodeFrame(kindEntry, []byte(`null`))},
		{"empty", encodeFrame(kindEntry, nil)},
		{"invalid-state", frameOf(t, kindEntry, makeEntry(5))},
		{"wrong-kind-shape", frameOf(t, kindMarker, makeEntry(1))},
		{"unknown-kind", encodeFrame(9, []byte(`{}`))},
	} {
		for _, final := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-final-%t", test.name, final), func(t *testing.T) {
				log := append(bytes.Clone(first), test.frame...)
				if !final {
					log = append(log, frameOf(t, kindMarker, marker{"after"})...)
				}
				path := rawStore(t, log)
				before := tree(t, path)
				var replayed state
				_, err := Replay(path, testOptions(false, nil), replayed.consume)
				var corruption *CorruptionError
				if !errors.As(err, &corruption) || corruption.Offset != int64(len(first)) || corruption.Records != 1 {
					t.Fatalf("Replay error = %v", err)
				}
				if len(replayed.entries) != 1 || len(replayed.markers) != 0 {
					t.Fatalf("consumer applied part of a rejected record: %#v", replayed)
				}
				if _, _, err := OpenWriter(path, testOptions(false, nil), (&state{}).consume); !errors.Is(err, ErrCorrupt) {
					t.Fatalf("OpenWriter error = %v", err)
				}
				assertTree(t, path, before)
			})
		}
	}
}
