package world

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)

func TestCompressedSnapshotPreservesCanonicalSnapshotBytes(t *testing.T) {
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
	legacy := w.Snapshot()
	cardinality := w.DiagnosticCardinality()
	want, err := EncodeSnapshot(legacy)
	if err != nil {
		t.Fatal(err)
	}
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
	var envelope struct {
		Gzip []byte `json:"gzip"`
	}
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(envelope.Gzip))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("expanded snapshot changed: got %s, want %s", actual, want)
	}
	summary, err := DecodeCompressedSnapshot(encoded, CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	if summary.StateDigest != legacy.StateDigest || summary.TranscriptDigest != legacy.TranscriptDigest || summary.RequestCount != 1 || summary.EventCount != 1 || summary.TransitionCount != 5 {
		t.Fatalf("summary = %#v", summary)
	}
	if _, err := detached.Encode(); err == nil {
		t.Fatal("detached snapshot encoded twice")
	}
	if _, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile()); err == nil {
		t.Fatal("model detached twice")
	}
	for name, mutate := range map[string]func() error{
		"register": func() error { _, err := w.Register(Request{}); return err },
		"ready":    func() error { _, err := w.Ready(Readiness{}); return err },
		"cancel":   func() error { _, err := w.Cancel(id); return err },
		"quiesce":  func() error { _, err := w.Quiesce(); return err },
		"record":   func() error { _, err := w.StartRecording(1024); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := mutate(); err == nil {
				t.Fatal("detached model mutated")
			}
		})
	}
}

func TestCompressedSnapshotEmptyPreservesCanonicalArrays(t *testing.T) {
	w := newTestWorld(t, 0)
	want, err := EncodeSnapshot(w.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	detached, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := detached.Encode()
	if err != nil {
		t.Fatal(err)
	}
	var e compressedSnapshotEnvelope
	if err := json.Unmarshal(encoded, &e); err != nil {
		t.Fatal(err)
	}
	if err := compressedGzipPass(e, func(r io.Reader) error {
		got, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("expanded empty snapshot = %s, want %s", got, want)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeCompressedSnapshot(encoded, CompressedSnapshotProfile()); err != nil {
		t.Fatal(err)
	}
}

func TestCompressedSnapshotRejectsMalformedExpandedState(t *testing.T) {
	w := newTestWorld(t, 0)
	id, err := w.Register(Request{Kind: "read", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}, Payload: []byte("r")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Ready(Readiness{RequestID: id, At: InitialTime, Kind: "done", Payload: []byte("v")}); err != nil {
		t.Fatal(err)
	}
	valid := w.Snapshot()
	expanded, err := EncodeSnapshot(valid)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(Snapshot) Snapshot{
		"missing register body": func(s Snapshot) Snapshot { s.Transitions[0].Register = nil; return s },
		"missing ready body":    func(s Snapshot) Snapshot { s.Transitions[1].Ready = nil; return s },
		"transition sequence":   func(s Snapshot) Snapshot { s.Transitions[1].Sequence = 1; return s },
		"transition digest":     func(s Snapshot) Snapshot { s.Transitions[0].Digest = emptyDigest(); return s },
		"request state":         func(s Snapshot) Snapshot { s.Requests[0].State = RequestCanceled; return s },
		"event identity":        func(s Snapshot) Snapshot { s.Events[0].Readiness.RequestID = 2; return s },
		"request order":         func(s Snapshot) Snapshot { s.Requests = append(s.Requests, s.Requests[0]); return s },
		"event order":           func(s Snapshot) Snapshot { s.Events = append(s.Events, s.Events[0]); return s },
		"payload accounting":    func(s Snapshot) Snapshot { s.PayloadBytes++; return s },
		"schema":                func(s Snapshot) Snapshot { s.SchemaVersion++; return s },
		"state digest":          func(s Snapshot) Snapshot { s.StateDigest = emptyDigest(); return s },
		"transcript digest":     func(s Snapshot) Snapshot { s.TranscriptDigest = emptyDigest(); return s },
		"replay cursor":         func(s Snapshot) Snapshot { s.Replay.Cursor = 1; return s },
		"config capacity":       func(s Snapshot) Snapshot { s.Config.Limits.MaxRequests = 250001; return s },
		"strings":               func(s Snapshot) Snapshot { s.Requests[0].Request.Kind = strings.Repeat("x", 1025); return s },
	} {
		t.Run(name, func(t *testing.T) {
			s := valid
			s.Requests = append([]RequestSnapshot(nil), valid.Requests...)
			s.Events = append([]EventSnapshot(nil), valid.Events...)
			s.Transitions = copyTransitions(valid.Transitions)
			bad, err := canonicaljson.CanonicalJSON(mutate(s))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeCompressedSnapshot(compressedExpandedFixture(t, bad), CompressedSnapshotProfile()); err == nil {
				t.Fatal("malformed expanded snapshot admitted")
			}
		})
	}
	for name, bad := range map[string][]byte{
		"whitespace":           append([]byte(" "), expanded...),
		"duplicate key":        []byte(strings.Replace(string(expanded), `"schemaVersion":1`, `"schemaVersion":1,"schemaVersion":1`, 1)),
		"duplicate nested key": []byte(strings.Replace(string(expanded), `"seed":"0"`, `"seed":"0","seed":"0"`, 1)),
		"unknown key":          []byte(strings.Replace(string(expanded), `"schemaVersion":1`, `"schemaVersion":1,"unexpected":0`, 1)),
		"unknown nested key":   []byte(strings.Replace(string(expanded), `"seed":"0"`, `"seed":"0","unexpected":0`, 1)),
		"invalid UTF8":         []byte(strings.Replace(string(expanded), `"key":"a"`, "\"key\":\"\xff\"", 1)),
		"noncanonical unicode": []byte(strings.Replace(string(expanded), `"key":"a"`, `"key":"\u0061"`, 1)),
		"truncated":            expanded[:len(expanded)-1],
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeCompressedSnapshot(compressedExpandedFixture(t, bad), CompressedSnapshotProfile()); err == nil {
				t.Fatal("malformed expanded bytes admitted")
			}
		})
	}
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
	encoded, err := canonicaljson.CanonicalJSON(compressedSnapshotEnvelope{
		Schema: CompressedSnapshotSchema, ExpandedBytes: uint64(len(expanded)), CompressedBytes: uint64(compressed.Len()),
		ExpandedSHA256: Digest(hex.EncodeToString(digest[:])), Gzip: compressed.Bytes(),
	})
	if err != nil {
		t.Fatal(err)
	}
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
	if got := w.Snapshot(); got.SchemaVersion != 0 || got.StateDigest != "" {
		t.Fatalf("detached model published snapshot: %#v", got)
	}
}

func TestCompressedSnapshotRejectsInvalidGzipStreams(t *testing.T) {
	w := newTestWorld(t, 0)
	expanded, err := EncodeSnapshot(w.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	var valid compressedSnapshotEnvelope
	if err := json.Unmarshal(compressedExpandedFixture(t, expanded), &valid); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*compressedSnapshotEnvelope){
		"multiple members": func(e *compressedSnapshotEnvelope) {
			e.Gzip = append(append([]byte(nil), e.Gzip...), e.Gzip...)
			e.CompressedBytes = uint64(len(e.Gzip))
		},
		"trailing data": func(e *compressedSnapshotEnvelope) {
			e.Gzip = append(append([]byte(nil), e.Gzip...), 0)
			e.CompressedBytes++
		},
		"truncated trailer":                 func(e *compressedSnapshotEnvelope) { e.Gzip = e.Gzip[:len(e.Gzip)-1]; e.CompressedBytes-- },
		"truncated header":                  func(e *compressedSnapshotEnvelope) { e.Gzip = e.Gzip[:5]; e.CompressedBytes = 5 },
		"decompression exceeds declaration": func(e *compressedSnapshotEnvelope) { e.ExpandedBytes = 1 },
		"oversize expansion declaration":    func(e *compressedSnapshotEnvelope) { e.ExpandedBytes = 384<<20 + 1 },
		"oversize compressed declaration":   func(e *compressedSnapshotEnvelope) { e.CompressedBytes = 64<<20 + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			e := valid
			mutate(&e)
			data, err := canonicaljson.CanonicalJSON(e)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeCompressedSnapshot(data, CompressedSnapshotProfile()); err == nil {
				t.Fatal("invalid compressed stream admitted")
			}
		})
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
	detached, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recorder.Finish(); err == nil {
		t.Fatal("stale recorder finished after detach")
	}
	if _, err := detached.Encode(); err != nil {
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
		"requests":    func(l *Limits) { l.MaxRequests++ },
		"events":      func(l *Limits) { l.MaxEvents++ },
		"queued":      func(l *Limits) { l.MaxQueuedEvents++; l.MaxEvents++ },
		"transitions": func(l *Limits) { l.MaxTransitions++ },
		"payload":     func(l *Limits) { l.MaxPayloadBytes++ },
		"strings":     func(l *Limits) { l.MaxStringBytes++ },
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
			if _, err := w.Quiesce(); err != nil {
				t.Fatalf("failed detach froze model: %v", err)
			}
		})
	}
}

