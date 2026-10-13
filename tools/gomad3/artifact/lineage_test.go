package artifact

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishWritesTheStorageEnvelopeOutsideStoredBytes(t *testing.T) {
	published, err := (Store{Root: t.TempDir()}).PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := os.ReadFile(filepath.Join(published.Path, "storage.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(envelope) != `{"format":"gomad3.artifact-storage/v1"}`+"\n" {
		t.Fatalf("storage envelope = %q", envelope)
	}
	info, err := os.Stat(filepath.Join(published.Path, "storage.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("storage envelope mode = %v, %v; want 0600", info, err)
	}
	for _, file := range published.Manifest.Files {
		if file.Path == "storage.json" {
			t.Fatal("manifest lists the storage envelope")
		}
	}
	manifest, err := os.ReadFile(filepath.Join(published.Path, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := uint64(len(manifest))
	for _, file := range published.Manifest.Files {
		want += uint64(file.Size)
	}
	if published.StoredBytes != want {
		t.Fatalf("published stored bytes = %d, want manifest+files = %d without the envelope", published.StoredBytes, want)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer mustClose(t, opened)
	if opened.StoredBytes() != published.StoredBytes {
		t.Fatalf("opened stored bytes = %d, want %d", opened.StoredBytes(), published.StoredBytes)
	}
}

func TestPublishRejectsAPayloadNamedLikeTheStorageEnvelope(t *testing.T) {
	input := artifactInput(t)
	input.Payloads[1].Path = "storage.json"
	root := t.TempDir()
	if _, err := (Store{Root: root}).PublishArtifact(input); err == nil || !strings.Contains(err.Error(), "invalid artifact payload path") {
		t.Fatalf("Publish() error = %v", err)
	}
}

// Every lineage other than the current one is refused with the named
// unsupported format, and the manifest is never decoded: each case also
// replaces the manifest with bytes that would fail to decode.
func TestOpenRefusesEveryOtherStorageLineageBeforeDecodingTheManifest(t *testing.T) {
	for _, test := range []struct {
		name      string
		envelope  *string
		format    string
		malformed bool
		message   string
	}{
		{name: "pre-lineage", format: PreLineageFormat, message: `unsupported artifact storage format "pre-lineage": the artifact has no storage.json`},
		{name: "older", envelope: new(`{"format":"gomad3.artifact-storage/v0"}`), format: "gomad3.artifact-storage/v0", message: `unsupported artifact storage format "gomad3.artifact-storage/v0"`},
		{name: "newer", envelope: new(`{"format":"gomad3.artifact-storage/v2"}` + "\n"), format: "gomad3.artifact-storage/v2", message: `unsupported artifact storage format "gomad3.artifact-storage/v2"`},
		{name: "not-json", envelope: new(`gomad3.artifact-storage/v1`), malformed: true},
		{name: "empty", envelope: new(``), malformed: true},
		{name: "null", envelope: new(`null`), malformed: true},
		{name: "no-format", envelope: new(`{}`), malformed: true},
		{name: "unknown-field", envelope: new(`{"format":"gomad3.artifact-storage/v1","migrated":true}`), malformed: true},
		{name: "trailing-value", envelope: new(`{"format":"gomad3.artifact-storage/v1"}{}`), malformed: true},
		{name: "wrong-type", envelope: new(`{"format":1}`), malformed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			published, err := (Store{Root: t.TempDir()}).PublishArtifact(artifactInput(t))
			if err != nil {
				t.Fatal(err)
			}
			envelopePath := filepath.Join(published.Path, "storage.json")
			if test.envelope == nil {
				mustDo(t, os.Remove(envelopePath))
			} else {
				mustDo(t, os.WriteFile(envelopePath, []byte(*test.envelope), 0o600))
			}
			mustDo(t, os.WriteFile(filepath.Join(published.Path, "manifest.json"), []byte("not a manifest"), 0o600))
			opened, err := OpenArtifact(published.Path)
			if opened != nil {
				t.Fatal("OpenArtifact returned a handle for an unsupported lineage")
			}
			var unsupported *UnsupportedFormatError
			if !errors.Is(err, ErrUnsupportedFormat) || !errors.As(err, &unsupported) {
				t.Fatalf("OpenArtifact() error = %v, want the unsupported storage format", err)
			}
			if strings.Contains(err.Error(), "manifest") {
				t.Fatalf("OpenArtifact() read the manifest of an unsupported lineage: %v", err)
			}
			if test.malformed {
				if unsupported.Malformed == nil || unsupported.Format != "" || !strings.HasPrefix(err.Error(), "unsupported artifact storage format: malformed storage.json") {
					t.Fatalf("OpenArtifact() error = %#v (%v), want a malformed envelope", unsupported, err)
				}
				return
			}
			if unsupported.Malformed != nil || unsupported.Format != test.format || !strings.HasPrefix(err.Error(), test.message) {
				t.Fatalf("OpenArtifact() error = %#v (%v), want format %q", unsupported, err, test.format)
			}
			if !strings.Contains(err.Error(), `this Gomad reads only "gomad3.artifact-storage/v1" artifacts and does not migrate others`) {
				t.Fatalf("OpenArtifact() error does not name the supported format: %v", err)
			}
		})
	}
}

func TestOpenKeepsFilesystemGuardsOnTheStorageEnvelope(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(t *testing.T, envelope string)
		want   string
	}{
		{"symbolic link", func(t *testing.T, envelope string) {
			mustDo(t, os.Rename(envelope, envelope+".moved"))
			mustDo(t, os.Symlink("storage.json.moved", envelope))
		}, "symbolic link"},
		{"mode", func(t *testing.T, envelope string) { mustDo(t, os.Chmod(envelope, 0o644)) }, "mode"},
		{"oversized", func(t *testing.T, envelope string) {
			mustDo(t, os.WriteFile(envelope, []byte(strings.Repeat(" ", maximumStorageEnvelopeBytes+1)), 0o600))
		}, "exceeds its bound"},
	} {
		t.Run(test.name, func(t *testing.T) {
			published, err := (Store{Root: t.TempDir()}).PublishArtifact(artifactInput(t))
			if err != nil {
				t.Fatal(err)
			}
			test.change(t, filepath.Join(published.Path, "storage.json"))
			if _, err := OpenArtifact(published.Path); err == nil || errors.Is(err, ErrUnsupportedFormat) || !strings.Contains(err.Error(), "read artifact storage envelope") || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("OpenArtifact() error = %v, want the %s guard", err, test.want)
			}
		})
	}
}

// A publication that finds an artifact of an earlier lineage at its
// destination refuses it rather than reusing or replacing it.
func TestPublishRefusesToReuseAnArtifactOfAnotherLineage(t *testing.T) {
	store := Store{Root: t.TempDir()}
	published, err := store.PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	mustDo(t, os.Remove(filepath.Join(published.Path, "storage.json")))
	if _, err := store.PublishArtifact(artifactInput(t)); !errors.Is(err, ErrUnsupportedFormat) || !strings.Contains(err.Error(), "existing artifact") {
		t.Fatalf("Publish() error = %v, want the existing artifact's unsupported format", err)
	}
	if _, err := os.Lstat(filepath.Join(published.Path, "storage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("publication changed the refused artifact: %v", err)
	}
}

func mustClose(t *testing.T, opened *Opened) {
	t.Helper()
	if err := opened.Close(); err != nil {
		t.Error(err)
	}
}
