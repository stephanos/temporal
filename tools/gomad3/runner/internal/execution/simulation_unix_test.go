//go:build unix

package execution

import "testing"

func TestSimulationCoordinatorKeepsCompletedNodeResponseDeliverable(t *testing.T) {
	coordinator, err := newSimulationCoordinator(Spec{})
	if err != nil {
		t.Fatal(err)
	}
	nodeTime, err := coordinator.time.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	close(done)
	node := &simulationNodeProcess{node: "node", incarnation: 1, done: done, time: nodeTime}
	coordinator.time.remove(nodeTime)

	if err := coordinator.beginNodeResponseBarrier(1, 0, node); err != nil {
		t.Fatal(err)
	}
	coordinator.handleCoordinatorDelivery(simulationFrame{Request: 1})
	if err := coordinator.time.acknowledgeExternal(coordinator.coordinator, 1); err != nil {
		t.Fatal(err)
	}
}

func TestSimulationCoordinatorStaleRemovalPreservesReplacementNode(t *testing.T) {
	coordinator, err := newSimulationCoordinator(Spec{})
	if err != nil {
		t.Fatal(err)
	}
	oldTime, err := coordinator.time.register("server/1")
	if err != nil {
		t.Fatal(err)
	}
	oldNode := &simulationNodeProcess{node: "server", incarnation: 1, time: oldTime}
	coordinator.nodes["server/1"] = oldNode
	coordinator.time.remove(oldTime)
	replacementTime, err := coordinator.time.register("server/1")
	if err != nil {
		t.Fatal(err)
	}
	replacement := &simulationNodeProcess{node: "server", incarnation: 1, time: replacementTime}
	coordinator.nodes["server/1"] = replacement

	coordinator.removeNode(oldNode)

	if actual := coordinator.nodes["server/1"]; actual != replacement {
		t.Fatalf("registered node = %p, want replacement %p", actual, replacement)
	}
}
