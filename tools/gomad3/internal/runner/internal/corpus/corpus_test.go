package corpus

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/strictjson"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/logstore"
)

func TestCorpusPublishesCanonicalSnapshotOnlyAfterMatchingReplay(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	input, coverage, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	corpus, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
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

	reopened, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("Close() = %v", err)
		}
	}()
	if reopened.Snapshot().SnapshotSHA256 != snapshot.SnapshotSHA256 || len(reopened.Snapshot().Entries) != 1 {
		t.Fatalf("reopened snapshot = %#v", reopened.Snapshot())
	}
}

func TestCorpusRejectsIdentityChangesAndNonMatchingReplay(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	input, coverage, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	corpus, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
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
	if _, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{}); err == nil {
		t.Fatal("Open accepted a changed instrumentation identity")
	}
}

func TestCorpusRejectsFilesystemRootAndSymbolicLink(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	if _, err := Open(context.Background(), string(filepath.Separator), identity, corpusDescription(), corpusEvidenceProvider{}); err == nil {
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
	if _, err := Open(context.Background(), link, identity, corpusDescription(), corpusEvidenceProvider{}); err == nil {
		t.Fatal("Open accepted a symbolic-link corpus")
	}
}

func TestCorpusCleansUnreferencedCasesAndRejectsUnexpectedEntries(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	root := filepath.Join(t.TempDir(), "corpus")
	orphan := filepath.Join(root, "cases", "sha256-0123456789abcdef0123456789abcdef")
	if err := os.MkdirAll(orphan, 0o700); err != nil {
		t.Fatal(err)
	}
	corpus, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("unreferenced case remains: %v", err)
	}
	if err := corpus.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "cases", "unexpected"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{}); err == nil {
		t.Fatal("Open accepted an unexpected corpus entry")
	}
}

func TestCorpusAllowsOnlyOneWriter(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	root := filepath.Join(t.TempDir(), "corpus")
	first, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{}); !errors.Is(err, logstore.ErrInUse) || !strings.Contains(err.Error(), "guided corpus is already in use") {
		t.Fatalf("Open(concurrent writer) = %v, want an in-use refusal", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func guideArtifactInput(t *testing.T, seed uint64) (artifact.ArtifactInput, record.SemanticCoverage, []Feature) {
	t.Helper()
	targetBytes := []byte("guided target")
	targetPath := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(targetPath, targetBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	transcriptBytes, err := transcript.EncodeTranscript([]transcript.Operation{{Name: "os.open"}})
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := transcript.SummarizeSemanticProbes(nil)
	if err != nil {
		t.Fatal(err)
	}
	world, payloads := record.NoneWorld()
	profile := transcript.Default()
	exitCode := record.Uint64String(0)
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactSuccess, CreatedAt: "2026-08-13T00:00:00Z", CampaignID: "guided-test", SelectionOrdinal: 0, Seed: record.Uint64String(seed), ReplayMode: record.ReplayExact,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "runner", HostOS: "darwin", HostArch: "arm64"},
		Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target: record.Target{
			Kind: "go-run", Source: ".", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)), Argv: []string{"gomad3-target"}, BuildTags: []string{},
			Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
		},
		IOProfile: record.IOProfile{
			Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256()),
			Transcript: &record.IOTranscript{Schema: "gomad3.io-transcript/v1", File: "io/transcript.bin", SHA256: record.HashBytes(transcriptBytes), Bytes: record.Uint64String(len(transcriptBytes)), Records: 1},
		},
		Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: profile.Name()}, {Name: "GOMADSEED", Value: strconv.FormatUint(seed, 10)}, {Name: "TZ", Value: "UTC"}},
		Limits:      record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64, IOTranscriptBytes: 64 << 20},
		World:       world, Outcome: record.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode},
		Streams: record.Streams{Stdout: record.Stream{FullSHA256: record.HashBytes(nil)}, Stderr: record.Stream{FullSHA256: record.HashBytes(nil)}},
		Host:    record.Host{StartedAt: "2026-08-13T00:00:00Z", FinishedAt: "2026-08-13T00:00:01Z", ElapsedNanos: 1},
	}
	features, err := semanticFeatures(manifest, coverage, []backends.OperationOutcome{{Name: "os.open"}}, payloads.Transitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	return artifact.ArtifactInput{Manifest: manifest, TargetPath: targetPath, IOTranscript: transcriptBytes, World: payloads}, coverage, features
}

