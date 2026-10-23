package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestConcurrentPublicationPublishesOnce(t *testing.T) {
	run := recordGood(t, statefulGuest(t))
	store := t.TempDir()
	const writers = 16
	publications := make([]Publication, writers)
	errs := make([]error, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Go(func() { publications[i], errs[i] = Publish(context.Background(), store, run) })
	}
	wg.Wait()
	fresh := 0
	for i := range writers {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		if !publications[i].Existing {
			fresh++
		}
		if publications[i].Key != publications[0].Key {
			t.Fatalf("keys differ: %s %s", publications[i].Key, publications[0].Key)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(publications[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	if fresh != 1 || len(entries) != 1 {
		t.Fatalf("%d fresh publications, %d entries", fresh, len(entries))
	}
	loaded, err := Load(publications[0].Path)
	if err != nil || !proto.Equal(loaded, run) {
		t.Fatalf("load: %v", err)
	}
}

func TestPublicationNeverOverwrites(t *testing.T) {
	run := recordGood(t, statefulGuest(t))
	data, digest, err := Canonical(run)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := func(store string) string {
		return filepath.Join(store, run.GetModelHash(), "1", digest+RunExtension)
	}
	prepare := func(t *testing.T, occupy func(path string) error) string {
		store := t.TempDir()
		path := keyPath(store)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := occupy(path); err != nil {
			t.Fatal(err)
		}
		return store
	}
	for _, tc := range []struct {
		name   string
		occupy func(path string) error
	}{
		{"other bytes", func(path string) error { return os.WriteFile(path, []byte("other"), 0o644) }},
		{"longer bytes", func(path string) error { return os.WriteFile(path, append(bytes.Clone(data), 0), 0o644) }},
		{"directory", func(path string) error { return os.Mkdir(path, 0o755) }},
		{"symlink", func(path string) error {
			target := filepath.Join(filepath.Dir(path), "target")
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return err
			}
			return os.Symlink(target, path)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := prepare(t, tc.occupy)
			before, _ := os.Lstat(keyPath(store))
			_, err := Publish(context.Background(), store, run)
			var conflict *ConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("publish over %s: %v", tc.name, err)
			}
			after, _ := os.Lstat(keyPath(store))
			if before.Mode() != after.Mode() || before.Size() != after.Size() {
				t.Fatal("the key changed")
			}
		})
	}
	t.Run("seed directory is a file", func(t *testing.T) {
		store := t.TempDir()
		if err := os.MkdirAll(filepath.Join(store, run.GetModelHash()), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(store, run.GetModelHash(), "1"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		var conflict *ConflictError
		if _, err := Publish(context.Background(), store, run); !errors.As(err, &conflict) {
			t.Fatalf("publish: %v", err)
		}
	})
	t.Run("interrupted", func(t *testing.T) {
		store := t.TempDir()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := Publish(ctx, store, run); !errors.Is(err, context.Canceled) {
			t.Fatalf("publish: %v", err)
		}
		entries, err := os.ReadDir(filepath.Dir(keyPath(store)))
		if err != nil || len(entries) != 0 {
			t.Fatalf("an interrupted publication left %d entries: %v", len(entries), err)
		}
	})
	t.Run("model hash is not a path", func(t *testing.T) {
		altered := proto.CloneOf(run)
		altered.ModelHash = "../escape"
		if _, err := Publish(context.Background(), t.TempDir(), altered); err == nil {
			t.Fatal("published under a traversing model hash")
		}
	})
}

func TestLoadRejectsCorruptRuns(t *testing.T) {
	run := recordGood(t, statefulGuest(t))
	store := t.TempDir()
	publication, err := Publish(context.Background(), store, run)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(publication.Path)
	if err != nil {
		t.Fatal(err)
	}
	corrupt := bytes.Clone(data)
	corrupt[len(corrupt)-1] ^= 1
	if err := os.WriteFile(publication.Path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(publication.Path); err == nil {
		t.Fatal("loaded a corrupted Run")
	}
	var conflict *ConflictError
	if _, err := Publish(context.Background(), store, run); !errors.As(err, &conflict) {
		t.Fatalf("republishing over corruption: %v", err)
	}

	renamed := filepath.Join(store, "renamed.binpb")
	if err := os.WriteFile(renamed, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(renamed); err != nil {
		t.Fatalf("a canonical Run under another name: %v", err)
	}
	misnamed := filepath.Join(store, "0000000000000000000000000000000000000000000000000000000000000000"+RunExtension)
	if err := os.WriteFile(misnamed, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(misnamed); err == nil {
		t.Fatal("loaded a Run under another digest")
	}

	unseeded := proto.CloneOf(run)
	unseeded.Seed = 0
	reordered, err := proto.MarshalOptions{Deterministic: true}.Marshal(unseeded)
	if err != nil {
		t.Fatal(err)
	}
	reordered = protowire.AppendVarint(protowire.AppendTag(reordered, 4, protowire.VarintType), run.GetSeed())
	path := filepath.Join(store, "reordered.binpb")
	if err := os.WriteFile(path, reordered, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("loaded a non-canonical Run")
	}
}
