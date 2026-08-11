package mailbox

import (
	"errors"
	"fmt"
	"testing"

	"go.temporal.io/server/tools/gomadv3/world"
)

func TestMailboxPilotPreservesFIFOForDistinctReceivers(t *testing.T) {
	for seed := world.Seed(0); seed < 32; seed++ {
		first := mailboxOrder(t, seed)
		second := mailboxOrder(t, seed)
		if first != "123" || second != first {
			t.Fatalf("seed %d order = %q then %q, want FIFO", seed, first, second)
		}
	}
}

func TestMailboxPilotCancellationDeadlockSnapshotAndReplay(t *testing.T) {
	core := newCore(t, 9)
	adapter, err := New(core)
	if err != nil {
		t.Fatal(err)
	}
	initialCore := core.Snapshot()
	first, err := adapter.Receive("inbox", 0, []byte("waiter"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Receive("inbox", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Cancel(first); err != nil {
		t.Fatal(err)
	}
	deadlock, err := adapter.Drive()
	if err != nil {
		t.Fatal(err)
	}
	if deadlock.Kind != world.QuiescenceDeadlock || len(deadlock.Blocked) != 1 || deadlock.Blocked[0] != second {
		t.Fatalf("deadlock = %#v", deadlock)
	}
	if _, err := adapter.MessageReady(second, world.InitialTime+2, []byte("message")); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Drive(); err != nil {
		t.Fatal(err)
	}
	adapterSnapshot, err := adapter.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(adapterSnapshot.Receives) != 2 || adapterSnapshot.Digest == "" {
		t.Fatalf("adapter snapshot = %#v", adapterSnapshot)
	}
	finalCore := core.Snapshot()
	restoredCore, err := world.Restore(finalCore, nil)
	if err != nil {
		t.Fatal(err)
	}
	restoredAdapter, err := Restore(restoredCore, adapterSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	restoredSnapshot, err := restoredAdapter.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if restoredSnapshot.Digest != adapterSnapshot.Digest {
		t.Fatal("restored mailbox snapshot changed")
	}
	corruptSnapshot := adapterSnapshot
	corruptSnapshot.Digest = world.Digest("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if _, err := Restore(restoredCore, corruptSnapshot); err == nil {
		t.Fatal("Restore() accepted corrupt mailbox snapshot")
	}
	plan := world.ReplayPlan{SchemaVersion: world.SchemaVersion, InitialDigest: initialCore.StateDigest, Transitions: finalCore.Transitions, FinalDigest: finalCore.StateDigest}
	replayCore, err := world.Restore(initialCore, &plan)
	if err != nil {
		t.Fatal(err)
	}
	replayAdapter, err := New(replayCore)
	if err != nil {
		t.Fatal(err)
	}
	replayFirst, err := replayAdapter.Receive("inbox", 0, []byte("waiter"))
	if err != nil {
		t.Fatal(err)
	}
	replaySecond, err := replayAdapter.Receive("inbox", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := replayAdapter.Cancel(replayFirst); err != nil {
		t.Fatal(err)
	}
	if _, err := replayAdapter.Drive(); err != nil {
		t.Fatal(err)
	}
	if _, err := replayAdapter.MessageReady(replaySecond, world.InitialTime+2, []byte("message")); err != nil {
		t.Fatal(err)
	}
	if _, err := replayAdapter.Drive(); err != nil {
		t.Fatal(err)
	}
	if progress := replayCore.ReplayProgress(); progress.Cursor != progress.Expected {
		t.Fatalf("replay progress = %#v", progress)
	}
}

func TestMailboxPilotSurfacesWorldCapacityWithoutFallback(t *testing.T) {
	limits := limits()
	limits.MaxRequests = 1
	core, err := world.New(world.Config{Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := New(core)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Receive("inbox", 0, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Receive("inbox", 0, nil); !errors.Is(err, world.ErrCapacity) {
		t.Fatalf("capacity error = %v", err)
	}
}

func TestMailboxRestoreRejectsOmittedOwnedWorldRequest(t *testing.T) {
	core := newCore(t, 1)
	adapter, err := New(core)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := adapter.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.Register(world.Request{Kind: "receive", Resource: world.ResourceID{Adapter: "mailbox", Kind: "queue", Key: "inbox"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Restore(core, snapshot); err == nil {
		t.Fatal("Restore() accepted an omitted mailbox-owned World request")
	}
}

func mailboxOrder(t *testing.T, seed world.Seed) string {
	t.Helper()
	core := newCore(t, seed)
	adapter, err := New(core)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 3; index++ {
		id, err := adapter.Receive("inbox", 0, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := adapter.MessageReady(id, world.InitialTime, []byte("message")); err != nil {
			t.Fatal(err)
		}
	}
	delivery, err := adapter.Drive()
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%d%d%d", delivery.Deliveries[0].RequestID, delivery.Deliveries[1].RequestID, delivery.Deliveries[2].RequestID)
}

func newCore(t *testing.T, seed world.Seed) *world.World {
	t.Helper()
	core, err := world.New(world.Config{Seed: seed, Limits: limits()})
	if err != nil {
		t.Fatal(err)
	}
	return core
}

func limits() world.Limits {
	return world.Limits{MaxRequests: 100, MaxEvents: 100, MaxQueuedEvents: 100, MaxTransitions: 500, MaxPayloadBytes: 1 << 20, MaxStringBytes: 1024}
}
