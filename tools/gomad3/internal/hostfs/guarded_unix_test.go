//go:build unix

package hostfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardedOpenRejectsHardLinkAliases(t *testing.T) {
	directory := t.TempDir()
	root := openTestRoot(t, directory)
	path := filepath.Join(directory, "payload")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := SingleLink(info); err != nil {
		t.Fatalf("single link rejected: %v", err)
	}
	if err := os.Link(path, filepath.Join(directory, "alias")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := OpenRootWritable(root, "payload"); err == nil || !strings.Contains(err.Error(), "link count") {
		t.Fatalf("OpenRootWritable(hard link) error = %v", err)
	}
	if info, err = os.Lstat(path); err != nil {
		t.Fatal(err)
	}
	if err := SingleLink(info); err == nil {
		t.Fatal("hard-linked file reported as single link")
	}
}
