package artifact

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestPrivatePayloadWritesLiteralMetadataAndBytes(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(source, []byte("target bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		payload Payload
		want    record.File
		bytes   string
	}{
		{Payload{Path: "target", Mode: 0o700, SourcePath: source}, record.File{Path: "target", Mode: "0700", Size: 12, SHA256: "sha256:0350a9d9ffa2b933c2e6a8d73d4ee7398415e547178881241eca6bb865de5393"}, "target bytes"},
		{Payload{Path: "stdout", Mode: 0o600, Data: []byte("stdout")}, record.File{Path: "stdout", Mode: "0600", Size: 6, SHA256: "sha256:63d42d26156fcc761e57da4128e9881d5bdf3bf933f0f6e9c93d6e26b9b90ae7"}, "stdout"},
	} {
		t.Run(test.payload.Path, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "payloads", test.payload.Path)
			file, err := placePayload(context.Background(), test.payload, destination)
			if err != nil || file != test.want {
				t.Fatalf("placePayload() = %#v, %v, want %#v", file, err, test.want)
			}
			data, err := os.ReadFile(destination)
			if err != nil || string(data) != test.bytes {
				t.Fatalf("payload bytes = %q, %v, want %q", data, err, test.bytes)
			}
			info, err := os.Stat(destination)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != test.payload.Mode {
				t.Fatalf("payload mode = %#o, want %#o", info.Mode().Perm(), test.payload.Mode)
			}
			parent, err := os.Stat(filepath.Dir(destination))
			if err != nil || parent.Mode().Perm() != 0o700 {
				t.Fatalf("payload parent = %v, %v, want private directory", parent, err)
			}
		})
	}
}

func TestPrivatePayloadCancellationKeepsPrimaryErrorAndPartialDestination(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(source, []byte("target bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		payload Payload
		want    string
	}{
		{Payload{Path: "target", Mode: 0o700, SourcePath: source}, "copy payload target: context canceled"},
		{Payload{Path: "stdout", Mode: 0o600, Data: []byte("stdout")}, "write payload stdout: context canceled"},
	} {
		t.Run(test.payload.Path, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			destination := filepath.Join(t.TempDir(), test.payload.Path)
			file, err := placePayload(ctx, test.payload, destination)
			if file != (record.File{}) || err == nil || err.Error() != test.want || !errors.Is(err, context.Canceled) || errors.Unwrap(err) != context.Canceled {
				t.Fatalf("placePayload() = %#v, %v, want zero metadata and single cancellation wrapper %q", file, err, test.want)
			}
			data, err := os.ReadFile(destination)
			if err != nil || len(data) != 0 {
				t.Fatalf("partial destination = %q, %v, want retained empty file", data, err)
			}
			info, err := os.Stat(destination)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != test.payload.Mode {
				t.Fatalf("partial destination mode = %#o, want %#o", info.Mode().Perm(), test.payload.Mode)
			}
		})
	}
}

func TestPrivatePayloadCollisionPreservesDestinationAndPathError(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(source, []byte("target bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, payload := range []Payload{
		{Path: "target", Mode: 0o700, SourcePath: source},
		{Path: "stdout", Mode: 0o600, Data: []byte("stdout")},
	} {
		t.Run(payload.Path, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), payload.Path)
			if err := os.WriteFile(destination, []byte("sentinel"), 0o600); err != nil {
				t.Fatal(err)
			}
			file, err := placePayload(context.Background(), payload, destination)
			var pathErr *os.PathError
			if file != (record.File{}) || !errors.Is(err, os.ErrExist) || !errors.As(err, &pathErr) {
				t.Fatalf("placePayload() = %#v, %v, want zero metadata and destination collision", file, err)
			}
			if errors.Unwrap(err) != pathErr || pathErr.Op != "open" || pathErr.Path != destination || err.Error() != "create payload "+payload.Path+": "+pathErr.Error() {
				t.Fatalf("collision error shape = %#v, %v", pathErr, err)
			}
			data, err := os.ReadFile(destination)
			if err != nil || string(data) != "sentinel" {
				t.Fatalf("existing destination = %q, %v, want sentinel", data, err)
			}
		})
	}
}

