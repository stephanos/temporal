package volume_test

import (
	"errors"
	"reflect"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func capturedSnapshot(t *testing.T, lookups uint64) *volume.CapturedSnapshot {
	t.Helper()
	entries := []volume.CapturedEntry{
		{Path: "/inputs", Entry: volume.LoadEntry{Kind: volume.KindDirectory, Mode: 0o755, Children: []volume.Child{{Name: "value", Kind: volume.KindFile, Mode: 0o444}}}},
		{Path: "/inputs/value", Entry: volume.LoadEntry{Kind: volume.KindFile, Mode: 0o444, Data: []byte("captured")}},
	}
	snapshot, err := volume.NewCapturedSnapshot([]string{"/inputs"}, entries, volume.CapturedSnapshotLimits{Entries: 8, Bytes: 1024, Lookups: lookups})
	if err != nil {
		t.Fatal(err)
	}
	entries[0].Entry.Children[0].Name = "changed"
	entries[1].Entry.Data[0] = 'X'
	return snapshot
}

func TestCapturedSnapshotOwnsInputsAndDetachedLookupResults(t *testing.T) {
	snapshot := capturedSnapshot(t, 16)
	entry, status, err := snapshot.Load("/inputs/value")
	if err != nil || status != volume.MountOK || string(entry.Data) != "captured" {
		t.Fatalf("load = %#v, %v, %v", entry, status, err)
	}
	entry.Data[0] = 'X'
	entry, status, err = snapshot.Load("/inputs/value")
	if err != nil || status != volume.MountOK || string(entry.Data) != "captured" {
		t.Fatalf("detached load = %#v, %v, %v", entry, status, err)
	}
	directory, _, err := snapshot.Load("/inputs")
	if err != nil || directory.Children[0].Name != "value" {
		t.Fatalf("directory = %#v, %v", directory, err)
	}
	directory.Children[0].Name = "changed"
	directory, _, err = snapshot.Load("/inputs")
	if err != nil || directory.Children[0].Name != "value" {
		t.Fatalf("detached directory = %#v, %v", directory, err)
	}
	for path, expected := range map[string]volume.MountStatus{"/inputs/missing": volume.MountNotExist, "/ambient": volume.MountUnmounted} {
		if _, status, err := snapshot.Load(path); err != nil || status != expected {
			t.Fatalf("load %s = %v, %v", path, status, err)
		}
	}
	fs := volume.New()
	fs.SetCapturedSnapshot(snapshot)
	if got := string(readFile(t, fs, "/inputs/value")); got != "captured" {
		t.Fatalf("captured FS contents = %q", got)
	}
	if _, err := fs.Open("/inputs/value", volume.OpenFlags{Write: true}, 0); !errors.Is(err, syscall.EROFS) {
		t.Fatalf("captured write = %v", err)
	}
}

func TestCapturedSnapshotBoundsAndPerFilesystemLookupAccounting(t *testing.T) {
	snapshot := capturedSnapshot(t, 1)
	if _, _, err := snapshot.Load("/inputs/value"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := snapshot.Load("/inputs/value"); !errors.Is(err, volume.ErrVolumeCapacity) {
		t.Fatalf("lookup capacity = %v", err)
	}
	first, second := volume.New(), volume.New()
	first.SetCapturedSnapshot(snapshot)
	second.SetCapturedSnapshot(snapshot)
	if _, err := first.Stat("/inputs/value"); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Stat("/inputs/value"); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Stat("/inputs/missing"); !errors.Is(err, volume.ErrVolumeCapacity) {
		t.Fatalf("FS lookup capacity = %v", err)
	}
	if _, err := first.Stat("/inputs/value"); err != nil {
		t.Fatalf("cached captured lookup consumed capacity: %v", err)
	}
	if _, err := volume.NewCapturedSnapshot([]string{"/inputs"}, []volume.CapturedEntry{{Path: "/inputs/value", Entry: volume.LoadEntry{Kind: volume.KindFile, Data: make([]byte, 32)}}}, volume.CapturedSnapshotLimits{Entries: 1, Bytes: 16, Lookups: 1}); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("capture bytes = %v", err)
	}
	if _, err := volume.NewCapturedSnapshot([]string{"/inputs"}, []volume.CapturedEntry{{Path: "/outside", Entry: volume.LoadEntry{Kind: volume.KindFile}}}, volume.CapturedSnapshotLimits{Entries: 1, Bytes: 64, Lookups: 1}); !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("unmounted capture = %v", err)
	}
}

