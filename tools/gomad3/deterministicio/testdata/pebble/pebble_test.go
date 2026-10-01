package pebble_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cockroachdb/pebble/vfs"
)

func TestMemFS(t *testing.T) {
	fs := vfs.NewMem()
	f, err := fs.Create("old", vfs.WriteCategoryUnspecified)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("retained-memory-data")); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := fs.Rename("old", "new"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Link("new", "linked"); err != nil {
		t.Fatal(err)
	}
	f, err = fs.Open("linked")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "retained-memory-data" || f.Fd() != vfs.InvalidFd {
		t.Fatalf("MemFS data=%q, descriptor=%d", data, f.Fd())
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	f, err = fs.OpenReadWrite("new", vfs.WriteCategoryUnspecified)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("MEMORY"), 9); err != nil {
		t.Fatal(err)
	}
	updated := make([]byte, len("retained-MEMORY-data"))
	if _, err := f.ReadAt(updated, 0); err != nil {
		t.Fatal(err)
	}
	if string(updated) != "retained-MEMORY-data" {
		t.Fatalf("MemFS positional read/write = %q", updated)
	}
	if err := f.SyncData(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.SyncTo(int64(len(updated))); err != nil {
		t.Fatal(err)
	}
	if err := f.Preallocate(0, int64(len(updated))); err != nil {
		t.Fatal(err)
	}
	if err := f.Prefetch(0, int64(len(updated))); err != nil {
		t.Fatal(err)
	}
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != int64(len(updated)) {
		t.Fatalf("MemFS stat size = %d", info.Size())
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat("old"); !os.IsNotExist(err) {
		t.Fatalf("renamed old file still exists: %v", err)
	}
}

func TestOSFileConstructionRefused(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	for _, test := range []struct {
		name string
		call func() (vfs.File, error)
	}{
		{"create", func() (vfs.File, error) { return vfs.Default.Create(path, vfs.WriteCategoryUnspecified) }},
		{"open", func() (vfs.File, error) { return vfs.Default.Open(path) }},
		{"open-read-write", func() (vfs.File, error) { return vfs.Default.OpenReadWrite(path, vfs.WriteCategoryUnspecified) }},
		{"open-directory", func() (vfs.File, error) { return vfs.Default.OpenDir(dir) }},
		{"reuse", func() (vfs.File, error) {
			return vfs.Default.ReuseForWrite(path, path+"-new", vfs.WriteCategoryUnspecified)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f, err := test.call()
			if f != nil || err == nil || !strings.Contains(err.Error(), "gomad: Pebble OS-backed files are unavailable") {
				if f != nil {
					if err := f.Close(); err != nil {
						t.Error(err)
					}
				}
				t.Fatalf("OS-backed file construction: file=%v error=%v", f, err)
			}
		})
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("refused constructors changed filesystem: %v", files)
	}
}

func TestOSHardLinkRefused(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old")
	if err := os.WriteFile(old, []byte("native"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := vfs.Default.Link(old, filepath.Join(dir, "new"))
	if err == nil || !strings.Contains(err.Error(), "gomad: Pebble hard links are unavailable") {
		t.Fatalf("expected explicit hard-link refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new")); !os.IsNotExist(err) {
		t.Fatalf("refused hard link was created: %v", err)
	}
}
