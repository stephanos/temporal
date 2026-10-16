package simulation

import (
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/world"
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
			old := &node{handle: gomadsim.NodeHandle{Node: "a", Incarnation: 1}, done: make(chan struct{})}
			close(old.done)
			e := &execution{ctx: t.Context(), broker: b, provider: Provider{Boots: map[gomadsim.BootID]wasi.Request{"boot": test.request}}, nodes: map[gomadsim.NodeID]*node{"a": old}, results: map[gomadsim.NodeHandle]wasi.Result{}, requests: map[gomadsim.NodeHandle]wasi.Request{}, elapsed: map[gomadsim.NodeHandle]uint64{}}
			if err := e.Start(gomadsim.NodeSpec{ID: "a", Boot: "boot"}, gomadsim.NodeHandle{Node: "a", Incarnation: test.incarnation}); err == nil {
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
