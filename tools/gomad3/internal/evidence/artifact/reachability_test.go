package artifact

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

var publicationSteps = []publicationStep{stepPayloadsDurable, stepEnvelopeDurable, stepManifestDurable, stepStagingDurable, stepRenamed, stepStoreDurable}

func observePublication(t *testing.T, checkpoint func(publicationStep) error) *[]string {
	t.Helper()
	var events []string
	previousObserver, previousCheckpoint := syncObserver, publicationCheckpoint
	syncObserver = func(path string) { events = append(events, "sync "+path) }
	publicationCheckpoint = func(step publicationStep) error {
		events = append(events, "step "+string(step))
		return checkpoint(step)
	}
	t.Cleanup(func() { syncObserver, publicationCheckpoint = previousObserver, previousCheckpoint })
	return &events
}

func indexOf(t *testing.T, events []string, match func(string) bool, what string) int {
	t.Helper()
	index := slices.IndexFunc(events, match)
	if index < 0 {
		t.Fatalf("publication never made %s durable:\n%s", what, strings.Join(events, "\n"))
	}
	return index
}

// Removing any sync from publication, or moving one after the step that
// relies on it, fails this control.
func TestPublicationIsDurableBeforeItReturnsAReference(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "new", "store")
	events := observePublication(t, func(publicationStep) error { return nil })
	published, err := (Store{Root: root}).PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	log := *events
	step := func(step publicationStep) int {
		return indexOf(t, log, func(event string) bool { return event == "step "+string(step) }, string(step))
	}
	synced := func(suffix string) int {
		return indexOf(t, log, func(event string) bool {
			return strings.HasPrefix(event, "sync ") && strings.HasSuffix(event, suffix) && strings.Contains(event, string(filepath.Separator)+".publish-")
		}, suffix)
	}
	for index := 1; index < len(publicationSteps); index++ {
		if step(publicationSteps[index-1]) > step(publicationSteps[index]) {
			t.Fatalf("step %s precedes %s", publicationSteps[index], publicationSteps[index-1])
		}
	}
	for _, payload := range []string{"/target", "/stdout", "/stderr", "/world/snapshot.json", "/world/transitions.jsonl", "/world/final-snapshot.json", "/world"} {
		if synced(payload) > step(stepPayloadsDurable) {
			t.Fatalf("payload %s was synced after the payloads were declared durable", payload)
		}
	}
	if index := synced("/storage.json"); index < step(stepPayloadsDurable) || index > step(stepEnvelopeDurable) {
		t.Fatal("the storage envelope is not synced between the payloads and its step")
	}
	if index := synced("/manifest.json"); index < step(stepEnvelopeDurable) || index > step(stepManifestDurable) {
		t.Fatal("the manifest is not synced after the envelope and before its step")
	}
	staging := indexOf(t, log, func(event string) bool {
		return strings.HasPrefix(event, "sync "+filepath.Join(root, ".publish-")) && !strings.Contains(strings.TrimPrefix(event, "sync "+filepath.Join(root, ".publish-")), string(filepath.Separator))
	}, "the staging directory")
	if staging < step(stepManifestDurable) || staging > step(stepStagingDurable) {
		t.Fatal("the staging directory is not synced after the manifest and before the rename")
	}
	rootSync := slices.Index(log[step(stepRenamed):], "sync "+root)
	if rootSync < 0 || step(stepRenamed)+rootSync > step(stepStoreDurable) {
		t.Fatal("the store root is not synced after the rename and before the reference is returned")
	}
	for _, created := range []string{parent, filepath.Join(parent, "new")} {
		if index := slices.Index(log, "sync "+created); index < 0 || index > step(stepPayloadsDurable) {
			t.Fatalf("created store parent %s is not synced before publication", created)
		}
	}
	if step(stepStoreDurable) != len(log)-1 {
		t.Fatalf("publication did more after its store became durable:\n%s", strings.Join(log[step(stepStoreDurable):], "\n"))
	}
	mustDo(t, validateArtifactReference(published.Path))
}

