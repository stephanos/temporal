package guide

import (
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/iowire"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/worldrecord"
	"go.temporal.io/server/tools/gomadv3/world"
)

func TestIdentityForTargetIgnoresDiagnosticSourceButBindsExecutionArguments(t *testing.T) {
	target := record.Target{
		Kind: "go-run", Source: "/first/workspace", SHA256: record.HashBytes([]byte("target")), Size: 6,
		Argv: []string{"target", "first"}, BuildTags: []string{}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{},
		BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
	}
	toolchain := record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"}
	first, err := IdentityFor(target, toolchain, "boundary-v1", record.HashBytes([]byte("boundary")))
	if err != nil {
		t.Fatal(err)
	}
	target.Source = "/another/workspace"
	second, err := IdentityFor(target, toolchain, "boundary-v1", record.HashBytes([]byte("boundary")))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("diagnostic source changed identity: %#v, %#v", first, second)
	}
	target.Argv[1] = "second"
	third, err := IdentityFor(target, toolchain, "boundary-v1", record.HashBytes([]byte("boundary")))
	if err != nil {
		t.Fatal(err)
	}
	if third.TargetSHA256 == first.TargetSHA256 {
		t.Fatal("target arguments did not change identity")
	}
}

func TestIdentityForBindsFeatureAndProbeInstrumentation(t *testing.T) {
	target := record.Target{
		Kind: "go-run", SHA256: record.HashBytes([]byte("target")), Size: 6, Argv: []string{"target"}, BuildTags: []string{},
		Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
	}
	toolchain := record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"}
	identity, err := IdentityFor(target, toolchain, "boundary-v1", record.HashBytes([]byte("boundary")))
	if err != nil {
		t.Fatal(err)
	}
	if identity.InstrumentationSchema != SemanticFeatureSchema || identity.InstrumentationSHA256 == ioprofile.SemanticInstrumentationIdentity() {
		t.Fatalf("guided instrumentation identity = %#v", identity)
	}
}

