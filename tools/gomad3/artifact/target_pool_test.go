package artifact

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

// seededInput is artifactInput with the record identity of the given seed, so
// record-keyed stores keep one artifact per seed of the same target.
func seededInput(t *testing.T, seed uint64) Publication {
	t.Helper()
	input := artifactInput(t)
	input.Record.Seed = record.Uint64String(seed)
	input.Record.Environment[1].Value = fmt.Sprint(seed)
	return input
}

func publishSeed(t *testing.T, store Store, seed uint64) Artifact {
	t.Helper()
	published, err := store.PublishArtifact(seededInput(t, seed))
	if err != nil {
		t.Fatal(err)
	}
	return published
}

// distinctFiles counts the files the paths name, counting hard links to one
// file once.
func distinctFiles(t *testing.T, paths []string) int {
	t.Helper()
	var distinct []os.FileInfo
next:
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, seen := range distinct {
			if os.SameFile(seen, info) {
				continue next
			}
		}
		distinct = append(distinct, info)
	}
	return len(distinct)
}

func poolEntries(t *testing.T, pool string) []string {
	t.Helper()
	entries, err := os.ReadDir(pool)
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, filepath.Join(pool, entry.Name()))
	}
	return paths
}

func TestPublishSharesOneTargetAcrossTheStoresOfOnePool(t *testing.T) {
	owner := t.TempDir()
	pool := TargetPool(owner)
	staged := filepath.Join(owner, "v1", "campaign-b", ".partial", "round")
	stores := []Store{
		{Root: filepath.Join(owner, "v1", "campaign-a", "failures"), Key: StoreKeyRecord, TargetPool: pool},
		{Root: filepath.Join(owner, "v1", "campaign-a", "successes"), Key: StoreKeyRecord, TargetPool: pool},
		{Root: filepath.Join(staged, "successes"), Key: StoreKeyRecord, TargetPool: pool},
	}
	var artifacts []string
	for _, store := range stores {
		for _, seed := range []uint64{7, 8} {
			published := publishSeed(t, store, seed)
			if published.TargetSharing != TargetShared {
				t.Fatalf("target sharing = %q, want %q", published.TargetSharing, TargetShared)
			}
			again := publishSeed(t, store, seed)
			if again.Path != published.Path || again.TargetSharing != TargetShared {
				t.Fatalf("republished artifact = %s (%q), want %s shared", again.Path, again.TargetSharing, published.Path)
			}
			artifacts = append(artifacts, published.Path)
		}
	}
	committed := filepath.Join(owner, "v1", "campaign-b", "round-1")
	if err := os.Rename(staged, committed); err != nil {
		t.Fatal(err)
	}
	for index, path := range artifacts {
		if strings.HasPrefix(path, staged) {
			artifacts[index] = filepath.Join(committed, strings.TrimPrefix(path, staged))
		}
	}
	entries := poolEntries(t, pool)
	if len(entries) != 1 {
		t.Fatalf("pool entries = %v, want one", entries)
	}
	targets := entries
	for _, path := range artifacts {
		opened, err := OpenArtifact(path)
		if err != nil {
			t.Fatal(err)
		}
		copied := filepath.Join(t.TempDir(), "target")
		if err := opened.CopyPayload("target", copied, 0o500); err != nil {
			t.Fatal(err)
		}
		if content, err := os.ReadFile(copied); err != nil || string(content) != "target bytes" {
			t.Fatalf("copied target = %q, %v", content, err)
		}
		if mode := listedFile(opened, "target").Mode; mode != "0700" {
			t.Fatalf("recorded target mode = %s, want 0700", mode)
		}
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
		info, err := os.Lstat(filepath.Join(path, "target"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("target mode = %#o, want 0700", info.Mode().Perm())
		}
		targets = append(targets, filepath.Join(path, "target"))
	}
	if len(artifacts) != 6 || distinctFiles(t, targets) != 1 {
		t.Fatalf("%d artifacts hold %d target files, want 6 artifacts and one file", len(artifacts), distinctFiles(t, targets))
	}

	private := publishSeed(t, Store{Root: filepath.Join(owner, "private"), Key: StoreKeyRecord}, 7)
	if private.TargetSharing != TargetPrivate || distinctFiles(t, append(targets, filepath.Join(private.Path, "target"))) != 2 {
		t.Fatalf("store without a pool published %q", private.TargetSharing)
	}
}

func TestPublishConcurrentlyIntoOnePoolLeavesOneEntry(t *testing.T) {
	owner := t.TempDir()
	store := Store{Root: filepath.Join(owner, "failures"), Key: StoreKeyRecord, TargetPool: TargetPool(owner)}
	const publishers = 8
	inputs := make([]Publication, publishers)
	for index := range inputs {
		inputs[index] = seededInput(t, uint64(index))
	}
	published := make([]Artifact, publishers)
	failures := make([]error, publishers)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := range inputs {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			published[index], failures[index] = store.PublishArtifact(inputs[index])
		}()
	}
	close(start)
	wait.Wait()
	entries := poolEntries(t, store.TargetPool)
	if len(entries) != 1 {
		t.Fatalf("pool entries = %v, want one", entries)
	}
	targets := entries
	for index, err := range failures {
		if err != nil {
			t.Fatal(err)
		}
		if published[index].TargetSharing != TargetShared {
			t.Fatalf("publisher %d target sharing = %q", index, published[index].TargetSharing)
		}
		if _, err := OpenArtifact(published[index].Path); err != nil {
			t.Fatal(err)
		}
		targets = append(targets, filepath.Join(published[index].Path, "target"))
	}
	if distinctFiles(t, targets) != 1 {
		t.Fatalf("concurrent publishers hold %d target files, want one", distinctFiles(t, targets))
	}
}