// A crash at each publication step returns no reference, so no caller can
// record one; what it leaves behind is an unreferenced artifact or staging
// directory that the owner's sweep reclaims, after which publication works.
func TestEveryPublicationCrashWindowLeavesOnlyReclaimableState(t *testing.T) {
	for _, crashed := range publicationSteps {
		t.Run(string(crashed), func(t *testing.T) {
			root := t.TempDir()
			injected := errors.New("crash at " + string(crashed))
			observePublication(t, func(step publicationStep) error {
				if step == crashed {
					return injected
				}
				return nil
			})
			published, err := (Store{Root: root}).PublishArtifact(artifactInput(t))
			if !errors.Is(err, injected) || published.Path != "" || published.StoredBytes != 0 {
				t.Fatalf("PublishArtifact() = %#v, %v; want no reference and the injected crash", published, err)
			}
			publicationCheckpoint = func(publicationStep) error { return nil }
			// A real crash also keeps the staging directory the failed call removed.
			mustDo(t, os.Mkdir(filepath.Join(root, ".publish-crashed"), 0o700))
			mustDo(t, os.WriteFile(filepath.Join(root, ".publish-crashed", "manifest.json"), []byte("{}"), 0o600))
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			renamed := slices.Index(publicationSteps, crashed) >= slices.Index(publicationSteps, stepRenamed)
			var artifacts []string
			for _, entry := range entries {
				if isArtifactDirectoryName(entry.Name()) {
					artifacts = append(artifacts, entry.Name())
				}
			}
			if renamed != (len(artifacts) == 1) || len(artifacts) > 1 {
				t.Fatalf("store entries after a crash at %s = %v", crashed, entries)
			}
			if renamed {
				mustDo(t, validateArtifactReference(filepath.Join(root, artifacts[0])))
			}
			removed, err := SweepOwnedRoot(root, References{})
			if err != nil {
				t.Fatal(err)
			}
			if want := append([]string{".publish-crashed"}, artifacts...); !slices.Equal(removed, want) {
				t.Fatalf("swept %v, want %v", removed, want)
			}
			if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
				t.Fatalf("store after sweep = %v, %v", entries, err)
			}
			published, err = (Store{Root: root}).PublishArtifact(artifactInput(t))
			if err != nil {
				t.Fatal(err)
			}
			mustDo(t, validateArtifactReference(published.Path))
		})
	}
}

// The publisher of an existing artifact may have stopped between its rename
// and its store sync; reusing that artifact makes it durable first.
func TestReusedArtifactIsDurableBeforeItIsReturned(t *testing.T) {
	root := t.TempDir()
	store := Store{Root: root}
	if _, err := store.PublishArtifact(artifactInput(t)); err != nil {
		t.Fatal(err)
	}
	events := observePublication(t, func(publicationStep) error { return nil })
	reused, err := store.PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	if last := (*events)[len(*events)-1]; last != "sync "+root {
		t.Fatalf("reuse of %s ended with %q, want the store sync", reused.Path, last)
	}
}

