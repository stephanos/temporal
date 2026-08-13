package gomadfs

import (
	"errors"
	"io"
	"strings"
	"syscall"
	"testing"
)

func TestFilesystemEnforcesPathAndFileBounds(t *testing.T) {
	filesystem := New()
	if _, _, err := Normalize("/" + strings.Repeat("x", maximumPathBytes)); !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("Normalize() error = %v", err)
	}
	file, err := filesystem.Open("/file", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maximumFileBytes + 1); !errors.Is(err, syscall.EFBIG) {
		t.Fatalf("Truncate() error = %v", err)
	}
}

func TestFilesystemAccountsReleasedBytes(t *testing.T) {
	filesystem := New()
	file, err := filesystem.Open("/file", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(make([]byte, 1024)); err != nil {
		t.Fatal(err)
	}
	if filesystem.usedBytes != 1024 {
		t.Fatalf("used bytes = %d", filesystem.usedBytes)
	}
	if err := file.Truncate(1); err != nil {
		t.Fatal(err)
	}
	if filesystem.usedBytes != 1 {
		t.Fatalf("used bytes after truncate = %d", filesystem.usedBytes)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if filesystem.openHandles != 0 {
		t.Fatalf("open handles = %d", filesystem.openHandles)
	}
}

func TestFilesystemFailsMountedMutationsClosed(t *testing.T) {
	filesystem := New()
	filesystem.SetLoader(func(name string) (LoadEntry, MountStatus, error) {
		if name == "/mounted" {
			return LoadEntry{Mode: 0o755, Kind: KindDirectory}, MountOK, nil
		}
		return LoadEntry{}, MountUnmounted, nil
	})
	if err := filesystem.Mkdir("/mounted/new", 0o700); !errors.Is(err, syscall.EROFS) {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := filesystem.Rename("/mounted", "/renamed"); !errors.Is(err, syscall.EXDEV) {
		t.Fatalf("Rename() error = %v", err)
	}
	file, err := filesystem.Open("/file", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Rename("/file", "/mounted"); !errors.Is(err, syscall.EXDEV) {
		t.Fatalf("rename over mount error = %v", err)
	}
}

func TestFilesystemWorkingDirectoryAndMetadata(t *testing.T) {
	filesystem := New()
	logicalTime := int64(100)
	filesystem.SetClock(func() int64 { return logicalTime })
	if err := filesystem.MkdirAll("/workspace/nested", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Chdir("/workspace"); err != nil {
		t.Fatal(err)
	}
	if got := filesystem.Getwd(); got != "/workspace" {
		t.Fatalf("Getwd() = %q", got)
	}
	file, err := filesystem.Open("nested/file", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	logicalTime = 110
	if err := file.Chmod(0o640); err != nil {
		t.Fatal(err)
	}
	if err := file.Chtimes(123); err != nil {
		t.Fatal(err)
	}
	entry, err := filesystem.Stat("nested/file")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Mode != 0o640 || entry.ModTime != 123 {
		t.Fatalf("Stat() = %#v", entry)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Chdir("nested/file"); !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("Chdir() error = %v", err)
	}
}

func TestFilesystemRejectsRenameIntoDescendant(t *testing.T) {
	filesystem := New()
	if err := filesystem.MkdirAll("/tree/nested", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Rename("/tree", "/tree/nested/moved"); !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("Rename() error = %v", err)
	}
	if _, err := filesystem.Stat("/tree/nested"); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestFilesystemRemoveAllRetainsOpenNode(t *testing.T) {
	filesystem := New()
	if err := filesystem.MkdirAll("/tree/nested", 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.Open("/tree/nested/file", OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("contents")); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.RemoveAll("/tree"); err != nil {
		t.Fatal(err)
	}
	if _, err := filesystem.Stat("/tree"); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("Stat() error = %v", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, len("contents"))
	if _, err := file.Read(buffer); err != nil {
		t.Fatal(err)
	}
	if string(buffer) != "contents" {
		t.Fatalf("Read() = %q", buffer)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if filesystem.usedBytes != 0 {
		t.Fatalf("used bytes = %d", filesystem.usedBytes)
	}
}
