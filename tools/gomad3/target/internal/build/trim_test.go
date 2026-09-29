package build

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrimCacheDropsLeastRecentlyUsedEntriesToTheBound(t *testing.T) {
	cache := t.TempDir()
	write := func(name string, size int, age time.Duration) string {
		t.Helper()
		path := filepath.Join(cache, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
		used := time.Now().Add(-age)
		if err := os.Chtimes(path, used, used); err != nil {
			t.Fatal(err)
		}
		return path
	}
	oldest := write("aa/old-d", 300, 3*time.Hour)
	middle := write("bb/middle-a", 300, 2*time.Hour)
	newest := write("cc/new-d", 300, time.Hour)
	bookkeeping := write("trim.txt", 300, 4*time.Hour)
	if err := TrimCache(cache, 900); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{oldest, middle, newest, bookkeeping} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("trim within the bound removed %s: %v", path, err)
		}
	}
	if err := TrimCache(cache, 400); err != nil {
		t.Fatal(err)
	}
	for path, wantRemoved := range map[string]bool{oldest: true, middle: true, newest: false, bookkeeping: false} {
		_, err := os.Stat(path)
		if removed := os.IsNotExist(err); removed != wantRemoved {
			t.Fatalf("%s removed = %t, want %t (%v)", path, removed, wantRemoved, err)
		}
	}
}
