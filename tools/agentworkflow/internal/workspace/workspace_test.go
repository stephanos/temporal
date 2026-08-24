package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPrepareDiffAndApplyPreserveOriginalUntilPromotion(t *testing.T) {
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "keep.txt"), "before")
	writeFile(t, filepath.Join(source, "delete.txt"), "delete")
	writeFile(t, filepath.Join(source, ".git", "private"), "ignored")
	writeFile(t, filepath.Join(source, "node_modules", "large"), "ignored")
	options := Options{MaxBytes: 1 << 20, MaxFiles: 100, Exclude: []string{"node_modules"}}
	prepared, err := Prepare(context.Background(), source, t.TempDir(), options)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(prepared.Base, ".git")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf(".git was copied: %v", err)
	}
	writeFile(t, filepath.Join(prepared.Candidate, "keep.txt"), "after")
	writeFile(t, filepath.Join(prepared.Candidate, "added.txt"), "added")
	if err := os.Remove(filepath.Join(prepared.Candidate, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	changes, digest, err := Diff(context.Background(), prepared)
	if err != nil {
		t.Fatal(err)
	}
	if digest == prepared.Digest || !hasChange(changes, "keep.txt", "modified") || !hasChange(changes, "added.txt", "added") || !hasChange(changes, "delete.txt", "deleted") {
		t.Fatalf("changes = %#v, digest = %s", changes, digest)
	}
	if data, _ := os.ReadFile(filepath.Join(source, "keep.txt")); string(data) != "before" {
		t.Fatalf("source changed before apply: %q", data)
	}
	backup := filepath.Join(t.TempDir(), "backup")
	if err := Apply(context.Background(), prepared, backup); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(source, "keep.txt")); string(data) != "after" {
		t.Fatalf("source after apply = %q", data)
	}
	if _, err := os.Stat(filepath.Join(source, "delete.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted source path exists: %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(backup, "keep.txt")); string(data) != "before" {
		t.Fatalf("backup = %q", data)
	}
}

func TestApplyHandlesDirectoryToFileReplacement(t *testing.T) {
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "swap", "child.txt"), "child")
	prepared, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 1 << 20, MaxFiles: 100})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(prepared.Candidate, "swap")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(prepared.Candidate, "swap"), "file")
	if err := Apply(context.Background(), prepared, filepath.Join(t.TempDir(), "backup")); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(source, "swap")); err != nil || string(data) != "file" {
		t.Fatalf("replacement = %q, %v", data, err)
	}
}

func TestApplyHandlesFileToDirectoryAndSymlinkReplacement(t *testing.T) {
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "swap"), "file")
	writeFile(t, filepath.Join(source, "target-one"), "one")
	writeFile(t, filepath.Join(source, "target-two"), "two")
	if err := os.Symlink("target-one", filepath.Join(source, "link")); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	prepared, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 1 << 20, MaxFiles: 100})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(prepared.Candidate, "swap")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(prepared.Candidate, "swap", "child.txt"), "child")
	if err := os.Remove(filepath.Join(prepared.Candidate, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target-two", filepath.Join(prepared.Candidate, "link")); err != nil {
		t.Fatal(err)
	}
	if err := Apply(context.Background(), prepared, filepath.Join(t.TempDir(), "backup")); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(source, "swap", "child.txt")); err != nil || string(data) != "child" {
		t.Fatalf("replacement child=%q error=%v", data, err)
	}
	if target, err := os.Readlink(filepath.Join(source, "link")); err != nil || target != "target-two" {
		t.Fatalf("replacement link=%q error=%v", target, err)
	}
}

func TestPrepareRejectsEscapingSymlink(t *testing.T) {
	source := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	writeFile(t, outside, "secret")
	if err := os.Symlink(outside, filepath.Join(source, "escape")); err != nil {
		t.Fatal(err)
	}
	_, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 1 << 20, MaxFiles: 100})
	if err == nil || !stringsContains(err.Error(), "escapes source root") {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPrepareEnforcesFileAndByteBounds(t *testing.T) {
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "one"), "12345")
	if _, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 4, MaxFiles: 10}); err == nil {
		t.Fatal("byte overflow was accepted")
	}
	if _, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 100, MaxFiles: 0}); err == nil {
		t.Fatal("invalid file bound was accepted")
	}
}

