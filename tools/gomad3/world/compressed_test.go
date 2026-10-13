package world

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"testing"
)

func TestSnapshotUsesOneBinaryEnvelope(t *testing.T) {
	w := newTestWorld(t, 0)
	plain, err := EncodeSnapshot(w.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	detached, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := detached.Encode()
	if err != nil {
		t.Fatal(err)
	}
	for name, encoded := range map[string][]byte{"snapshot": plain, "detached": compressed} {
		t.Run(name, func(t *testing.T) {
			if len(encoded) < len(compressedEnvelopeMagic) || !bytes.Equal(encoded[:len(compressedEnvelopeMagic)], compressedEnvelopeMagic[:]) {
				t.Fatalf("World snapshot envelope magic = %x", encoded)
			}
			if bytes.Contains(encoded[:compressedEnvelopeHeaderBytes], []byte{'{'}) {
				t.Fatal("World snapshot envelope is JSON")
			}
		})
	}
	if !bytes.Equal(plain, compressed) {
		t.Fatal("ordinary and detached World snapshots use different profiles")
	}
	if _, err := DecodeSnapshot(plain); err != nil {
		t.Fatal(err)
	}
}

func TestCompressedSnapshotRoundTripSummary(t *testing.T) {
	w := newTestWorld(t, 17)
	id, err := w.Register(Request{Kind: "read", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "雪<&>\u2028"}, Payload: []byte{0, 1, 255}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Ready(Readiness{RequestID: id, At: InitialTime + 3, Kind: "ready", Payload: []byte("résultat"), EquivalenceClass: "同"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Quiesce(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Cancel(id); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Quiesce(); err != nil {
		t.Fatal(err)
	}
	snapshot := w.Snapshot()
	cardinality := w.DiagnosticCardinality()
	detached, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	if got := w.DiagnosticCardinality(); got != cardinality {
		t.Fatalf("detached diagnostic cardinality = %#v, want %#v", got, cardinality)
	}
	encoded, err := detached.Encode()
	if err != nil {
		t.Fatal(err)
	}
	summary, err := DecodeCompressedSnapshot(encoded, CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	if summary.StateDigest != snapshot.StateDigest || summary.TranscriptDigest != snapshot.TranscriptDigest || summary.RequestCount != 1 || summary.EventCount != 1 || summary.TransitionCount != 5 {
		t.Fatalf("summary = %#v", summary)
	}
	if _, err := detached.Encode(); err == nil {
		t.Fatal("detached snapshot encoded twice")
	}
	if _, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile()); err == nil {
		t.Fatal("model detached twice")
	}
}

func TestCompressedSnapshotRejectsMalformedExpandedWire(t *testing.T) {
	encoded, err := EncodeSnapshot(newTestWorld(t, 0).Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	expanded := compressedExpandedBytes(t, encoded)
	if len(expanded) != 208 {
		t.Fatalf("empty expanded snapshot bytes = %d, want 208", len(expanded))
	}
	malformedMagic := bytes.Clone(expanded)
	malformedMagic[0] ^= 1
	oversizedCount := bytes.Clone(expanded)
	binary.BigEndian.PutUint64(oversizedCount[184:192], CompressedSnapshotProfile().Limits.MaxRequests+1)
	tamperedState := bytes.Clone(expanded)
	tamperedState[152] ^= 1
	for name, wire := range map[string][]byte{
		"malformed":       malformedMagic,
		"truncated":       expanded[:len(expanded)-1],
		"duplicate field": append(bytes.Clone(expanded), expanded[len(expanded)-8:]...),
		"oversized count": oversizedCount,
		"tampered state":  tamperedState,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeCompressedSnapshot(compressedExpandedFixture(t, wire), CompressedSnapshotProfile()); err == nil {
				t.Fatal("malformed expanded snapshot admitted")
			}
		})
	}
}

func TestCompressedSnapshotRejectsCorruptedEnvelopeAndGzip(t *testing.T) {
	encoded, err := EncodeSnapshot(newTestWorld(t, 0).Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	wrongMagic := bytes.Clone(encoded)
	wrongMagic[0] ^= 1
	wrongHeader := bytes.Clone(encoded)
	binary.BigEndian.PutUint32(wrongHeader[8:12], compressedEnvelopeHeaderBytes+1)
	wrongExpanded := bytes.Clone(encoded)
	binary.BigEndian.PutUint64(wrongExpanded[16:24], 1)
	wrongCompressed := bytes.Clone(encoded)
	binary.BigEndian.PutUint64(wrongCompressed[24:32], 1)
	wrongDigest := bytes.Clone(encoded)
	wrongDigest[32] ^= 1
	wrongChecksum := bytes.Clone(encoded)
	wrongChecksum[len(wrongChecksum)-8] ^= 1
	for name, data := range map[string][]byte{
		"magic": wrongMagic, "header": wrongHeader, "expanded length": wrongExpanded,
		"compressed length": wrongCompressed, "digest": wrongDigest, "checksum": wrongChecksum,
		"truncated": encoded[:len(encoded)-1], "trailing": append(bytes.Clone(encoded), 0),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeCompressedSnapshot(data, CompressedSnapshotProfile()); err == nil {
				t.Fatal("corrupt envelope admitted")
			}
		})
	}
}

func compressedExpandedBytes(t *testing.T, encoded []byte) []byte {
	t.Helper()
	envelope, err := decodeCompressedEnvelope(encoded, CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(envelope.Gzip))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return expanded
}

func compressedExpandedFixture(t *testing.T, expanded []byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	if _, err := w.Write(expanded); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(expanded)
	encoded := make([]byte, compressedEnvelopeHeaderBytes+compressed.Len())
	copy(encoded[:8], compressedEnvelopeMagic[:])
	binary.BigEndian.PutUint32(encoded[8:12], compressedEnvelopeHeaderBytes)
	binary.BigEndian.PutUint64(encoded[16:24], uint64(len(expanded)))
	binary.BigEndian.PutUint64(encoded[24:32], uint64(compressed.Len()))
	copy(encoded[32:64], digest[:])
	copy(encoded[compressedEnvelopeHeaderBytes:], compressed.Bytes())
	return encoded
}

func TestCompressedSnapshotCopiedHandlesConsumeOnce(t *testing.T) {
	w := newTestWorld(t, 0)
	detached, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	copied := *detached
	var wait sync.WaitGroup
	errors := make(chan error, 2)
	for _, handle := range []*DetachedSnapshot{detached, &copied} {
		wait.Add(1)
		go func() { defer wait.Done(); _, err := handle.Encode(); errors <- err }()
	}
	wait.Wait()
	close(errors)
	var successes int
	for err := range errors {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful encodings = %d", successes)
	}
}

func TestCompressedSnapshotCapacityFailureConsumesHandle(t *testing.T) {
	for name, limit := range map[string]func(*ProfileLimits){
		"expanded":   func(l *ProfileLimits) { l.MaxExpandedBytes = 1 },
		"compressed": func(l *ProfileLimits) { l.MaxCompressedBytes = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			limits := CompressedSnapshotProfile()
			limit(&limits)
			detached, err := newTestWorld(t, 0).DetachCompressedSnapshot(limits)
			if err != nil {
				t.Fatal(err)
			}
			data, err := detached.Encode()
			var capacity *CapacityError
			if !errors.As(err, &capacity) || !errors.Is(err, ErrCapacity) || data != nil {
				t.Fatalf("untyped capacity result = %d bytes, %v", len(data), err)
			}
			if _, err := detached.Encode(); err == nil {
				t.Fatal("failed encoding reused transferred state")
			}
		})
	}
}

func TestCompressedSnapshotRejectsActiveRecordingWithoutFreezingModel(t *testing.T) {
	w := newTestWorld(t, 0)
	recorder, err := w.StartRecording(4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile()); err == nil {
		t.Fatal("active recorder detached")
	}
	if _, err := w.Register(Request{Kind: "read", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Quiesce(); err != nil {
		t.Fatal(err)
	}
	if _, err := recorder.Finish(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile()); err != nil {
		t.Fatal(err)
	}
}

func TestCompressedSnapshotInvalidProfileDoesNotFreezeModel(t *testing.T) {
	for name, mutate := range map[string]func(*ProfileLimits){
		"zero expanded":    func(l *ProfileLimits) { l.MaxExpandedBytes = 0 },
		"zero compressed":  func(l *ProfileLimits) { l.MaxCompressedBytes = 0 },
		"large expanded":   func(l *ProfileLimits) { l.MaxExpandedBytes++ },
		"large compressed": func(l *ProfileLimits) { l.MaxCompressedBytes++ },
		"zero requests":    func(l *ProfileLimits) { l.Limits.MaxRequests = 0 },
		"large requests":   func(l *ProfileLimits) { l.Limits.MaxRequests++ },
	} {
		t.Run(name, func(t *testing.T) {
			limits := CompressedSnapshotProfile()
			mutate(&limits)
			w := newTestWorld(t, 0)
			if detached, err := w.DetachCompressedSnapshot(limits); err == nil || detached != nil {
				t.Fatal("invalid profile detached")
			}
			if _, err := w.Quiesce(); err != nil {
				t.Fatalf("invalid profile froze model: %v", err)
			}
		})
	}
}

func TestCompressedSnapshotConfiguredCapsRejectBeforeTransfer(t *testing.T) {
	if err := ValidateCompressedSnapshotConfiguration(Config{Limits: CompressedSnapshotProfile().Limits}, CompressedSnapshotProfile()); err != nil {
		t.Fatal(err)
	}
	for name, raise := range map[string]func(*Limits){
		"requests": func(l *Limits) { l.MaxRequests++ }, "events": func(l *Limits) { l.MaxEvents++ },
		"queued": func(l *Limits) { l.MaxQueuedEvents++; l.MaxEvents++ }, "transitions": func(l *Limits) { l.MaxTransitions++ },
		"payload": func(l *Limits) { l.MaxPayloadBytes++ }, "strings": func(l *Limits) { l.MaxStringBytes++ },
	} {
		t.Run(name, func(t *testing.T) {
			limits := CompressedSnapshotProfile()
			raise(&limits.Limits)
			w, err := New(Config{Limits: limits.Limits})
			if err != nil {
				t.Fatal(err)
			}
			detached, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile())
			var capacity *CapacityError
			if !errors.As(err, &capacity) || !errors.Is(err, ErrCapacity) || detached != nil {
				t.Fatalf("configured over-capacity model did not return typed capacity: %v", err)
			}
		})
	}
}
