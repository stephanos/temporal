package corpus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
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
			Kind: "go-run", Source: ".", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)), Argv: []string{"gomad3-target"}, BuildTags: []string{},
			Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
		},
		IOProfile: record.IOProfile{
			Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256()),
			Transcript: &record.IOTranscript{Schema: "gomad3.io-transcript/v1", File: "io/transcript.bin", SHA256: record.HashBytes(transcript), Bytes: record.Uint64String(len(transcript)), Records: 1},
		},
		Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: profile.Name()}, {Name: "GOMADSEED", Value: strconv.FormatUint(seed, 10)}, {Name: "TZ", Value: "UTC"}},
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
	identity, err := IdentityFor(manifest.Target, manifest.Toolchain, version, record.SHA256(boundary), manifest.Environment)
	if err != nil {
		t.Fatal(err)
	}
	return identity
}

func TestCorpusRejectsPreviousAndFutureSchemaBeforeChangedIdentity(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	for _, schema := range []string{"gomad3.guide-corpus/v1", "gomad3.guide-corpus/v999"} {
		t.Run(schema, func(t *testing.T) {
			root := t.TempDir()
			snapshot, _, err := finalizeSnapshot(Snapshot{Schema: CorpusSchema, Identity: identity, Entries: []Entry{}})
			if err != nil {
				t.Fatal(err)
			}
			snapshot.Schema = schema
			snapshot.Identity.TargetSHA256 = record.HashBytes([]byte("another target"))
			encoded, err := canonicaljson.CanonicalJSON(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]any
			if err := json.Unmarshal(encoded, &wire); err != nil {
				t.Fatal(err)
			}
			if schema == "gomad3.guide-corpus/v1" {
				delete(wire["identity"].(map[string]any), "environment_sha256")
			} else {
				wire["future_field"] = true
			}
			encoded, err = canonicaljson.CanonicalJSON(wire)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "corpus.json"), encoded, 0o600); err != nil {
				t.Fatal(err)
			}
			opened, err := Open(context.Background(), root, identity)
			if opened != nil {
				if closeErr := opened.Close(); closeErr != nil {
					t.Fatal(closeErr)
				}
			}
			if err == nil || !strings.Contains(err.Error(), "guided corpus schema is invalid") {
				t.Fatalf("schema rejection = %v", err)
			}
		})
	}
}

func TestCorpusRejectsCaseWithChangedEnvironment(t *testing.T) {
	input, coverage, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	corpus, err := Open(context.Background(), t.TempDir(), identity)
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

func TestCorpusRejectsMalformedAndNoncanonicalCurrentSchema(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	_, encoded, err := finalizeSnapshot(Snapshot{Schema: CorpusSchema, Identity: identity, Entries: []Entry{}})
	if err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string][]byte{
		"malformed":        []byte(`{"schema":`),
		"noncanonical":     append([]byte(" "), encoded...),
		"unknown field":    append([]byte(`{"future_field":true,`), encoded[1:]...),
		"duplicate schema": []byte(strings.Replace(string(encoded), `"schema":`, `"schema":"`+CorpusSchema+`","schema":`, 1)),
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "corpus.json"), contents, 0o600); err != nil {
				t.Fatal(err)
			}
			opened, err := Open(context.Background(), root, identity)
			if opened != nil {
				if closeErr := opened.Close(); closeErr != nil {
					t.Fatal(closeErr)
				}
			}
			if err == nil {
				t.Fatal("Open accepted invalid current-schema snapshot")
			}
		})
	}
}

func TestCorpusCasesShareOneTarget(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	first, coverage, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, first.Manifest)
	corpus, err := Open(context.Background(), root, identity)
	if err != nil {
		t.Fatal(err)
	}
	second, _, _ := guideArtifactInput(t, 8)
	second.IOTranscript, err = deterministicio.EncodeTranscript([]deterministicio.Operation{{Name: "os.open"}, {Ordinal: 1, Name: "os.read"}})
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
	reopened, err := Open(context.Background(), root, identity)
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
	corpus, err := Open(context.Background(), root, identity)
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
	reopened, err := Open(context.Background(), root, identity)
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
			selected, err := boundedEntries(entries)
			if err != nil || len(selected) != test.want {
				t.Fatalf("boundedEntries() kept %d cases, %v, want %d", len(selected), err, test.want)
			}
		})
	}
}

