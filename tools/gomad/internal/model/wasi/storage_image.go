package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"slices"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

// StorageImageIdentity names recovery image v1.
const StorageImageIdentity = "gomad.volume-recovery-image/v1"

// storageImageMetadataBytes bounds recovery history and metadata beyond the
// filesystem byte limit; the helper uses the same bound. Measured on the
// Temporal crash/reopen gate (fs limit 512 MiB, images <= 7.5 MB) and the
// SQLite fixtures (fs limit 8 MiB, images <= 85 KB).
const storageImageMetadataBytes = 1 << 20

func storageImageLimit(config Config) uint64 {
	return config.Limits.FilesystemBytes + storageImageMetadataBytes
}

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

// encodeStorageImage writes recovery image v1 (see wasmhost protocol.md):
// the storage-model identity, sorted fault occurrences and the complete
// durable volume state. Its SHA-256 is the image identity.
func encodeStorageImage(source *volume.FS, modelJSON []byte, counts map[string]uint64, maximum uint64) ([]byte, string, error) {
	state := source.ConformanceState()
	var model, occurrences, fs volumeImageEncoder
	modelDigest := sha256.Sum256(modelJSON)
	model.text("sha256:" + hex.EncodeToString(modelDigest[:]))
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	occurrences.number(uint64(len(keys)))
	for _, key := range keys {
		occurrences.text(key)
		occurrences.number(counts[key])
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
