package artifact

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

func TestPublishFailsBeforePublicationWhenByteCapacityIsExceeded(t *testing.T) {
	root := t.TempDir()
	_, err := (Store{Root: root, MaximumBytes: 1}).Publish(artifactInput(t))
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
	published, err := store.Publish(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filepath.Base(published.Path), "sha256-") {
		t.Fatalf("artifact path = %q", published.Path)
	}
	opened, err := Open(published.Path)
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
	first, err := store.Publish(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Publish(input)
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
	if _, err := store.Publish(input); err == nil || !strings.Contains(err.Error(), "existing artifact") {
		t.Fatalf("Publish() error = %v", err)
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
			published, err := store.Publish(input)
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
	if _, err := Open(expected); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRejectsSymlinksAndUnlistedFiles(t *testing.T) {
	store := Store{Root: t.TempDir()}
	published, err := store.Publish(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("stdout", filepath.Join(published.Path, "extra")); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(published.Path); err == nil || !strings.Contains(err.Error(), "unlisted") {
		t.Fatalf("Open() error = %v", err)
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
	if _, err := Open(published.Path); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("Open() error = %v", err)
	}
}

func TestOpenedArtifactRemainsPinnedAcrossPathReplacement(t *testing.T) {
	store := Store{Root: t.TempDir()}
	published, err := store.Publish(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(published.Path)
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

func TestPublishRejectsWorldPathEscapeBeforeWriting(t *testing.T) {
	input := artifactInput(t)
	input.Manifest.World.Initial.File = "world/../../escaped"
	root := t.TempDir()
	if _, err := (Store{Root: root}).Publish(input); err == nil || !strings.Contains(err.Error(), "World payload paths") {
		t.Fatalf("Publish() error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "escaped")); !os.IsNotExist(err) {
		t.Fatalf("escaped payload was written: %v", err)
	}
}

func artifactInput(t *testing.T) Input {
	t.Helper()
	targetPath := filepath.Join(t.TempDir(), "target")
	targetBytes := []byte("target bytes")
	if err := os.WriteFile(targetPath, targetBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	world, payloads := record.NoneWorld()
	exitCode := record.Uint64String(2)
	stdout := []byte("stdout")
	stderr := []byte("stderr")
	return Input{
		Manifest: record.Manifest{
			SchemaVersion:    record.SchemaVersion,
			ArtifactKind:     record.ArtifactTargetFailure,
			CreatedAt:        "2026-08-10T12:00:00Z",
			BatchID:          "batch-1",
			SelectionOrdinal: 0,
			Seed:             7,
			ReplayMode:       record.ReplayExact,
			Runner:           record.Runner{RecordContract: "gomadv3.run-record/v1", RunnerBuild: "test", HostOS: "darwin", HostArch: "arm64"},
			Toolchain:        record.Toolchain{GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
			Target: record.Target{
				Kind: "go-run", Source: ".", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)), Argv: []string{"gomadv3-target"}, BuildTags: []string{},
				BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
			},
			Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
			Limits: record.Limits{
				RunTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64,
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
		},
		TargetPath: targetPath,
		Stdout:     stdout,
		Stderr:     stderr,
		World:      payloads,
	}
}
