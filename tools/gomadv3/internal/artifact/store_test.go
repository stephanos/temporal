package artifact

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
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

func TestPublishWritesAndValidatesChoiceTrace(t *testing.T) {
	input := artifactInput(t)
	trace := choiceTracePayload(t)
	input.Manifest.ChoiceProfile = &record.ChoiceProfile{
		Name:                 choicewire.Profile,
		ImplementationSHA256: choiceImplementationIdentity(t, input.Manifest.Toolchain.BuildKey),
		Trace: record.ChoiceTrace{
			Schema: "gomadv3.choice-trace/v1", SHA256: record.HashBytes(trace), Bytes: record.Uint64String(len(trace)),
			Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: choicewire.HeaderBytes + choicewire.RecordBytes,
		},
	}
	input.Manifest.Limits.ChoiceTraceBytes = choicewire.HeaderBytes + choicewire.RecordBytes
	input.Manifest.Environment = append(input.Manifest.Environment, record.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choicewire.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right record.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = trace

	published, err := (Store{Root: t.TempDir()}).Publish(input)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if opened.Manifest.ChoiceProfile == nil || opened.Manifest.ChoiceProfile.Trace.File != "choices.bin" {
		t.Fatalf("choice profile = %#v", opened.Manifest.ChoiceProfile)
	}
	observed, err := ReadPayload(opened, "choices.bin", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if string(observed) != string(trace) {
		t.Fatalf("choice trace = %q", observed)
	}
}

func TestPublishRejectsMalformedChoiceTraceWithMatchingIdentity(t *testing.T) {
	input := artifactInput(t)
	trace := make([]byte, choicewire.RecordBytes)
	input.Manifest.ChoiceProfile = &record.ChoiceProfile{
		Name: choicewire.Profile, ImplementationSHA256: choiceImplementationIdentity(t, input.Manifest.Toolchain.BuildKey),
		Trace: record.ChoiceTrace{
			Schema: "gomadv3.choice-trace/v1", SHA256: record.HashBytes(trace), Bytes: record.Uint64String(len(trace)),
			Records: 1, BranchingRecords: 0, TerminalState: "complete", Limit: choicewire.HeaderBytes + choicewire.RecordBytes,
		},
	}
	input.Manifest.Limits.ChoiceTraceBytes = choicewire.HeaderBytes + choicewire.RecordBytes
	input.Manifest.Environment = append(input.Manifest.Environment, record.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choicewire.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right record.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = trace

	if _, err := (Store{Root: t.TempDir()}).Publish(input); err == nil || !strings.Contains(err.Error(), "validate choice trace payload") {
		t.Fatalf("Publish() error = %v", err)
	}
}

func TestPublishRejectsChangedChoiceTraceIdentity(t *testing.T) {
	input := artifactInput(t)
	input.Manifest.ChoiceProfile = &record.ChoiceProfile{
		Name: "gomadv3-choice-trace/v1", ImplementationSHA256: record.HashBytes([]byte("choice implementation")),
		Trace: record.ChoiceTrace{Schema: "gomadv3.choice-trace/v1", SHA256: record.HashBytes([]byte("expected")), Bytes: 8, Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: 1 << 20},
	}
	input.Manifest.Limits.ChoiceTraceBytes = 1 << 20
	input.Manifest.Environment = append(input.Manifest.Environment, record.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: "gomadv3-choice-trace/v1"})
	slices.SortFunc(input.Manifest.Environment, func(left, right record.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = []byte("changed!")
	if _, err := (Store{Root: t.TempDir()}).Publish(input); err == nil || !strings.Contains(err.Error(), "choice trace implementation identity") {
		t.Fatalf("Publish() error = %v", err)
	}
}

func choiceTracePayload(t *testing.T) []byte {
	t.Helper()
	recordBytes, err := choicewire.EncodeRecord(choicewire.Record{
		Ordinal: 0, Kind: choicewire.KindRunnable, Flags: choicewire.FlagDecision, Alternatives: 2, Selected: 1, SiteOffset: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	return recordBytes[:]
}

func choiceImplementationIdentity(t *testing.T, buildKey string) record.SHA256 {
	t.Helper()
	implementation, err := choicewire.ImplementationIdentity(buildKey)
	if err != nil {
		t.Fatal(err)
	}
	return record.SHA256FromSum(implementation)
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

func TestPublishCanKeyCorpusCasesByRecordIdentity(t *testing.T) {
	input := artifactInput(t)
	store := Store{Root: t.TempDir(), Key: StoreKeyRecord}
	first, err := store.Publish(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Manifest.Seed = 8
	input.Manifest.Environment[0].Value = "8"
	second, err := store.Publish(input)
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
	profile := ioprofile.Default()
	return Input{
		Manifest: record.Manifest{
			SchemaVersion:    record.SchemaVersion,
			ArtifactKind:     record.ArtifactTargetFailure,
			CreatedAt:        "2026-08-10T12:00:00Z",
			BatchID:          "batch-1",
			SelectionOrdinal: 0,
			Seed:             7,
			ReplayMode:       record.ReplayExact,
			Runner:           record.Runner{RecordContract: record.RecordContract, RunnerBuild: "test", HostOS: "darwin", HostArch: "arm64"},
			Toolchain:        record.Toolchain{GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
			Target: record.Target{
				Kind: "go-run", Source: ".", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)), Argv: []string{"gomadv3-target"}, BuildTags: []string{},
				Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
			},
			IOProfile: record.IOProfile{
				Name: profile.Name(), ImplementationSHA256: profile.ImplementationSHA256(), Inventory: string(profile.Inventory()), InventorySHA256: profile.InventorySHA256(),
			},
			Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
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