func TestSweepKeepsLiveReferencesAndReclaimsWhatCommittedStateRetired(t *testing.T) {
	root := t.TempDir()
	store := Store{Root: root, Key: StoreKeyRecord}
	live := publishSeed(t, store, 1)
	evicted := publishSeed(t, store, 2)
	pending := publishSeed(t, store, 3)
	diagnostic, err := PublishOwnedFile(context.Background(), root, "00000000000000000003-diagnostic.bin", []byte("diagnostic"))
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := PublishOwnedFile(context.Background(), root, "00000000000000000004-diagnostic.bin", []byte("orphan"))
	if err != nil {
		t.Fatal(err)
	}
	removed, err := SweepOwnedRoot(root, References{
		Artifacts: []string{filepath.Base(live.Path), filepath.Base(pending.Path)},
		Files:     []FileReference{diagnostic},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{orphan.Name, filepath.Base(evicted.Path)}; !slices.Equal(removed, want) {
		t.Fatalf("swept %v, want %v", removed, want)
	}
	for _, kept := range []string{live.Path, pending.Path} {
		mustDo(t, validateArtifactReference(kept))
	}
	mustDo(t, validateFileReference(root, diagnostic))
	// Once an outcome transaction retires the pending receipt, its payload is
	// reclaimable.
	removed, err = SweepOwnedRoot(root, References{Artifacts: []string{filepath.Base(live.Path)}, Files: []FileReference{diagnostic}})
	if err != nil || !slices.Equal(removed, []string{filepath.Base(pending.Path)}) {
		t.Fatalf("SweepOwnedRoot() = %v, %v; want the retired pending payload", removed, err)
	}
}

// Older PublishOwnedFile builds linked the final name before removing their
// temporary. A process death in that window leaves both names on the same
// two-link inode; recovery must recognize that closed, root-local pair rather
// than treating it as a foreign hard link.
func TestSweepReclaimsInterruptedOwnedFilePublication(t *testing.T) {
	root := t.TempDir()
	temporary := filepath.Join(root, ".publish-crashed")
	final := filepath.Join(root, "pending.bin")
	mustDo(t, os.WriteFile(temporary, []byte("pending result"), 0o600))
	mustDo(t, os.Link(temporary, final))

	removed, err := SweepOwnedRoot(root, References{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{".publish-crashed", "pending.bin"}; !slices.Equal(removed, want) {
		t.Fatalf("SweepOwnedRoot() removed %v, want %v", removed, want)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("owned root after sweep = %v, %v; want empty", entries, err)
	}
}

// Missing or corrupt live state is corruption: the sweep reports it and
// removes nothing, not even an orphan.
func TestSweepRemovesNothingWhenALiveReferenceIsMissingOrCorrupt(t *testing.T) {
	for _, test := range []struct {
		name   string
		damage func(t *testing.T, root string, artifact Artifact, file FileReference) References
		cause  error
	}{
		{"missing artifact", func(t *testing.T, root string, artifact Artifact, file FileReference) References {
			mustDo(t, os.RemoveAll(artifact.Path))
			return References{Artifacts: []string{filepath.Base(artifact.Path)}, Files: []FileReference{file}}
		}, os.ErrNotExist},
		{"corrupt artifact", func(t *testing.T, root string, artifact Artifact, file FileReference) References {
			mustDo(t, os.WriteFile(filepath.Join(artifact.Path, "stdout"), []byte("STDOUT"), 0o600))
			return References{Artifacts: []string{filepath.Base(artifact.Path)}, Files: []FileReference{file}}
		}, nil},
		{"pre-lineage artifact", func(t *testing.T, root string, artifact Artifact, file FileReference) References {
			mustDo(t, os.Remove(filepath.Join(artifact.Path, "storage.json")))
			return References{Artifacts: []string{filepath.Base(artifact.Path)}, Files: []FileReference{file}}
		}, ErrUnsupportedFormat},
		{"missing pending payload", func(t *testing.T, root string, artifact Artifact, file FileReference) References {
			mustDo(t, os.Remove(filepath.Join(root, file.Name)))
			return References{Artifacts: []string{filepath.Base(artifact.Path)}, Files: []FileReference{file}}
		}, os.ErrNotExist},
		{"corrupt diagnostic sidecar", func(t *testing.T, root string, artifact Artifact, file FileReference) References {
			file.SHA256 = record.HashBytes([]byte("other"))
			return References{Artifacts: []string{filepath.Base(artifact.Path)}, Files: []FileReference{file}}
		}, nil},
		{"borrowed shard artifact", func(t *testing.T, root string, artifact Artifact, file FileReference) References {
			return References{Artifacts: []string{filepath.Base(artifact.Path)}, Files: []FileReference{file}, Borrowed: []string{filepath.Join(t.TempDir(), "sha256-missing")}}
		}, os.ErrNotExist},
		{"reference outside the root", func(t *testing.T, root string, artifact Artifact, file FileReference) References {
			return References{Artifacts: []string{"../" + filepath.Base(artifact.Path)}}
		}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			store := Store{Root: root, Key: StoreKeyRecord}
			live := publishSeed(t, store, 1)
			orphan := publishSeed(t, store, 2)
			file, err := PublishOwnedFile(context.Background(), root, "pending.bin", []byte("pending result"))
			if err != nil {
				t.Fatal(err)
			}
			removed, err := SweepOwnedRoot(root, test.damage(t, root, live, file))
			var reference *ReferenceError
			if !errors.Is(err, ErrCorruptReference) || !errors.As(err, &reference) || (test.cause != nil && !errors.Is(err, test.cause)) {
				t.Fatalf("SweepOwnedRoot() error = %v, want corruption caused by %v", err, test.cause)
			}
			if len(removed) != 0 {
				t.Fatalf("SweepOwnedRoot() removed %v despite corruption", removed)
			}
			mustDo(t, validateArtifactReference(orphan.Path))
		})
	}
}

func TestSweepNeverTakesSharedPoolsBorrowedArtifactsOrForeignEntries(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, root string) References
		want  string
	}{
		{"target pool inside the root", func(t *testing.T, root string) References {
			publishSeed(t, Store{Root: filepath.Join(t.TempDir(), "store"), TargetPool: TargetPool(root)}, 5)
			return References{}
		}, "unexpected entry targets"},
		{"pool entry", func(t *testing.T, root string) References {
			mustDo(t, os.WriteFile(filepath.Join(root, poolEntryName(record.HashBytes([]byte("target")))), []byte("target"), 0o700))
			return References{}
		}, "shared target pool entry"},
		{"file with another link", func(t *testing.T, root string) References {
			other := filepath.Join(t.TempDir(), "shared")
			mustDo(t, os.WriteFile(other, []byte("shared"), 0o600))
			mustDo(t, os.Link(other, filepath.Join(root, "shared.bin")))
			return References{}
		}, "has other links"},
		{"symbolic link", func(t *testing.T, root string) References {
			mustDo(t, os.Symlink(t.TempDir(), filepath.Join(root, "sha256-0123456789abcdef0123456789abcdef")))
			return References{}
		}, "symbolic link"},
		{"other directory", func(t *testing.T, root string) References {
			mustDo(t, os.Mkdir(filepath.Join(root, "campaign"), 0o700))
			return References{}
		}, "unexpected entry campaign"},
		{"borrowed artifact inside the root", func(t *testing.T, root string) References {
			return References{Borrowed: []string{filepath.Join(root, "sha256-0123456789abcdef0123456789abcdef")}}
		}, "inside the owned root"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			orphan := publishSeed(t, Store{Root: root, Key: StoreKeyRecord}, 1)
			removed, err := SweepOwnedRoot(root, test.setup(t, root))
			if err == nil || !strings.Contains(err.Error(), test.want) || len(removed) != 0 {
				t.Fatalf("SweepOwnedRoot() = %v, %v; want refusal %q before any removal", removed, err, test.want)
			}
			mustDo(t, validateArtifactReference(orphan.Path))
		})
	}
}

