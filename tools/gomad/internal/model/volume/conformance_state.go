package volume

import (
	"slices"
	"sort"
)

type ConformanceState struct {
	NextInode  uint64
	Generation uint64
	Volumes    []ConformanceVolumeState
}

type ConformanceVolumeState struct {
	ID            string
	Mount         string
	CapacityBytes uint64
	Persisted     ConformanceDiskState
	Pending       []ConformancePersistenceOperation
	History       []ConformancePersistenceOperation
	NextOperation uint64
	Last          []ConformanceLast
}

type ConformanceDiskState struct {
	Objects []ConformanceDiskObject
	Paths   []ConformancePath
	Owned   []uint64
}

type ConformanceDiskObject struct {
	Inode   uint64
	Mode    uint32
	Kind    Kind
	ModTime int64
	Data    []byte
}

type ConformancePath struct {
	Path  string
	Inode uint64
}

type ConformanceLast struct {
	Key       string
	Operation uint64
}

type ConformancePersistenceOperation struct {
	ID           uint64
	Kind         string
	Dependencies []uint64
	Stores       []string
	Effect       ConformanceEffect
}

type ConformanceEffect struct {
	Kind      string
	Inode     uint64
	Parent    uint64
	Parents   []uint64
	Offset    uint64
	Size      uint64
	Mode      uint32
	ModTime   int64
	Data      []byte
	Object    *ConformanceDiskObject
	Namespace []ConformancePath
}

// ConformanceState retains raw recovery inputs omitted by public observations;
// the WASI storage recovery image encodes exactly this state.
func (fs *FS) ConformanceState() ConformanceState {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	state := ConformanceState{NextInode: fs.nextInode, Generation: fs.generation}
	for _, id := range fs.volumeIDsLocked() {
		volume := fs.volumes[id]
		result := ConformanceVolumeState{
			ID: volume.id, Mount: volume.mount, CapacityBytes: volume.capacityBytes,
			Persisted: conformanceDiskState(volume.persisted),
			Pending:   conformanceOperations(volume.pending), History: conformanceOperations(volume.history),
			NextOperation: volume.nextOperation,
		}
		for key, operation := range volume.last {
			result.Last = append(result.Last, ConformanceLast{Key: key, Operation: operation})
		}
		sort.Slice(result.Last, func(left, right int) bool { return result.Last[left].Key < result.Last[right].Key })
		state.Volumes = append(state.Volumes, result)
	}
	return state
}

func conformanceDiskState(source *diskState) ConformanceDiskState {
	var state ConformanceDiskState
	for _, object := range source.objects {
		state.Objects = append(state.Objects, conformanceDiskObject(object))
	}
	sort.Slice(state.Objects, func(left, right int) bool { return state.Objects[left].Inode < state.Objects[right].Inode })
	for path, inode := range source.paths {
		state.Paths = append(state.Paths, ConformancePath{Path: path, Inode: inode})
	}
	sort.Slice(state.Paths, func(left, right int) bool { return state.Paths[left].Path < state.Paths[right].Path })
	for inode := range source.owned {
		state.Owned = append(state.Owned, inode)
	}
	slices.Sort(state.Owned)
	return state
}

func conformanceDiskObject(source *diskObject) ConformanceDiskObject {
	return ConformanceDiskObject{
		Inode: source.inode, Mode: source.mode, Kind: source.kind, ModTime: source.modTime,
		Data: slices.Clone(source.data),
	}
}

func conformanceOperations(source []*persistenceOperation) []ConformancePersistenceOperation {
	if source == nil {
		return nil
	}
	operations := make([]ConformancePersistenceOperation, len(source))
	for index, operation := range source {
		operations[index] = ConformancePersistenceOperation{
			ID: operation.id, Kind: operation.kind,
			Dependencies: slices.Clone(operation.deps), Stores: slices.Clone(operation.stores),
			Effect: conformanceEffect(operation.effect),
		}
	}
	return operations
}

func conformanceEffect(source persistenceEffect) ConformanceEffect {
	effect := ConformanceEffect{
		Kind: source.kind, Inode: source.inode, Parent: source.parent, Parents: slices.Clone(source.parents),
		Offset: source.offset, Size: source.size, Mode: source.mode, ModTime: source.modTime,
		Data: slices.Clone(source.data),
	}
	if source.object != nil {
		object := conformanceDiskObject(source.object)
		effect.Object = &object
	}
	if source.namespace != nil {
		effect.Namespace = make([]ConformancePath, len(source.namespace))
		for index, part := range source.namespace {
			effect.Namespace[index] = ConformancePath{Path: part.path, Inode: part.inode}
		}
	}
	return effect
}
