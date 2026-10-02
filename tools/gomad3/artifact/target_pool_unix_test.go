//go:build unix

package artifact

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestOpenRejectsAnotherLinkToAPayloadThatIsNotTheTarget(t *testing.T) {
	owner := t.TempDir()
	store := Store{Root: filepath.Join(owner, "failures"), TargetPool: TargetPool(owner)}
	published, err := store.PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(published.Path, "stdout"), filepath.Join(owner, "stdout-alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenArtifact(published.Path); err == nil || !strings.Contains(err.Error(), "link count") {
		t.Fatalf("OpenArtifact() error = %v", err)
	}
}

// otherTargetInput is seededInput with a target of different bytes.
func otherTargetInput(t *testing.T, seed uint64) Publication {
	t.Helper()
	input := seededInput(t, seed)
	targetBytes := []byte("another target")
	if err := os.WriteFile(input.Payloads[0].SourcePath, targetBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	input.Record.Target.SHA256, input.Record.Target.Size = record.HashBytes(targetBytes), record.Uint64String(len(targetBytes))
	input.Payloads[0].SHA256, input.Payloads[0].Size = input.Record.Target.SHA256, input.Record.Target.Size
	return input
}

func poolNames(t *testing.T, pool string) []string {
	t.Helper()
	names := []string{}
	for _, path := range poolEntries(t, pool) {
		names = append(names, filepath.Base(path))
	}
	sort.Strings(names)
	return names
}

func TestPruneTargetPoolRemovesOnlyEntriesNothingShares(t *testing.T) {
	owner := t.TempDir()
	pool := TargetPool(owner)
	store := Store{Root: filepath.Join(owner, "v1", "campaign-a", "successes"), Key: StoreKeyRecord, TargetPool: pool}
	first, second := publishSeed(t, store, 7), publishSeed(t, store, 8)
	other, err := store.PublishArtifact(otherTargetInput(t, 9))
	if err != nil {
		t.Fatal(err)
	}
	kept, dropped := poolEntryName(first.Manifest.Target.SHA256), poolEntryName(other.Manifest.Target.SHA256)
	staging := ".publish-crashed"
	if err := os.Mkdir(filepath.Join(pool, staging), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pool, staging, "target"), []byte("partial"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pool, "notes"), []byte("not an entry"), 0o600); err != nil {
		t.Fatal(err)
	}
	steps := []struct {
		name      string
		remove    string
		exclusive bool
		want      []string
	}{
		{name: "every entry is shared", want: []string{staging, "notes", dropped, kept}},
		{name: "one target lost its only artifact", remove: other.Path, want: []string{staging, "notes", kept}},
		{name: "one of two artifacts of a target is gone", remove: first.Path, want: []string{staging, "notes", kept}},
		{name: "the pool's only writer also clears staging", exclusive: true, want: []string{"notes", kept}},
		{name: "the last artifact of the target is gone", remove: second.Path, want: []string{"notes"}},
	}
	for _, step := range steps {
		if step.remove != "" {
			if err := os.RemoveAll(step.remove); err != nil {
				t.Fatal(err)
			}
		}
		if err := PruneTargetPool(pool, step.exclusive); err != nil {
			t.Fatalf("%s: PruneTargetPool() error = %v", step.name, err)
		}
		sort.Strings(step.want)
		if got := poolNames(t, pool); !reflect.DeepEqual(got, step.want) {
			t.Fatalf("%s: pool holds %v, want %v", step.name, got, step.want)
		}
	}
	// The retained artifact was never touched: it opened and hashed its target
	// through every step before it was removed.
	republished := publishSeed(t, store, 8)
	opened, err := OpenArtifact(republished.Path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPayload(opened, "target", 64); err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPruneTargetPoolRefusesAPoolThatIsNotADirectoryOfItsOwner(t *testing.T) {
	if err := PruneTargetPool(TargetPool(t.TempDir()), true); err != nil {
		t.Fatalf("PruneTargetPool(missing pool) error = %v", err)
	}
	outside := t.TempDir()
	unshared := filepath.Join(outside, poolEntryName(record.HashBytes([]byte("outside"))))
	if err := os.WriteFile(unshared, []byte("outside"), 0o700); err != nil {
		t.Fatal(err)
	}
	pool := TargetPool(t.TempDir())
	if err := os.Symlink(outside, pool); err != nil {
		t.Fatal(err)
	}
	if err := PruneTargetPool(pool, true); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("PruneTargetPool(symbolic link) error = %v", err)
	}
	if _, err := os.Lstat(unshared); err != nil {
		t.Fatalf("pruning through a symbolic link removed %s: %v", unshared, err)
	}
}

func TestPublishCreatesAgainAPoolEntryPrunedBeforeItsLink(t *testing.T) {
	link := linkFile
	t.Cleanup(func() { linkFile = link })
	owner := t.TempDir()
	store := Store{Root: filepath.Join(owner, "failures"), Key: StoreKeyRecord, TargetPool: TargetPool(owner)}
	var links int
	linkFile = func(oldPath, newPath string) error {
		// The second and third attempts find the entry they just created; a
		// pruner takes it before each link.
		if links++; links == 2 || links == 3 {
			if err := PruneTargetPool(store.TargetPool, false); err != nil {
				t.Error(err)
			}
		}
		return os.Link(oldPath, newPath)
	}
	published := publishSeed(t, store, 7)
	if published.TargetSharing != TargetShared || links != 4 {
		t.Fatalf("target sharing = %q after %d link attempts, want %q after 4", published.TargetSharing, links, TargetShared)
	}
	if distinct := distinctFiles(t, append(poolEntries(t, store.TargetPool), filepath.Join(published.Path, "target"))); distinct != 1 {
		t.Fatalf("the artifact and its pool entry are %d files, want 1", distinct)
	}
}
