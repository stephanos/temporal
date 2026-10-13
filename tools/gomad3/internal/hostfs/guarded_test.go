//go:build unix

package hostfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func openTestRoot(t *testing.T, directory string) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	return root
}

func TestOpenRootWritableWritesTheValidatedFileWithoutCreating(t *testing.T) {
	directory := t.TempDir()
	root := openTestRoot(t, directory)
	if _, _, err := OpenRootWritable(root, "missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("OpenRootWritable(missing) error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(directory, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("writable open created a file: %v", err)
	}
	path := filepath.Join(directory, "payload")
	if err := os.WriteFile(path, []byte("payload"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	file, info, err := OpenRootWritable(root, "payload")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 || info.Size() != 7 {
		t.Fatalf("opened info = %v %d", info.Mode(), info.Size())
	}
	if _, err := file.WriteAt([]byte("P"), 0); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "Payload" {
		t.Fatalf("written payload = %q, %v", data, err)
	}
	if after, err := os.Lstat(path); err != nil || after.Mode().Perm() != 0o640 {
		t.Fatalf("writable open changed mode: %v, %v", after, err)
	}
}

func TestOpenRootWritableRejectsLinksAndNonregularFiles(t *testing.T) {
	directory := t.TempDir()
	root := openTestRoot(t, directory)
	if err := os.WriteFile(filepath.Join(directory, "target"), []byte("target"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(directory, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "directory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := OpenRootWritable(root, "link"); !errors.Is(err, ErrSymbolicLink) {
		t.Fatalf("OpenRootWritable(symlink) error = %v", err)
	}
	if _, _, err := OpenRootWritable(root, "directory"); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("OpenRootWritable(directory) error = %v", err)
	}
	if _, _, err := OpenRootWritable(nil, "target"); err == nil {
		t.Fatal("nil root accepted")
	}
	if data, err := os.ReadFile(filepath.Join(directory, "target")); err != nil || string(data) != "target" {
		t.Fatalf("rejected open changed the link target: %q, %v", data, err)
	}
}

func TestCreateRootExclusiveSetsPrivateModeDespiteUmask(t *testing.T) {
	directory := t.TempDir()
	root := openTestRoot(t, directory)
	previous := syscall.Umask(0o777)
	file, info, err := CreateRootExclusive(root, "created", 0o600)
	syscall.Umask(previous)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode() != 0o600 {
		t.Fatalf("created mode = %v", info.Mode())
	}
	if _, err := io.WriteString(file, "created"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateRootExclusive(root, "created", 0o600); !errors.Is(err, os.ErrExist) {
		t.Fatalf("second exclusive create error = %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(directory, "created")); err != nil || string(data) != "created" {
		t.Fatalf("exclusive create changed existing file: %q, %v", data, err)
	}
	if err := os.Symlink("created", filepath.Join(directory, "link")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateRootExclusive(root, "link", 0o600); !errors.Is(err, os.ErrExist) {
		t.Fatalf("exclusive create through symlink error = %v", err)
	}
	if _, _, err := CreateRootExclusive(nil, "other", 0o600); err == nil {
		t.Fatal("nil root accepted")
	}
}

func TestSyncDirectoriesFlushPinnedAndNamedDirectories(t *testing.T) {
	directory := t.TempDir()
	if err := SyncRoot(openTestRoot(t, directory)); err != nil {
		t.Fatal(err)
	}
	if err := SyncRoot(nil); err == nil {
		t.Fatal("nil root accepted")
	}
	if err := SyncDirectory(directory); err != nil {
		t.Fatal(err)
	}
	if err := SyncDirectory(filepath.Join(directory, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("SyncDirectory(missing) error = %v", err)
	}
}

func TestOpenRootGrowingAdmitsOnlyGrowthBetweenLstatAndOpen(t *testing.T) {
	directory := t.TempDir()
	root := openTestRoot(t, directory)
	path := filepath.Join(directory, "log")
	for _, test := range []struct {
		name     string
		contents string
		growing  bool
		accepted bool
	}{
		{"unchanged", "four", true, true},
		{"grown", "fourmore", true, true},
		{"shrunk", "fo", true, false},
		{"grown-exact", "fourmore", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte("four"), 0o600); err != nil {
				t.Fatal(err)
			}
			open := func(name string) (*os.File, error) {
				if err := os.WriteFile(path, []byte(test.contents), 0o600); err != nil {
					return nil, err
				}
				return root.Open(name)
			}
			file, info, err := openRegularSized("log", root.Lstat, open, test.growing)
			if (err == nil) != test.accepted {
				t.Fatalf("open after %s = %v", test.name, err)
			}
			if err != nil {
				return
			}
			if info.Size() != int64(len(test.contents)) {
				t.Fatalf("opened size = %d", info.Size())
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
	file, _, err := OpenRootGrowing(root, "log")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("x")); err == nil {
		t.Fatal("growing open is writable")
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := OpenRootGrowing(nil, "log"); err == nil {
		t.Fatal("nil root accepted")
	}
}
