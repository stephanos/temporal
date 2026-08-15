package world

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestWorldSnapshotRestoreContinuesWithIdenticalState(t *testing.T) {
	original := newTestWorld(t, 17)
	pending, err := original.Register(Request{Kind: "pending", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "p"}, Payload: []byte("request")})
	if err != nil {
		t.Fatal(err)
	}
	queued, err := original.Register(Request{Kind: "queued", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "q"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := original.Ready(Readiness{RequestID: queued, At: InitialTime + 5, Kind: "done", Payload: []byte("result")}); err != nil {
		t.Fatal(err)
	}
	canceled, err := original.Register(Request{Kind: "canceled", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "c"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := original.Cancel(canceled); err != nil {
		t.Fatal(err)
	}
	snapshot := original.Snapshot()
	restored, err := Restore(snapshot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := restored.Snapshot().StateDigest; got != snapshot.StateDigest {
		t.Fatalf("restored digest = %s, want %s", got, snapshot.StateDigest)
	}
	for name, world := range map[string]*Model{"original": original, "restored": restored} {
		batch, quiesceErr := world.Quiesce()
		if quiesceErr != nil {
			t.Fatal(quiesceErr)
		}
		if batch.Kind != QuiescenceDelivered || len(batch.Deliveries) != 1 || batch.Deliveries[0].RequestID != queued {
			t.Fatalf("%s batch = %#v", name, batch)
		}
		if _, cancelErr := world.Cancel(pending); cancelErr != nil {
			t.Fatal(cancelErr)
		}
		id, registerErr := world.Register(Request{Kind: "next", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "n"}})
		if registerErr != nil {
			t.Fatal(registerErr)
		}
		if id != 4 {
			t.Fatalf("%s next ID = %d", name, id)
		}
	}
	if got, want := restored.Snapshot().StateDigest, original.Snapshot().StateDigest; got != want {
		t.Fatalf("continued restored digest = %s, want %s", got, want)
	}
}

func TestRestoreRejectsCorruptionWithoutPublishingWorld(t *testing.T) {
	w := newTestWorld(t, 3)
	id, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}, Payload: []byte("payload")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Ready(Readiness{RequestID: id, At: InitialTime + 1, Kind: "done"}); err != nil {
		t.Fatal(err)
	}
	valid := w.Snapshot()
	tests := map[string]func(*Snapshot){
		"schema": func(snapshot *Snapshot) { snapshot.SchemaVersion++ },
		"digest": func(snapshot *Snapshot) {
			snapshot.StateDigest = Digest("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		},
		"request order":      func(snapshot *Snapshot) { snapshot.Requests = append(snapshot.Requests, snapshot.Requests[0]) },
		"event reference":    func(snapshot *Snapshot) { snapshot.Events[0].Readiness.RequestID++ },
		"payload accounting": func(snapshot *Snapshot) { snapshot.PayloadBytes++ },
		"time":               func(snapshot *Snapshot) { snapshot.Now = InitialTime + 2 },
		"transition digest": func(snapshot *Snapshot) {
			snapshot.Transitions[0].Digest = Digest("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		},
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			snapshot := valid
			snapshot.Requests = append([]RequestSnapshot(nil), valid.Requests...)
			snapshot.Events = append([]EventSnapshot(nil), valid.Events...)
			snapshot.Transitions = copyTransitions(valid.Transitions)
			corrupt(&snapshot)
			if restored, err := Restore(snapshot, nil); !errors.Is(err, ErrInvalidSnapshot) || restored != nil {
				t.Fatalf("Restore() = %#v, %v", restored, err)
			}
		})
	}
}

func TestWorldJSONIdentitiesRequireCanonicalDecimalStrings(t *testing.T) {
	for _, input := range []string{`0`, `"01"`, `"+1"`, `"-1"`, `"18446744073709551616"`} {
		var id RequestID
		if err := json.Unmarshal([]byte(input), &id); err == nil {
			t.Fatalf("Unmarshal(%s) succeeded with %d", input, id)
		}
	}
	var seed Seed
	if err := json.Unmarshal([]byte(`"0"`), &seed); err != nil || seed != 0 {
		t.Fatalf("seed zero = %d, %v", seed, err)
	}
	encoded, err := json.Marshal(struct {
		Seed Seed      `json:"seed"`
		ID   RequestID `json:"id"`
	}{Seed: ^Seed(0), ID: RequestID(^uint64(0))})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(encoded), `{"seed":"18446744073709551615","id":"18446744073709551615"}`; got != want {
		t.Fatalf("encoded identities = %s, want %s", got, want)
	}
}

func TestWorldStateDigestGolden(t *testing.T) {
	w := newTestWorld(t, 0)
	empty := w.Snapshot().StateDigest
	id, err := w.Register(Request{Kind: "read", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "golden"}, Payload: []byte{0, 1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Ready(Readiness{RequestID: id, At: InitialTime + 9, Kind: "result", Payload: []byte("value"), EquivalenceClass: "choice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Quiesce(); err != nil {
		t.Fatal(err)
	}
	populated := w.Snapshot().StateDigest
	if empty != "26d1d224d5b91771591d11db918a8123f9b40305a818c24cbbab5e228c4f4551" || populated != "a12cef38dd0aee9573d8eba5aefec220f1ab177b74d9c0bca7af9e8ea9f66e8d" {
		t.Fatalf("World digest goldens: empty=%s populated=%s", empty, populated)
	}
}