func TestWorkspaceRejectsInvalidRootsAndExclusions(t *testing.T) {
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "one"), "one")
	for name, options := range map[string]Options{
		"absolute": {MaxBytes: 100, MaxFiles: 10, Exclude: []string{"/outside"}},
		"escape":   {MaxBytes: 100, MaxFiles: 10, Exclude: []string{"../outside"}},
		"dot":      {MaxBytes: 100, MaxFiles: 10, Exclude: []string{"."}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Prepare(context.Background(), source, t.TempDir(), options); err == nil {
				t.Fatal("invalid exclusion was accepted")
			}
		})
	}
	if _, err := Prepare(context.Background(), string(filepath.Separator), t.TempDir(), Options{MaxBytes: 100, MaxFiles: 10}); err == nil {
		t.Fatal("filesystem root was accepted")
	}
	if _, err := Snapshot(context.Background(), filepath.Join(t.TempDir(), "missing"), Options{MaxBytes: 100, MaxFiles: 10}); err == nil {
		t.Fatal("missing snapshot root was accepted")
	}
}

func TestCandidateInventoryRejectsEscapingSymlink(t *testing.T) {
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "one"), "one")
	prepared, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 1 << 20, MaxFiles: 100})
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	writeFile(t, outside, "outside")
	if err := os.Symlink(outside, filepath.Join(prepared.Candidate, "escape")); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	if _, _, err := Diff(context.Background(), prepared); err == nil {
		t.Fatal("escaping candidate symlink was accepted")
	}
}

func TestApplyRejectsSourceDriftBeforeWriting(t *testing.T) {
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "base"), "one")
	prepared, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 1 << 20, MaxFiles: 100})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(prepared.Candidate, "new"), "candidate")
	writeFile(t, filepath.Join(source, "base"), "drift")
	if err := Apply(context.Background(), prepared, filepath.Join(t.TempDir(), "backup")); err == nil {
		t.Fatal("source drift was accepted")
	}
	if _, err := os.Stat(filepath.Join(source, "new")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial apply after drift: %v", err)
	}
}

func TestApplyRejectsDriftImmediatelyBeforeReplacement(t *testing.T) {
	source := t.TempDir()
	sourcePath := filepath.Join(source, "base")
	writeFile(t, sourcePath, "one")
	prepared, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 1 << 20, MaxFiles: 100})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(prepared.Candidate, "base"), "candidate")
	backup := filepath.Join(t.TempDir(), "backup")
	ctx := mutateAfterBackupContext{path: sourcePath, backup: backup}
	if err := Apply(ctx, prepared, backup); err == nil || !stringsContains(err.Error(), "source drift") {
		t.Fatalf("Apply() error = %v", err)
	}
	if data, err := os.ReadFile(sourcePath); err != nil || string(data) != "concurrent" {
		t.Fatalf("concurrent source = %q, %v", data, err)
	}
}

func TestApplyRollsBackWhenPostWriteVerificationFails(t *testing.T) {
	source := t.TempDir()
	sourcePath := filepath.Join(source, "base")
	writeFile(t, sourcePath, "one")
	prepared, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 1 << 20, MaxFiles: 100})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(prepared.Candidate, "base"), "candidate")
	ctx := cancelWhenFileEqualsContext{path: sourcePath, value: "candidate"}
	backup := filepath.Join(t.TempDir(), "backup")
	if err := Apply(ctx, prepared, backup); !errors.Is(err, context.Canceled) {
		t.Fatalf("Apply() error = %v", err)
	}
	if data, err := os.ReadFile(sourcePath); err != nil || string(data) != "one" {
		t.Fatalf("rolled-back source = %q, %v", data, err)
	}
	if err := Apply(context.Background(), prepared, backup); err != nil {
		t.Fatalf("retry Apply() = %v", err)
	}
}

