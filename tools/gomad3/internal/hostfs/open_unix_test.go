//go:build unix

package hostfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenPathRejectsMultipleLinks(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "payload")
	link := filepath.Join(directory, "hard-link")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(path, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := OpenPath(path); err == nil || !strings.Contains(err.Error(), "link count") {
		t.Fatalf("OpenPath(hard link) error = %v", err)
	}
}
