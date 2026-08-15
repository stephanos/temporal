package deterministicio

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestCaptureCachesRegularFileContents(t *testing.T) {
	source := t.TempDir()
	file := filepath.Join(source, "schema.sql")
	if err := os.WriteFile(file, []byte("first"), 0o640); err != nil {
		t.Fatal(err)
	}
	broker, err := Prepare([]Mapping{{Source: source, Target: "/schema"}}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := broker.Close(); err != nil {
			t.Error(err)
		}
	})

	first, err := broker.Lookup("/schema/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("second"), 0o640); err != nil {
		t.Fatal(err)
	}
	second, err := broker.Lookup("/schema/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if string(first.Data) != "first" || string(second.Data) != "first" || first.Mode != 0o640 || first.Kind != KindFile {
		t.Fatalf("captured entries = %#v, %#v", first, second)
	}
	if snapshot := broker.Captured(); len(snapshot.Entries) != 1 || snapshot.TotalBytes != 5 || snapshot.Requests != 2 {
		t.Fatalf("Captured() = %#v", snapshot)
	}
}

func TestCaptureCachesMissingMountedPath(t *testing.T) {
	source := t.TempDir()
	broker, err := Prepare([]Mapping{{Source: source, Target: "/schema"}}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close() })

	for range 2 {
		if _, err := broker.Lookup("/schema/missing.sql"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Lookup() error = %v", err)
		}
	}
	snapshot := broker.Captured()
	if len(snapshot.NotExist) != 1 || snapshot.NotExist[0] != "/schema/missing.sql" || snapshot.Requests != 2 {
		t.Fatalf("Captured() = %#v", snapshot)
	}
}

func TestCaptureReturnsSortedDirectoryEntries(t *testing.T) {
	source := t.TempDir()
	for _, name := range []string{"z", "a"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	broker, err := Prepare([]Mapping{{Source: source, Target: "/mounted"}}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	entry, err := broker.Lookup("/mounted")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Kind != KindDirectory || len(entry.Children) != 2 || entry.Children[0].Name != "a" || entry.Children[1].Name != "z" {
		t.Fatalf("directory entry = %#v", entry)
	}
}

func TestCaptureRejectsSymlinksHardLinksAndTraversal(t *testing.T) {
	source := t.TempDir()
	file := filepath.Join(source, "file")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(file, filepath.Join(source, "hard")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("file", filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}
	broker, err := Prepare([]Mapping{{Source: source, Target: "/mounted"}}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	for _, path := range []string{"/mounted/link", "/mounted/hard", "/mounted/../file"} {
		if _, err := broker.Lookup(path); err == nil {
			t.Fatalf("Lookup(%q) succeeded", path)
		}
	}
}

func TestCaptureEnforcesLimitsBeforePublishingEntry(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "file"), []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	limits := DefaultLimits()
	limits.SingleFileBytes = 4
	broker, err := Prepare([]Mapping{{Source: source, Target: "/mounted"}}, limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	if _, err := broker.Lookup("/mounted/file"); !errors.Is(err, ErrCapacity) {
		t.Fatalf("Lookup() error = %v", err)
	}
	if snapshot := broker.Captured(); len(snapshot.Entries) != 0 || snapshot.TotalBytes != 0 {
		t.Fatalf("Captured() after rejected lookup = %#v", snapshot)
	}
}

func TestCaptureSerializesConcurrentFirstLookup(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "file"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	broker, err := Prepare([]Mapping{{Source: source, Target: "/mounted"}}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	var wait sync.WaitGroup
	wait.Add(8)
	for range 8 {
		go func() {
			defer wait.Done()
			entry, lookupErr := broker.Lookup("/mounted/file")
			if lookupErr != nil || string(entry.Data) != "data" {
				t.Errorf("Lookup() = %#v, %v", entry, lookupErr)
			}
		}()
	}
	wait.Wait()
	if snapshot := broker.Captured(); len(snapshot.Entries) != 1 || snapshot.Requests != 8 {
		t.Fatalf("Captured() = %#v", snapshot)
	}
}