func TestReviewWorkspaceIsImmutableSnapshotByIdentity(t *testing.T) {
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "base"), "one")
	prepared, err := Prepare(context.Background(), source, t.TempDir(), Options{MaxBytes: 1 << 20, MaxFiles: 100})
	if err != nil {
		t.Fatal(err)
	}
	first, err := CopyReview(context.Background(), prepared, "correctness")
	if err != nil {
		t.Fatal(err)
	}
	second, err := CopyReview(context.Background(), prepared, "correctness")
	if err != nil || first != second {
		t.Fatalf("reused review = %q, %v", second, err)
	}
	writeFile(t, filepath.Join(prepared.Candidate, "new"), "changed")
	if _, err := CopyReview(context.Background(), prepared, "correctness"); err == nil {
		t.Fatal("stale review snapshot was reused")
	}
}

func TestReopenAndSnapshotValidatePersistedWorkspaceIdentity(t *testing.T) {
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "base"), "one")
	runDirectory := t.TempDir()
	options := Options{MaxBytes: 1 << 20, MaxFiles: 100}
	prepared, err := Prepare(context.Background(), source, runDirectory, options)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := Reopen(source, runDirectory, prepared.Digest, options)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Base != prepared.Base || reopened.Candidate != prepared.Candidate {
		t.Fatalf("reopened = %#v, prepared = %#v", reopened, prepared)
	}
	digest, err := Snapshot(context.Background(), source, options)
	if err != nil || digest != prepared.Digest {
		t.Fatalf("snapshot = %s, %v, want %s", digest, err, prepared.Digest)
	}
	if _, err := Reopen(source, runDirectory, "sha256:wrong", options); err == nil {
		t.Fatal("wrong persisted digest was accepted")
	}
}

func TestSnapshotExactDetectsControlDirectoryMutation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "base"), "one")
	options := Options{MaxBytes: 1 << 20, MaxFiles: 100}
	filteredBefore, err := Snapshot(context.Background(), root, options)
	if err != nil {
		t.Fatal(err)
	}
	exactBefore, err := SnapshotExact(context.Background(), root, options)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".agentworkflow", "intrusion"), "bad")
	filteredAfter, err := Snapshot(context.Background(), root, options)
	if err != nil {
		t.Fatal(err)
	}
	exactAfter, err := SnapshotExact(context.Background(), root, options)
	if err != nil {
		t.Fatal(err)
	}
	if filteredBefore != filteredAfter || exactBefore == exactAfter {
		t.Fatalf("filtered %s/%s, exact %s/%s", filteredBefore, filteredAfter, exactBefore, exactAfter)
	}
}

func TestValidateChangesRejectsForbiddenPrefix(t *testing.T) {
	err := ValidateChanges([]Change{{Path: "secrets/key", Kind: "added"}}, []string{"secrets"})
	if err == nil {
		t.Fatal("forbidden change was accepted")
	}
	if err := ValidateChanges([]Change{{Path: "safe/key", Kind: "added"}}, []string{"secrets"}); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func hasChange(changes []Change, path, kind string) bool {
	for _, change := range changes {
		if change.Path == path && change.Kind == kind {
			return true
		}
	}
	return false
}

func stringsContains(value, target string) bool {
	return len(value) >= len(target) && (value == target || containsAt(value, target))
}

func containsAt(value, target string) bool {
	for index := 0; index+len(target) <= len(value); index++ {
		if value[index:index+len(target)] == target {
			return true
		}
	}
	return false
}

type mutateAfterBackupContext struct {
	path   string
	backup string
}

func (ctx mutateAfterBackupContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (ctx mutateAfterBackupContext) Done() <-chan struct{}       { return nil }
func (ctx mutateAfterBackupContext) Value(any) any               { return nil }
func (ctx mutateAfterBackupContext) Err() error {
	if _, err := os.Stat(ctx.backup); err == nil {
		_ = os.WriteFile(ctx.path, []byte("concurrent"), 0o600)
	}
	return nil
}

type cancelWhenFileEqualsContext struct {
	path  string
	value string
}

func (ctx cancelWhenFileEqualsContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (ctx cancelWhenFileEqualsContext) Done() <-chan struct{}       { return nil }
func (ctx cancelWhenFileEqualsContext) Value(any) any               { return nil }
func (ctx cancelWhenFileEqualsContext) Err() error {
	data, _ := os.ReadFile(ctx.path)
	if string(data) == ctx.value {
		return context.Canceled
	}
	return nil
}
