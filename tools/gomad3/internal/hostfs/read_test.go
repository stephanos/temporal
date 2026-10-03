package hostfs

import (
	"os"
	"path/filepath"
	"testing"
)

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