func TestCorpusReadSnapshotPreservesResultsAndErrorOrder(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	snapshot, encoded, err := finalizeSnapshot(Snapshot{Schema: CorpusSchema, Identity: identity, Entries: []Entry{}})
	if err != nil {
		t.Fatal(err)
	}
	changed := snapshot
	changed.Identity.TargetSHA256 = record.HashBytes([]byte("changed"))
	_, changedBytes, err := finalizeSnapshot(changed)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		contents []byte
		mode     os.FileMode
		want     string
	}{
		{name: "success", contents: encoded, mode: 0o600},
		{name: "mode before malformed", contents: []byte(`{"schema":`), mode: 0o644, want: "guided corpus snapshot mode or size is invalid"},
		{name: "malformed", contents: []byte(`{"schema":`), mode: 0o600, want: "decode guided corpus snapshot schema: unexpected end of JSON input"},
		{name: "schema before identity", contents: []byte(`{"schema":"gomad3.guide-corpus/v1","identity":{}}`), mode: 0o600, want: "guided corpus schema is invalid"},
		{name: "identity", contents: changedBytes, mode: 0o600, want: "guided corpus identity does not match the prepared target and toolchain"},
		{name: "digest before identity", contents: bytes.Replace(changedBytes, []byte(changed.CoverageSHA256), []byte(record.HashBytes(nil)), 1), mode: 0o600, want: "guided corpus snapshot identity mismatch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "corpus.json")
			if err := os.WriteFile(path, test.contents, test.mode); err != nil {
				t.Fatal(err)
			}
			corpus := &Corpus{path: root, identity: identity, snapshot: snapshot}
			got, err := corpus.readSnapshot()
			if test.want == "" {
				if err != nil || !reflect.DeepEqual(got, snapshot) {
					t.Fatalf("readSnapshot() = %#v, %v", got, err)
				}
			} else {
				if err == nil || err.Error() != test.want || !reflect.DeepEqual(got, Snapshot{}) {
					t.Fatalf("readSnapshot() = %#v, %v, want zero result and %q", got, err, test.want)
				}
				if test.name == "malformed" {
					if _, ok := errors.Unwrap(err).(*json.SyntaxError); !ok {
						t.Fatalf("decode error lost its concrete syntax cause: %T", errors.Unwrap(err))
					}
				} else if reflect.TypeOf(err) != reflect.TypeOf(errors.New("")) {
					t.Fatalf("primary error gained a cleanup wrapper: %T", err)
				}
			}
			actual, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(actual, test.contents) || !reflect.DeepEqual(corpus.snapshot, snapshot) {
				t.Fatalf("readSnapshot changed disk or memory: %v", readErr)
			}
		})
	}
}

func TestCorpusValidateEntryPreservesResultsAndErrorOrder(t *testing.T) {
	input, coverage, _ := guideArtifactInput(t, 7)
	corpus, err := Open(context.Background(), t.TempDir(), guideIdentity(t, input.Manifest))
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
	corpus, err := Open(context.Background(), t.TempDir(), guideIdentity(t, input.Manifest))
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
	if _, err := os.Stat(filepath.Join(corpus.path, "corpus.json")); !os.IsNotExist(err) {
		t.Fatalf("failed validation published a snapshot: %v", err)
	}
}

func TestCorpusCanonicalSnapshotBaseline(t *testing.T) {
	input, coverage, _ := guideArtifactInput(t, 7)
	corpus, err := Open(context.Background(), t.TempDir(), guideIdentity(t, input.Manifest))
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
	encoded, err := os.ReadFile(filepath.Join(corpus.path, "corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"coverage_sha256":"sha256:6133c1926850e76c20dadfb2f3bc78162729785eb67cf45711ec80d0dcd7cc5b","entries":[{"artifact":"cases/sha256-988727643f3e5380fef260f35d88bdd3","captured_inputs":{"io_transcript_records":"1","io_transcript_sha256":"sha256:e81de92b0622e7645f313e8c7b95f2a0481d122a093d89f03d79dd2e5b8a0496","world_initial_sha256":"sha256:74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b","world_transcript_sha256":"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","world_transition_records":"0"},"features":[{"kind":"outcome","value":"success/success/exit"},{"kind":"io_outcome","value":"os.open/0"}],"novelty_reasons":[{"kind":"outcome","value":"success/success/exit"},{"kind":"io_outcome","value":"os.open/0"}],"payload_bytes":"149","record_hash":"sha256:988727643f3e5380fef260f35d88bdd3ce9fcfe09fbecafb649b67f19613e33e","replay":{"diagnostic":false,"match":true,"verified":true},"seed":"7","semantic_coverage":{"digest":"sha256:b14c6b04e0e4bd4b4431562e77b1b78eb7940a9cc505e47a8df9f713f4058356","probes":[],"schema":"gomad3.semantic-coverage/v1"},"stored_bytes":"7838"}],"generation":"1","identity":{"boundary_sha256":"sha256:ca18b6934d906b95235e04f83dfc2eef0a94d17d5417eb032cd086f7425ebbd0","boundary_version":"go1.27.1-v1","environment_sha256":"sha256:8a15523da42b0647d93aa136c19f86bec6d0117878938d7f86310c94a38bfd5b","instrumentation_schema":"gomad3.semantic-features/v1","instrumentation_sha256":"sha256:6b0e45f581c4be050d0caac7bb11394e111373c0212bd200394f509555773e2b","manifest_record_contract":"gomad3.execution-record/v1","manifest_schema_version":1,"target_sha256":"sha256:1876ff5982ede7673a78a35d78aa868749ba1d629c50582c92101bbfe0da2254","toolchain":{"build_key":"cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e","go_version":"go1.26.4","target_goarch":"arm64","target_goos":"darwin"}},"schema":"gomad3.guide-corpus/v2","snapshot_sha256":"sha256:b465cb3046d07998b31810e6a10167507ead9c68d33cade81d212552d9f57833"}`
	if string(encoded) != want || record.HashBytes(encoded) != "sha256:e243d461e78c4a4a1a7f9832091709dc8bd2f8ade34e61d318914b8269fecf2f" || corpus.snapshot.SnapshotSHA256 != "sha256:b465cb3046d07998b31810e6a10167507ead9c68d33cade81d212552d9f57833" {
		t.Fatalf("canonical snapshot changed: %s", encoded)
	}
	got, err := corpus.readSnapshot()
	if err != nil || !reflect.DeepEqual(got, corpus.snapshot) {
		t.Fatalf("readSnapshot() = %#v, %v", got, err)
	}
}
