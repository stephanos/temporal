package volume_test

import (
	"errors"
	"reflect"
	"runtime"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad3/volume"
)

func TestMetadataReturnsDetachedAttributesWithoutPayload(t *testing.T) {
	fs := newFilesystem(t)
	file := openFile(t, fs, "/data/value")
	writeAt(t, file, "value", 0)
	if err := file.Chmod(0o640); err != nil {
		t.Fatal(err)
	}
	if err := file.Chtimes(456); err != nil {
		t.Fatal(err)
	}
	entry, size, err := file.Metadata()
	if err != nil {
		t.Fatal(err)
	}
	expected := volume.Entry{Name: "value", Mode: 0o640, Kind: volume.KindFile, ModTime: 456}
	if !reflect.DeepEqual(entry, expected) || size != 5 {
		t.Fatalf("metadata = %#v, %d", entry, size)
	}
	entry.Name, entry.Mode, entry.ModTime = "changed", 0, 0
	next, _, err := file.Metadata()
	if err != nil || !reflect.DeepEqual(next, expected) {
		t.Fatalf("detached metadata = %#v, %v", next, err)
	}
	if err := file.Truncate(2); err != nil {
		t.Fatal(err)
	}
	if _, size, err := file.Metadata(); err != nil || size != 2 {
		t.Fatalf("truncated metadata = %d, %v", size, err)
	}
	directory, err := fs.Open("/data", volume.OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := directory.Close(); err != nil {
			t.Error(err)
		}
	}()
	entry, size, err = directory.Metadata()
	if err != nil || entry.Name != "data" || entry.Kind != volume.KindDirectory || size != 0 || entry.Data != nil || entry.Children != nil {
		t.Fatalf("directory metadata = %#v, %d, %v", entry, size, err)
	}
}

func TestMetadataRejectsClosedAndStaleHandles(t *testing.T) {
	fs := newFilesystem(t)
	closed, err := fs.Open("/data/value", volume.OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	if entry, size, err := closed.Metadata(); !errors.Is(err, syscall.EBADF) || size != 0 || !reflect.DeepEqual(entry, volume.Entry{}) {
		t.Fatalf("closed metadata = %#v, %d, %v", entry, size, err)
	}
	stale := openFile(t, fs, "/data/stale")
	if err := fs.CrashVolumes(nil); err != nil {
		t.Fatal(err)
	}
	if entry, size, err := stale.Metadata(); !errors.Is(err, syscall.ESTALE) || size != 0 || !reflect.DeepEqual(entry, volume.Entry{}) {
		t.Fatalf("stale metadata = %#v, %d, %v", entry, size, err)
	}
	for _, invalid := range []*volume.Handle{nil, {}} {
		if _, _, err := invalid.Metadata(); !errors.Is(err, syscall.EBADF) {
			t.Fatalf("invalid handle metadata = %v", err)
		}
	}
}

func TestMetadataAllocationsDoNotCopyLargeFile(t *testing.T) {
	fs := newFilesystem(t)
	file := openFile(t, fs, "/data/large")
	if err := file.Truncate(1 << 19); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 100 {
		entry, size, err := file.Metadata()
		if err != nil || size != 1<<19 || entry.Data != nil {
			t.Fatalf("metadata = %#v, %d, %v", entry, size, err)
		}
	}
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 1<<20 {
		t.Fatalf("metadata allocated %d bytes for 100 reads; file contents must not be copied", allocated)
	}
}
