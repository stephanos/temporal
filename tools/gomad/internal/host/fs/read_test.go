package hostfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadAcceptsCompleteBoundedBytesAndRejectsOverflowOrLinks(t *testing.T) {
	name := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(name, []byte("frozen"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := ReadBounded(name, 6)
	if err != nil || string(data) != "frozen" {
		t.Fatalf("bounded input = %q, %v", data, err)
	}
	if _, err := ReadBounded(name, 5); err == nil {
		t.Fatal("oversize input accepted")
	}
	link := name + "-link"
	if err := os.Symlink(name, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBounded(link, 6); err == nil {
		t.Fatal("symlink input accepted")
	}
	if _, _, err := OpenPath(link); !errors.Is(err, ErrSymbolicLink) {
		t.Fatalf("symlink open: %v", err)
	}
}

func TestReadBoundedReadsOnlyBoundedRegularFiles(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "file")
	if err := os.WriteFile(path, []byte("four"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, err := ReadBounded(path, 4); err != nil || string(data) != "four" {
		t.Fatalf("ReadBounded() = %q, %v", data, err)
	}
	if _, err := ReadBounded(path, 3); err == nil || err.Error() != path+" exceeds its size bound" {
		t.Fatalf("ReadBounded() over its bound error = %v", err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBounded(link, 4); err == nil || err.Error() != link+" is not a regular file" {
		t.Fatalf("ReadBounded() through a symbolic link error = %v", err)
	}
}
