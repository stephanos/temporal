package volume

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"path"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
)

// DecodeCrashRecoveryImage imports full image state for the original crash
// enumerator. It does not recover the namespace or discard pending effects.
func DecodeCrashRecoveryImage(data []byte, digest, model string, configs []VolumeConfig, limits VolumeLimits, locks []string, maximum uint64) (*FS, error) {
	if uint64(len(data)) > maximum {
		return nil, errors.New("capacity:image-bytes")
	}
	if len(digest) != 64 || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
		return nil, errors.New("recovery image digest mismatch")
	}
	frame := crashImageReader{data: data}
	if string(frame.take(8)) != "GMVOLIMG" || frame.small() != 1 || frame.small() != 3 {
		return nil, errors.New("recovery image format")
	}
	var sections [3][]byte
	for i := range sections {
		if frame.small() != uint32(i+1) {
			return nil, errors.New("recovery image sections")
		}
		sections[i] = frame.blob()
		if frame.err != nil {
			return nil, frame.err
		}
	}
	if err := frame.finish(); err != nil {
		return nil, err
	}
	identity := crashImageReader{data: sections[0]}
	actualModel := identity.text()
	if err := identity.finish(); err != nil {
		return nil, err
	}
	if actualModel != model || !validCrashImageDigest(model) {
		return nil, errors.New("recovery storage model identity")
	}
	occurrences := crashImageReader{data: sections[1]}
	previous := ""
	for n := occurrences.count(); n > 0; n-- {
		key, count := occurrences.text(), occurrences.number()
		if occurrences.err != nil {
			return nil, occurrences.err
		}
		if key <= previous {
			return nil, errors.New("recovery state ordering")
		}
		previous = key
		resource, operation, ok := strings.Cut(key, "\x00")
		if !ok || count == 0 || !crashImagePath(resource) || !slices.Contains([]string{"read", "write", "sync", "truncate", "open", "mkdir", "remove", "rename"}, operation) {
			return nil, errors.New("recovery fault occurrence")
		}
		if crashImageLock(resource, locks) {
			return nil, errors.New("durable image contains process-owned SQLite lock")
		}
	}
	if err := occurrences.finish(); err != nil {
		return nil, err
	}
	fs := New()
	if err := fs.ConfigureVolumes(configs, limits); err != nil {
		return nil, errors.New("recovery volume configuration")
	}
	fs.volumes = make(map[string]*volumeState)
	state := crashImageReader{data: sections[2]}
	fs.nextInode, fs.generation = state.number(), state.number()
	if state.err != nil {
		return nil, state.err
	}
	if fs.nextInode == 0 || fs.generation == 0 {
		return nil, errors.New("recovery filesystem counters")
	}
	previous = ""
	inodes := map[uint64]bool{1: true, 2: true}
	var maximumInode uint64
	for n := state.count(); n > 0; n-- {
		v := &volumeState{id: state.text(), mount: state.text(), capacityBytes: state.number(), persisted: &diskState{objects: make(map[uint64]*diskObject), paths: make(map[string]uint64), owned: make(map[uint64]struct{})}}
		if state.err != nil {
			return nil, state.err
		}
		if v.id <= previous {
			return nil, errors.New("recovery state ordering")
		}
		previous = v.id
		if !slices.Contains(configs, VolumeConfig{ID: v.id, Path: v.mount, CapacityBytes: v.capacityBytes}) {
			return nil, errors.New("recovery volume configuration")
		}
		var priorInode uint64
		for k := state.count(); k > 0; k-- {
			object := state.object()
			if state.err != nil {
				return nil, state.err
			}
			if object.inode <= priorInode {
				return nil, errors.New("recovery state ordering")
			}
			priorInode = object.inode
			if inodes[object.inode] {
				return nil, errors.New("recovery object inode alias")
			}
			inodes[object.inode] = true
			maximumInode = max(maximumInode, object.inode)
			v.persisted.objects[object.inode] = object
		}
		priorPath := ""
		pathInodes := make(map[uint64]bool)
		for k := state.count(); k > 0; k-- {
			name, inode := state.text(), state.number()
			if state.err != nil {
				return nil, state.err
			}
			if name <= priorPath {
				return nil, errors.New("recovery state ordering")
			}
			priorPath = name
			if !crashImagePath(name) || v.persisted.objects[inode] == nil {
				return nil, errors.New("recovery durable tree")
			}
			if pathInodes[inode] {
				return nil, errors.New("recovery path inode alias")
			}
			pathInodes[inode] = true
			resource := v.mount
			if name != "/" {
				resource += name
			}
			if crashImageLock(resource, locks) {
				return nil, errors.New("durable image contains process-owned SQLite lock")
			}
			v.persisted.paths[name] = inode
		}
		priorInode = 0
		for _, inode := range state.numbers() {
			if inode <= priorInode {
				return nil, errors.New("recovery state ordering")
			}
			priorInode = inode
			if v.persisted.objects[inode] == nil {
				return nil, errors.New("recovery object ownership")
			}
			v.persisted.owned[inode] = struct{}{}
		}
		v.pending, v.history = state.operations(), state.operations()
		if uint64(len(v.pending)) > limits.PendingOperations || uint64(len(v.history)) > limits.Transitions {
			return nil, errors.New("recovery operation limits")
		}
		v.nextOperation = state.number()
		if state.err != nil {
			return nil, state.err
		}
		if v.nextOperation != uint64(len(v.history))+1 {
			return nil, errors.New("recovery operation counter")
		}
		for i, operation := range v.history {
			if operation.id != uint64(i)+1 {
				return nil, errors.New("recovery operation history")
			}
			if err := validateCrashImageOperation(operation); err != nil {
				return nil, err
			}
			maximumInode = max(maximumInode, operation.effect.inode)
			if operation.effect.object != nil {
				maximumInode = max(maximumInode, operation.effect.object.inode)
			}
			for _, part := range operation.effect.namespace {
				if crashImageLock(v.mount+part.path, locks) {
					return nil, errors.New("durable image contains process-owned SQLite lock")
				}
			}
		}
		priorOperation := uint64(0)
		expectedLast := make(map[string]uint64)
		for _, operation := range v.pending {
			if operation.id <= priorOperation {
				return nil, errors.New("recovery state ordering")
			}
			priorOperation = operation.id
			if operation.id > uint64(len(v.history)) || !reflect.DeepEqual(operation, v.history[operation.id-1]) {
				return nil, errors.New("recovery pending operation")
			}
			for _, key := range operation.stores {
				expectedLast[key] = operation.id
			}
		}
		v.last = make(map[string]uint64)
		priorKey := ""
		for k := state.count(); k > 0; k-- {
			key, operation := state.text(), state.number()
			if state.err != nil {
				return nil, state.err
			}
			if key <= priorKey {
				return nil, errors.New("recovery state ordering")
			}
			priorKey = key
			v.last[key] = operation
		}
		if !reflect.DeepEqual(v.last, expectedLast) {
			return nil, errors.New("recovery pending operation stores")
		}
		root := v.persisted.objects[v.persisted.paths["/"]]
		if root == nil || root.kind != KindDirectory {
			return nil, errors.New("recovery durable tree")
		}
		fs.volumes[v.id] = v
	}
	if err := state.finish(); err != nil {
		return nil, err
	}
	if len(fs.volumes) != len(configs) {
		return nil, errors.New("recovery volume configuration")
	}
	// Rebuilding consumes one temporary inode and one for each missing mount component.
	mountPaths := map[string]bool{"/": true, TempDirectory: true}
	allocations := uint64(1)
	for _, config := range configs {
		for current := config.Path; current != "/"; current = path.Dir(current) {
			if !mountPaths[current] {
				mountPaths[current] = true
				allocations++
			}
		}
	}
	if fs.nextInode <= maximumInode || fs.nextInode > math.MaxUint64-allocations {
		return nil, errors.New("recovery filesystem counters")
	}
	return fs, nil
}

func validCrashImageDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, c := range value[7:] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
func crashImagePath(value string) bool {
	normalized, _, err := Normalize(value)
	return err == nil && strings.HasPrefix(value, "/") && normalized == value
}
func crashImageLock(resource string, databases []string) bool {
	for _, database := range databases {
		if resource == database+".lock" || resource == database+"-shm" || strings.HasPrefix(resource, database+"-shm/") {
			return true
		}
	}
	return false
}
func validateCrashImageOperation(operation *persistenceOperation) error {
	effect := operation.effect
	if operation.kind != effect.kind || !slices.Contains([]string{"alloc", "resize", "write", "metadata", "namespace"}, effect.kind) {
		return errors.New("recovery persistence effect")
	}
	previous := uint64(0)
	for _, dependency := range operation.deps {
		if dependency <= previous {
			return errors.New("recovery state ordering")
		}
		previous = dependency
		if dependency >= operation.id {
			return errors.New("recovery operation dependencies")
		}
	}
	if effect.kind == "alloc" && (effect.object == nil || effect.object.inode != effect.inode) || effect.kind != "alloc" && effect.object != nil {
		return errors.New("recovery persistence effect")
	}
	if effect.size > MaximumFileBytes || effect.offset > MaximumFileBytes || uint64(len(effect.data)) > MaximumFileBytes-effect.offset {
		return errors.New("recovery persistence effect")
	}
	for _, part := range effect.namespace {
		if part.path == "/" || !crashImagePath(part.path) {
			return errors.New("recovery persistence effect")
		}
	}
	return nil
}

