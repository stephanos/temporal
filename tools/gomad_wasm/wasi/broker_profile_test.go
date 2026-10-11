package wasi

import (
	"errors"
	"testing"

	"go.temporal.io/server/tools/gomad3/world"
)

func TestBrokerCompressedWorldProfileAdmissionAndTerminalOwnership(t *testing.T) {
	config := BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, WorldProfile: world.CompressedSnapshotSchema, Limits: world.CompressedSnapshotProfile().Limits}
	oversize := config
	oversize.Limits.MaxRequests++
	if broker, err := NewBroker(oversize); !errors.Is(err, world.ErrCapacity) || broker != nil {
		t.Fatalf("oversize configured profile admitted: %v", err)
	}
	unknown := config
	unknown.WorldProfile = "unknown"
	if broker, err := NewBroker(unknown); err == nil || broker != nil {
		t.Fatal("unknown World profile admitted")
	}
	b, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.EncodeWorldEvidence(); err == nil {
		t.Fatal("live participant World was detached")
	}
	if err := b.Session("a").Finished(); err != nil {
		t.Fatal(err)
	}
	before := b.DiagnosticCardinality()
	data, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := world.DecodeCompressedSnapshot(data, world.CompressedSnapshotProfile()); err != nil {
		t.Fatal(err)
	}
	if b.DiagnosticCardinality() != before {
		t.Fatal("terminal ownership transfer changed diagnostic cardinality")
	}
	if _, err := b.EncodeWorldEvidence(); err == nil {
		t.Fatal("terminal World encoded twice")
	}
	if _, err := b.Restart("a"); err == nil {
		t.Fatal("terminal detached World restarted")
	}
}

func TestBrokerAbsentWorldProfilePreservesLegacyOwnerCodec(t *testing.T) {
	b, err := NewBroker(BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, Limits: world.CompressedSnapshotProfile().Limits})
	if err != nil {
		t.Fatal(err)
	}
	data, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := world.DecodeSnapshot(data); err != nil {
		t.Fatal(err)
	}
}
