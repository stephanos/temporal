package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
)

func TestPreMoveWASMArtifactReportsModelIdentityMismatch(t *testing.T) {
	t.Run("original", func(t *testing.T) {
		path := os.Getenv("GOMAD_WASM_PRE_MOVE_ARTIFACT")
		if path == "" {
			t.Skip("retained pre-move artifact required: GOMAD_WASM_PRE_MOVE_ARTIFACT")
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
			ModelSHA256: currentModel, GoVersion: pr.GoVersion, Engine: pr.Engine,
			MemoryBytes: pr.MemoryBytes, Fuel: pr.Fuel, Config: pr.Config,
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = provider.ValidateReplay(t.Context(), opened)
		if err == nil || !strings.Contains(err.Error(), "model identity mismatch") || !strings.Contains(err.Error(), recordedModel) || !strings.Contains(err.Error(), currentModel) {
			t.Fatalf("replay must name recorded and current model identities: %v", err)
		}
		t.Logf("record=%s recorded=%s current=%s rejection=%v", manifest.RecordHash, recordedModel, currentModel, err)
	})
}
