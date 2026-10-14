package qualification

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

func TestResultPublicationIsCompleteAndConcurrentNoClobber(t *testing.T) {
	root := t.TempDir()
	data := bytes.Repeat([]byte("measured result\n"), 16384)
	final := filepath.Join(root, "wasm-qualifications", "v1", "result-"+strings.TrimPrefix(string(record.HashBytes(data)), "sha256:")+".json")
	start, done := make(chan struct{}), make(chan struct{})
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			path, err := publishResultBytes(root, data)
			if err != nil || path != final {
				t.Errorf("same-content publication: %s %v", path, err)
			}
		}()
	}
	close(start)
	defer workers.Wait()
	go func() { workers.Wait(); close(done) }()
	for {
		read, err := os.ReadFile(final)
		if err == nil && !bytes.Equal(read, data) {
			t.Fatal("partial final content visible before publication")
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		select {
		case <-done:
			info, err := os.Stat(final)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("published file is not private: %v %v", info, err)
			}
			entries, err := os.ReadDir(filepath.Dir(final))
			if err != nil || len(entries) != 1 {
				t.Fatalf("staging files leaked: %v %v", entries, err)
			}
			return
		default:
		}
	}
}

func TestResultPublicationRejectsRedirectedNamespace(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "wasm-qualifications")); err != nil {
		t.Fatal(err)
	}
	if _, err := publishResultBytes(root, []byte("result")); err == nil {
		t.Fatal("symbolic-link publication namespace accepted")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("redirected namespace was written: %v %v", entries, err)
	}
}

func TestResultPublicationPreservesPartialConflictsAndRejectsSymlinks(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		root := t.TempDir()
		data := []byte("complete measured result\n")
		directory := filepath.Join(root, "wasm-qualifications", "v1")
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
		final := filepath.Join(directory, "result-"+strings.TrimPrefix(string(record.HashBytes(data)), "sha256:")+".json")
		outside := filepath.Join(root, "outside")
		if symlink {
			if err := os.WriteFile(outside, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, final); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(final, data[:3], 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := publishResultBytes(root, data); err == nil {
			t.Fatal("partial conflict or symlink accepted as immutable publication")
		}
		info, err := os.Lstat(final)
		if err != nil || (info.Mode()&os.ModeSymlink != 0) != symlink {
			t.Fatalf("conflicting entry replaced: %v %v", info, err)
		}
		retained, err := os.ReadFile(final)
		if err != nil || !bytes.Equal(retained, map[bool][]byte{false: data[:3], true: data}[symlink]) {
			t.Fatal("conflicting bytes overwritten")
		}
	}
}

func TestResultPublicationIgnoresAbandonedStagingWithoutPublishingIt(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "wasm-qualifications", "v1")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	abandoned := filepath.Join(directory, ".result-abandoned.partial")
	if err := os.WriteFile(abandoned, []byte("interrupted partial"), 0600); err != nil {
		t.Fatal(err)
	}
	data := []byte("complete result\n")
	path, err := publishResultBytes(root, data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal("abandoned staging affected complete publication")
	}
	got, err = os.ReadFile(abandoned)
	if err != nil || string(got) != "interrupted partial" {
		t.Fatal("foreign interrupted staging was overwritten or removed")
	}
}

func TestResultPublicationRejectsPersistentHardlink(t *testing.T) {
	root := t.TempDir()
	data := []byte("complete measured result\n")
	path, err := publishResultBytes(root, data)
	if err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(root, "foreign-hardlink")
	if err := os.Link(path, foreign); err != nil {
		t.Fatal(err)
	}
	if _, err := publishResultBytes(root, data); err == nil {
		t.Fatal("persistent second link accepted as complete immutable publication")
	}
	for _, retained := range []string{path, foreign} {
		got, err := os.ReadFile(retained)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("foreign hardlink or publication changed: %s %v", retained, err)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("own staging leaked: %v %v", entries, err)
	}
}
