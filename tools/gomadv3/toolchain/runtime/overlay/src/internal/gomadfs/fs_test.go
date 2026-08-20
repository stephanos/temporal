package gomadfs_test

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strings"
	"syscall"
	"testing"

	. "internal/gomadfs"
)

func TestFilesystemEnforcesPathAndFileBounds(t *testing.T) {
	filesystem := New()
	if _, _, err := Normalize("/" + strings.Repeat("x", MaximumPathBytes)); !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("Normalize() error = %v", err)
	}
	file, err := filesystem.Open("/file", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MaximumFileBytes + 1); !errors.Is(err, syscall.EFBIG) {
		t.Fatalf("Truncate() error = %v", err)
	}
}

func TestSimulationFilesystemDoesNotLoadAmbientHostPaths(t *testing.T) {
	loaded := false
	Default.SetLoader(func(string) (LoadEntry, MountStatus, error) {
		loaded = true
		return LoadEntry{}, MountNotExist, nil
	})
	t.Cleanup(func() { Default.SetLoader(nil) })

	filesystem := NewSimulation()
	_, err := filesystem.Stat("/ambient")
	if !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("Stat() error = %v, want ENOENT", err)
	}
	if loaded {
		t.Fatal("simulation filesystem consulted the ambient host loader")
	}
}

