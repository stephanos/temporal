package backend

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
)

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

// The fixture is a stock-profile success retained by the fn-156 WASM
// qualification before the backend move.
func TestPreMoveWASMArtifactReportsModelIdentityMismatch(t *testing.T) {
	opened, err := artifact.OpenArtifact(stageRetainedArtifact(t, filepath.Join("testdata", "premove-stock-artifact")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	manifest := opened.Manifest()
	if manifest.RecordHash != "sha256:ff8eac83b09b83de585a03b0602153457ef6ef6ee3a2753ebb3925d8954b6ed6" {
		t.Fatalf("pre-move record changed: %s", manifest.RecordHash)
	}
	data, err := opened.ReadPayload(manifest.Target.Backend.Provenance.File, maximumProvenanceBytes)
	if err != nil {
		t.Fatal(err)
	}
	pr, err := decodeProvenance(data)
	if err != nil {
		t.Fatal(err)
	}
	const recordedModel = "sha256:f101f3f203bd3cac097966f14203e8f21c61b2d8c2ce3ec4a1fe1114fb7bab0f"
	currentModel := wasi.ImplementationSHA256()
	if pr.ModelSHA256 != recordedModel || currentModel == recordedModel {
		t.Fatalf("pre-move model fixture changed: recorded=%s current=%s", pr.ModelSHA256, currentModel)
	}
	helper := filepath.Join(t.TempDir(), "helper")
	options := Options{
		CompilerPath: filepath.Join(t.TempDir(), "compiler"), CompilerSHA256: pr.CompilerSHA256,
		HelperPath: helper, HelperSHA256: pr.HelperSHA256,
		ModelSHA256: currentModel, GoVersion: pr.GoVersion, Engine: wasi.PinnedEngine(),
		MemoryBytes: pr.MemoryBytes, Fuel: pr.Fuel, Config: pr.Config,
	}
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	want := "WASM model identity mismatch: recorded " + recordedModel + ", current " + currentModel
	if _, err := provider.ValidateReplay(t.Context(), opened); err == nil || err.Error() != want {
		t.Fatalf("replay preflight = %v, want %q", err, want)
	}
	// New admits only this binary's model, so the provider that built the
	// artifact is reconstructed directly. With its recorded identities the
	// artifact passes every identity check and stops at the absent helper.
	options.ModelSHA256, options.Engine = recordedModel, pr.Engine
	if _, err := (&Provider{options: options}).ValidateReplay(t.Context(), opened); err == nil || strings.Contains(err.Error(), "identity") || strings.Contains(err.Error(), "mismatch") || !strings.Contains(err.Error(), helper) {
		t.Fatalf("unchanged-identity replay preflight = %v", err)
	}
}

func TestPreMoveMinimizedWASMArtifactReportsModelIdentityMismatch(t *testing.T) {
	path := os.Getenv("GOMAD_PRE_MOVE_MINIMIZED_ARTIFACT")
	if path == "" {
		t.Skip("retained pre-move artifact required: GOMAD_PRE_MOVE_MINIMIZED_ARTIFACT")
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
	data, err := opened.ReadPayload(manifest.Target.Backend.Provenance.File, maximumProvenanceBytes)
	if err != nil {
		t.Fatal(err)
	}
	pr, err := decodeProvenance(data)
	if err != nil {
		t.Fatal(err)
	}
	const recordedModel = "sha256:19c26be08c13427544ff7b198c00f57179dd67793bbe6d8ec3059f907d3e0cb1"
	currentModel := wasi.ImplementationSHA256()
	if pr.ModelSHA256 != recordedModel || currentModel == recordedModel {
		t.Fatalf("pre-move model fixture changed: recorded=%s current=%s", pr.ModelSHA256, currentModel)
	}
	absent := t.TempDir()
	provider, err := New(Options{
		CompilerPath: filepath.Join(absent, "compiler"), CompilerSHA256: pr.CompilerSHA256,
		HelperPath: filepath.Join(absent, "helper"), HelperSHA256: pr.HelperSHA256,
		ModelSHA256: currentModel, GoVersion: pr.GoVersion, Engine: wasi.PinnedEngine(),
		MemoryBytes: pr.MemoryBytes, Fuel: pr.Fuel, Config: pr.Config,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.ValidateReplay(t.Context(), opened)
	if err == nil || !strings.Contains(err.Error(), "model identity mismatch") || !strings.Contains(err.Error(), recordedModel) || !strings.Contains(err.Error(), currentModel) {
		t.Fatalf("replay must name recorded and current model identities: %v", err)
	}
}
