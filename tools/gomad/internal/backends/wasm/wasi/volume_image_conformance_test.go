//go:build test_dep

package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

const volumeConformanceBinaryImageIdentity = "gomad3.volume-recovery-image/v1"

type volumeImageEncoder struct{ bytes.Buffer }

func (e *volumeImageEncoder) number(value uint64) {
	var data [8]byte
	binary.LittleEndian.PutUint64(data[:], value)
	e.Write(data[:])
}
func (e *volumeImageEncoder) blob(value []byte) { e.number(uint64(len(value))); e.Write(value) }
func (e *volumeImageEncoder) text(value string) { e.blob([]byte(value)) }
func (e *volumeImageEncoder) numbers(values []uint64) {
	e.number(uint64(len(values)))
	for _, value := range values {
		e.number(value)
	}
}
func (e *volumeImageEncoder) texts(values []string) {
	e.number(uint64(len(values)))
	for _, value := range values {
		e.text(value)
	}
}
func (e *volumeImageEncoder) object(object volume.ConformanceDiskObject) {
	e.number(object.Inode)
	e.number(uint64(object.Mode))
	e.number(uint64(object.Kind))
	e.number(uint64(object.ModTime))
	e.blob(object.Data)
}
func (e *volumeImageEncoder) paths(paths []volume.ConformancePath) {
	e.number(uint64(len(paths)))
	for _, entry := range paths {
		e.text(entry.Path)
		e.number(entry.Inode)
	}
}
func (e *volumeImageEncoder) operations(operations []volume.ConformancePersistenceOperation) {
	e.number(uint64(len(operations)))
	for _, operation := range operations {
		e.number(operation.ID)
		e.text(operation.Kind)
		e.numbers(operation.Dependencies)
		e.texts(operation.Stores)
		effect := operation.Effect
		e.text(effect.Kind)
		e.number(effect.Inode)
		e.number(effect.Parent)
		e.numbers(effect.Parents)
		e.number(effect.Offset)
		e.number(effect.Size)
		e.number(uint64(effect.Mode))
		e.number(uint64(effect.ModTime))
		e.blob(effect.Data)
		if effect.Object == nil {
			e.number(0)
		} else {
			e.number(1)
			e.object(*effect.Object)
		}
		e.paths(effect.Namespace)
	}
}

func volumeConformanceEncodeImage(image *StorageImage, maximum uint64) ([]byte, string, error) {
	if image == nil || image.fs == nil {
		return nil, "", invalid("durable storage image")
	}
	state := image.fs.ConformanceState()
	var model, occurrences, fs volumeImageEncoder
	modelDigest := sha256.Sum256(image.model)
	model.text("sha256:" + hex.EncodeToString(modelDigest[:]))
	keys := make([]string, 0, len(image.occurrences))
	for key := range image.occurrences {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	occurrences.number(uint64(len(keys)))
	for _, key := range keys {
		occurrences.text(key)
		occurrences.number(image.occurrences[key])
	}
	fs.number(state.NextInode)
	fs.number(state.Generation)
	fs.number(uint64(len(state.Volumes)))
	for _, v := range state.Volumes {
		fs.text(v.ID)
		fs.text(v.Mount)
		fs.number(v.CapacityBytes)
		fs.number(uint64(len(v.Persisted.Objects)))
		for _, object := range v.Persisted.Objects {
			fs.object(object)
		}
		fs.paths(v.Persisted.Paths)
		fs.numbers(v.Persisted.Owned)
		fs.operations(v.Pending)
		fs.operations(v.History)
		fs.number(v.NextOperation)
		fs.number(uint64(len(v.Last)))
		for _, entry := range v.Last {
			fs.text(entry.Key)
			fs.number(entry.Operation)
		}
	}
	var result volumeImageEncoder
	result.WriteString("GMVOLIMG")
	var header [8]byte
	binary.LittleEndian.PutUint32(header[:4], 1)
	binary.LittleEndian.PutUint32(header[4:], 3)
	result.Write(header[:])
	for index, section := range [][]byte{model.Bytes(), occurrences.Bytes(), fs.Bytes()} {
		var tag [4]byte
		binary.LittleEndian.PutUint32(tag[:], uint32(index+1))
		result.Write(tag[:])
		result.blob(section)
	}
	if uint64(result.Len()) > maximum {
		return nil, "", capacity("storage-image-bytes")
	}
	data := result.Bytes()
	digest := sha256.Sum256(data)
	return slices.Clone(data), hex.EncodeToString(digest[:]), nil
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