func TestFilesystemAccountsReleasedBytes(t *testing.T) {
	filesystem := New()
	file, err := filesystem.Open("/file", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(make([]byte, 1024)); err != nil {
		t.Fatal(err)
	}
	if statistics := filesystem.Statistics(); statistics.UsedBytes != 1024 {
		t.Fatalf("used bytes = %d", statistics.UsedBytes)
	}
	if err := file.Truncate(1); err != nil {
		t.Fatal(err)
	}
	if statistics := filesystem.Statistics(); statistics.UsedBytes != 1 {
		t.Fatalf("used bytes after truncate = %d", statistics.UsedBytes)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if statistics := filesystem.Statistics(); statistics.OpenHandles != 0 {
		t.Fatalf("open handles = %d", statistics.OpenHandles)
	}
}

func TestFilesystemFailsMountedMutationsClosed(t *testing.T) {
	filesystem := New()
	filesystem.SetLoader(func(name string) (LoadEntry, MountStatus, error) {
		if name == "/mounted" {
			return LoadEntry{Mode: 0o755, Kind: KindDirectory}, MountOK, nil
		}
		return LoadEntry{}, MountUnmounted, nil
	})
	if err := filesystem.Mkdir("/mounted/new", 0o700); !errors.Is(err, syscall.EROFS) {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := filesystem.Rename("/mounted", "/renamed"); !errors.Is(err, syscall.EXDEV) {
		t.Fatalf("Rename() error = %v", err)
	}
	file, err := filesystem.Open("/file", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Rename("/file", "/mounted"); !errors.Is(err, syscall.EXDEV) {
		t.Fatalf("rename over mount error = %v", err)
	}
}

func TestFilesystemWorkingDirectoryAndMetadata(t *testing.T) {
	filesystem := New()
	logicalTime := int64(100)
	filesystem.SetClock(func() int64 { return logicalTime })
	if err := filesystem.MkdirAll("/workspace/nested", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Chdir("/workspace"); err != nil {
		t.Fatal(err)
	}
	if got := filesystem.Getwd(); got != "/workspace" {
		t.Fatalf("Getwd() = %q", got)
	}
	file, err := filesystem.Open("nested/file", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	logicalTime = 110
	if err := file.Chmod(0o640); err != nil {
		t.Fatal(err)
	}
	if err := file.Chtimes(123); err != nil {
		t.Fatal(err)
	}
	entry, err := filesystem.Stat("nested/file")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Mode != 0o640 || entry.ModTime != 123 {
		t.Fatalf("Stat() = %#v", entry)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Chdir("nested/file"); !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("Chdir() error = %v", err)
	}
}

func TestFilesystemRejectsRenameIntoDescendant(t *testing.T) {
	filesystem := New()
	if err := filesystem.MkdirAll("/tree/nested", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Rename("/tree", "/tree/nested/moved"); !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("Rename() error = %v", err)
	}
	if _, err := filesystem.Stat("/tree/nested"); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestFilesystemRemoveAllRetainsOpenNode(t *testing.T) {
	filesystem := New()
	if err := filesystem.MkdirAll("/tree/nested", 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.Open("/tree/nested/file", OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("contents")); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.RemoveAll("/tree"); err != nil {
		t.Fatal(err)
	}
	if _, err := filesystem.Stat("/tree"); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("Stat() error = %v", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, len("contents"))
	if _, err := file.Read(buffer); err != nil {
		t.Fatal(err)
	}
	if string(buffer) != "contents" {
		t.Fatalf("Read() = %q", buffer)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if statistics := filesystem.Statistics(); statistics.UsedBytes != 0 {
		t.Fatalf("used bytes = %d", statistics.UsedBytes)
	}
}

func TestVolumeFileAndDirectorySyncHaveSeparateDurability(t *testing.T) {
	filesystem := newTestVolumeFilesystem(t)
	file, err := filesystem.Open("/data/value", OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("value")); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}

	states := enumerateAllCrashStates(t, filesystem, "data", CrashEnumerationLimits{States: 16, Operations: 16, Depth: 16, Bytes: 1 << 20, WallNanos: 1_000_000_000})
	assertCrashContents(t, states, []map[string][]byte{{}, {"/value": []byte("value")}})

	directory, err := filesystem.Open("/data", OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Sync(); err != nil {
		t.Fatal(err)
	}
	states = enumerateAllCrashStates(t, filesystem, "data", CrashEnumerationLimits{States: 16, Operations: 16, Depth: 16, Bytes: 1 << 20, WallNanos: 1_000_000_000})
	assertCrashContents(t, states, []map[string][]byte{{"/value": []byte("value")}})
}

func TestVolumeEnumerationIsCompleteDependencyValidAndResumable(t *testing.T) {
	filesystem := newTestVolumeFilesystem(t)
	file, err := filesystem.Open("/data/value", OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(2); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	directory, err := filesystem.Open("/data", OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Sync(); err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte("a"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte("b"), 1); err != nil {
		t.Fatal(err)
	}

	limits := CrashEnumerationLimits{States: 1, Operations: 16, Depth: 16, Bytes: 1 << 20, WallNanos: 1_000_000_000}
	var frontier *CrashFrontier
	var states []CrashState
	for {
		page, err := filesystem.EnumerateCrashStates("data", limits, frontier)
		if err != nil {
			t.Fatal(err)
		}
		states = append(states, page.States...)
		if page.Complete {
			break
		}
		if page.Frontier == nil || page.Capacity != CrashCapacityStates {
			t.Fatalf("incomplete page = %#v", page)
		}
		frontier = page.Frontier
	}
	assertCrashContents(t, states, []map[string][]byte{
		{"/value": {0, 0}},
		{"/value": {'a', 0}},
		{"/value": {0, 'b'}},
		{"/value": {'a', 'b'}},
	})
	identities := make([]string, 0, len(states))
	for _, state := range states {
		identities = append(identities, state.Identity)
	}
	if len(slices.Compact(append([]string(nil), identities...))) != len(identities) {
		t.Fatalf("crash identities contain adjacent duplicates: %v", identities)
	}
}

func TestVolumeRenameCrashStatesRespectNamespaceDependencies(t *testing.T) {
	filesystem := newTestVolumeFilesystem(t)
	file, err := filesystem.Open("/data/a", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	directory, err := filesystem.Open("/data", OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Rename("/data/a", "/data/b"); err != nil {
		t.Fatal(err)
	}

	states := enumerateAllCrashStates(t, filesystem, "data", CrashEnumerationLimits{States: 16, Operations: 16, Depth: 16, Bytes: 1 << 20, WallNanos: 1_000_000_000})
	assertCrashContents(t, states, []map[string][]byte{{"/a": []byte("a")}, {"/b": []byte("a")}})
}

func TestVolumeTruncateCrashStatesRespectResizeDependency(t *testing.T) {
	filesystem := newTestVolumeFilesystem(t)
	file, err := filesystem.Open("/data/value", OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("abcd")); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	directory, err := filesystem.Open("/data", OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(2); err != nil {
		t.Fatal(err)
	}
	states := enumerateAllCrashStates(t, filesystem, "data", CrashEnumerationLimits{States: 16, Operations: 16, Depth: 16, Bytes: 1 << 20, WallNanos: 1_000_000_000})
	assertCrashContents(t, states, []map[string][]byte{{"/value": []byte("abcd")}, {"/value": []byte("ab")}})
}

func TestVolumeCrashRestoresSelectedStateAndRevokesHandles(t *testing.T) {
	filesystem := newTestVolumeFilesystem(t)
	file, err := filesystem.Open("/data/value", OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("value")); err != nil {
		t.Fatal(err)
	}
	states := enumerateAllCrashStates(t, filesystem, "data", CrashEnumerationLimits{States: 16, Operations: 16, Depth: 16, Bytes: 1 << 20, WallNanos: 1_000_000_000})
	var selected CrashState
	for _, state := range states {
		if bytes.Equal(crashContents(state)["/value"], []byte("value")) {
			selected = state
			break
		}
	}
	if selected.Identity == "" {
		t.Fatal("missing selected crash state")
	}
	if err := filesystem.CrashVolumes(map[string][]uint64{"data": selected.SelectedOperations}); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("stale")); !errors.Is(err, syscall.ESTALE) {
		t.Fatalf("stale handle write error = %v", err)
	}
	reopened, err := filesystem.Open("/data/value", OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := io.ReadAll(reopened)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "value" {
		t.Fatalf("restarted contents = %q", contents)
	}
}

func TestVolumeCrashRevokesReadOnlyMappings(t *testing.T) {
	filesystem := newTestVolumeFilesystem(t)
	file, err := filesystem.Open("/data/value", OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("mapped")); err != nil {
		t.Fatal(err)
	}
	mapping, err := file.Map(16)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := mapping.Bytes()
	if err != nil || string(contents[:6]) != "mapped" {
		t.Fatalf("mapping before crash = %q, %v", contents, err)
	}
	if err := filesystem.CrashVolumes(map[string][]uint64{"data": {}}); err != nil {
		t.Fatal(err)
	}
	if _, err := mapping.Bytes(); !errors.Is(err, syscall.ESTALE) {
		t.Fatalf("mapping after crash error = %v, want ESTALE", err)
	}
	if !bytes.Equal(contents, make([]byte, len(contents))) {
		t.Fatalf("mapping contents after crash = %q, want revoked zero bytes", contents)
	}
}

func TestReadOnlyMappingTracksWritesAndTruncation(t *testing.T) {
	filesystem := newTestVolumeFilesystem(t)
	file, err := filesystem.Open("/data/value", OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("initial")); err != nil {
		t.Fatal(err)
	}
	mapping, err := file.Map(16)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := mapping.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte("changed"), 0); err != nil {
		t.Fatal(err)
	}
	if string(contents[:7]) != "changed" {
		t.Fatalf("mapping after write = %q", contents[:7])
	}
	if err := file.Truncate(3); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents, append([]byte("cha"), make([]byte, 13)...)) {
		t.Fatalf("mapping after truncate = %q", contents)
	}
}

func TestVolumeOperationCapacityFailsBeforeMutation(t *testing.T) {
	filesystem := New()
	if err := filesystem.ConfigureVolumes([]VolumeConfig{{ID: "data", Path: "/data", CapacityBytes: 1024}}, VolumeLimits{PendingOperations: 1, Transitions: 16}); err != nil {
		t.Fatal(err)
	}
	if _, err := filesystem.Open("/data/value", OpenFlags{Write: true, Create: true}, 0o600); !errors.Is(err, ErrVolumeCapacity) {
		t.Fatalf("create error = %v", err)
	}
	if _, err := filesystem.Stat("/data/value"); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("stat after rejected create error = %v", err)
	}
}

func TestVolumeObserverRejectsMkdirAllBeforePartialMutation(t *testing.T) {
	filesystem := New()
	if err := filesystem.ConfigureVolumes([]VolumeConfig{{ID: "data", Path: "/data", CapacityBytes: 1 << 20}}, VolumeLimits{PendingOperations: 16, Transitions: 16}); err != nil {
		t.Fatal(err)
	}
	filesystem.SetVolumeObserver(rejectingVolumeObserver{err: syscall.EIO})
	if err := filesystem.MkdirAll("/data/first/second", 0o755); !errors.Is(err, syscall.EIO) {
		t.Fatalf("MkdirAll error = %v, want EIO", err)
	}
	if _, err := filesystem.Stat("/data/first"); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("Stat after rejected MkdirAll = %v, want ENOENT", err)
	}
	if operations := filesystem.Operations()["data"]; len(operations) != 0 {
		t.Fatalf("rejected MkdirAll committed %d operations", len(operations))
	}
}

func TestVolumeEnumerationWallCapacityReturnsResumableFrontier(t *testing.T) {
	filesystem := newTestVolumeFilesystem(t)
	file, err := filesystem.Open("/data/value", OpenFlags{Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("value")); err != nil {
		t.Fatal(err)
	}
	page, err := filesystem.EnumerateCrashStates("data", CrashEnumerationLimits{States: 16, Operations: 16, Depth: 16, Bytes: 1 << 20, WallNanos: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if page.Capacity != CrashCapacityWall || page.Frontier == nil || page.Complete {
		t.Fatalf("wall-bounded page = %#v", page)
	}
	resumed, err := filesystem.EnumerateCrashStates("data", CrashEnumerationLimits{States: 16, Operations: 16, Depth: 16, Bytes: 1 << 20, WallNanos: 1_000_000_000}, page.Frontier)
	if err != nil {
		t.Fatal(err)
	}
	if !resumed.Complete || len(resumed.States) == 0 {
		t.Fatalf("resumed wall-bounded enumeration = %#v", resumed)
	}
}

type rejectingVolumeObserver struct {
	err error
}

func (observer rejectingVolumeObserver) BeforeVolumeOperations(string, []Operation) error {
	return observer.err
}

func (observer rejectingVolumeObserver) BeforeVolumeControl(string, string, []uint64) error {
	return observer.err
}

func newTestVolumeFilesystem(t *testing.T) *FS {
	t.Helper()
	filesystem := New()
	if err := filesystem.ConfigureVolumes([]VolumeConfig{{ID: "data", Path: "/data", CapacityBytes: 1 << 20}}, VolumeLimits{PendingOperations: 1024, Transitions: 1024}); err != nil {
		t.Fatal(err)
	}
	return filesystem
}

func enumerateAllCrashStates(t *testing.T, filesystem *FS, volume string, limits CrashEnumerationLimits) []CrashState {
	t.Helper()
	page, err := filesystem.EnumerateCrashStates(volume, limits, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !page.Complete || page.Frontier != nil {
		t.Fatalf("enumeration did not complete: %#v", page)
	}
	return page.States
}

func assertCrashContents(t *testing.T, states []CrashState, expected []map[string][]byte) {
	t.Helper()
	actual := make([]map[string][]byte, 0, len(states))
	for _, state := range states {
		actual = append(actual, crashContents(state))
	}
	if len(actual) != len(expected) {
		t.Fatalf("crash state count = %d, want %d: %#v", len(actual), len(expected), actual)
	}
	for _, wanted := range expected {
		found := false
		for _, candidate := range actual {
			if equalCrashContents(candidate, wanted) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing crash contents %#v in %#v", wanted, actual)
		}
	}
}

func crashContents(state CrashState) map[string][]byte {
	contents := make(map[string][]byte)
	for _, entry := range state.Entries {
		if entry.Kind == KindFile {
			contents[entry.Path] = append([]byte(nil), entry.Data...)
		}
	}
	return contents
}

func equalCrashContents(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for path, contents := range left {
		if !bytes.Equal(contents, right[path]) {
			return false
		}
		if _, ok := right[path]; !ok {
			return false
		}
	}
	return true
}
