package simulation

import (
	"encoding/json"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
)

func TestCompressedWorldEvidenceUsesSharedOwnerPreflight(t *testing.T) {
	config := world.Config{Seed: 7, Limits: world.CompressedSnapshotProfile().Limits}
	b, err := wasi.NewBroker(wasi.BrokerConfig{Nodes: []wasi.BrokerNode{{ID: "a", Incarnation: 1}}, Seed: 7, Limits: config.Limits, WorldProfile: world.CompressedSnapshotSchema})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Session("a").Finished(); err != nil {
		t.Fatal(err)
	}
	e := &execution{broker: b, provider: Provider{Broker: wasi.BrokerConfig{WorldProfile: world.CompressedSnapshotSchema}}, limits: gomad3sim.DefaultLimits()}
	evidence, err := e.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyWorldEvidence(evidence.Execution, world.CompressedSnapshotSchema, config); err != nil {
		t.Fatal(err)
	}
	changed := config
	changed.Seed++
	if err := verifyWorldEvidence(evidence.Execution, world.CompressedSnapshotSchema, changed); err == nil {
		t.Fatal("World seed mismatch admitted")
	}
	changed = config
	changed.Limits.MaxRequests--
	if err := verifyWorldEvidence(evidence.Execution, world.CompressedSnapshotSchema, changed); err == nil {
		t.Fatal("World capacity mismatch admitted")
	}
	if err := verifyWorldEvidence(evidence.Execution, "", config); err == nil {
		t.Fatal("profile discriminator mismatch admitted")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(evidence.Execution.World, &raw); err != nil {
		t.Fatal(err)
	}
	raw["unknown"] = json.RawMessage("true")
	changedWorld, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	evidence.Execution.World = changedWorld
	if err := verifyWorldEvidence(evidence.Execution, world.CompressedSnapshotSchema, config); err == nil {
		t.Fatal("owner-invalid compressed evidence admitted")
	}
}

func TestExplicitWorldProfileRequiresExplicitCapacities(t *testing.T) {
	p := &Provider{Broker: wasi.BrokerConfig{WorldProfile: world.CompressedSnapshotSchema}}
	_, err := p.Open(t.Context(), gomad3sim.Spec{Backend: gomad3sim.BackendWASM})
	if err == nil || !strings.Contains(err.Error(), "explicit configured capacities") {
		t.Fatalf("profile defaulted capacities: %v", err)
	}
}
