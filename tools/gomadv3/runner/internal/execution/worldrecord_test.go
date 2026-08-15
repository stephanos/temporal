package execution

import (
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/world"
	"go.temporal.io/server/tools/gomadv3/world/mailbox"
)

func TestComposePreservesWorldSemanticAndRawIdentities(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: worldLimits()})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	id, err := core.Register(world.Request{Kind: "wait", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.Ready(world.Readiness{RequestID: id, At: world.InitialTime + 1, Kind: "done"}); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	final := core.Snapshot()
	bundle, err := Compose(initial, final, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Manifest.Initial.Schema != "gomadv3.world.snapshot/v1" || bundle.Manifest.Transitions.Schema != "gomadv3.world.transitions/v1" || bundle.Manifest.Final.Schema != "gomadv3.world.snapshot/v1" {
		t.Fatalf("World schemas = %#v", bundle.Manifest)
	}
	if bundle.Manifest.Initial.SemanticDigest != evidence.SHA256("sha256:"+string(initial.StateDigest)) || bundle.Manifest.Final.SemanticDigest != evidence.SHA256("sha256:"+string(final.StateDigest)) {
		t.Fatalf("World semantic digests = %#v", bundle.Manifest)
	}
	if bundle.Manifest.Transitions.Count != 3 || len(bundle.Payloads.Transitions) == 0 || bundle.Payloads.Transitions[len(bundle.Payloads.Transitions)-1] != '\n' {
		t.Fatalf("World transition payload = count %d bytes %q", bundle.Manifest.Transitions.Count, bundle.Payloads.Transitions)
	}
	if _, err := world.DecodeSnapshot(bundle.Payloads.Initial); err != nil {
		t.Fatal(err)
	}
	if _, err := world.DecodeSnapshot(bundle.Payloads.Final); err != nil {
		t.Fatal(err)
	}
}

func TestComposeRecordsDerivedMailboxSchemaIdentity(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: worldLimits()})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	adapter, err := mailbox.New(core)
	if err != nil {
		t.Fatal(err)
	}
	requestID, err := adapter.Receive("inbox", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.MessageReady(requestID, world.InitialTime, []byte("same")); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Drive(); err != nil {
		t.Fatal(err)
	}
	bundle, err := Compose(initial, core.Snapshot(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Manifest.Adapters) != 1 || bundle.Manifest.Adapters[0].Schema != "gomadv3.world.adapter/mailbox/v1" {
		t.Fatalf("World adapters = %#v", bundle.Manifest.Adapters)
	}
	if _, _, err := Validate(bundle.Manifest, bundle.Payloads); err != nil {
		t.Fatal(err)
	}
}

func TestComposeRejectsTransitionLimitAndIncompatibleSnapshots(t *testing.T) {
	core, err := world.New(world.Config{Seed: 1, Limits: worldLimits()})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	final := core.Snapshot()
	if _, err := Compose(initial, final, 1); err == nil {
		t.Fatal("Compose() accepted undersized transition limit")
	}
	other, err := world.New(world.Config{Seed: 2, Limits: worldLimits()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compose(initial, other.Snapshot(), 1<<20); err == nil {
		t.Fatal("Compose() accepted different World config")
	}
}

func TestValidateRejectsSemanticAndTransitionDivergence(t *testing.T) {
	core, err := world.New(world.Config{Seed: 4, Limits: worldLimits()})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	bundle, err := Compose(initial, core.Snapshot(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Validate(bundle.Manifest, bundle.Payloads); err != nil {
		t.Fatal(err)
	}
	changedManifest := bundle.Manifest
	changedManifest.Final.SemanticDigest = evidence.HashBytes([]byte("changed"))
	if _, _, err := Validate(changedManifest, bundle.Payloads); err == nil {
		t.Fatal("Validate() accepted changed semantic digest")
	}
	changedPayloads := bundle.Payloads
	changedPayloads.Transitions = append([]byte(nil), bundle.Payloads.Transitions...)
	changedPayloads.Transitions[0] = 'X'
	if _, _, err := Validate(bundle.Manifest, changedPayloads); err == nil {
		t.Fatal("Validate() accepted changed transitions")
	}
}

func worldLimits() world.Limits {
	return world.Limits{MaxRequests: 100, MaxEvents: 100, MaxQueuedEvents: 100, MaxTransitions: 500, MaxPayloadBytes: 1 << 20, MaxStringBytes: 1024}
}
