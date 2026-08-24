package artifact

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

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
	if opened.Manifest.RecordHash != published.Manifest.RecordHash || opened.Manifest.Outcome.FailureSignature != published.Manifest.Outcome.FailureSignature {
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
	input.Record.Environment[0].Value = "8"
	second, err := store.PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Path == second.Path || first.Manifest.Outcome.FailureSignature != second.Manifest.Outcome.FailureSignature || first.Manifest.RecordHash == second.Manifest.RecordHash {
		t.Fatalf("record-keyed artifacts = %#v and %#v", first, second)
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
	stdout, err := ReadPayload(opened, "stdout", 64)
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
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMAD3_IO_PROFILE", Value: "gomad3-deterministic/v1"}, {Name: "TZ", Value: "UTC"}},
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
