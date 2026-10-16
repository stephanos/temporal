package wasi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"path"
	"slices"
	"strings"
	"syscall"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

const StorageProfile = "gomad.wasi-native-volume/v1"

type StorageModel struct {
	Volumes []volume.VolumeConfig `json:"volumes"`
	Limits  volume.VolumeLimits   `json:"limits"`
	Faults  []StorageFault        `json:"faults,omitempty"`
	// Only ncruces sqlite3_dotlk's process-owned lock artifacts are ephemeral.
	// Database, rollback journal, and WAL files retain the native volume contract.
	SQLiteLockDatabases []string `json:"sqlite_lock_databases,omitempty"`
}

type StorageFault struct {
	Name       string `json:"name"`
	Resource   string `json:"resource"`
	Operation  string `json:"operation"`
	Occurrence uint64 `json:"occurrence"`
	Action     string `json:"action"`
	Bytes      uint64 `json:"bytes,omitempty"`
}

type StorageEvent struct {
	Sequence   uint64 `json:"sequence"`
	Resource   string `json:"resource"`
	Operation  string `json:"operation"`
	Occurrence uint64 `json:"occurrence"`
	Fault      string `json:"fault,omitempty"`
	Action     string `json:"action"`
	Bytes      uint64 `json:"bytes,omitempty"`
}

type StorageCrash struct{ Fault string }

func (e *StorageCrash) Error() string { return "modeled storage crash: " + e.Fault }

type storageState struct {
	fs          *volume.FS
	model       StorageModel
	occurrences map[string]uint64
	log         []byte
	sequence    uint64
	clock       *volume.Clock
}

// StorageImage contains only the durable filesystem of a discarded guest.
// Its private owner prevents callers from mutating a recovery observation.
// Its identity is the SHA-256 of the self-contained recovery image v1 blob.
// Go-owned images also keep the Go volume for Go-owned recovery; helper
// images hold the blob when the request asked for it.
type StorageImage struct {
	fs          *volume.FS
	data        []byte
	identity    string
	model       []byte
	occurrences map[string]uint64
}

func (image *StorageImage) Identity() string { return image.identity }

// Bytes is the size of the recovery image blob, or zero when only its
// identity was returned.
func (image *StorageImage) Bytes() int { return len(image.data) }

