//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package hostfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTryOwnsPrivateExclusiveLockLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.lock")
	first, err := Try(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("lock mode = %#o", info.Mode().Perm())
	}
	if _, err := Try(path); !errors.Is(err, ErrContended) {
		t.Fatalf("second Try() error = %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	second, err := Try(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestTryRejectsSymbolicLink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Try(link); !errors.Is(err, ErrSymbolicLink) {
		t.Fatalf("Try(symlink) error = %v", err)
	}
}

func TestSharedLocksExcludeTryButNotEachOther(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.lock")
	first, err := Shared(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Shared(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Try(path); !errors.Is(err, ErrContended) {
		t.Fatalf("Try() under shared holders error = %v", err)
	}
	if err := errors.Join(first.Release(), second.Release()); err != nil {
		t.Fatal(err)
	}
	exclusive, err := Try(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := exclusive.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestTryDescriptorLocksTheHeldFileAndExcludesPathLocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.lock")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	held, err := TryDescriptor(file)
	if err != nil {
		t.Fatal(errors.Join(err, file.Close()))
	}
	if _, err := Try(path); !errors.Is(err, ErrContended) {
		t.Fatalf("Try() under descriptor lock error = %v", err)
	}
	other, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := TryDescriptor(other); !errors.Is(err, ErrContended) {
		t.Fatalf("second TryDescriptor() error = %v", err)
	}
	if _, err := other.Stat(); err != nil {
		t.Fatalf("failed TryDescriptor closed the caller's file: %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Release left the owned descriptor open: %v", err)
	}
	reacquired, err := TryDescriptor(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := reacquired.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := TryDescriptor(nil); err == nil {
		t.Fatal("nil descriptor accepted")
	}
}