func guideIdentity(t *testing.T, manifest record.ExecutionRecord) Identity {
	t.Helper()
	identity, err := IdentityFor(manifest.Target, manifest.Toolchain, corpusDescription(), manifest.Environment)
	if err != nil {
		t.Fatal(err)
	}
	return identity
}

func TestCorpusRejectsCaseWithChangedEnvironment(t *testing.T) {
	input, coverage, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	corpus, err := Open(context.Background(), t.TempDir(), identity, corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := corpus.Close(); err != nil {
			t.Errorf("Close() = %v", err)
		}
	}()
	input.Manifest.Environment = append(input.Manifest.Environment, record.Environment{Name: "MODE", Value: "changed"})
	sort.Slice(input.Manifest.Environment, func(i, j int) bool { return input.Manifest.Environment[i].Name < input.Manifest.Environment[j].Name })
	replayed := false
	added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, func(context.Context, string) (ReplayResult, error) {
		replayed = true
		return ReplayResult{Verified: true, Match: true}, nil
	})
	if err == nil || added || !replayed || len(corpus.Snapshot().Entries) != 0 || !strings.Contains(err.Error(), "guided corpus case target identity mismatch") {
		t.Fatalf("changed case admission = %t, %v, replayed = %t", added, err, replayed)
	}
}

func TestCorpusCasesShareOneTarget(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	first, coverage, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, first.Manifest)
	corpus, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	second, _, _ := guideArtifactInput(t, 8)
	second.IOTranscript, err = transcript.EncodeTranscript([]transcript.Operation{{Name: "os.open"}, {Ordinal: 1, Name: "os.read"}})
	if err != nil {
		t.Fatal(err)
	}
	second.Manifest.IOProfile.Transcript.SHA256 = record.HashBytes(second.IOTranscript)
	second.Manifest.IOProfile.Transcript.Bytes = record.Uint64String(len(second.IOTranscript))
	second.Manifest.IOProfile.Transcript.Records = 2
	for _, input := range []artifact.ArtifactInput{first, second} {
		added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, func(context.Context, string) (ReplayResult, error) {
			return ReplayResult{Verified: true, Match: true}, nil
		})
		if err != nil || !added {
			t.Fatalf("Admit(seed %d) = %t, %v", input.Manifest.Seed, added, err)
		}
	}
	if err := corpus.Close(); err != nil {
		t.Fatal(err)
	}
	targets, err := filepath.Glob(filepath.Join(root, "cases", "*", "target"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := filepath.Glob(filepath.Join(artifact.TargetPool(root), "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 || len(entries) != 1 {
		t.Fatalf("corpus holds cases %v and pool entries %v, want two cases and one entry", targets, entries)
	}
	entry, err := os.Lstat(entries[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		info, err := os.Lstat(target)
		if err != nil {
			t.Fatal(err)
		}
		if !os.SameFile(entry, info) {
			t.Fatalf("corpus case target %s is a separate copy", target)
		}
	}
	reopened, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("Close() = %v", err)
		}
	}()
	if len(reopened.Snapshot().Entries) != 2 {
		t.Fatalf("reopened snapshot = %#v", reopened.Snapshot())
	}
}

func corpusTargets(t *testing.T, root string) []string {
	t.Helper()
	entries, err := filepath.Glob(filepath.Join(artifact.TargetPool(root), "*"))
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestCorpusKeepsItsTargetUntilNoCaseSharesIt(t *testing.T) {
	matching := func(context.Context, string) (ReplayResult, error) {
		return ReplayResult{Verified: true, Match: true}, nil
	}
	diverged := func(context.Context, string) (ReplayResult, error) {
		return ReplayResult{Verified: true, Divergence: "outcome"}, nil
	}
	root := filepath.Join(t.TempDir(), "corpus")
	input, coverage, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	corpus, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	if added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, diverged); err == nil || added {
		t.Fatalf("Admit(diverging replay) = %t, %v", added, err)
	}
	if targets := corpusTargets(t, root); len(targets) != 0 {
		t.Fatalf("the target of a discarded case stays in an empty corpus: %v", targets)
	}
	if added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, matching); err != nil || !added {
		t.Fatalf("Admit() = %t, %v", added, err)
	}
	duplicate, duplicateCoverage, _ := guideArtifactInput(t, 8)
	if added, err := corpus.Admit(context.Background(), Candidate{Artifact: duplicate, Coverage: duplicateCoverage}, matching); err != nil || added {
		t.Fatalf("Admit(duplicate) = %t, %v", added, err)
	}
	if err := corpus.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(artifact.TargetPool(root), ".publish-crashed"), 0o700); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("Close() = %v", err)
		}
	}()
	targets := corpusTargets(t, root)
	if len(targets) != 1 {
		t.Fatalf("corpus targets after a discarded case and a crashed publisher = %v, want the retained case's one", targets)
	}
	entry, err := os.Lstat(targets[0])
	if err != nil {
		t.Fatal(err)
	}
	retained, err := os.Lstat(filepath.Join(root, filepath.FromSlash(reopened.Snapshot().Entries[0].Artifact), "target"))
	if err != nil || !os.SameFile(entry, retained) {
		t.Fatalf("the retained case no longer shares the corpus target: %v", err)
	}
}

