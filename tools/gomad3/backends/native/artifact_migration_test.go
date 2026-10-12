package native

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
)

// preMoveNativeArtifact is a native success artifact retained by fn-113 before
// the backend move. Its target is a Darwin binary that this test never runs.
const preMoveNativeArtifact = "../../../../.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-4/core-qualification-artifacts/v1/campaign-20261003T095108.150283000Z-f19cf22b25bc9df20ac82dfdf302641b/successes/sha256-57dc45d96a58b85a4480d2b2c067b0ac"

// stageRetainedArtifact copies a checked-in artifact with the private modes
// that OpenArtifact requires and version control does not keep.
func stageRetainedArtifact(t *testing.T, source string) string {
	t.Helper()
	destination := filepath.Join(t.TempDir(), "artifact")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.Mkdir(target, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if info.Mode()&0o100 != 0 {
			mode = 0o700
		}
		return os.WriteFile(target, data, mode)
	})
	if err != nil {
		t.Fatal(err)
	}
	return destination
}

func TestPreMoveNativeArtifactReadOnlyPreflight(t *testing.T) {
	opened, err := artifact.OpenArtifact(stageRetainedArtifact(t, preMoveNativeArtifact))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	manifest := opened.Manifest()
	if manifest.RecordHash != "sha256:f9e09cfb8a34933e1fdd43b86ef6bb6e05104e53c7815e75012528ab3e1c66ed" || manifest.Target.Backend != nil || manifest.Toolchain.TargetGOOS != "darwin" || manifest.Toolchain.TargetGOARCH != "arm64" {
		t.Fatalf("historical native artifact identity changed: %#v", manifest)
	}
	// The native schema has no backend model identity. The retained I/O
	// implementation predates this Runner's, so preflight rejects it on every
	// host before platform or toolchain checks, exactly as the pre-move Runner
	// did.
	const recordedImplementation = "sha256:9cd0cff9595bb7f79ec3de247031165cc96d7b40c0e8133051ec17afbf6ac7c0"
	if manifest.IOProfile.ImplementationSHA256 != recordedImplementation || transcript.Default().MatchesRecorded(manifest.IOProfile.Name, string(manifest.IOProfile.ImplementationSHA256), string(manifest.IOProfile.InventorySHA256), manifest.IOProfile.Inventory) {
		t.Fatalf("historical native I/O profile identity changed: %#v", manifest.IOProfile)
	}
	_, err = New(Config{ToolchainRoot: "/must-not-resolve-historical-toolchain"}).ValidateReplay(t.Context(), opened)
	if err == nil || err.Error() != "artifact I/O profile identity does not match this Runner" {
		t.Fatalf("read-only preflight = %v", err)
	}
}