func TestDamagedSharedTargetFailsOpenAndLaterPublication(t *testing.T) {
	tests := []struct {
		name string
		// damage changes the artifact or its pool entry.
		damage func(t *testing.T, artifactPath, entry string)
		// poisoned says the pool entry itself no longer matches its identity.
		poisoned bool
	}{
		{name: "altered", poisoned: true, damage: func(t *testing.T, _, entry string) {
			if err := os.WriteFile(entry, []byte("TARGET BYTES"), 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "truncated", poisoned: true, damage: func(t *testing.T, _, entry string) {
			if err := os.Truncate(entry, 3); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "mode changed", poisoned: true, damage: func(t *testing.T, _, entry string) {
			if err := os.Chmod(entry, 0o500); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "missing", damage: func(t *testing.T, artifactPath, _ string) {
			if err := os.Remove(filepath.Join(artifactPath, "target")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "symbolic link", damage: func(t *testing.T, artifactPath, entry string) {
			if err := os.Remove(filepath.Join(artifactPath, "target")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(entry, filepath.Join(artifactPath, "target")); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			owner := t.TempDir()
			store := Store{Root: filepath.Join(owner, "failures"), Key: StoreKeyRecord, TargetPool: TargetPool(owner)}
			published := publishSeed(t, store, 7)
			test.damage(t, published.Path, poolEntries(t, store.TargetPool)[0])
			if opened, err := OpenArtifact(published.Path); err == nil {
				t.Fatalf("OpenArtifact() opened a damaged artifact: %#v", opened.Manifest().Target)
			}
			_, err := store.PublishArtifact(seededInput(t, 8))
			if test.poisoned && (err == nil || !strings.Contains(err.Error(), "shared target pool entry")) {
				t.Fatalf("PublishArtifact() over a damaged pool entry error = %v", err)
			}
			if !test.poisoned && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCopiedArtifactOpensWithoutItsStore(t *testing.T) {
	owner := filepath.Join(t.TempDir(), "artifacts")
	store := Store{Root: filepath.Join(owner, "failures"), TargetPool: TargetPool(owner)}
	published, err := store.PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	exported := filepath.Join(t.TempDir(), "exported")
	copyTree(t, published.Path, exported)
	if err := os.RemoveAll(owner); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenArtifact(exported)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	content, err := opened.ReadPayload("target", 64)
	if err != nil || string(content) != "target bytes" {
		t.Fatalf("exported target = %q, %v", content, err)
	}
	if opened.Manifest().RecordHash != published.Manifest.RecordHash {
		t.Fatal("exported artifact identity changed")
	}
}

// copyTree copies a directory the way a plain recursive copy does: new files
// with the source's content and permissions.
func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.Mkdir(filepath.Join(destination, relative), info.Mode().Perm())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(destination, relative), content, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublishWithoutHardLinksKeepsPrivateCopiesAndSaysSo(t *testing.T) {
	link := linkFile
	t.Cleanup(func() { linkFile = link })
	for _, unavailable := range []error{syscall.EXDEV, syscall.ENOTSUP, syscall.EPERM, syscall.EMLINK} {
		t.Run(unavailable.Error(), func(t *testing.T) {
			linkFile = func(oldPath, newPath string) error {
				if _, err := os.Lstat(oldPath); err != nil {
					return &os.LinkError{Op: "link", Old: oldPath, New: newPath, Err: syscall.ENOENT}
				}
				return &os.LinkError{Op: "link", Old: oldPath, New: newPath, Err: unavailable}
			}
			owner := t.TempDir()
			store := Store{Root: filepath.Join(owner, "failures"), Key: StoreKeyRecord, TargetPool: TargetPool(owner)}
			var targets []string
			for _, seed := range []uint64{7, 8} {
				published := publishSeed(t, store, seed)
				if published.TargetSharing != TargetUnshared {
					t.Fatalf("target sharing = %q, want %q", published.TargetSharing, TargetUnshared)
				}
				if again := publishSeed(t, store, seed); again.TargetSharing != TargetUnshared {
					t.Fatalf("republished target sharing = %q, want %q", again.TargetSharing, TargetUnshared)
				}
				opened, err := OpenArtifact(published.Path)
				if err != nil {
					t.Fatal(err)
				}
				content, err := opened.ReadPayload("target", 64)
				if err != nil || !bytes.Equal(content, []byte("target bytes")) {
					t.Fatalf("private target = %q, %v", content, err)
				}
				if err := opened.Close(); err != nil {
					t.Fatal(err)
				}
				targets = append(targets, filepath.Join(published.Path, "target"))
			}
			if distinct := distinctFiles(t, append(targets, poolEntries(t, store.TargetPool)...)); distinct != 3 {
				t.Fatalf("two private copies and the pool entry are %d files, want 3", distinct)
			}
		})
	}
	t.Run("another link failure fails publication", func(t *testing.T) {
		linkFile = func(oldPath, newPath string) error {
			return &os.LinkError{Op: "link", Old: oldPath, New: newPath, Err: syscall.EIO}
		}
		owner := t.TempDir()
		store := Store{Root: filepath.Join(owner, "failures"), TargetPool: TargetPool(owner)}
		if _, err := store.PublishArtifact(artifactInput(t)); !errors.Is(err, syscall.EIO) {
			t.Fatalf("PublishArtifact() error = %v, want the link failure", err)
		}
		if entries, err := os.ReadDir(store.Root); err != nil || len(entries) != 0 {
			t.Fatalf("store entries after a failed publication = %v, %v", entries, err)
		}
	})
}
