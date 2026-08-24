package hostfs

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenPathValidatesTheOpenedRegularFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "payload")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, info, err := OpenPath(path)
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read error = %v, close error = %v", readErr, closeErr)
	}
	if string(data) != "payload" || info.Size() != int64(len(data)) {
		t.Fatalf("opened payload = %q, info = %#v", data, info)
	}
}

func TestOpenPathRejectsSymbolicLink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	link := filepath.Join(directory, "link")
	if err := os.WriteFile(target, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := OpenPath(link); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("OpenPath(symlink) error = %v", err)
	}
}

func TestOpenRootUsesTheSameInvariant(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "payload"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	file, info, err := OpenRoot(root, "payload")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name() != "payload" {
		t.Fatalf("opened info = %#v", info)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