func TestCorpusByteCapCountsTheSharedTargetOnce(t *testing.T) {
	const targetBytes, caseBytes = 160 << 20, 170 << 20
	shared := artifact.SharedTarget{SHA256: record.HashBytes([]byte("target")), Bytes: targetBytes}
	for name, test := range map[string]struct {
		cases  int
		shared artifact.SharedTarget
		want   int
	}{
		// 6 x 170 MiB is the last sum under 1 GiB.
		"private copies": {cases: 200, want: 6},
		// 170 MiB for the first case, 10 MiB for each later one.
		"one shared target":        {cases: 200, shared: shared, want: 86},
		"fewer cases than the cap": {cases: 40, shared: shared, want: 40},
	} {
		t.Run(name, func(t *testing.T) {
			entries := make([]Entry, test.cases)
			for index := range entries {
				entries[index] = Entry{
					Seed: record.Uint64String(index), RecordHash: record.HashBytes([]byte(strconv.Itoa(index))), StoredBytes: caseBytes,
					Features: []Feature{{Kind: FeatureBoundaryProbe, Value: strconv.Itoa(index)}}, sharedTarget: test.shared,
				}
			}
			selected, err := boundedEntries(entries, defaultCapacity)
			if err != nil || len(selected) != test.want {
				t.Fatalf("boundedEntries() kept %d cases, %v, want %d", len(selected), err, test.want)
			}
		})
	}
}

func TestCorpusValidateEntryPreservesResultsAndErrorOrder(t *testing.T) {
	input, coverage, _ := guideArtifactInput(t, 7)
	corpus, err := Open(context.Background(), t.TempDir(), guideIdentity(t, input.Manifest), corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := corpus.Close(); err != nil {
			t.Errorf("Close() = %v", err)
		}
	}()
	if added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, func(context.Context, string) (ReplayResult, error) {
		return ReplayResult{Verified: true, Match: true}, nil
	}); err != nil || !added {
		t.Fatalf("Admit() = %t, %v", added, err)
	}
	before := corpus.Snapshot()
	entry := before.Entries[0]
	shared, err := corpus.validateEntry(entry)
	if err != nil || shared != entry.sharedTarget || shared.Bytes != 13 || shared.SHA256 != input.Manifest.Target.SHA256 {
		t.Fatalf("validateEntry() = %#v, %v", shared, err)
	}
	for _, test := range []struct {
		name   string
		change func(*Entry)
		want   string
	}{
		{name: "entry before coverage", change: func(e *Entry) { e.Replay.Match = false; e.Coverage.Schema = "bad" }, want: "guided corpus entry identity is invalid"},
		{name: "coverage before features", change: func(e *Entry) { e.Coverage.Schema = "bad"; e.Features = nil }, want: "guided corpus semantic coverage is invalid"},
		{name: "features before case", change: func(e *Entry) { e.Features = nil; e.Seed++ }, want: "guided corpus features or novelty reasons are invalid"},
		{name: "case before payload", change: func(e *Entry) { e.Seed++; e.PayloadBytes++ }, want: "guided corpus case identity does not match its entry"},
		{name: "payload before captured inputs", change: func(e *Entry) { e.PayloadBytes++; e.Inputs.IOTranscriptRecords++ }, want: "guided corpus payload size mismatch"},
		{name: "captured inputs before mounts", change: func(e *Entry) {
			e.Inputs.IOTranscriptRecords++
			digest := record.HashBytes(nil)
			e.Inputs.ReadOnlyMountsSHA256 = &digest
		}, want: "guided corpus captured input identity mismatch"},
		{name: "mounts", change: func(e *Entry) { digest := record.HashBytes(nil); e.Inputs.ReadOnlyMountsSHA256 = &digest }, want: "guided corpus read-only mount identity mismatch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := entry
			test.change(&changed)
			got, err := corpus.validateEntry(changed)
			if err == nil || err.Error() != test.want || got != (artifact.SharedTarget{}) {
				t.Fatalf("validateEntry() = %#v, %v, want zero result and %q", got, err, test.want)
			}
			if !reflect.DeepEqual(corpus.Snapshot(), before) {
				t.Fatal("validateEntry changed the in-memory snapshot")
			}
		})
	}
}