func storageModelIdentity(model []byte) string {
	digest := sha256.Sum256(model)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validateStorageModel(model *StorageModel, config Config) error {
	if model == nil {
		return nil
	}
	if len(model.Volumes) == 0 || len(model.Volumes) > 64 || len(model.Faults) > 4096 {
		return invalid("storage model bounds")
	}
	if len(model.SQLiteLockDatabases) > 64 {
		return invalid("SQLite lock adapter bounds")
	}
	for i, database := range model.SQLiteLockDatabases {
		mounted := false
		for _, mount := range model.Volumes {
			mounted = mounted || strings.HasPrefix(database, mount.Path+"/")
		}
		if !absoluteClean(database) || len(database)+5 > volume.MaximumPathBytes || !mounted || i > 0 && database <= model.SQLiteLockDatabases[i-1] {
			return invalid("SQLite lock adapter database paths")
		}
		for _, other := range model.SQLiteLockDatabases {
			if database != other && (sqliteLockPath(model, database) || strings.HasPrefix(other, database+"/")) {
				return invalid("SQLite lock adapter path collision")
			}
		}
	}
	seen := map[string]bool{}
	var capacityBytes uint64
	for _, mount := range model.Volumes {
		if !slices.Contains(config.WritableDirectories, mount.Path) || mount.CapacityBytes > config.Limits.FilesystemBytes {
			return invalid("storage volume writable preopen or capacity")
		}
		if mount.CapacityBytes > config.Limits.FilesystemBytes-capacityBytes {
			return invalid("storage total volume capacity")
		}
		capacityBytes += mount.CapacityBytes
	}
	for _, fault := range model.Faults {
		mounted := false
		for _, mount := range model.Volumes {
			mounted = mounted || fault.Resource == mount.Path || strings.HasPrefix(fault.Resource, mount.Path+"/")
		}
		if fault.Name == "" || len(fault.Name) > 128 || !utf8.ValidString(fault.Name) || strings.IndexByte(fault.Name, 0) >= 0 || seen[fault.Name] || !absoluteClean(fault.Resource) || len(fault.Resource) > volume.MaximumPathBytes || !mounted || fault.Occurrence == 0 {
			return invalid("storage fault identity")
		}
		if sqliteLockPath(model, fault.Resource) {
			return invalid("storage fault targets process-owned SQLite lock")
		}
		seen[fault.Name] = true
		switch fault.Operation {
		case "read", "write", "sync", "truncate", "open", "mkdir", "remove", "rename":
		default:
			return invalid("storage fault operation")
		}
		switch fault.Action {
		case "io-error", "crash-before":
			if fault.Bytes != 0 {
				return invalid("storage error payload")
			}
		case "sync-error":
			if fault.Operation != "sync" || fault.Bytes != 0 {
				return invalid("storage sync fault")
			}
		case "no-space":
			if fault.Operation != "write" && fault.Operation != "truncate" && fault.Operation != "open" && fault.Operation != "mkdir" || fault.Bytes != 0 {
				return invalid("storage space fault")
			}
		case "short-write", "torn-write":
			if fault.Operation != "write" || fault.Bytes == 0 || fault.Bytes > bufferLimit {
				return invalid("storage bounded write fault")
			}
		default:
			return invalid("storage fault action")
		}
		for _, prior := range model.Faults {
			if prior.Name != fault.Name && prior.Resource == fault.Resource && prior.Operation == fault.Operation && prior.Occurrence == fault.Occurrence {
				return invalid("ambiguous storage fault match")
			}
		}
	}
	return nil
}

func (ns *namespace) initializeStorage(config Config, fs *volume.FS) error {
	if err := validateStorageModel(config.Storage, config); err != nil {
		return err
	}
	if config.Storage == nil {
		return nil
	}
	model := *config.Storage
	model.Volumes = slices.Clone(model.Volumes)
	model.Faults = slices.Clone(model.Faults)
	model.SQLiteLockDatabases = slices.Clone(model.SQLiteLockDatabases)
	if fs == nil {
		fs = volume.New()
		if err := fs.ConfigureVolumes(model.Volumes, model.Limits); err != nil {
			return invalid("storage volume configuration")
		}
	}
	ns.storage = &storageState{fs: fs, model: model, occurrences: map[string]uint64{}, clock: &volume.Clock{}}
	fs.SetClock(ns.storage.clock)
	for _, mount := range model.Volumes {
		n := ns.lookup(mount.Path)
		if n == nil || !n.directory || len(n.children) != 0 {
			return invalid("storage mount must be an empty writable directory")
		}
		n.storagePath = mount.Path
		if err := ns.loadStorageTree(n); err != nil {
			return err
		}
	}
	if ns.files > config.Limits.Files || ns.bytes > config.Limits.FilesystemBytes {
		return capacity("recovered-namespace")
	}
	for _, d := range ns.descriptors {
		if d.node.storagePath != "" {
			if d.storage != nil {
				if err := d.storage.Close(); err != nil {
					return err
				}
			}
			h, err := fs.Open(d.node.storagePath, volume.OpenFlags{Read: true}, 0)
			if err != nil {
				return err
			}
			d.storage = h
		}
	}
	return nil
}

func (ns *namespace) loadStorageTree(n *node) error {
	h, err := ns.storage.fs.Open(n.storagePath, volume.OpenFlags{Read: true}, 0)
	if err != nil {
		return err
	}
	entries, err := h.ReadDir(-1)
	closeErr := h.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	for _, entry := range entries {
		child := ns.newNode(entry.Kind == volume.KindDirectory, false)
		child.storagePath = path.Join(n.storagePath, entry.Name)
		if sqliteLockPath(&ns.storage.model, child.storagePath) {
			return invalid("durable image contains process-owned SQLite lock")
		}
		child.modified = uint64(max(0, entry.ModTime))
		child.storageSize = uint64(len(entry.Data))
		ns.bytes += child.storageSize
		n.children[entry.Name] = child
		if child.directory {
			if err := ns.loadStorageTree(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Environment) storageDecision(resource, operation string, count uint64) (uint64, uint16, error) {
	s := e.namespace.storage
	s.clock.Value = int64(e.now)
	key := resource + "\x00" + operation
	s.occurrences[key]++
	event := StorageEvent{Sequence: s.sequence, Resource: resource, Operation: operation, Occurrence: s.occurrences[key], Action: "complete", Bytes: count}
	var errno uint16
	var crash error
	for _, fault := range s.model.Faults {
		if fault.Resource != resource || fault.Operation != operation || fault.Occurrence != event.Occurrence {
			continue
		}
		event.Fault, event.Action = fault.Name, fault.Action
		switch fault.Action {
		case "io-error", "sync-error":
			errno = 29
		case "no-space":
			errno = 51
		case "short-write", "torn-write":
			if count <= fault.Bytes {
				return 0, 0, invalid("storage write fault must shorten the matched write")
			}
			count = fault.Bytes
			event.Bytes = count
			if fault.Action == "torn-write" {
				errno = 29
			}
		case "crash-before":
			crash = &StorageCrash{Fault: fault.Name}
		}
		break
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return 0, 0, err
	}
	if uint64(len(s.log)+len(encoded)+1) > e.config.Limits.TranscriptBytes {
		return 0, 0, capacity("storage-transcript")
	}
	s.log = append(s.log, encoded...)
	s.log = append(s.log, '\n')
	s.sequence++
	return count, errno, crash
}

func storageErrno(err error) uint16 {
	if err == nil || err == io.EOF {
		return 0
	}
	if err == volume.ErrClosed {
		return 8
	}
	if space, ok := err.(*volume.VolumeCapacityError); ok && space.Resource == "volume_bytes" {
		return 51
	}
	native, ok := err.(syscall.Errno)
	if !ok {
		return 29
	}
	for _, pair := range []struct {
		err   syscall.Errno
		errno uint16
	}{{syscall.EACCES, 2}, {syscall.EBADF, 8}, {syscall.EBUSY, 10}, {syscall.EEXIST, 20}, {syscall.EFBIG, 22}, {syscall.EINVAL, 28}, {syscall.EIO, 29}, {syscall.EISDIR, 31}, {syscall.EMFILE, 33}, {syscall.ENOENT, 44}, {syscall.ENOMEM, 48}, {syscall.ENOSPC, 51}, {syscall.ENOTDIR, 54}, {syscall.ENOTEMPTY, 55}, {syscall.EPERM, 63}, {syscall.EROFS, 69}, {syscall.ESTALE, 72}, {syscall.EXDEV, 75}} {
		if native == pair.err {
			return pair.errno
		}
	}
	return 29
}

func storageBoundary(err error) error {
	if exhausted, ok := err.(*volume.VolumeCapacityError); ok && exhausted.Resource != "volume_bytes" {
		return capacity("storage-" + exhausted.Resource)
	}
	return nil
}

func (e *Environment) StorageTranscript() []byte {
	if e.namespace.storage == nil {
		return nil
	}
	return slices.Clone(e.namespace.storage.log)
}
func (e *Environment) StorageSnapshots() []volume.VolumeSnapshot {
	if e.namespace.storage == nil {
		return nil
	}
	return e.namespace.storage.fs.VolumeSnapshots()
}
func (e *Environment) StorageOperations() map[string][]volume.Operation {
	if e.namespace.storage == nil {
		return nil
	}
	return e.namespace.storage.fs.Operations()
}
func (e *Environment) EnumerateStorageCrashStates(id string, limits volume.CrashEnumerationLimits, exploration *volume.CrashExploration) (volume.CrashEnumeration, error) {
	if e.namespace.storage == nil {
		return volume.CrashEnumeration{}, invalid("storage is not configured")
	}
	return e.namespace.storage.fs.EnumerateCrashStates(id, limits, exploration)
}
func (e *Environment) CrashStorage(selections map[string][]uint64) (*StorageImage, error) {
	if e.namespace.storage == nil {
		return nil, invalid("storage is not configured")
	}
	fs, err := e.namespace.storage.fs.AdvanceVolumeLifecycle(false, selections)
	if err != nil {
		return nil, err
	}
	occurrences := cloneStorageOccurrences(e.namespace.storage.occurrences)
	model, err := json.Marshal(e.namespace.storage.model)
	if err != nil {
		return nil, err
	}
	data, identity, err := encodeStorageImage(fs, model, occurrences, storageImageLimit(e.config))
	if err != nil {
		return nil, err
	}
	// Descriptor numbers and pending runtime work belong to the discarded incarnation.
	e.namespace.descriptors = map[uint32]*descriptor{}
	return &StorageImage{fs: fs, data: data, identity: identity, model: model, occurrences: occurrences}, nil
}
func NewRecoveredEnvironment(config Config, image *StorageImage) (*Environment, error) {
	if image == nil || image.fs == nil && image.data == nil {
		return nil, invalid("durable storage image")
	}
	model, err := json.Marshal(config.Storage)
	if err != nil {
		return nil, err
	}
	if string(model) != string(image.model) {
		return nil, invalid("recovery storage model identity")
	}
	e, err := NewEnvironment(config)
	if err != nil {
		return nil, err
	}
	if image.fs == nil {
		// Only the helper can load a helper image; a Go-owned run refuses it.
		e.storageRecovery = image
		e.identity = recoveredEnvironmentIdentity(e.identity, image.identity)
		return e, nil
	}
	fs := image.fs.CloneDurable()
	if err := e.namespace.initializeStorage(config, fs); err != nil {
		return nil, err
	}
	e.namespace.storage.occurrences = cloneStorageOccurrences(image.occurrences)
	e.storageRecovery = image
	e.identity = recoveredEnvironmentIdentity(e.identity, image.identity)
	return e, nil
}

func recoveredEnvironmentIdentity(base, imageIdentity string) string {
	digest := sha256.Sum256([]byte(base + "\x00" + imageIdentity))
	return hex.EncodeToString(digest[:])
}

func cloneStorageOccurrences(source map[string]uint64) map[string]uint64 {
	result := make(map[string]uint64, len(source))
	for resource, occurrence := range source {
		result[resource] = occurrence
	}
	return result
}

func (ns *namespace) storagePath(parent *node, name string) string {
	if parent.storagePath == "" {
		return ""
	}
	if name == "" {
		return parent.storagePath
	}
	resource := path.Join(parent.storagePath, name)
	if sqliteLockPath(&ns.storage.model, resource) {
		return ""
	}
	return resource
}

func sqliteLockPath(model *StorageModel, resource string) bool {
	for _, database := range model.SQLiteLockDatabases {
		if resource == database+".lock" || resource == database+"-shm" || strings.HasPrefix(resource, database+"-shm/") {
			return true
		}
	}
	return false
}
func relocateStorageNode(n *node, name string) {
	n.storagePath = name
	for childName, child := range n.children {
		relocateStorageNode(child, path.Join(name, childName))
	}
}

func (e *Environment) storageWrite(d *descriptor, input []byte, offset uint64) (any, uint16, string, error) {
	count, errno, err := e.storageDecision(d.node.storagePath, "write", uint64(len(input)))
	if err != nil {
		return nil, 0, "", err
	}
	if errno != 0 && count == uint64(len(input)) {
		return nil, errno, "", nil
	}
	finalSize := d.node.storageSize
	if count > 0 {
		finalSize = max(finalSize, offset+count)
	}
	if err := e.storageResizeAllowed(d.node, finalSize); err != nil {
		return nil, 0, "", err
	}
	written, writeErr := d.storage.WriteAt(input[:count], int64(offset))
	if writeErr != nil {
		return nil, storageErrno(writeErr), "", storageBoundary(writeErr)
	}
	if errno != 0 {
		return nil, errno, "", nil
	}
	return writtenOutput{uint32(written)}, 0, "", nil
}

func (e *Environment) storageResizeAllowed(n *node, size uint64) error {
	maximum := e.config.Limits.FilesystemBytes
	if size > maximum || e.namespace.bytes-n.storageSize > maximum-size {
		return capacity("filesystem-bytes")
	}
	return nil
}

func (e *Environment) refreshStorageNode(d *descriptor) error {
	entry, size, err := d.storage.Metadata()
	if err != nil {
		return err
	}
	n := d.node
	e.namespace.bytes = e.namespace.bytes - n.storageSize + size
	n.storageSize = size
	n.modified = uint64(max(0, entry.ModTime))
	return nil
}