// A borrowed shard artifact is validated and survives the aggregate's sweep,
// and a pool that the swept artifacts shared is left to PruneTargetPool.
func TestSweepValidatesButKeepsBorrowedArtifactsAndSharedTargets(t *testing.T) {
	owner := t.TempDir()
	pool := TargetPool(owner)
	shard := publishSeed(t, Store{Root: filepath.Join(owner, "shard"), Key: StoreKeyRecord, TargetPool: pool}, 1)
	root := filepath.Join(owner, "aggregate")
	orphan := publishSeed(t, Store{Root: root, Key: StoreKeyRecord, TargetPool: pool}, 2)
	removed, err := SweepOwnedRoot(root, References{Borrowed: []string{shard.Path}})
	if err != nil || !slices.Equal(removed, []string{filepath.Base(orphan.Path)}) {
		t.Fatalf("SweepOwnedRoot() = %v, %v", removed, err)
	}
	mustDo(t, validateArtifactReference(shard.Path))
	if entries := poolEntries(t, pool); len(entries) != 1 {
		t.Fatalf("pool after sweep = %v, want the shared entry kept", entries)
	}
}

func TestSweepRemovalFailureKeepsItsIdentity(t *testing.T) {
	root := t.TempDir()
	store := Store{Root: root, Key: StoreKeyRecord}
	first := publishSeed(t, store, 1)
	second := publishSeed(t, store, 2)
	injected := errors.New("injected removal failure")
	previous := removeOwnedEntry
	removeOwnedEntry = func(root *os.Root, name string) error {
		if name == filepath.Base(second.Path) {
			return &os.PathError{Op: "unlinkat", Path: name, Err: injected}
		}
		return previous(root, name)
	}
	t.Cleanup(func() { removeOwnedEntry = previous })
	removed, err := SweepOwnedRoot(root, References{})
	if !errors.Is(err, injected) || !strings.Contains(err.Error(), "remove unreferenced owned payload") {
		t.Fatalf("SweepOwnedRoot() error = %v, want the injected failure", err)
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr.Path != filepath.Base(second.Path) {
		t.Fatalf("SweepOwnedRoot() error = %#v, want the path error", err)
	}
	if !slices.Equal(removed, []string{filepath.Base(first.Path)}) && !slices.Equal(removed, []string(nil)) {
		t.Fatalf("SweepOwnedRoot() removed %v", removed)
	}
	mustDo(t, validateArtifactReference(second.Path))
}

func TestSweepOfAMissingRootWithNoLiveStateDoesNothing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if removed, err := SweepOwnedRoot(missing, References{}); err != nil || removed != nil {
		t.Fatalf("SweepOwnedRoot() = %v, %v", removed, err)
	}
	if _, err := SweepOwnedRoot(missing, References{Artifacts: []string{"sha256-0123456789abcdef0123456789abcdef"}}); !errors.Is(err, ErrCorruptReference) || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("SweepOwnedRoot() error = %v, want a missing live reference", err)
	}
}

