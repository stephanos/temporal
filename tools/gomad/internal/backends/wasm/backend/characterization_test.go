package backend

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
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
// qualification before the backend move, written before artifacts declared a
// storage lineage. It is now a negative control: the current build refuses it
// with the named unsupported format and never decodes its manifest. With only
// a current envelope added, the same bytes open, so the lineage gate alone
// causes the refusal.
func TestPreMoveWASMArtifactIsRefusedAsUnsupportedStorageFormat(t *testing.T) {
	staged := stageRetainedArtifact(t, filepath.Join("testdata", "premove-stock-artifact"))
	opened, err := artifact.OpenArtifact(staged)
	var unsupported *artifact.UnsupportedFormatError
	if opened != nil || !errors.Is(err, artifact.ErrUnsupportedFormat) || !errors.As(err, &unsupported) || unsupported.Format != artifact.PreLineageFormat {
		t.Fatalf("OpenArtifact() = %v, %v; want the pre-lineage refusal", opened, err)
	}
	want := `unsupported artifact storage format "pre-lineage": the artifact has no storage.json; this Gomad reads only "gomad.artifact-storage/v1" artifacts and does not migrate others`
	if err.Error() != want {
		t.Fatalf("OpenArtifact() error = %q, want %q", err, want)
	}
	if err := os.WriteFile(filepath.Join(staged, "storage.json"), []byte(`{"format":"`+artifact.StorageFormat+`"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opened, err = artifact.OpenArtifact(staged)
	if err != nil {
		t.Fatalf("control with a current envelope: %v", err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	if manifest := opened.Manifest(); manifest.RecordHash != "sha256:1437ebb347ea4cc3d31b463117c76dcd981c2c9f7285f5c289fb643402a44aaf" {
		t.Fatalf("pre-move record changed: %s", manifest.RecordHash)
	}
}

func TestPreMoveMinimizedWASMArtifactIsRefusedAsUnsupportedStorageFormat(t *testing.T) {
	path := os.Getenv("GOMAD_PRE_MOVE_MINIMIZED_ARTIFACT")
	if path == "" {
		t.Skip("retained pre-move artifact required: GOMAD_PRE_MOVE_MINIMIZED_ARTIFACT")
	}
	opened, err := artifact.OpenArtifact(path)
	var unsupported *artifact.UnsupportedFormatError
	if opened != nil || !errors.As(err, &unsupported) || unsupported.Format != artifact.PreLineageFormat {
		t.Fatalf("OpenArtifact() = %v, %v; want the pre-lineage refusal", opened, err)
	}
}
