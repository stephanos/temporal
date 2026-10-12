package volume_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad3/volume"
)

func newFilesystem(t *testing.T) *volume.FS {
	t.Helper()
	fs := volume.New()
	fs.SetClock(&volume.Clock{Value: 123})
	if err := fs.ConfigureVolumes([]volume.VolumeConfig{{ID: "data", Path: "/data", CapacityBytes: 1 << 20}}, volume.VolumeLimits{PendingOperations: 128, Transitions: 128}); err != nil {
		t.Fatal(err)
	}
	return fs
}

func openFile(t *testing.T, fs *volume.FS, name string) *volume.Handle {
	t.Helper()
	handle, err := fs.Open(name, volume.OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := handle.Close(); err != nil && !errors.Is(err, syscall.ESTALE) {
			t.Error(err)
		}
	})
	return handle
}

func syncDirectory(t *testing.T, fs *volume.FS) {
	t.Helper()
	handle, err := fs.Open("/data", volume.OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeAt(t *testing.T, handle *volume.Handle, data string, offset int64) {
	t.Helper()
	if n, err := handle.WriteAt([]byte(data), offset); err != nil || n != len(data) {
		t.Fatalf("write = %d, %v", n, err)
	}
}

func readFile(t *testing.T, fs *volume.FS, name string) []byte {
	t.Helper()
	handle, err := fs.Open(name, volume.OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(handle)
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	return data
}

func enumerate(t *testing.T, fs *volume.FS, pageSize uint64) []volume.CrashState {
	t.Helper()
	limits := volume.CrashEnumerationLimits{States: pageSize, Operations: 128, Depth: 128, Bytes: 1 << 20, Work: 1_000_000}
	var cursor *volume.CrashExploration
	var states []volume.CrashState
	for count := 0; count < 256; count++ {
		page, err := fs.EnumerateCrashStates("data", limits, cursor)
		if err != nil {
			t.Fatal(err)
		}
		states = append(states, page.States...)
		if page.Complete {
			return states
		}
		if page.Capacity != volume.CrashCapacityStates || page.ChoiceExploration == nil {
			t.Fatalf("unexpected enumeration capacity: %#v", page)
		}
		cursor = page.ChoiceExploration
	}
	t.Fatal("enumeration did not complete")
	return nil
}

func contents(state volume.CrashState) map[string]string {
	result := make(map[string]string)
	for _, entry := range state.Entries {
		if entry.Kind == volume.KindFile {
			result[entry.Path] = string(entry.Data)
		}
	}
	return result
}

func TestVisibleDataFileSyncAndDirectorySync(t *testing.T) {
	fs := newFilesystem(t)
	file := openFile(t, fs, "/data/value")
	writeAt(t, file, "value", 0)
	if got := string(readFile(t, fs, "/data/value")); got != "value" {
		t.Fatalf("visible contents = %q", got)
	}
	initial := fs.VolumeSnapshots()[0]
	if len(initial.Persisted) != 1 || len(initial.Volatile) != 2 || initial.PendingOperations != 4 {
		t.Fatalf("initial snapshot: %#v", initial)
	}
	if _, err := fs.CloneDurable().Stat("/data/value"); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("unsynced clone stat = %v", err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	states := enumerate(t, fs, 16)
	if len(states) != 2 {
		t.Fatalf("file sync crash states = %d", len(states))
	}
	if _, err := fs.CloneDurable().Stat("/data/value"); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("file sync made directory link durable: %v", err)
	}
	syncDirectory(t, fs)
	states = enumerate(t, fs, 16)
	if len(states) != 1 || contents(states[0])["/value"] != "value" {
		t.Fatalf("directory sync states: %#v", states)
	}
	durable := fs.VolumeSnapshots()[0]
	if durable.PendingOperations != 0 || len(durable.Persisted) != 2 || durable.Persisted[1].Path != "/value" || durable.Persisted[1].DataSHA256 != durable.Volatile[1].DataSHA256 {
		t.Fatalf("durable snapshot: %#v", durable)
	}
	if got := string(readFile(t, fs.CloneDurable(), "/data/value")); got != "value" {
		t.Fatalf("durable clone contents = %q", got)
	}
}

func TestCrashEnumerationIsResumableAndSnapshotsAreDetached(t *testing.T) {
	fs := newFilesystem(t)
	file := openFile(t, fs, "/data/value")
	if err := file.Truncate(2); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	syncDirectory(t, fs)
	writeAt(t, file, "a", 0)
	writeAt(t, file, "b", 1)
	states := enumerate(t, fs, 1)
	if len(states) != 4 {
		t.Fatalf("partial persistence states = %d, want 4", len(states))
	}
	wanted := map[string]bool{"\x00\x00": false, "a\x00": false, "\x00b": false, "ab": false}
	seen := make(map[string]bool)
	for _, state := range states {
		if seen[state.Identity] {
			t.Fatal("duplicate crash state")
		}
		seen[state.Identity] = true
		value := contents(state)["/value"]
		if _, exists := wanted[value]; !exists {
			t.Fatalf("unexpected crash contents %q", value)
		}
		wanted[value] = true
	}
	for value, found := range wanted {
		if !found {
			t.Fatalf("missing crash contents %q", value)
		}
	}
	before := fs.VolumeSnapshots()
	detached := fs.VolumeSnapshots()
	detached[0].Persisted[0].Path = "changed"
	detached[0].Volatile[0].DataSHA256 = "changed"
	states[0].Entries[0].Path = "changed"
	operations := fs.Operations()
	operations["data"][0].Kind = "changed"
	if !reflect.DeepEqual(fs.VolumeSnapshots(), before) {
		t.Fatal("caller mutation changed the model snapshot")
	}
	if !reflect.DeepEqual(enumerate(t, fs, 16)[1:], enumerate(t, fs, 1)[1:]) {
		t.Fatal("pagination changed crash states")
	}
}

func TestCrashEnforcesWriteResizeAndNamespaceOrdering(t *testing.T) {
	fs := newFilesystem(t)
	file := openFile(t, fs, "/data/value")
	writeAt(t, file, "old", 0)
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	syncDirectory(t, fs)
	writeAt(t, file, "new", 0)
	writeAt(t, file, "last", 0)
	if err := fs.Rename("/data/value", "/data/final"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Rename("/data/final", "/data/done"); err != nil {
		t.Fatal(err)
	}
	operations := fs.Operations()["data"]
	resize, write, rename := operations[len(operations)-4], operations[len(operations)-3], operations[len(operations)-1]
	if resize.Kind != "resize" || write.Kind != "write" || !containsID(write.Dependencies, resize.ID) {
		t.Fatalf("write lacks resize dependency: %#v", operations)
	}
	before := fs.VolumeSnapshots()
	for _, selected := range [][]uint64{{write.ID}, {rename.ID}, {999999}} {
		if err := fs.CrashVolumes(map[string][]uint64{"data": selected}); err == nil {
			t.Fatalf("invalid crash selection accepted: %v", selected)
		}
		if !reflect.DeepEqual(before, fs.VolumeSnapshots()) {
			t.Fatal("rejected crash mutated snapshots")
		}
	}
	var complete volume.CrashState
	for _, state := range enumerate(t, fs, 64) {
		if contents(state)["/done"] == "last" {
			complete = state
			break
		}
	}
	if complete.Identity == "" {
		t.Fatal("latest ordered writes and renames missing")
	}
	if err := fs.CrashVolumes(map[string][]uint64{"data": complete.SelectedOperations}); err != nil {
		t.Fatal(err)
	}
	if got := string(readFile(t, fs, "/data/done")); got != "last" {
		t.Fatalf("crashed file = %q", got)
	}
	if _, err := file.Write([]byte("stale")); !errors.Is(err, syscall.ESTALE) {
		t.Fatalf("old handle error = %v", err)
	}
}

func containsID(ids []uint64, expected uint64) bool {
	for _, id := range ids {
		if id == expected {
			return true
		}
	}
	return false
}

func TestDurableCloneDoesNotRevokeOwnerOrShareMutations(t *testing.T) {
	fs := newFilesystem(t)
	file := openFile(t, fs, "/data/value")
	writeAt(t, file, "base", 0)
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	syncDirectory(t, fs)
	first, second := fs.CloneDurable(), fs.CloneDurable()
	writeAt(t, file, "live", 0)
	cloneFile := openFile(t, first, "/data/value")
	writeAt(t, cloneFile, "fork", 0)
	if err := cloneFile.Sync(); err != nil {
		t.Fatal(err)
	}
	if got := string(readFile(t, second, "/data/value")); got != "base" {
		t.Fatalf("second clone contents = %q", got)
	}
	if got := string(readFile(t, fs, "/data/value")); got != "live" {
		t.Fatalf("original contents = %q", got)
	}
	if got := string(readFile(t, first, "/data/value")); got != "fork" {
		t.Fatalf("first clone contents = %q", got)
	}
}

func faultRecorder(t *testing.T) *volume.Recorder {
	t.Helper()
	recorder, err := volume.NewRecorder(volume.RecorderLimits{Records: 128, Operations: 128}, []volume.RecorderFault{{Ordinal: 1, Errno: syscall.EIO}})
	if err != nil {
		t.Fatal(err)
	}
	return recorder
}

func TestFaultObserverAndCapacityRejectBeforeMutation(t *testing.T) {
	fs := newFilesystem(t)
	before := fs.VolumeSnapshots()
	fs.SetVolumeRecorder(faultRecorder(t))
	if _, err := fs.Open("/data/rejected", volume.OpenFlags{Write: true, Create: true}, 0o600); !errors.Is(err, syscall.EIO) {
		t.Fatalf("faulted create = %v", err)
	}
	if !reflect.DeepEqual(before, fs.VolumeSnapshots()) {
		t.Fatal("faulted create changed snapshots")
	}
	fs.SetVolumeRecorder(nil)
	file := openFile(t, fs, "/data/value")
	writeAt(t, file, "value", 0)
	before = fs.VolumeSnapshots()
	fs.SetVolumeRecorder(faultRecorder(t))
	if err := file.Sync(); !errors.Is(err, syscall.EIO) {
		t.Fatalf("faulted sync = %v", err)
	}
	if !reflect.DeepEqual(before, fs.VolumeSnapshots()) {
		t.Fatal("faulted sync changed snapshots")
	}
	limited := volume.New()
	if err := limited.ConfigureVolumes([]volume.VolumeConfig{{ID: "data", Path: "/data", CapacityBytes: 1024}}, volume.VolumeLimits{PendingOperations: 1, Transitions: 16}); err != nil {
		t.Fatal(err)
	}
	if _, err := limited.Open("/data/value", volume.OpenFlags{Write: true, Create: true}, 0o600); !errors.Is(err, volume.ErrVolumeCapacity) {
		t.Fatalf("capacity create = %v", err)
	}
	if _, err := limited.Stat("/data/value"); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("rejected file exists: %v", err)
	}
}

func TestCrashEnumerationWorkBudgetResumesAndRejectsStaleCursor(t *testing.T) {
	fs := newFilesystem(t)
	file := openFile(t, fs, "/data/value")
	writeAt(t, file, "value", 0)
	limits := volume.CrashEnumerationLimits{States: 16, Operations: 128, Depth: 128, Bytes: 1 << 20, Work: 1}
	page, err := fs.EnumerateCrashStates("data", limits, nil)
	if err != nil {
		t.Fatal(err)
	}
	if page.Capacity != volume.CrashCapacityWork || page.ChoiceExploration == nil || page.Complete || len(page.States) != 1 {
		t.Fatalf("work bound: %#v", page)
	}
	limits.Work = 1_000_000
	resumed, err := fs.EnumerateCrashStates("data", limits, page.ChoiceExploration)
	if err != nil || !resumed.Complete {
		t.Fatalf("resume = %#v, %v", resumed, err)
	}
	writeAt(t, file, "changed", 0)
	if _, err := fs.EnumerateCrashStates("data", limits, page.ChoiceExploration); err == nil {
		t.Fatal("stale enumeration cursor accepted")
	}
}

func TestLifecycleRestartDropsVolatileAndRevokesOldFS(t *testing.T) {
	fs := newFilesystem(t)
	file := openFile(t, fs, "/data/durable")
	writeAt(t, file, "durable", 0)
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	syncDirectory(t, fs)
	volatile := openFile(t, fs, "/data/volatile")
	writeAt(t, volatile, "lost", 0)
	next, err := fs.AdvanceVolumeLifecycle(false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readFile(t, next, "/data/durable"), []byte("durable")) {
		t.Fatal("restart lost synced bytes")
	}
	if _, err := next.Stat("/data/volatile"); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("restart preserved volatile bytes: %v", err)
	}
	if _, err := fs.Stat("/data/durable"); !errors.Is(err, syscall.ESTALE) {
		t.Fatalf("old FS stat = %v", err)
	}
	if _, err := file.Read(make([]byte, 1)); !errors.Is(err, syscall.ESTALE) {
		t.Fatalf("old handle read = %v", err)
	}
}