func TestCorpusValidationFailureDoesNotPublish(t *testing.T) {
	input, coverage, features := guideArtifactInput(t, 7)
	corpus, err := Open(context.Background(), t.TempDir(), guideIdentity(t, input.Manifest), corpusDescription(), corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := corpus.Close(); err != nil {
			t.Errorf("Close() = %v", err)
		}
	}()
	before := corpus.Snapshot()
	published, err := artifact.PublishArtifact(artifact.Store{Root: corpus.casesPath(), Context: context.Background(), MaximumBytes: maximumBytes, Key: artifact.StoreKeyRecord, TargetPool: corpus.targetPool()}, input)
	if err != nil {
		t.Fatal(err)
	}
	published.Manifest.Seed++
	added, err := corpus.merge(published, coverage, features, ReplayResult{Verified: true, Match: true})
	if added || err == nil || err.Error() != "guided corpus case identity does not match its entry" || !reflect.DeepEqual(corpus.Snapshot(), before) {
		t.Fatalf("merge(invalid case) = %t, %v, snapshot = %#v", added, err, corpus.Snapshot())
	}
	if history := corpus.History(); history.Records != 1 {
		t.Fatalf("failed validation committed an admission: %#v", history)
	}
}

func TestCorpusSnapshotSemanticRoundTripAndRepeatability(t *testing.T) {
	input, coverage, _ := guideArtifactInput(t, 7)
	description := corpusDescription()
	const retainedBoundary = record.SHA256("sha256:ca18b6934d906b95235e04f83dfc2eef0a94d17d5417eb032cd086f7425ebbd0")
	description.IOProfile.Inventory = strings.ReplaceAll(description.IOProfile.Inventory, string(description.BoundarySHA256), string(retainedBoundary))
	description.IOProfile.InventorySHA256 = record.HashBytes([]byte(description.IOProfile.Inventory))
	description.IOProfile.ImplementationSHA256 = record.HashBytes([]byte("gomad3.deterministic-io/v1/implementation-v14\x00" + string(description.IOProfile.InventorySHA256)))
	description.BoundarySHA256 = retainedBoundary
	input.Manifest.IOProfile.Name = description.IOProfile.Name
	input.Manifest.IOProfile.Inventory = description.IOProfile.Inventory
	input.Manifest.IOProfile.InventorySHA256 = description.IOProfile.InventorySHA256
	input.Manifest.IOProfile.ImplementationSHA256 = description.IOProfile.ImplementationSHA256
	identity, err := IdentityFor(input.Manifest.Target, input.Manifest.Toolchain, description, input.Manifest.Environment)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	corpus, err := Open(context.Background(), root, identity, description, corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	if added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, func(context.Context, string) (ReplayResult, error) {
		return ReplayResult{Verified: true, Match: true}, nil
	}); err != nil || !added {
		t.Fatalf("Admit() = %t, %v", added, err)
	}
	// The snapshot projection, not a file, carries the identity a campaign
	// freezes; its stored bytes keep the artifact capacity accounting.
	finalized, encoded, err := finalizeSnapshot(corpus.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	var decoded Snapshot
	if err := strictjson.Decode(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	repeated, repeatedBytes, err := finalizeSnapshot(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repeated, decoded) || repeated.SnapshotSHA256 != finalized.SnapshotSHA256 || string(repeatedBytes) != string(encoded) {
		t.Fatalf("snapshot round trip changed: %#v, %s", repeated, repeatedBytes)
	}
	committed := corpus.snapshot
	if err := corpus.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), root, identity, description, corpusEvidenceProvider{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("Close() = %v", err)
		}
	}()
	if !reflect.DeepEqual(reopened.snapshot, committed) {
		t.Fatalf("replayed snapshot = %#v, want %#v", reopened.snapshot, committed)
	}
}

func TestFinalizeSnapshotRejectsInvalidOriginalStrings(t *testing.T) {
	snapshot := Snapshot{Schema: CorpusSchema, Entries: []Entry{{Features: []Feature{{Kind: string([]byte{0xff}), Value: "value"}}}}}
	if _, _, err := finalizeSnapshot(snapshot); err == nil {
		t.Fatal("finalizeSnapshot() accepted invalid UTF-8")
	}
}
