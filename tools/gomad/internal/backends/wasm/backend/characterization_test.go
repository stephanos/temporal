package backend

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
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
// with the named unsupported format before decoding its manifest. Adding a
// current envelope still cannot admit its historical manifest namespace.
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
	if opened != nil || err == nil || errors.Is(err, artifact.ErrUnsupportedFormat) || err.Error() != `decode artifact manifest: unknown artifact kind "gomad3.success/v1"` {
		t.Fatalf("historical manifest with a current envelope = %v, %v; want the old namespace refusal", opened, err)
	}
	if got := stringMustRead(t, filepath.Join(staged, "manifest.json")); got != stringMustRead(t, filepath.Join("testdata", "premove-stock-artifact", "manifest.json")) {
		t.Fatal("historical manifest bytes changed")
	}
}

func TestCurrentWASMArtifactStorageLineageOpensOwnerPublishedSuccess(t *testing.T) {
	module := []byte("current artifact module")
	modulePath := filepath.Join(t.TempDir(), "module.wasm")
	if err := os.WriteFile(modulePath, module, 0o700); err != nil {
		t.Fatal(err)
	}
	provenance := []byte(`{"schema":"gomad-wasm.provenance/v1"}`)
	metadata := record.BackendMetadata{Name: Name, ReplayMode: record.ReplayObserved, Provenance: record.BackendPayload{
		Schema: provenanceSchema, File: "backend/provenance.json", SHA256: record.HashBytes(provenance), Bytes: record.Uint64String(len(provenance)),
	}}
	world, worldPayloads := record.NoneWorld()
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactSuccess, CreatedAt: "2026-10-16T00:00:00Z", CampaignID: "current-namespace-control", Seed: 7, ReplayMode: record.ReplayObserved,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "fixture", HostOS: "linux", HostArch: "arm64"},
		Toolchain: record.Toolchain{GoVersion: "go1.27.1", BuildKey: strings.Repeat("0", 64), TargetGOOS: "wasip1", TargetGOARCH: "wasm"},
		Target:    record.Target{Kind: "go-run", Source: ".", SHA256: record.HashBytes(module), Size: record.Uint64String(len(module)), Argv: []string{"gomad-target"}, BuildTags: []string{}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.27.1", Path: "example.test/current"}, Backend: &metadata},
		IOProfile: record.BackendIOProfile(metadata), Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMAD_IO_PROFILE", Value: Name}, {Name: "TZ", Value: "UTC"}},
		Limits: record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64}, World: world,
		Outcome: record.Outcome{Domain: "success", Reason: "exit_zero", Termination: "exit", ExitCode: new(record.Uint64String)},
		Streams: record.Streams{Stdout: record.Stream{FullSHA256: record.HashBytes(nil)}, Stderr: record.Stream{FullSHA256: record.HashBytes(nil)}},
		Host:    record.Host{StartedAt: "2026-10-16T00:00:00Z", FinishedAt: "2026-10-16T00:00:00Z"},
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: modulePath, World: worldPayloads,
		BackendPayloads: []target.BackendPayload{{Reference: metadata.Provenance, Data: provenance}},
	})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := artifact.OpenArtifact(published.Path)
	if err != nil {
		t.Fatalf("owner-published current artifact: %v", err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	if got := opened.Manifest(); got.ArtifactKind != record.ArtifactSuccess || got.RecordHash != published.Manifest.RecordHash {
		t.Fatalf("current artifact changed: %+v", got)
	}
	if got, err := opened.ReadPayload("target", uint64(len(module))); err != nil || string(got) != string(module) {
		t.Fatalf("current artifact target = %q, %v", got, err)
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