func TestCompressedSnapshotRejectsCorruptedEnvelope(t *testing.T) {
	w := newTestWorld(t, 0)
	detached, err := w.DetachCompressedSnapshot(CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := detached.Encode()
	if err != nil {
		t.Fatal(err)
	}
	var valid map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &valid); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]json.RawMessage){
		"schema":            func(e map[string]json.RawMessage) { e["schema"] = json.RawMessage(`"other"`) },
		"expanded length":   func(e map[string]json.RawMessage) { e["expanded_bytes"] = json.RawMessage(`1`) },
		"compressed length": func(e map[string]json.RawMessage) { e["compressed_bytes"] = json.RawMessage(`1`) },
		"digest": func(e map[string]json.RawMessage) {
			e["expanded_sha256"] = json.RawMessage(`"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`)
		},
		"unknown": func(e map[string]json.RawMessage) { e["unknown"] = json.RawMessage(`0`) },
		"checksum": func(e map[string]json.RawMessage) {
			var data []byte
			if err := json.Unmarshal(e["gzip"], &data); err != nil {
				t.Fatal(err)
			}
			data[len(data)-8] ^= 1
			b, err := json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			e["gzip"] = b
		},
	} {
		t.Run(name, func(t *testing.T) {
			e := make(map[string]json.RawMessage, len(valid))
			for k, v := range valid {
				e[k] = v
			}
			mutate(e)
			corrupt, err := canonicaljson.CanonicalJSON(e)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeCompressedSnapshot(corrupt, CompressedSnapshotProfile()); err == nil {
				t.Fatal("corrupt envelope admitted")
			}
		})
	}
}
