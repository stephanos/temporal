package adapterregen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleEnvelopeStaysOutsidePublicationAndDetectsChangedInputs(t *testing.T) {
	checkout := t.TempDir()
	root := filepath.Join(checkout, "tools", "gomad")
	if err := os.MkdirAll(filepath.Join(root, "internal/backends/native"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"go.mod", "go.sum", "tools/gomad/go.mod", "tools/gomad/go.sum", "tools/gomad/internal/backends/native/go.mod", "tools/gomad/internal/backends/native/go.sum"} {
		if err := os.WriteFile(filepath.Join(checkout, relative), []byte(relative+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	work := t.TempDir()
	stage := filepath.Join(work, "tools", "gomad")
	read, err := copyCheckout(root, stage)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := copyModuleEnvelope(root, stage)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		original, err := os.ReadFile(filepath.Join(checkout, name))
		if err != nil {
			t.Fatal(err)
		}
		copied, err := os.ReadFile(filepath.Join(work, name))
		if err != nil || !bytes.Equal(original, copied) {
			t.Fatalf("outer %s = %q, %v; want original %q", name, copied, err, original)
		}
	}
	entries, err := stagedChanges(stage, read)
	if err != nil || len(entries) != 0 {
		t.Fatalf("metadata entered publication: %+v, %v", entries, err)
	}
	if err := unchanged(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkout, "go.mod"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := unchanged(); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("changed metadata = %v", err)
	}
	if _, err := copyModuleEnvelope(root, filepath.Join(work, "root")); err == nil {
		t.Fatal("accepted a stage whose relative replacement escapes its module envelope")
	}
}
