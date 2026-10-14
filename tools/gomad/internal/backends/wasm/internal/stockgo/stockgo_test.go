package stockgo

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func TestSharedCopyIsCreatedOnceAndReused(t *testing.T) {
	cache := t.TempDir()
	t.Setenv(Environment, cache)
	command := Command()
	if _, err := exec.Command(command, "version").Output(); err != nil {
		t.Skipf("stock go command unavailable: %v", err)
	}
	// Concurrent first callers from one process share the single copy.
	roots := make([]string, 4)
	var group sync.WaitGroup
	for index := range roots {
		group.Add(1)
		go func() {
			defer group.Done()
			roots[index] = Root(t)
		}()
	}
	group.Wait()
	for _, root := range roots[1:] {
		if root != roots[0] {
			t.Fatalf("shared roots differ: %q %q", roots[0], root)
		}
	}
	root := roots[0]
	if filepath.Dir(filepath.Dir(filepath.Dir(root))) != cache {
		t.Fatalf("root %q is not beneath the configured cache %q", root, cache)
	}
	if _, err := os.Stat(filepath.Join(root, "bin", "go")); err != nil {
		t.Fatal(err)
	}
	stampPath := filepath.Join(filepath.Dir(root), "stamp.json")
	before, err := os.Stat(stampPath)
	if err != nil {
		t.Fatal(err)
	}
	if stale, _ := filepath.Glob(filepath.Dir(root) + ".staging-*"); len(stale) != 0 {
		t.Fatalf("staging left behind: %v", stale)
	}
	// A fresh process-level resolution finds the published copy without copying again.
	shared.Delete(command)
	if again, err := Shared(command); err != nil || again != root {
		t.Fatalf("reuse: %q %v", again, err)
	}
	after, err := os.Stat(stampPath)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("reuse rewrote the stamp")
	}
	// The copied compiler reports the same version as its source.
	want, err := exec.Command(command, "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	copied := exec.Command(filepath.Join(root, "bin", "go"), "version")
	copied.Env = append(os.Environ(), "GOROOT="+root, "GOTOOLCHAIN=local")
	got, err := copied.Output()
	if err != nil || string(got) != string(want) {
		t.Fatalf("copied compiler version %q %v, want %q", got, err, want)
	}
}

func TestCacheDirectoryRequiresAbsoluteOverride(t *testing.T) {
	t.Setenv(Environment, "relative/cache")
	if _, err := cacheDirectory(); err == nil {
		t.Fatal("accepted a relative cache override")
	}
}
