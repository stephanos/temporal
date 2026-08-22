package corpus

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"go.temporal.io/server/tools/gomadv3/artifact"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/record"
)

func TestCorpusPublishesCanonicalSnapshotOnlyAfterMatchingReplay(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	input, coverage, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	corpus, err := Open(context.Background(), root, identity)
	if err != nil {
		t.Fatal(err)
	}
	added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, func(_ context.Context, path string) (ReplayResult, error) {
		if path == "" {
			t.Fatal("replay path is empty")
		}
		return ReplayResult{Verified: true, Match: true}, nil
	})
	if err != nil || !added {
		t.Fatalf("Admit() = %t, %v", added, err)
	}
	snapshot := corpus.Snapshot()
	if len(snapshot.Entries) != 1 || snapshot.Entries[0].Seed != 7 || snapshot.Entries[0].Replay != (ReplayResult{Verified: true, Match: true}) || len(snapshot.Entries[0].NoveltyReasons) == 0 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	duplicate, duplicateCoverage, _ := guideArtifactInput(t, 8)
	replayed := false
	added, err = corpus.Admit(context.Background(), Candidate{Artifact: duplicate, Coverage: duplicateCoverage}, func(context.Context, string) (ReplayResult, error) {
		replayed = true
		return ReplayResult{Verified: true, Match: true}, nil
	})
	if err != nil || added || replayed || len(corpus.Snapshot().Entries) != 1 {
		t.Fatalf("Admit(duplicate) = %t, %v, replayed = %t, snapshot = %#v", added, err, replayed, corpus.Snapshot())
	}
	if err := corpus.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(context.Background(), root, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if reopened.Snapshot().SnapshotSHA256 != snapshot.SnapshotSHA256 || len(reopened.Snapshot().Entries) != 1 {
		t.Fatalf("reopened snapshot = %#v", reopened.Snapshot())
	}
}

func TestCorpusRejectsIdentityChangesAndNonMatchingReplay(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	input, coverage, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	corpus, err := Open(context.Background(), root, identity)
	if err != nil {
		t.Fatal(err)
	}
	if added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, func(context.Context, string) (ReplayResult, error) {
		return ReplayResult{Verified: true, Match: false, Divergence: "stdout"}, nil
	}); err == nil || added {
		t.Fatalf("Admit(non-match) = %t, %v", added, err)
	}
	if len(corpus.Snapshot().Entries) != 0 {
		t.Fatal("non-matching replay changed corpus coverage")
	}
	caseEntries, err := os.ReadDir(filepath.Join(root, "cases"))
	if err != nil || len(caseEntries) != 0 {
		t.Fatalf("divergent case cleanup = %v, %v", caseEntries, err)
	}
	if added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, func(context.Context, string) (ReplayResult, error) {
		return ReplayResult{Verified: true, Match: true}, nil
	}); err != nil || !added {
		t.Fatalf("Admit(match) = %t, %v", added, err)
	}
	if err := corpus.Close(); err != nil {
		t.Fatal(err)
	}

	identity.InstrumentationSHA256 = record.HashBytes([]byte("changed"))
	if _, err := Open(context.Background(), root, identity); err == nil {
		t.Fatal("Open accepted a changed instrumentation identity")
	}
}

func TestCorpusRejectsFilesystemRootAndSymbolicLink(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	if _, err := Open(context.Background(), string(filepath.Separator), identity); err == nil {
		t.Fatal("Open accepted the filesystem root")
	}
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), link, identity); err == nil {
		t.Fatal("Open accepted a symbolic-link corpus")
	}
}

func TestCorpusCleansUnreferencedCasesAndRejectsUnexpectedEntries(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	root := filepath.Join(t.TempDir(), "corpus")
	orphan := filepath.Join(root, "cases", "sha256-orphan")
	if err := os.MkdirAll(orphan, 0o700); err != nil {
		t.Fatal(err)
	}
	corpus, err := Open(context.Background(), root, identity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("unreferenced case remains: %v", err)
	}
	if err := corpus.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cases", "unexpected"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), root, identity); err == nil {
		t.Fatal("Open accepted an unexpected corpus entry")
	}
}

func TestCorpusAllowsOnlyOneWriter(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	root := filepath.Join(t.TempDir(), "corpus")
	first, err := Open(context.Background(), root, identity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), root, identity); err == nil {
		t.Fatal("Open accepted a concurrent writer")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), root, identity)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCorpusRejectsEntryCapacityOverflowBeforeOpeningCases(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	root := filepath.Join(t.TempDir(), "corpus")
	if err := os.MkdirAll(filepath.Join(root, "cases"), 0o700); err != nil {
		t.Fatal(err)
	}
	snapshot, encoded, err := finalizeSnapshot(Snapshot{
		Schema: CorpusSchema, Identity: identity, Entries: make([]Entry, maximumEntries+1),
	})
	if err != nil || snapshot.SnapshotSHA256 == "" {
		t.Fatalf("finalizeSnapshot() = %#v, %v", snapshot, err)
	}
	if err := os.WriteFile(filepath.Join(root, "corpus.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), root, identity); err == nil {
		t.Fatal("Open accepted too many corpus entries")
	}
}

func guideArtifactInput(t *testing.T, seed uint64) (artifact.ArtifactInput, deterministicio.SemanticCoverage, []Feature) {
	t.Helper()
	targetBytes := []byte("guided target")
	targetPath := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(targetPath, targetBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	transcript, err := deterministicio.EncodeTranscript([]deterministicio.Operation{{Name: "os.open"}})
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := deterministicio.SummarizeSemanticProbes(nil)
	if err != nil {
		t.Fatal(err)
	}
	world, payloads := record.NoneWorld()
	profile := deterministicio.Default()
	exitCode := record.Uint64String(0)
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactSuccess, CreatedAt: "2026-08-13T00:00:00Z", CampaignID: "guided-test", SelectionOrdinal: 0, Seed: record.Uint64String(seed), ReplayMode: record.ReplayExact,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "runner", HostOS: "darwin", HostArch: "arm64"},
		Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target: record.Target{
			Kind: "go-run", Source: ".", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)), Argv: []string{"gomadv3-target"}, BuildTags: []string{},
			Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
		},
		IOProfile: record.IOProfile{
			Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256()),
			Transcript: &record.IOTranscript{Schema: "gomadv3.io-transcript/v1", File: "io/transcript.bin", SHA256: record.HashBytes(transcript), Bytes: record.Uint64String(len(transcript)), Records: 1},
		},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: strconv.FormatUint(seed, 10)}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
		Limits:      record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64, IOTranscriptBytes: 64 << 20},
		World:       world, Outcome: record.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode},
		Streams: record.Streams{Stdout: record.Stream{FullSHA256: record.HashBytes(nil)}, Stderr: record.Stream{FullSHA256: record.HashBytes(nil)}},
		Host:    record.Host{StartedAt: "2026-08-13T00:00:00Z", FinishedAt: "2026-08-13T00:00:01Z", ElapsedNanos: 1},
	}
	features, err := semanticFeatures(manifest, coverage, transcript, payloads.Transitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	return artifact.ArtifactInput{Manifest: manifest, TargetPath: targetPath, IOTranscript: transcript, World: payloads}, coverage, features
}

func guideIdentity(t *testing.T, manifest record.ExecutionRecord) Identity {
	t.Helper()
	version, boundary := deterministicio.BoundaryManifestIdentity()
	identity, err := IdentityFor(manifest.Target, manifest.Toolchain, version, record.SHA256(boundary))
	if err != nil {
		t.Fatal(err)
	}
	return identity
}
