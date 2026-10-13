package simulation

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	gomad3sim "go.temporal.io/server/tools/gomad3/internal/model/cluster"
	"go.temporal.io/server/tools/gomad3/internal/model/world"
)

func TestRejectedRestartLeavesNoUnownedBrokerSession(t *testing.T) {
	for _, test := range []struct {
		name        string
		incarnation uint64
		request     wasi.Request
	}{
		{"missing-storage-image", 2, wasi.Request{Config: wasi.Config{Storage: &wasi.StorageModel{}}}},
		{"stale-incarnation", 3, wasi.Request{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			b, err := wasi.NewBroker(wasi.BrokerConfig{Nodes: []wasi.BrokerNode{{ID: "a", Incarnation: 1}, {ID: "b", Incarnation: 1}}, Limits: world.CompressedSnapshotProfile().Limits})
			if err != nil {
				t.Fatal(err)
			}
			if err := b.Session("a").Finished(); err != nil {
				t.Fatal(err)
			}
			old := &node{handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 1}, done: make(chan struct{})}
			close(old.done)
			e := &execution{ctx: t.Context(), broker: b, provider: Provider{Boots: map[gomad3sim.BootID]wasi.Request{"boot": test.request}}, nodes: map[gomad3sim.NodeID]*node{"a": old}, results: map[gomad3sim.NodeHandle]wasi.Result{}, requests: map[gomad3sim.NodeHandle]wasi.Request{}, elapsed: map[gomad3sim.NodeHandle]uint64{}}
			if err := e.Start(gomad3sim.NodeSpec{ID: "a", Boot: "boot"}, gomad3sim.NodeHandle{Node: "a", Incarnation: test.incarnation}); err == nil {
				t.Fatal("invalid restart started")
			}
			if session := b.Session("a"); session.Node.Incarnation != 1 {
				t.Fatalf("rejected restart scheduled incarnation %d", session.Node.Incarnation)
			}
			if _, err := b.Restart("a"); err != nil {
				t.Fatalf("rejected restart consumed the broker lifecycle: %v", err)
			}
		})
	}
}
