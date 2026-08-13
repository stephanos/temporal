//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package filelock

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