func TestSemanticFeaturesUseStableOutcomesAndOperationPairs(t *testing.T) {
	transcript := append(transcriptRecord(t, 0, "os.open", 0), transcriptRecord(t, 1, "os.read", 3)...)
	coverage, err := ioprofile.SummarizeSemanticProbes([]string{"stdlib.os.openfile"})
	if err != nil {
		t.Fatal(err)
	}
	manifest := record.Manifest{
		Outcome: record.Outcome{Domain: "target", Reason: "world_deadlock", Termination: "exit", FailureSignature: record.HashBytes([]byte("failure"))},
		World: record.World{
			Initial:     record.WorldPayload{Schema: "gomadv3.world.snapshot/v1", SemanticDigest: record.HashBytes([]byte("initial"))},
			Transitions: record.WorldTransitions{Schema: "gomadv3.world.transitions/v1"},
			Final:       record.WorldPayload{Schema: "gomadv3.world.snapshot/v1", SemanticDigest: record.HashBytes([]byte("world"))},
			Terminal:    record.WorldTerminal{Kind: "deadlock", Detail: "mailbox is empty"},
		},
	}
	features, err := SemanticFeatures(manifest, coverage, transcript, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []Feature{
		{Kind: FeatureFailure, Value: string(manifest.Outcome.FailureSignature)},
		{Kind: FeatureInvariant, Value: "target/world_deadlock"},
		{Kind: FeatureTerminal, Value: "deadlock/sha256:0cd0e400d1c7a939d03a9b0c5b9b2f2091f4e27942141c3def892dff8042eb72"},
		{Kind: FeatureOutcome, Value: "target/world_deadlock/exit"},
		{Kind: FeatureWorld, Value: "state/gomadv3.world.snapshot/v1/changed"},
		{Kind: FeatureIOOutcome, Value: "os.open/0"},
		{Kind: FeatureIOOutcome, Value: "os.read/3"},
		{Kind: FeatureOperationPair, Value: "os.open/0->os.read/3"},
		{Kind: FeatureBoundaryProbe, Value: "stdlib.os.openfile"},
	}
	if !slices.Equal(features, want) {
		t.Fatalf("features = %#v, want %#v", features, want)
	}
}

func TestSemanticFeaturesSummarizeWorldTransitionsWithoutSeedOrPayloadIdentity(t *testing.T) {
	coverage, err := ioprofile.SummarizeSemanticProbes(nil)
	if err != nil {
		t.Fatal(err)
	}
	firstWorld, firstTransitions := semanticWorld(t, 7, "first-key", []byte("first-payload"), world.InitialTime)
	secondWorld, secondTransitions := semanticWorld(t, 99, "second-key", []byte("second-payload"), world.InitialTime+100)
	manifest := record.Manifest{Outcome: record.Outcome{Domain: "success", Reason: "success", Termination: "exit"}, World: firstWorld}
	first, err := SemanticFeatures(manifest, coverage, nil, firstTransitions)
	if err != nil {
		t.Fatal(err)
	}
	manifest.World = secondWorld
	second, err := SemanticFeatures(manifest, coverage, nil, secondTransitions)
	if err != nil {
		t.Fatal(err)
	}
	want := []Feature{
		{Kind: FeatureTerminal, Value: "delivered/sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{Kind: FeatureOutcome, Value: "success/success/exit"},
		{Kind: FeatureWorld, Value: "state/gomadv3.world.snapshot/v1/changed"},
		{Kind: FeatureWorld, Value: "transition/quiesce/delivered/deliveries=1/blocked=0"},
		{Kind: FeatureWorld, Value: "transition/ready/done"},
		{Kind: FeatureWorld, Value: "transition/register/memory/cell/wait"},
		{Kind: FeatureOperationPair, Value: "world.ready/done->world.quiesce/delivered/deliveries=1/blocked=0"},
		{Kind: FeatureOperationPair, Value: "world.register/memory/cell/wait->world.ready/done"},
	}
	if !slices.Equal(first, want) || !slices.Equal(second, want) {
		t.Fatalf("SemanticFeatures() = %#v and %#v, want %#v", first, second, want)
	}
}

func TestSnapshotPrioritizesReproducibleFailuresThenRareDomains(t *testing.T) {
	snapshot := Snapshot{Entries: []Entry{
		{Seed: 1, RecordHash: record.HashBytes([]byte("one")), StoredBytes: 20, Features: []Feature{{Kind: FeatureOutcome, Value: "success"}, {Kind: FeatureBoundaryProbe, Value: "common"}}},
		{Seed: 2, RecordHash: record.HashBytes([]byte("two")), StoredBytes: 20, Features: []Feature{{Kind: FeatureOutcome, Value: "success"}, {Kind: FeatureBoundaryProbe, Value: "rare"}}},
		{Seed: 3, RecordHash: record.HashBytes([]byte("three")), StoredBytes: 20, Features: []Feature{{Kind: FeatureOutcome, Value: "success"}, {Kind: FeatureBoundaryProbe, Value: "common"}}},
		{Seed: 4, RecordHash: record.HashBytes([]byte("four")), StoredBytes: 30, Features: []Feature{{Kind: FeatureFailure, Value: "failure"}}},
	}}
	if got, want := snapshot.PrioritizedSeeds(), []uint64{4, 2, 1, 3}; !slices.Equal(got, want) {
		t.Fatalf("PrioritizedSeeds() = %v, want %v", got, want)
	}
}

func transcriptRecord(t *testing.T, ordinal uint64, operation string, result uint32) []byte {
	t.Helper()
	encoded, err := iowire.EncodeTranscriptRecord(iowire.TranscriptRecord{Ordinal: ordinal, Operation: operation, Result: result})
	if err != nil {
		t.Fatal(err)
	}
	return encoded[:]
}

func semanticWorld(t *testing.T, seed uint64, key string, payload []byte, readyAt world.LogicalTime) (record.World, []byte) {
	t.Helper()
	limits := world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 20, MaxPayloadBytes: 1 << 20, MaxStringBytes: 1024}
	model, err := world.New(world.Config{Seed: world.Seed(seed), Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := model.StartRecording(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	requestID, err := model.Register(world.Request{Kind: "wait", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: key}, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := model.Ready(world.Readiness{RequestID: requestID, At: readyAt, Kind: "done", Payload: payload}); err != nil {
		t.Fatal(err)
	}
	if _, err := model.Quiesce(); err != nil {
		t.Fatal(err)
	}
	recording, err := recorder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := worldrecord.ComposeRecording(recording, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return bundle.Manifest, bundle.Payloads.Transitions
}
