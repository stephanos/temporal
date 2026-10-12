package hostfs

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadBoundedAllocatesProportionallyToOpenedFileSize(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 1<<20)
	path := filepath.Join(t.TempDir(), "module")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	actual, err := ReadBounded(path, 512<<20)
	runtime.ReadMemStats(&after)
	if err != nil || !bytes.Equal(actual, data) {
		t.Fatalf("bounded read changed bytes: err=%v len=%d", err, len(actual))
	}
	allocated := after.TotalAlloc - before.TotalAlloc
	if allocated > 2*uint64(len(data))+64<<10 {
		t.Fatalf("bounded read allocated %d bytes for %d-byte opened file; exceeds proportional budget", allocated, len(data))
	}
}
