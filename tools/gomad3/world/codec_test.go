package world

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"testing"
)

func TestDecodeSnapshotRejectsMalformedEnvelope(t *testing.T) {
	snapshot := newTestWorld(t, 0).Snapshot()
	encoded, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSnapshot(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.StateDigest != snapshot.StateDigest {
		t.Fatalf("decoded digest = %s, want %s", decoded.StateDigest, snapshot.StateDigest)
	}
	snapshot.Replay = ReplayProgress{Cursor: 1, Expected: 2}
	encodedWithReplay, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	decodedWithReplay, err := DecodeSnapshot(encodedWithReplay)
	if err != nil {
		t.Fatal(err)
	}
	if decodedWithReplay.Replay != snapshot.Replay {
		t.Fatalf("decoded replay = %#v, want %#v", decodedWithReplay.Replay, snapshot.Replay)
	}
	badMagic := bytes.Clone(encoded)
	badMagic[0] ^= 1
	badHeader := bytes.Clone(encoded)
	binary.BigEndian.PutUint32(badHeader[8:12], compressedEnvelopeHeaderBytes+1)
	badFlags := bytes.Clone(encoded)
	badFlags[12] = 1
	for name, malformed := range map[string][]byte{
		"magic": badMagic, "header": badHeader, "flags": badFlags,
		"trailing": append(bytes.Clone(encoded), 0), "truncated": encoded[:len(encoded)-1],
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeSnapshot(malformed); err == nil {
				t.Fatal("DecodeSnapshot() succeeded")
			}
		})
	}
}

func TestMinimumLimitSnapshotRoundTrips(t *testing.T) {
	limits := Limits{MaxRequests: 1, MaxEvents: 1, MaxQueuedEvents: 1, MaxTransitions: 1, MaxPayloadBytes: 1, MaxStringBytes: 1}
	core, err := New(Config{Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.Register(Request{Kind: "k", Resource: ResourceID{Adapter: "a", Kind: "r"}}); err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeSnapshot(core.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeSnapshot(encoded); err != nil {
		t.Fatal(err)
	}
}

func TestPopulatedHighCardinalitySnapshotRoundTrips(t *testing.T) {
	const requests = 1024
	limits := Limits{
		MaxRequests: requests, MaxEvents: requests, MaxQueuedEvents: requests, MaxTransitions: requests * 2,
		MaxPayloadBytes: 1 << 20, MaxStringBytes: 64,
	}
	core, err := New(Config{Seed: 7, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	for index := range requests {
		id, err := core.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: strconv.Itoa(index)}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := core.Cancel(id); err != nil {
			t.Fatal(err)
		}
	}
	encoded, err := EncodeSnapshot(core.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeSnapshot(encoded); err != nil {
		t.Fatal(err)
	}
}

func FuzzDecodeSnapshot(f *testing.F) {
	w, err := New(Config{Seed: 1, Limits: testLimits()})
	if err != nil {
		f.Fatal(err)
	}
	encoded, err := EncodeSnapshot(w.Snapshot())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(encoded)
	f.Add([]byte("GMWORLD2"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		_, _ = DecodeSnapshot(data)
	})
}
