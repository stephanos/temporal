package artifact

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestRetainedBytesCountsASharedTargetOnce(t *testing.T) {
	first := SharedTarget{SHA256: record.HashBytes([]byte("first")), Bytes: 100}
	second := SharedTarget{SHA256: record.HashBytes([]byte("second")), Bytes: 40}
	var retained RetainedBytes
	var costs []uint64
	for _, added := range []struct {
		storedBytes uint64
		shared      SharedTarget
	}{
		{110, first},
		{125, first},
		{110, SharedTarget{}},
		{50, second},
		{45, second},
		{101, first},
	} {
		cost, err := retained.Add(added.storedBytes, added.shared)
		if err != nil {
			t.Fatal(err)
		}
		costs = append(costs, cost)
	}
	if want := []uint64{110, 25, 110, 50, 5, 1}; !reflect.DeepEqual(costs, want) || retained.Total() != 301 {
		t.Fatalf("costs = %v total = %d, want %v and 301", costs, retained.Total(), want)
	}
}

func TestRetainedBytesRejectsATargetThatIsNotPartOfTheArtifact(t *testing.T) {
	digest := record.HashBytes([]byte("target"))
	for name, test := range map[string]struct {
		storedBytes uint64
		shared      SharedTarget
		want        string
	}{
		"target as large as the artifact": {100, SharedTarget{SHA256: digest, Bytes: 100}, "not a part of its stored bytes"},
		"target without an identity":      {100, SharedTarget{Bytes: 10}, "not a part of its stored bytes"},
		"target without bytes":            {100, SharedTarget{SHA256: digest}, "not a part of its stored bytes"},
		"sum past uint64":                 {^uint64(0), SharedTarget{}, "overflows"},
	} {
		t.Run(name, func(t *testing.T) {
			retained := RetainedBytes{}
			if _, err := retained.Add(1, SharedTarget{}); err != nil {
				t.Fatal(err)
			}
			if _, err := retained.Add(test.storedBytes, test.shared); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Add() error = %v, want %q", err, test.want)
			}
			if retained.Total() != 1 {
				t.Fatalf("a rejected artifact changed the total to %d", retained.Total())
			}
		})
	}
}

func TestPoolTargetIsTheTargetOnlyOfAnArtifactLinkedToThePool(t *testing.T) {
	owner := t.TempDir()
	pool := TargetPool(owner)
	shared := publishSeed(t, Store{Root: filepath.Join(owner, "failures"), Key: StoreKeyRecord, TargetPool: pool}, 7)
	private := publishSeed(t, Store{Root: filepath.Join(owner, "private"), Key: StoreKeyRecord}, 7)
	if shared.StoredBytes != private.StoredBytes {
		t.Fatalf("stored bytes are %d with a shared target and %d with a private one, want them equal", shared.StoredBytes, private.StoredBytes)
	}
	want := SharedTarget{SHA256: shared.Manifest.Target.SHA256, Bytes: uint64(len("target bytes"))}
	if got := PoolTarget(pool, shared.Path, shared.Manifest); got != want {
		t.Fatalf("PoolTarget(linked artifact) = %#v, want %#v", got, want)
	}
	for name, got := range map[string]SharedTarget{
		"private copy beside the pool": PoolTarget(pool, private.Path, private.Manifest),
		"no pool":                      PoolTarget("", shared.Path, shared.Manifest),
		"another owner's pool":         PoolTarget(TargetPool(t.TempDir()), shared.Path, shared.Manifest),
	} {
		if got != (SharedTarget{}) {
			t.Fatalf("PoolTarget(%s) = %#v, want none", name, got)
		}
	}
}
