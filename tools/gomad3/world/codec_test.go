package world

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestDecodeSnapshotRejectsUnknownDuplicateAndTrailingData(t *testing.T) {
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
	unknown := append([]byte(`{"unknown":true,`), encoded[1:]...)
	duplicate := append([]byte(`{"config":null,`), encoded[1:]...)
	for name, malformed := range map[string][]byte{
		"unknown":    unknown,
		"duplicate":  duplicate,
		"trailing":   append(append([]byte(nil), encoded...), []byte(` null`)...),
		"truncated":  encoded[:len(encoded)-1],
		"whitespace": append([]byte{' '}, encoded...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeSnapshot(malformed); err == nil {
				t.Fatal("DecodeSnapshot() succeeded")
			}
		})
	}
}

func TestDecodeSnapshotPreflightsConfiguredArrayLimits(t *testing.T) {
	limits := testLimits()
	limits.MaxRequests = 1
	limits.MaxEvents = 1
	limits.MaxQueuedEvents = 1
	world, err := New(Config{Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeSnapshot(world.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	oversized := bytes.Replace(encoded, []byte(`"events":[]`), []byte(`"events":[{},{}]`), 1)
	if _, err := DecodeSnapshot(oversized); !errors.Is(err, ErrInvalidSnapshot) || !strings.Contains(err.Error(), "element count exceeds configured limit") {
		t.Fatalf("DecodeSnapshot() error = %v", err)
	}
	nested := bytes.Replace(encoded, []byte(`"transitions":[]`), []byte(`"transitions":[{"nested":[{},{}]}]`), 1)
	if _, err := DecodeSnapshot(nested); !errors.Is(err, ErrInvalidSnapshot) || !strings.Contains(err.Error(), "aggregate element count exceeds configured limit") {
		t.Fatalf("DecodeSnapshot() nested error = %v", err)
	}
}

func TestSnapshotPreflightCountsObjectMembersAndScalarValues(t *testing.T) {
	decoder := json.NewDecoder(strings.NewReader(`{"first":1,"second":2}`))
	budget := uint64(4)
	if err := consumeBoundedJSONValue(decoder, 10, &budget, 0); err == nil || !strings.Contains(err.Error(), "node count") {
		t.Fatalf("consumeBoundedJSONValue() error = %v", err)
	}
}

func TestMinimumLimitSnapshotRoundTrips(t *testing.T) {
	limits := Limits{MaxRequests: 1, MaxEvents: 1, MaxQueuedEvents: 1, MaxTransitions: 1, MaxPayloadBytes: 1, MaxStringBytes: 1}
	core, err := New(Config{Limits: limits})
	if err != nil {
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
	encoded, err := json.Marshal(w.Snapshot())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(encoded)
	f.Add([]byte(`{"schemaVersion":1}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		_, _ = DecodeSnapshot(data)
	})
}
