package sourceinventory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The digest was captured from the implementation target owned before this
// package; adapter profiles pin inventories computed the same way.
func TestDigestPinsInventoryEncoding(t *testing.T) {
	root := t.TempDir()
	writeInventoryFile(t, root, "go.mod", "module example.com/adapter\n\ngo 1.26\n", 0o600)
	writeInventoryFile(t, root, "adapter.go", "package adapter\n", 0o600)
	writeInventoryFile(t, root, filepath.Join("internal", "deep", "z.go"), "package deep\n\nconst Z = 1\n", 0o600)
	writeInventoryFile(t, root, filepath.Join("internal", "a.s"), "TEXT ·a(SB),$0\n", 0o400)
	got, err := Digest(root)
	if err != nil {
		t.Fatal(err)
	}
	const want = "sha256:624ffd10d3b0e4126993be4d4c60de5dba62a7bc08df9a1b9f4db1a0260c07b3"
	if got != want {
		t.Fatalf("Digest() = %s, want %s", got, want)
	}
}

func TestDigestReturnsTypedCapacityError(t *testing.T) {
	root := t.TempDir()
	writeInventoryFile(t, root, "one.go", "1", 0o600)
	writeInventoryFile(t, root, "two.go", "2", 0o600)
	for _, test := range []struct {
		name         string
		maximumFiles int
		maximumBytes uint64
		want         CapacityError
	}{
		{name: "files", maximumFiles: 1, maximumBytes: 2, want: CapacityError{Resource: "files", Limit: 1}},
		{name: "bytes", maximumFiles: 2, maximumBytes: 1, want: CapacityError{Resource: "bytes", Limit: 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := digest(root, test.maximumFiles, test.maximumBytes)
			var capacity *CapacityError
			if !errors.As(err, &capacity) || *capacity != test.want {
				t.Fatalf("digest() error = %#v, want %#v", err, test.want)
			}
		})
	}
}

func TestDigestRejectsUnsafeOrEmptyInventories(t *testing.T) {
	for _, test := range []struct {
		name string
		make func(*testing.T, string)
		want string
	}{
		{name: "empty", make: func(*testing.T, string) {}, want: "adapter source inventory is empty"},
		{name: "symbolic link", make: func(t *testing.T, root string) {
			writeInventoryFile(t, root, "target.go", "package adapter\n", 0o600)
			if err := os.Symlink(filepath.Join(root, "target.go"), filepath.Join(root, "link.go")); err != nil {
				t.Fatal(err)
			}
		}, want: "adapter source inventory contains a symbolic link"},
		{name: "missing root", make: func(t *testing.T, root string) {
			if err := os.Remove(root); err != nil {
				t.Fatal(err)
			}
		}, want: "no such file or directory"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "module")
			if err := os.Mkdir(root, 0o700); err != nil {
				t.Fatal(err)
			}
			test.make(t, root)
			if _, err := Digest(root); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Digest() error = %v, want %q", err, test.want)
			}
		})
	}
	if _, err := digest(t.TempDir(), 0, 1); err == nil || err.Error() != "adapter source inventory limits must be positive" {
		t.Fatalf("digest() with zero files error = %v", err)
	}
}

func writeInventoryFile(t *testing.T, root, name, contents string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatal(err)
	}
}
