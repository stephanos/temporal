package safefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplacePublishesFileWithRequestedMode(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "generated")
	path := filepath.Join(directory, "artifact.go")
	if err := Replace(path, []byte("generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "generated\n" || info.Mode().Perm() != 0o644 || len(entries) != 1 {
		t.Fatalf("contents = %q, mode = %o, entries = %d", contents, info.Mode().Perm(), len(entries))
	}
}