func TestPrivatePayloadRejectsNonregularSourceBeforeCreatingDestination(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "target")
	file, err := copyPayload(context.Background(), t.TempDir(), destination, "target", 0o700)
	if file != (record.File{}) || err == nil || err.Error() != "payload target is not a regular file" || errors.Unwrap(err) != nil {
		t.Fatalf("copyPayload() = %#v, %v", file, err)
	}
	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nonregular source created destination: %v", err)
	}
}

func TestPublishRemovesStagingAfterNonregularSourceFailure(t *testing.T) {
	input := artifactInput(t)
	input.Payloads[0].SourcePath = t.TempDir()
	root := t.TempDir()
	published, err := (Store{Root: root}).PublishArtifact(input)
	if err == nil || err.Error() != "payload target is not a regular file" || published.Path != "" {
		t.Fatalf("PublishArtifact() = %#v, %v", published, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed publication entries = %v, %v, want no staging or published artifact", entries, err)
	}
}

func TestPublishFailsBeforePublicationWhenByteCapacityIsExceeded(t *testing.T) {
	root := t.TempDir()
	_, err := (Store{Root: root, MaximumBytes: 1}).PublishArtifact(artifactInput(t))
	var capacity *CapacityError
	if !errors.As(err, &capacity) || capacity.Maximum != 1 || capacity.Required <= capacity.Maximum {
		t.Fatalf("Publish() error = %#v", err)
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("artifact store entries = %v", entries)
	}
}

func TestPublishWritesPrivateAtomicArtifactAndOpenValidatesIt(t *testing.T) {
	input := artifactInput(t)
	store := Store{Root: t.TempDir()}
	published, err := store.PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filepath.Base(published.Path), "sha256-") {
		t.Fatalf("artifact path = %q", published.Path)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest().RecordHash != published.Manifest.RecordHash || opened.Manifest().Outcome.FailureSignature != published.Manifest.Outcome.FailureSignature {
		t.Fatal("opened artifact identity changed")
	}
	for path, mode := range map[string]os.FileMode{
		"manifest.json":             0o600,
		"target":                    0o700,
		"stdout":                    0o600,
		"stderr":                    0o600,
		"world/snapshot.json":       0o600,
		"world/transitions.jsonl":   0o600,
		"world/final-snapshot.json": 0o600,
	} {
		info, statErr := os.Stat(filepath.Join(published.Path, path))
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm() != mode {
			t.Fatalf("%s mode = %#o, want %#o", path, info.Mode().Perm(), mode)
		}
	}
}

func TestPublishReusesOnlyCompletelyMatchingArtifact(t *testing.T) {
	input := artifactInput(t)
	store := Store{Root: t.TempDir()}
	first, err := store.PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Path != second.Path || first.Manifest.RecordHash != second.Manifest.RecordHash {
		t.Fatalf("reused artifact = %#v, want %#v", second, first)
	}
	input.Record.Seed = 8
	input.Record.Environment[1].Value = "8"
	second, err = store.PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	if second.Path != first.Path || second.Manifest.RecordHash != first.Manifest.RecordHash {
		t.Fatalf("failure signature deduplication changed: %#v, want %#v", second, first)
	}
	if err := os.Chmod(filepath.Join(first.Path, "stdout"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(first.Path, "stdout"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublishArtifact(input); err == nil || !strings.Contains(err.Error(), "existing artifact") {
		t.Fatalf("Publish() error = %v", err)
	}
}

func TestPublishCanKeyCorpusCasesByRecordIdentity(t *testing.T) {
	input := artifactInput(t)
	store := Store{Root: t.TempDir(), Key: StoreKeyRecord}
	first, err := store.PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Record.Seed = 8
	input.Record.Environment[1].Value = "8"
	second, err := store.PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Path == second.Path || first.Manifest.Outcome.FailureSignature != second.Manifest.Outcome.FailureSignature || first.Manifest.RecordHash == second.Manifest.RecordHash {
		t.Fatalf("record-keyed artifacts = %#v and %#v", first, second)
	}
}

func TestPublishKeepsSuccessesWithOneSignatureAsDistinctReplayArtifacts(t *testing.T) {
	input := artifactInput(t)
	input.Record.ArtifactKind = record.ArtifactSuccess
	zero := record.Uint64String(0)
	input.Record.Outcome = record.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &zero}
	store := Store{Root: t.TempDir()}
	first, err := store.PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(first.Path) != "sha256-8804bc935588b0e0ac9fd7f890e4da67" || first.Manifest.RecordHash != "sha256:27c9b74965e1b7cb30ef6f914b8f028eda0072216f5df3794bd84f8536db5f9d" || first.Manifest.Outcome.FailureSignature != "sha256:8804bc935588b0e0ac9fd7f890e4da6718d567133466c52672ce1b11e9b454be" {
		t.Fatalf("noncolliding success identity changed: %#v", first)
	}
	paths := map[string]bool{first.Path: true}
	for _, seed := range []record.Uint64String{8, 9} {
		input.Record.Seed = seed
		input.Record.Environment[1].Value = fmt.Sprint(seed)
		published, err := store.PublishArtifact(input)
		if err != nil {
			t.Fatal(err)
		}
		if paths[published.Path] || published.Manifest.Seed != seed || published.Manifest.Outcome.FailureSignature != first.Manifest.Outcome.FailureSignature {
			t.Fatalf("success seed %d collapsed to %#v", seed, published)
		}
		paths[published.Path] = true
		again, err := store.PublishArtifact(input)
		if err != nil {
			t.Fatal(err)
		}
		if again.Path != published.Path || again.Manifest.RecordHash != published.Manifest.RecordHash {
			t.Fatalf("republication changed the success identity: %#v, want %#v", again, published)
		}
	}
	entries, err := os.ReadDir(store.Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("stored %d entries, want three successes", len(entries))
	}
}

// Two seeds can complete with one outcome signature. Under StoreKeyExecution
// each keeps its own artifact: the first under the signature's directory, as
// before, and the next under its execution's identity. Publishing one
// execution again, as a resumed campaign does, reuses its artifact.
func TestPublishKeepsEachExecutionOfOneOutcomeSignature(t *testing.T) {
	root := t.TempDir()
	store := Store{Root: root, Key: StoreKeyExecution}
	execution := func(ordinal, seed uint64) Publication {
		input := seededInput(t, seed)
		input.Record.ArtifactKind = record.ArtifactSuccess
		input.Record.Outcome = record.Outcome{Domain: "success", Reason: "exit_zero", Termination: "exit", ExitCode: new(record.Uint64String)}
		input.Record.SelectionOrdinal = record.Uint64String(ordinal)
		return input
	}
	var published []Artifact
	for ordinal, seed := range []uint64{7, 8, 9} {
		artifact, err := store.PublishArtifact(execution(uint64(ordinal), seed))
		if err != nil {
			t.Fatal(err)
		}
		published = append(published, artifact)
	}
	signature := published[0].Manifest.Outcome.FailureSignature
	if want := filepath.Join(root, identityDirectory(signature, false)); published[0].Path != want {
		t.Fatalf("first artifact path = %s, want the signature's directory %s", published[0].Path, want)
	}
	for index, artifact := range published {
		if artifact.Manifest.Outcome.FailureSignature != signature {
			t.Fatalf("artifact %d signature = %s, want %s", index, artifact.Manifest.Outcome.FailureSignature, signature)
		}
		opened, err := OpenArtifact(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
		if opened.Manifest().Seed != artifact.Manifest.Seed || opened.Manifest().SelectionOrdinal != record.Uint64String(index) {
			t.Fatalf("artifact %d at %s records seed %d ordinal %d", index, artifact.Path, opened.Manifest().Seed, opened.Manifest().SelectionOrdinal)
		}
		if index == 0 {
			continue
		}
		if want := filepath.Join(root, identityDirectory(executionIdentity(opened.Manifest()), true)); artifact.Path != want {
			t.Fatalf("artifact %d path = %s, want its execution's directory %s", index, artifact.Path, want)
		}
	}
	again := execution(1, 8)
	again.Record.CreatedAt, again.Record.Host.StartedAt, again.Record.Host.FinishedAt = "2026-08-10T13:00:00Z", "2026-08-10T13:00:00Z", "2026-08-10T13:00:01Z"
	again.Record.Limits.OverallTimeoutNanos = 3
	reused, err := store.PublishArtifact(again)
	if err != nil {
		t.Fatal(err)
	}
	if reused.Path != published[1].Path || reused.Manifest.CreatedAt != published[1].Manifest.CreatedAt {
		t.Fatalf("published execution again at %s (created %s), want reuse of %s", reused.Path, reused.Manifest.CreatedAt, published[1].Path)
	}
	first := execution(0, 7)
	first.Record.CreatedAt, first.Record.Host.StartedAt, first.Record.Host.FinishedAt = "2026-08-10T13:00:00Z", "2026-08-10T13:00:00Z", "2026-08-10T13:00:01Z"
	reusedFirst, err := store.PublishArtifact(first)
	if err != nil {
		t.Fatal(err)
	}
	if reusedFirst.Path != published[0].Path || reusedFirst.Manifest.CreatedAt != published[0].Manifest.CreatedAt {
		t.Fatalf("published first execution again at %s (created %s), want reuse of %s", reusedFirst.Path, reusedFirst.Manifest.CreatedAt, published[0].Path)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 3 {
		t.Fatalf("store entries = %v, %v, want three artifacts", entries, err)
	}
}

func TestPublishConcurrentlyNeverReplacesACompleteArtifact(t *testing.T) {
	input := artifactInput(t)
	store := Store{Root: t.TempDir()}
	const publishers = 8
	paths := make(chan string, publishers)
	errors := make(chan error, publishers)
	var wait sync.WaitGroup
	for range publishers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			published, err := store.PublishArtifact(input)
			if err != nil {
				errors <- err
				return
			}
			paths <- published.Path
		}()
	}
	wait.Wait()
	close(errors)
	close(paths)
	for err := range errors {
		t.Fatal(err)
	}
	var expected string
	for path := range paths {
		if expected == "" {
			expected = path
		}
		if path != expected {
			t.Fatalf("concurrent artifact path = %s, want %s", path, expected)
		}
	}
	if _, err := OpenArtifact(expected); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRejectsSymlinksAndUnlistedFiles(t *testing.T) {
	store := Store{Root: t.TempDir()}
	published, err := store.PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("stdout", filepath.Join(published.Path, "extra")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenArtifact(published.Path); err == nil || !strings.Contains(err.Error(), "unlisted") {
		t.Fatalf("OpenArtifact() error = %v", err)
	}
	if err := os.Remove(filepath.Join(published.Path, "extra")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(published.Path, "stderr")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("stdout", filepath.Join(published.Path, "stderr")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenArtifact(published.Path); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("OpenArtifact() error = %v", err)
	}
}

func TestOpenedArtifactRemainsPinnedAcrossPathReplacement(t *testing.T) {
	store := Store{Root: t.TempDir()}
	published, err := store.PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	moved := published.Path + ".moved"
	if err := os.Rename(published.Path, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), published.Path); err != nil {
		t.Fatal(err)
	}
	stdout, err := opened.ReadPayload("stdout", 64)
	if err != nil {
		t.Fatal(err)
	}
	if string(stdout) != "stdout" {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestPublishRejectsPayloadPathEscapeBeforeWriting(t *testing.T) {
	input := artifactInput(t)
	input.Payloads[3].Path = "world/../../escaped"
	root := t.TempDir()
	if _, err := (Store{Root: root}).PublishArtifact(input); err == nil || !strings.Contains(err.Error(), "invalid artifact payload path") {
		t.Fatalf("Publish() error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "escaped")); !os.IsNotExist(err) {
		t.Fatalf("escaped payload was written: %v", err)
	}
}

func artifactInput(t *testing.T) Publication {
	t.Helper()
	targetPath := filepath.Join(t.TempDir(), "target")
	targetBytes := []byte("target bytes")
	if err := os.WriteFile(targetPath, targetBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	world, worldPayloads := record.NoneWorld()
	exitCode := record.Uint64String(2)
	stdout := []byte("stdout")
	stderr := []byte("stderr")
	manifest := record.ExecutionRecord{
		SchemaVersion:    record.SchemaVersion,
		ArtifactKind:     record.ArtifactTargetFailure,
		CreatedAt:        "2026-08-10T12:00:00Z",
		CampaignID:       "batch-1",
		SelectionOrdinal: 0,
		Seed:             7,
		ReplayMode:       record.ReplayExact,
		Runner:           record.Runner{RecordContract: record.RecordContract, RunnerBuild: "test", HostOS: "darwin", HostArch: "arm64"},
		Toolchain:        record.Toolchain{GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target: record.Target{
			Kind: "go-run", Source: ".", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)), Argv: []string{"gomad3-target"}, BuildTags: []string{},
			Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
		},
		IOProfile:   record.IOProfile{Name: "gomad3-deterministic/v1", ImplementationSHA256: record.HashBytes([]byte("implementation")), Inventory: "{}", InventorySHA256: record.HashBytes([]byte("{}"))},
		Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: "gomad3-deterministic/v1"}, {Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Limits: record.Limits{
			ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64,
		},
		World: world,
		Outcome: record.Outcome{
			Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode,
		},
		Streams: record.Streams{
			Stdout: record.Stream{FullSHA256: record.HashBytes(stdout), TotalBytes: record.Uint64String(len(stdout)), RetainedBytes: record.Uint64String(len(stdout))},
			Stderr: record.Stream{FullSHA256: record.HashBytes(stderr), TotalBytes: record.Uint64String(len(stderr)), RetainedBytes: record.Uint64String(len(stderr))},
		},
		Host: record.Host{StartedAt: "2026-08-10T12:00:00Z", FinishedAt: "2026-08-10T12:00:01Z", ElapsedNanos: 1},
	}
	manifest.Target.File = "target"
	manifest.Streams.Stdout.File = "stdout"
	manifest.Streams.Stdout.RetainedSHA256 = record.HashBytes(stdout)
	manifest.Streams.Stderr.File = "stderr"
	manifest.Streams.Stderr.RetainedSHA256 = record.HashBytes(stderr)
	return Publication{Record: manifest, Payloads: []Payload{
		{Path: "target", Mode: 0o700, SourcePath: targetPath, SHA256: manifest.Target.SHA256, Size: manifest.Target.Size},
		{Path: "stdout", Mode: 0o600, Data: stdout, SHA256: record.HashBytes(stdout), Size: record.Uint64String(len(stdout))},
		{Path: "stderr", Mode: 0o600, Data: stderr, SHA256: record.HashBytes(stderr), Size: record.Uint64String(len(stderr))},
		{Path: manifest.World.Initial.File, Mode: 0o600, Data: worldPayloads.Initial, SHA256: manifest.World.Initial.RawSHA256, Size: record.Uint64String(len(worldPayloads.Initial))},
		{Path: manifest.World.Transitions.File, Mode: 0o600, Data: worldPayloads.Transitions, SHA256: manifest.World.Transitions.RawSHA256, Size: record.Uint64String(len(worldPayloads.Transitions))},
		{Path: manifest.World.Final.File, Mode: 0o600, Data: worldPayloads.Final, SHA256: manifest.World.Final.RawSHA256, Size: record.Uint64String(len(worldPayloads.Final))},
	}}
}
