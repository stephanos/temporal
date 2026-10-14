//go:build test_dep

package wasi

import (
	"bytes"
	"fmt"
	"slices"
	"testing"
)

const volumeConformanceBinaryImageIdentity = StorageImageIdentity

func volumeConformanceEncodeImage(image *StorageImage, maximum uint64) ([]byte, string, error) {
	if image == nil || image.fs == nil {
		return nil, "", invalid("durable storage image")
	}
	return encodeStorageImage(image.fs, image.model, image.occurrences, maximum)
}

func TestVolumeConformanceImageIncludesHistoryAndCounters(t *testing.T) {
	config := storageConfig()
	e := newTestEnvironment(t, config)
	fd := openPath(t, e, 4, "db", 1, rightsAll)
	writeStorage(t, e, fd, 0, "durable")
	call(t, e, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd))
	call(t, e, "fd_sync", `{"fd":4}`)
	image, err := e.CrashStorage(nil)
	if err != nil {
		t.Fatal(err)
	}
	data, identity, err := volumeConformanceEncodeImage(image, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("GMVOLIMG\x01\x00\x00\x00\x03\x00\x00\x00")) {
		t.Fatal("image framing missing")
	}
	if identity == image.Identity() {
		t.Fatal("binary image retained incomplete snapshot identity")
	}
	before := image.fs.ConformanceState()
	copyImage := &StorageImage{fs: image.fs.CloneDurable(), model: slices.Clone(image.model), occurrences: cloneStorageOccurrences(image.occurrences)}
	_, clonedIdentity, err := volumeConformanceEncodeImage(copyImage, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if identity == clonedIdentity || before.NextInode == copyImage.fs.ConformanceState().NextInode {
		t.Fatal("recovery inode continuation missing from image identity")
	}
	if _, _, err := volumeConformanceEncodeImage(image, uint64(len(data)-1)); err == nil {
		t.Fatal("oversized image admitted")
	}
	image.occurrences["/workspace/db\x00write"]++
	_, changedIdentity, err := volumeConformanceEncodeImage(image, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if identity == changedIdentity {
		t.Fatal("occurrence continuation missing from image identity")
	}
}
