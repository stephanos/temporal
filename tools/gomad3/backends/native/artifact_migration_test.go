package native

import (
	"os"
	"runtime"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
)

func TestPreMoveNativeArtifactReadOnlyPreflight(t *testing.T) {
	path := os.Getenv("GOMAD_NATIVE_PRE_MOVE_ARTIFACT")
	if path == "" {
		t.Skip("retained pre-move artifact required: GOMAD_NATIVE_PRE_MOVE_ARTIFACT")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("retained Darwin artifact preflight observation is scoped to linux/arm64")
	}
	opened, err := artifact.OpenArtifact(path)
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
	expected := "artifact platform does not match host linux/arm64"
	if !transcript.Default().MatchesRecorded(manifest.IOProfile.Name, string(manifest.IOProfile.ImplementationSHA256), string(manifest.IOProfile.InventorySHA256), manifest.IOProfile.Inventory) {
		expected = "artifact I/O profile identity does not match this Runner"
	}
	_, err = New(Config{ToolchainRoot: "/must-not-resolve-historical-toolchain"}).ValidateReplay(t.Context(), opened)
	if err == nil || err.Error() != expected {
		t.Fatalf("read-only preflight = %v, want %q", err, expected)
	}
	t.Logf("record=%s recorded-platform=%s/%s preflight=%v; native schema has no backend model identity", manifest.RecordHash, manifest.Toolchain.TargetGOOS, manifest.Toolchain.TargetGOARCH, err)
}