func TestPublishOwnedFileIsDurableAndContentAddressedByName(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "diagnostics")
	events := observePublication(t, func(publicationStep) error { return nil })
	reference, err := PublishOwnedFile(context.Background(), root, "00000000000000000001-trace.bin", []byte("trace"))
	if err != nil {
		t.Fatal(err)
	}
	if reference != (FileReference{Name: "00000000000000000001-trace.bin", SHA256: record.HashBytes([]byte("trace")), Bytes: 5}) {
		t.Fatalf("reference = %#v", reference)
	}
	log := *events
	parentSync := slices.Index(log, "sync "+parent)
	temporarySync := slices.IndexFunc(log, func(event string) bool { return strings.HasPrefix(event, "sync "+filepath.Join(root, ".publish-")) })
	if parentSync < 0 || temporarySync < parentSync || log[len(log)-1] != "sync "+root {
		t.Fatalf("owned file durability order:\n%s", strings.Join(log, "\n"))
	}
	if again, err := PublishOwnedFile(context.Background(), root, reference.Name, []byte("trace")); err != nil || again != reference {
		t.Fatalf("republish = %#v, %v", again, err)
	}
	if _, err := PublishOwnedFile(context.Background(), root, reference.Name, []byte("other")); err == nil || !strings.Contains(err.Error(), "differs") {
		t.Fatalf("conflicting publish error = %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("owned root = %v, %v; want only the published file", entries, err)
	}
	info, err := os.Stat(filepath.Join(root, reference.Name))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("owned file = %v, %v", info, err)
	}
	for _, name := range []string{"", "..", "a/b", ".publish-x", "sha256-0123456789abcdef0123456789abcdef", poolEntryName(record.HashBytes(nil))} {
		if _, err := PublishOwnedFile(context.Background(), root, name, nil); err == nil {
			t.Fatalf("PublishOwnedFile(%q) succeeded", name)
		}
	}
}
