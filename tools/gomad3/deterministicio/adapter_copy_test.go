package deterministicio

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCopyAdapterModuleAppliesReplacementsAndAdditions(t *testing.T) {
	source := t.TempDir()
	if err := os.Mkdir(filepath.Join(source, "internal"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "go.mod"), []byte("module example.com/adapter\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "internal", "source.go"), []byte("package internal\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "replacement")
	err := copyAdapterModule(source, destination, map[string][]byte{
		"internal/source.go": []byte("package internal\n\nconst rewritten = true\n"),
		"gomad.go":           []byte("package adapter\n"),
	}, adapterCopyLimits{Files: 3, Bytes: 256})
	if err != nil {
		t.Fatal(err)
	}
	for relative, want := range map[string]string{
		"go.mod":             "module example.com/adapter\n",
		"internal/source.go": "package internal\n\nconst rewritten = true\n",
		"gomad.go":           "package adapter\n",
	} {
		path := filepath.Join(destination, relative)
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(contents) != want {
			t.Fatalf("%s = %q, want %q", relative, contents, want)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o400 {
			t.Fatalf("%s mode = %o, want 400", relative, info.Mode().Perm())
		}
	}
}

func TestCopyAdapterModuleRejectsSymlink(t *testing.T) {
	source := t.TempDir()
	if err := os.Symlink("missing", filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}
	err := copyAdapterModule(source, filepath.Join(t.TempDir(), "replacement"), nil, adapterCopyLimits{Files: 2, Bytes: 256})
	if err == nil {
		t.Fatal("copyAdapterModule() accepted a symlink")
	}
}

func TestCopyAdapterModuleRejectsSpecialFile(t *testing.T) {
	source := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(source, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := copyAdapterModule(source, filepath.Join(t.TempDir(), "replacement"), nil, adapterCopyLimits{Files: 2, Bytes: 256})
	if err == nil {
		t.Fatal("copyAdapterModule() accepted a named pipe")
	}
}

func TestCopyAdapterModuleReturnsTypedFileCapacityError(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "one"), []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "two"), []byte("2"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := copyAdapterModule(source, filepath.Join(t.TempDir(), "replacement"), nil, adapterCopyLimits{Files: 1, Bytes: 256})
	var capacity *AdapterCapacityError
	if !errors.As(err, &capacity) || capacity.Resource != "files" {
		t.Fatalf("copyAdapterModule() error = %#v", err)
	}
}

func TestCopyAdapterModuleReturnsTypedByteCapacityError(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "source"), []byte("too large"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := copyAdapterModule(source, filepath.Join(t.TempDir(), "replacement"), nil, adapterCopyLimits{Files: 2, Bytes: 2})
	var capacity *AdapterCapacityError
	if !errors.As(err, &capacity) || capacity.Resource != "bytes" {
		t.Fatalf("copyAdapterModule() error = %#v", err)
	}
}