func TestOwnedClockAndRecorderPreserveNativeReferenceState(t *testing.T) {
	fs := newFilesystem(t)
	recorder, err := volume.NewRecorder(volume.RecorderLimits{Records: 16, Operations: 64}, nil)
	if err != nil {
		t.Fatal(err)
	}
	fs.SetVolumeRecorder(recorder)
	file := openFile(t, fs, "/data/value")
	writeAt(t, file, "value", 0)
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	syncDirectory(t, fs)
	// Retained from the unadapted native projection for clock=123 and these
	// exact create/write/file-sync/directory-sync inputs, including native mtime.
	const nativeSnapshot = "sha256:01afe1f791e56fa61e5fd86d5f372240a7556a3673956f1c9765dcda724886c8"
	if got := fs.VolumeSnapshots()[0].Identity; got != nativeSnapshot {
		t.Fatalf("native reference snapshot = %s", got)
	}
	records := recorder.Records()
	if len(records) != 4 || records[0].Kind != "operations" || records[1].Kind != "operations" || records[2].Kind != "file_sync" || records[3].Kind != "directory_sync" {
		t.Fatalf("reference record ordering = %#v", records)
	}
	if !reflect.DeepEqual(records[2].SelectedOperations, []uint64{1, 3, 4}) || !reflect.DeepEqual(records[3].SelectedOperations, []uint64{2}) {
		t.Fatalf("reference sync transcript = %#v", records)
	}
	if !reflect.DeepEqual(append(records[0].Operations, records[1].Operations...), fs.Operations()["data"]) {
		t.Fatal("recorder changed native operation transcript")
	}
	records[0].Operations[1].Dependencies[0] = 999
	records[2].SelectedOperations[0] = 999
	if recorder.Records()[0].Operations[1].Dependencies[0] == 999 || recorder.Records()[2].SelectedOperations[0] == 999 {
		t.Fatal("caller mutated recorder")
	}
	clock := &volume.Clock{Value: 7}
	fs.SetClock(clock)
	clock.Value = 42
	if err := fs.Mkdir("/clocked", 0o755); err != nil {
		t.Fatal(err)
	}
	entry, err := fs.Stat("/clocked")
	if err != nil || entry.ModTime != 42 {
		t.Fatalf("owned clock = %#v, %v", entry, err)
	}
}

func TestRecorderBoundAndFaultRejectBeforeMutation(t *testing.T) {
	fs := newFilesystem(t)
	recorder, err := volume.NewRecorder(volume.RecorderLimits{Records: 1, Operations: 16}, nil)
	if err != nil {
		t.Fatal(err)
	}
	fs.SetVolumeRecorder(recorder)
	file := openFile(t, fs, "/data/value")
	before := fs.VolumeSnapshots()
	if _, err := file.Write([]byte("rejected")); !errors.Is(err, volume.ErrVolumeCapacity) {
		t.Fatalf("recorder capacity write = %v", err)
	}
	if !reflect.DeepEqual(before, fs.VolumeSnapshots()) {
		t.Fatal("recorder capacity changed filesystem")
	}
	limited, err := volume.NewRecorder(volume.RecorderLimits{Records: 2, Operations: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := limited.BeforeVolumeOperations("data", []volume.Operation{{Dependencies: []uint64{1, 2}}}); !errors.Is(err, volume.ErrVolumeCapacity) {
		t.Fatalf("recorder dependency capacity = %v", err)
	}
	if len(limited.Records()) != 0 {
		t.Fatal("rejected recorder input was retained")
	}
}