type crashImageReader struct {
	data []byte
	err  error
}

func (r *crashImageReader) fail(message string) {
	if r.err == nil {
		r.err = errors.New(message)
	}
}
func (r *crashImageReader) take(n uint64) []byte {
	if r.err != nil {
		return nil
	}
	if n > uint64(len(r.data)) {
		r.fail("truncated recovery image")
		return nil
	}
	data := r.data[:int(n)]
	r.data = r.data[int(n):]
	return data
}
func (r *crashImageReader) number() uint64 {
	data := r.take(8)
	if len(data) != 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(data)
}
func (r *crashImageReader) small() uint32 {
	data := r.take(4)
	if len(data) != 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(data)
}
func (r *crashImageReader) count() uint64 {
	n := r.number()
	if n > uint64(len(r.data))/8 {
		r.fail("truncated recovery image")
		return 0
	}
	return n
}
func (r *crashImageReader) blob() []byte { return r.take(r.number()) }
func (r *crashImageReader) text() string {
	data := r.blob()
	if !utf8.Valid(data) {
		r.fail("recovery image UTF-8")
	}
	return string(data)
}
func (r *crashImageReader) finish() error {
	if r.err != nil {
		return r.err
	}
	if len(r.data) != 0 {
		return errors.New("recovery image trailing bytes")
	}
	return nil
}
func (r *crashImageReader) numbers() []uint64 {
	var values []uint64
	for n := r.count(); n > 0 && r.err == nil; n-- {
		values = append(values, r.number())
	}
	return values
}
func (r *crashImageReader) mode(message string) uint32 {
	value := r.number()
	if value > math.MaxUint32 {
		r.fail(message)
	}
	return uint32(value)
}
func (r *crashImageReader) object() *diskObject {
	inode, mode, kind := r.number(), r.mode("recovery object mode"), r.number()
	if kind != uint64(KindFile) && kind != uint64(KindDirectory) {
		r.fail("recovery object kind")
	}
	object := &diskObject{inode: inode, mode: mode, kind: Kind(kind), modTime: int64(r.number()), data: slices.Clone(r.blob())}
	if object.inode == 0 || len(object.data) > MaximumFileBytes || object.kind == KindDirectory && len(object.data) != 0 {
		r.fail("recovery object state")
	}
	return object
}
func (r *crashImageReader) operations() []*persistenceOperation {
	var operations []*persistenceOperation
	for n := r.count(); n > 0 && r.err == nil; n-- {
		operation := &persistenceOperation{id: r.number(), kind: r.text(), deps: r.numbers()}
		for k := r.count(); k > 0 && r.err == nil; k-- {
			operation.stores = append(operation.stores, r.text())
		}
		effect := persistenceEffect{kind: r.text(), inode: r.number(), parent: r.number(), parents: r.numbers(), offset: r.number(), size: r.number(), mode: r.mode("recovery effect mode"), modTime: int64(r.number()), data: slices.Clone(r.blob())}
		switch r.number() {
		case 0:
		case 1:
			effect.object = r.object()
		default:
			r.fail("recovery object presence")
		}
		for k := r.count(); k > 0 && r.err == nil; k-- {
			effect.namespace = append(effect.namespace, namespaceEffect{path: r.text(), inode: r.number()})
		}
		operation.effect = effect
		operations = append(operations, operation)
	}
	return operations
}
