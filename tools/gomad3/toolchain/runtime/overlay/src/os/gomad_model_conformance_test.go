package os_test

import (
	"errors"
	"internal/poll"
	"io"
	"os"
	"runtime"
	"syscall"
	"testing"
)

func TestGomadClosedFileHasPublicSentinel(t *testing.T) {
	file, err := os.OpenFile(t.TempDir()+"/file", os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	calls := []struct {
		name string
		call func() error
	}{
		{"close", file.Close},
		{"read", func() error { _, err := file.Read(make([]byte, 1)); return err }},
		{"read-at", func() error { _, err := file.ReadAt(make([]byte, 1), 0); return err }},
		{"write", func() error { _, err := file.Write([]byte("x")); return err }},
		{"write-at", func() error { _, err := file.WriteAt([]byte("x"), 0); return err }},
		{"seek", func() error { _, err := file.Seek(0, io.SeekStart); return err }},
		{"truncate", func() error { return file.Truncate(0) }},
		{"chmod", func() error { return file.Chmod(0600) }},
		{"sync", file.Sync},
		{"stat", func() error { _, err := file.Stat(); return err }},
	}
	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			err := call.call()
			if !errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.EBADF) {
				t.Fatalf("public closed error=%T %v", err, err)
			}
		})
	}
}

func TestGomadWrongFileAccessModeIsNotClosed(t *testing.T) {
	path := t.TempDir() + "/file"
	if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []int{os.O_RDONLY, os.O_WRONLY} {
		file, err := os.OpenFile(path, mode, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if mode == os.O_RDONLY {
			_, err = file.Write([]byte("x"))
		} else {
			_, err = file.Read(make([]byte, 1))
		}
		if !errors.Is(err, syscall.EBADF) || errors.Is(err, os.ErrClosed) {
			t.Fatalf("mode %d error=%T %v", mode, err, err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGomadClosedDirectoryMatchesNative(t *testing.T) {
	directory, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Close(); err != nil {
		t.Fatal(err)
	}
	for _, call := range []struct {
		name string
		call func() error
	}{
		{"chdir", directory.Chdir},
		{"read-dir", func() error { _, err := directory.ReadDir(-1); return err }},
		{"readdir", func() error { _, err := directory.Readdir(-1); return err }},
		{"readdirnames", func() error { _, err := directory.Readdirnames(-1); return err }},
	} {
		t.Run(call.name, func(t *testing.T) {
			err := call.call()
			closed := call.name == "chdir"
			if errors.Is(err, os.ErrClosed) != closed || errors.Is(err, poll.ErrFileClosing) == closed || errors.Is(err, syscall.EBADF) {
				t.Fatalf("closed directory error=%T %v", err, err)
			}
			var pathError *os.PathError
			operation := "readdirent"
			if runtime.GOOS == "darwin" {
				operation = ""
			}
			if closed {
				operation = "chdir"
			}
			if !errors.As(err, &pathError) || pathError.Op != operation || pathError.Path != directory.Name() {
				t.Fatalf("closed directory path error=%#v, want operation %q path %q", pathError, operation, directory.Name())
			}
		})
	}
}
