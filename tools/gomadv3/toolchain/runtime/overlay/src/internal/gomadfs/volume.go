// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"internal/gomadchoicewire"
)

var ErrVolumeCapacity = errors.New("simulation volume capacity exhausted")

type VolumeConfig struct {
	ID            string
	Path          string
	CapacityBytes uint64
}

type VolumeLimits struct {
	PendingOperations uint64
	Transitions       uint64
}

type VolumeCapacityError struct {
	Resource string
	Required uint64
	Maximum  uint64
}

func (err *VolumeCapacityError) Error() string {
	return ErrVolumeCapacity.Error() + ": resource=" + err.Resource + " required=" + strconv.FormatUint(err.Required, 10) + " maximum=" + strconv.FormatUint(err.Maximum, 10)
}

func (err *VolumeCapacityError) Unwrap() error {
	return ErrVolumeCapacity
}

type CrashCapacity string

const (
	CrashCapacityStates     CrashCapacity = "states"
	CrashCapacityOperations CrashCapacity = "operations"
	CrashCapacityDepth      CrashCapacity = "depth"
	CrashCapacityBytes      CrashCapacity = "bytes"
	CrashCapacityWall       CrashCapacity = "wall"
)

type CrashEnumerationLimits struct {
	States     uint64
	Operations uint64
	Depth      uint64
	Bytes      uint64
	WallNanos  uint64
}

type CrashExploration struct {
	Volume        string
	PendingSHA256 string
	Cursor        []byte
	Seen          []string
	Identity      string
}

type CrashEntry struct {
	Path    string
	Mode    uint32
	Kind    Kind
	ModTime int64
	Data    []byte
}

type CrashState struct {
	Volume             string
	PendingSHA256      string
	SelectedOperations []uint64
	Entries            []CrashEntry
	Identity           string
}

type CrashEnumeration struct {
	States            []CrashState
	ChoiceExploration *CrashExploration
	Complete          bool
	Capacity          CrashCapacity
}

type Operation struct {
	ID            uint64
	Kind          string
	Dependencies  []uint64
	Inode         uint64
	Offset        uint64
	Bytes         uint64
	PayloadSHA256 string
	EffectSHA256  string
	Path          string
	Destination   string
}

type SnapshotEntry struct {
	Path       string
	Mode       uint32
	Kind       string
	ModTime    int64
	Size       uint64
	DataSHA256 string
}

type VolumeSnapshot struct {
	ID                string
	Mount             string
	CapacityBytes     uint64
	Persisted         []SnapshotEntry
	Volatile          []SnapshotEntry
	PendingOperations uint64
	PendingSHA256     string
	NextOperation     uint64
	Identity          string
}

type operationObserver interface {
	BeforeVolumeOperations(string, []Operation) error
	BeforeVolumeControl(string, string, []uint64) error
}

type diskObject struct {
	inode   uint64
	mode    uint32
	kind    Kind
	modTime int64
	data    []byte
}

type diskState struct {
	objects map[uint64]*diskObject
	paths   map[string]uint64
	owned   map[uint64]struct{}
}

type persistenceEffect struct {
	kind      string
	inode     uint64
	parent    uint64
	parents   []uint64
	offset    uint64
	size      uint64
	mode      uint32
	modTime   int64
	data      []byte
	object    *diskObject
	namespace []namespaceEffect
}

type namespaceEffect struct {
	path  string
	inode uint64
}

type persistenceOperation struct {
	id     uint64
	kind   string
	deps   []uint64
	stores []string
	effect persistenceEffect
}

type operationPlan struct {
	kind    string
	effect  persistenceEffect
	stores  []string
	depends []string
}

type allocationLink struct {
	path   string
	node   *node
	parent *node
}

type volumeState struct {
	id            string
	mount         string
	capacityBytes uint64
	persisted     *diskState
	pending       []*persistenceOperation
	history       []*persistenceOperation
	nextOperation uint64
	last          map[string]uint64
	observer      operationObserver
}

func (fs *FS) ConfigureVolumes(configs []VolumeConfig, limits VolumeLimits) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.volumes != nil || len(fs.nodes) != 1 || fs.nodes["/"] == nil {
		return errors.New("simulation volumes must be configured on a new filesystem")
	}
	if limits.PendingOperations == 0 || limits.Transitions == 0 {
		return errors.New("simulation volume limits must be nonzero")
	}
	var previousPath string
	seenIDs := make(map[string]struct{}, len(configs))
	seenPaths := make(map[string]struct{}, len(configs))
	for _, config := range configs {
		path, _, err := Normalize(config.Path)
		if err != nil || path == "/" || path != config.Path {
			return errors.New("simulation volume has an invalid mount path")
		}
		if config.ID == "" || strings.IndexByte(config.ID, 0) >= 0 {
			return errors.New("simulation volume ID is invalid")
		}
		if _, exists := seenIDs[config.ID]; exists {
			return errors.New("simulation volume ID is duplicated")
		}
		if config.Path <= previousPath {
			return errors.New("simulation volume mount paths must be strictly sorted")
		}
		if config.CapacityBytes == 0 {
			return errors.New("simulation volume has zero capacity")
		}
		for existing := range seenPaths {
			if strings.HasPrefix(path+"/", existing+"/") || strings.HasPrefix(existing+"/", path+"/") {
				return errors.New("simulation volume mount paths must not overlap")
			}
		}
		seenPaths[path] = struct{}{}
		seenIDs[config.ID] = struct{}{}
		previousPath = config.Path
	}
	fs.volumes = make(map[string]*volumeState, len(configs))
	fs.volumeLimits = limits
	if fs.nextInode == 0 {
		fs.nextInode = 2
	}
	for _, config := range configs {
		if err := fs.addVolumeMountLocked(config); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FS) SetVolumeObserver(observer operationObserver) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	for _, volume := range fs.volumes {
		volume.observer = observer
	}
}

func (fs *FS) addVolumeMountLocked(config VolumeConfig) error {
	current := ""
	components := strings.Split(strings.TrimPrefix(config.Path, "/"), "/")
	for index, component := range components {
		current += "/" + component
		if existing := fs.nodes[current]; existing != nil {
			if existing.kind != KindDirectory {
				return syscall.ENOTDIR
			}
			if index == len(components)-1 {
				return syscall.EEXIST
			}
			continue
		}
		n := &node{inode: fs.allocateInodeLocked(), mode: 0o755, kind: KindDirectory, linked: true, modTime: fs.nowLocked()}
		if index == len(components)-1 {
			n.volume = config.ID
			n.mountRoot = true
		}
		fs.nodes[current] = n
		fs.liveNodes++
	}
	root := fs.nodes[config.Path]
	persisted := &diskState{
		objects: map[uint64]*diskObject{root.inode: diskObjectForNode(root)},
		paths:   map[string]uint64{"/": root.inode},
		owned:   map[uint64]struct{}{root.inode: {}},
	}
	fs.volumes[config.ID] = &volumeState{
		id: config.ID, mount: config.Path, capacityBytes: config.CapacityBytes,
		persisted: persisted, nextOperation: 1, last: make(map[string]uint64),
	}
	return nil
}

func (fs *FS) EnumerateCrashStates(volumeID string, limits CrashEnumerationLimits, exploration *CrashExploration) (CrashEnumeration, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	volume := fs.volumes[volumeID]
	if volume == nil {
		return CrashEnumeration{}, syscall.ENOENT
	}
	if limits.States == 0 || limits.Operations == 0 || limits.Depth == 0 || limits.Bytes == 0 || limits.WallNanos == 0 {
		return CrashEnumeration{}, errors.New("crash enumeration limits must be nonzero")
	}
	started := runtimeWallNanotime()
	pendingSHA256 := volume.pendingIdentity()
	cursor := make([]byte, len(volume.pending))
	seen := make(map[string]struct{})
	seenOrder := []string{}
	if exploration != nil {
		if err := validateCrashExploration(*exploration, volumeID, pendingSHA256, len(volume.pending)); err != nil {
			return CrashEnumeration{}, err
		}
		copy(cursor, exploration.Cursor)
		for _, identity := range exploration.Seen {
			seen[identity] = struct{}{}
			seenOrder = append(seenOrder, identity)
		}
	}
	if uint64(len(volume.pending)) > limits.Operations {
		return volume.capacityPage(pendingSHA256, cursor, seenOrder, CrashCapacityOperations), nil
	}
	page := CrashEnumeration{}
	var retainedBytes uint64
	for {
		now := runtimeWallNanotime()
		if now >= started && uint64(now-started) >= limits.WallNanos {
			return volume.capacityPage(pendingSHA256, cursor, seenOrder, CrashCapacityWall), nil
		}
		selected, depth := selectedOperations(volume.pending, cursor)
		next, complete := incrementCrashCursor(cursor)
		if depth > limits.Depth {
			return volume.capacityPage(pendingSHA256, cursor, seenOrder, CrashCapacityDepth), nil
		}
		if volume.selectionValid(selected) {
			state, err := volume.crashState(selected, pendingSHA256)
			if err != nil {
				return CrashEnumeration{}, err
			}
			if _, duplicate := seen[state.Identity]; !duplicate {
				stateBytes := crashStateBytes(state)
				if stateBytes > limits.Bytes-retainedBytes {
					return volume.capacityPage(pendingSHA256, cursor, seenOrder, CrashCapacityBytes), nil
				}
				page.States = append(page.States, state)
				retainedBytes += stateBytes
				seen[state.Identity] = struct{}{}
				seenOrder = append(seenOrder, state.Identity)
			}
		}
		cursor = next
		if complete {
			page.Complete = true
			return page, nil
		}
		if uint64(len(page.States)) == limits.States {
			page.Capacity = CrashCapacityStates
			page.ChoiceExploration = newCrashExploration(volumeID, pendingSHA256, cursor, seenOrder)
			return page, nil
		}
	}
}

func (volume *volumeState) capacityPage(pendingSHA256 string, cursor []byte, seen []string, capacity CrashCapacity) CrashEnumeration {
	return CrashEnumeration{
		ChoiceExploration: newCrashExploration(volume.id, pendingSHA256, cursor, seen),
		Capacity:          capacity,
	}
}

func newCrashExploration(volume, pendingSHA256 string, cursor []byte, seen []string) *CrashExploration {
	exploration := &CrashExploration{
		Volume: volume, PendingSHA256: pendingSHA256,
		Cursor: append([]byte(nil), cursor...), Seen: append([]string(nil), seen...),
	}
	exploration.Identity = crashExplorationIdentity(*exploration)
	return exploration
}

func validateCrashExploration(exploration CrashExploration, volume, pendingSHA256 string, operations int) error {
	if exploration.Volume != volume || exploration.PendingSHA256 != pendingSHA256 || len(exploration.Cursor) != operations || exploration.Identity != crashExplorationIdentity(exploration) {
		return errors.New("crash enumeration exploration does not match pending volume state")
	}
	seen := make(map[string]struct{}, len(exploration.Seen))
	for _, value := range exploration.Cursor {
		if value > 1 {
			return errors.New("crash enumeration exploration cursor is invalid")
		}
	}
	for _, identity := range exploration.Seen {
		if !validDigest(identity) {
			return errors.New("crash enumeration exploration contains an invalid state identity")
		}
		if _, ok := seen[identity]; ok {
			return errors.New("crash enumeration exploration contains duplicate state identities")
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func crashExplorationIdentity(exploration CrashExploration) string {
	exploration.Identity = ""
	hasher := gomadchoicewire.NewHasher()
	writeHashString(hasher, "gomadv3-volume-crash-exploration/v1")
	writeHashString(hasher, exploration.Volume)
	writeHashString(hasher, exploration.PendingSHA256)
	writeHashBytes(hasher, exploration.Cursor)
	for _, identity := range exploration.Seen {
		writeHashString(hasher, identity)
	}
	return hashDigest(hasher)
}

func selectedOperations(pending []*persistenceOperation, cursor []byte) (map[uint64]struct{}, uint64) {
	selected := make(map[uint64]struct{})
	var depth uint64
	for index, value := range cursor {
		if value == 0 {
			continue
		}
		selected[pending[index].id] = struct{}{}
		depth++
	}
	return selected, depth
}

func incrementCrashCursor(cursor []byte) ([]byte, bool) {
	next := append([]byte(nil), cursor...)
	for index := len(next) - 1; index >= 0; index-- {
		if next[index] == 0 {
			next[index] = 1
			return next, false
		}
		next[index] = 0
	}
	return next, true
}

func (volume *volumeState) selectionValid(selected map[uint64]struct{}) bool {
	pending := make(map[uint64]struct{}, len(volume.pending))
	for _, operation := range volume.pending {
		pending[operation.id] = struct{}{}
	}
	for _, operation := range volume.pending {
		if _, ok := selected[operation.id]; !ok {
			continue
		}
		for _, dependency := range operation.deps {
			if _, stillPending := pending[dependency]; !stillPending {
				continue
			}
			if _, ok := selected[dependency]; !ok {
				return false
			}
		}
	}
	return true
}

func (volume *volumeState) crashState(selected map[uint64]struct{}, pendingSHA256 string) (CrashState, error) {
	state := cloneDiskState(volume.persisted)
	selectedIDs := make([]uint64, 0, len(selected))
	for _, operation := range volume.pending {
		if _, ok := selected[operation.id]; !ok {
			continue
		}
		if err := applyPersistenceEffect(state, operation.effect); err != nil {
			return CrashState{}, err
		}
		selectedIDs = append(selectedIDs, operation.id)
	}
	entries := diskEntries(state)
	result := CrashState{Volume: volume.id, PendingSHA256: pendingSHA256, SelectedOperations: selectedIDs, Entries: entries}
	result.Identity = crashStateIdentity(result)
	return result, nil
}

func (fs *FS) CrashVolumes(selections map[string][]uint64) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.crashVolumesLocked(selections)
}

func (fs *FS) crashVolumesLocked(selections map[string][]uint64) error {
	if fs.volumes == nil {
		return errors.New("simulation volumes are not configured")
	}
	ids := fs.volumeIDsLocked()
	next := make(map[string]*diskState, len(fs.volumes))
	for _, id := range ids {
		volume := fs.volumes[id]
		selected := make(map[uint64]struct{})
		for _, operation := range selections[id] {
			if _, duplicate := selected[operation]; duplicate {
				return errors.New("crash selection contains a duplicate operation")
			}
			selected[operation] = struct{}{}
		}
		if !volume.selectionValid(selected) {
			return errors.New("crash selection is not dependency-closed")
		}
		for operation := range selected {
			if !volume.hasPending(operation) {
				return errors.New("crash selection contains an unknown operation")
			}
		}
		candidate := cloneDiskState(volume.persisted)
		for _, operation := range volume.pending {
			if _, ok := selected[operation.id]; ok {
				if err := applyPersistenceEffect(candidate, operation.effect); err != nil {
					return err
				}
			}
		}
		next[id] = candidate
	}
	for id := range selections {
		if fs.volumes[id] == nil {
			return errors.New("crash selection refers to an unknown volume")
		}
	}
	for _, id := range ids {
		volume := fs.volumes[id]
		if volume.observer != nil {
			if err := volume.observer.BeforeVolumeControl(id, "crash", selections[id]); err != nil {
				return err
			}
		}
	}
	for _, id := range ids {
		volume := fs.volumes[id]
		volume.persisted = next[id]
		volume.pending = nil
		volume.last = make(map[string]uint64)
	}
	fs.rebuildAfterLifecycleLocked()
	return nil
}

func (fs *FS) FlushVolumes() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.flushVolumesLocked()
}

func (fs *FS) flushVolumesLocked() error {
	for _, id := range fs.volumeIDsLocked() {
		volume := fs.volumes[id]
		selected := make(map[uint64]struct{}, len(volume.pending))
		selectedIDs := make([]uint64, 0, len(volume.pending))
		for _, operation := range volume.pending {
			selected[operation.id] = struct{}{}
			selectedIDs = append(selectedIDs, operation.id)
		}
		if volume.observer != nil {
			if err := volume.observer.BeforeVolumeControl(volume.id, "flush", selectedIDs); err != nil {
				return err
			}
		}
		if err := volume.persistSelected(selected); err != nil {
			return err
		}
	}
	fs.rebuildAfterLifecycleLocked()
	return nil
}

func (fs *FS) AdvanceVolumeLifecycle(graceful bool, selections map[string][]uint64) (*FS, error) {
	return fs.advanceVolumeLifecycle(graceful, selections, false)
}

func (fs *FS) AdvanceVolumeLifecycleAfterDivergence(graceful bool, selections map[string][]uint64) (*FS, error) {
	return fs.advanceVolumeLifecycle(graceful, selections, true)
}

func (fs *FS) advanceVolumeLifecycle(graceful bool, selections map[string][]uint64, ignoreObserver bool) (*FS, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if ignoreObserver {
		for _, volume := range fs.volumes {
			volume.observer = nil
		}
	}
	var err error
	if graceful {
		err = fs.flushVolumesLocked()
	} else {
		err = fs.crashVolumesLocked(selections)
	}
	if err != nil {
		return nil, err
	}
	next := fs.forkVolumesLocked()
	fs.unavailable = syscall.ESTALE
	for handle := range fs.handles {
		handle.revoked = true
	}
	return next, nil
}

func (fs *FS) forkVolumesLocked() *FS {
	next := New()
	next.loader = fs.loader
	next.clock = fs.clock
	next.nextInode = fs.nextInode
	next.volumeLimits = fs.volumeLimits
	next.volumes = make(map[string]*volumeState, len(fs.volumes))
	for _, id := range fs.volumeIDsLocked() {
		volume := fs.volumes[id]
		next.volumes[id] = &volumeState{
			id: volume.id, mount: volume.mount, capacityBytes: volume.capacityBytes,
			persisted: cloneDiskState(volume.persisted), history: append([]*persistenceOperation(nil), volume.history...),
			nextOperation: volume.nextOperation, last: make(map[string]uint64),
		}
	}
	next.rebuildAfterLifecycleLocked()
	return next
}

func (fs *FS) volumeIDsLocked() []string {
	ids := make([]string, 0, len(fs.volumes))
	for id := range fs.volumes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (fs *FS) SelectCrashOperations(seed uint64) map[string][]uint64 {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	result := make(map[string][]uint64, len(fs.volumes))
	ids := make([]string, 0, len(fs.volumes))
	for id := range fs.volumes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	state := seed
	for _, id := range ids {
		volume := fs.volumes[id]
		pending := make(map[uint64]struct{}, len(volume.pending))
		selected := make(map[uint64]struct{})
		for _, operation := range volume.pending {
			pending[operation.id] = struct{}{}
		}
		for _, operation := range volume.pending {
			eligible := true
			for _, dependency := range operation.deps {
				if _, ok := pending[dependency]; !ok {
					continue
				}
				if _, ok := selected[dependency]; !ok {
					eligible = false
					break
				}
			}
			state ^= state << 13
			state ^= state >> 7
			state ^= state << 17
			if eligible && state&1 != 0 {
				selected[operation.id] = struct{}{}
				result[id] = append(result[id], operation.id)
			}
		}
	}
	return result
}

func (fs *FS) Operations() map[string][]Operation {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	result := make(map[string][]Operation, len(fs.volumes))
	for id, volume := range fs.volumes {
		operations := make([]Operation, 0, len(volume.history))
		for _, operation := range volume.history {
			operations = append(operations, observedOperation(operation))
		}
		result[id] = operations
	}
	return result
}

func (fs *FS) VolumeSnapshots() []VolumeSnapshot {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	ids := make([]string, 0, len(fs.volumes))
	for id := range fs.volumes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]VolumeSnapshot, 0, len(ids))
	for _, id := range ids {
		volume := fs.volumes[id]
		snapshot := VolumeSnapshot{
			ID: id, Mount: volume.mount, CapacityBytes: volume.capacityBytes,
			Persisted: snapshotEntries(volume.persisted), Volatile: fs.volatileSnapshotEntriesLocked(volume),
			PendingOperations: uint64(len(volume.pending)), PendingSHA256: volume.pendingIdentity(), NextOperation: volume.nextOperation,
		}
		snapshot.Identity = volumeSnapshotIdentity(snapshot)
		result = append(result, snapshot)
	}
	return result
}

func (fs *FS) volatileSnapshotEntriesLocked(volume *volumeState) []SnapshotEntry {
	paths := make([]string, 0)
	for path, n := range fs.nodes {
		if n.volume == volume.id {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	entries := make([]SnapshotEntry, 0, len(paths))
	for _, path := range paths {
		n := fs.nodes[path]
		entries = append(entries, snapshotEntry(volume.relative(path), n.mode, n.kind, n.modTime, n.data))
	}
	return entries
}

func (fs *FS) rebuildAfterLifecycleLocked() {
	for handle := range fs.handles {
		handle.revoked = true
	}
	fs.handles = make(map[*Handle]struct{})
	for mapping := range fs.mappings {
		mapping.revoked = true
		clear(mapping.data)
	}
	fs.mappings = make(map[*Mapping]struct{})
	fs.generation++
	fs.openHandles = 0
	fs.cwd = "/"
	fs.nodes = map[string]*node{"/": {inode: 1, mode: 0o755, kind: KindDirectory, linked: true}}
	fs.liveNodes = 1
	fs.usedBytes = 0
	configs := make([]VolumeConfig, 0, len(fs.volumes))
	for _, volume := range fs.volumes {
		configs = append(configs, VolumeConfig{ID: volume.id, Path: volume.mount, CapacityBytes: volume.capacityBytes})
	}
	sort.Slice(configs, func(left, right int) bool { return configs[left].Path < configs[right].Path })
	for _, config := range configs {
		components := strings.Split(strings.TrimPrefix(config.Path, "/"), "/")
		current := ""
		for index, component := range components {
			current += "/" + component
			if fs.nodes[current] != nil {
				continue
			}
			n := &node{inode: fs.allocateInodeLocked(), mode: 0o755, kind: KindDirectory, linked: true}
			if index == len(components)-1 {
				rootInode := fs.volumes[config.ID].persisted.paths["/"]
				object := fs.volumes[config.ID].persisted.objects[rootInode]
				n = nodeForDiskObject(object, config.ID)
				n.mountRoot = true
			}
			fs.nodes[current] = n
			fs.liveNodes++
		}
		volume := fs.volumes[config.ID]
		paths := make([]string, 0, len(volume.persisted.paths))
		for relative := range volume.persisted.paths {
			if relative != "/" {
				paths = append(paths, relative)
			}
		}
		sort.Strings(paths)
		for _, relative := range paths {
			object := volume.persisted.objects[volume.persisted.paths[relative]]
			path := config.Path + relative
			n := nodeForDiskObject(object, config.ID)
			fs.nodes[path] = n
			fs.liveNodes++
			fs.usedBytes += uint64(len(n.data))
		}
	}
}

func (volume *volumeState) hasPending(identity uint64) bool {
	for _, operation := range volume.pending {
		if operation.id == identity {
			return true
		}
	}
	return false
}

func (volume *volumeState) persistSelected(selected map[uint64]struct{}) error {
	for _, operation := range volume.pending {
		if _, ok := selected[operation.id]; !ok {
			continue
		}
		if err := applyPersistenceEffect(volume.persisted, operation.effect); err != nil {
			return err
		}
	}
	retained := volume.pending[:0]
	for _, operation := range volume.pending {
		if _, ok := selected[operation.id]; !ok {
			retained = append(retained, operation)
		}
	}
	volume.pending = retained
	volume.rebuildLast()
	return nil
}

func (volume *volumeState) rebuildLast() {
	volume.last = make(map[string]uint64)
	for _, operation := range volume.pending {
		for _, key := range operation.stores {
			volume.last[key] = operation.id
		}
	}
}

func (fs *FS) syncNodeLocked(n *node) error {
	if n.volume == "" {
		return nil
	}
	volume := fs.volumes[n.volume]
	selected := make(map[uint64]struct{})
	for _, operation := range volume.pending {
		matches := operation.effect.inode == n.inode
		if n.kind == KindDirectory {
			matches = slicesContainsUint64(operation.effect.parents, n.inode) || operation.effect.parent == n.inode || operation.effect.inode == n.inode && operation.kind == "metadata"
		} else if operation.kind == "namespace" {
			matches = false
		}
		if matches {
			selected[operation.id] = struct{}{}
		}
	}
	volume.addDependencyClosure(selected)
	selectedIDs := make([]uint64, 0, len(selected))
	for _, operation := range volume.pending {
		if _, ok := selected[operation.id]; ok {
			selectedIDs = append(selectedIDs, operation.id)
		}
	}
	if volume.observer != nil {
		kind := "file_sync"
		if n.kind == KindDirectory {
			kind = "directory_sync"
		}
		if err := volume.observer.BeforeVolumeControl(volume.id, kind, selectedIDs); err != nil {
			return err
		}
	}
	return volume.persistSelected(selected)
}

func (volume *volumeState) addDependencyClosure(selected map[uint64]struct{}) {
	byID := make(map[uint64]*persistenceOperation, len(volume.pending))
	for _, operation := range volume.pending {
		byID[operation.id] = operation
	}
	queue := make([]uint64, 0, len(selected))
	for identity := range selected {
		queue = append(queue, identity)
	}
	for len(queue) != 0 {
		identity := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		operation := byID[identity]
		if operation == nil {
			continue
		}
		for _, dependency := range operation.deps {
			if byID[dependency] == nil {
				continue
			}
			if _, ok := selected[dependency]; ok {
				continue
			}
			selected[dependency] = struct{}{}
			queue = append(queue, dependency)
		}
	}
}

func (fs *FS) preflightVolumeOperationsLocked(volumeID string, count uint64) error {
	if volumeID == "" || count == 0 {
		return nil
	}
	volume := fs.volumes[volumeID]
	required := uint64(len(volume.pending)) + count
	if required > fs.volumeLimits.PendingOperations {
		return &VolumeCapacityError{Resource: "pending_operations", Required: required, Maximum: fs.volumeLimits.PendingOperations}
	}
	return nil
}

func (fs *FS) preflightVolumeGrowthLocked(n *node, finalSize int64) error {
	if n.volume == "" || finalSize <= int64(len(n.data)) {
		return nil
	}
	volume := fs.volumes[n.volume]
	var used uint64
	seen := make(map[uint64]struct{})
	for _, candidate := range fs.nodes {
		if candidate.volume != n.volume {
			continue
		}
		if _, ok := seen[candidate.inode]; ok {
			continue
		}
		seen[candidate.inode] = struct{}{}
		used += uint64(len(candidate.data))
	}
	required := used + uint64(finalSize-int64(len(n.data)))
	if required > volume.capacityBytes {
		return &VolumeCapacityError{Resource: "volume_bytes", Required: required, Maximum: volume.capacityBytes}
	}
	return nil
}

func (fs *FS) recordAllocationAndLinkLocked(path string, n, parent *node) error {
	if n.volume == "" {
		return nil
	}
	volume := fs.volumes[n.volume]
	allocation := persistenceEffect{kind: "alloc", inode: n.inode, object: diskObjectForNode(n)}
	relative := volume.relative(path)
	link := persistenceEffect{kind: "namespace", inode: n.inode, parent: parent.inode, namespace: []namespaceEffect{{path: relative, inode: n.inode}}}
	return volume.addOperations([]operationPlan{
		{kind: "alloc", effect: allocation, stores: []string{allocKey(n.inode)}},
		{kind: "namespace", effect: link, stores: []string{namespaceKey(relative)}, depends: []string{allocKey(n.inode), allocKey(parent.inode), namespaceKey(relative)}},
	})
}

func (fs *FS) recordAllocationLinksLocked(links []allocationLink) error {
	byVolume := make(map[string][]operationPlan)
	var order []string
	for _, link := range links {
		if link.node.volume == "" {
			continue
		}
		volume := fs.volumes[link.node.volume]
		if byVolume[link.node.volume] == nil {
			order = append(order, link.node.volume)
		}
		relative := volume.relative(link.path)
		allocation := persistenceEffect{kind: "alloc", inode: link.node.inode, object: diskObjectForNode(link.node)}
		namespace := persistenceEffect{kind: "namespace", inode: link.node.inode, parent: link.parent.inode, namespace: []namespaceEffect{{path: relative, inode: link.node.inode}}}
		byVolume[link.node.volume] = append(byVolume[link.node.volume],
			operationPlan{kind: "alloc", effect: allocation, stores: []string{allocKey(link.node.inode)}},
			operationPlan{kind: "namespace", effect: namespace, stores: []string{namespaceKey(relative)}, depends: []string{allocKey(link.node.inode), allocKey(link.parent.inode), namespaceKey(relative)}},
		)
	}
	for _, volumeID := range order {
		if err := fs.volumes[volumeID].addOperations(byVolume[volumeID]); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FS) recordResizeLocked(n *node, size uint64, modTime int64) error {
	if n.volume == "" {
		return nil
	}
	volume := fs.volumes[n.volume]
	effect := persistenceEffect{kind: "resize", inode: n.inode, size: size, modTime: modTime}
	return volume.addOperations([]operationPlan{{kind: "resize", effect: effect, stores: []string{resizeKey(n.inode)}, depends: []string{allocKey(n.inode), resizeKey(n.inode)}}})
}

func (fs *FS) recordWriteLocked(n *node, offset uint64, data []byte, finalSize uint64, modTime int64) error {
	if n.volume == "" {
		return nil
	}
	volume := fs.volumes[n.volume]
	plans := make([]operationPlan, 0, 2)
	if finalSize > uint64(len(n.data)) {
		resize := persistenceEffect{kind: "resize", inode: n.inode, size: finalSize, modTime: modTime}
		plans = append(plans, operationPlan{kind: "resize", effect: resize, stores: []string{resizeKey(n.inode)}, depends: []string{allocKey(n.inode), resizeKey(n.inode)}})
	}
	effect := persistenceEffect{kind: "write", inode: n.inode, offset: offset, data: append([]byte(nil), data...), modTime: modTime}
	plans = append(plans, operationPlan{kind: "write", effect: effect, stores: []string{writeKey(n.inode)}, depends: []string{allocKey(n.inode), resizeKey(n.inode)}})
	return volume.addOperations(plans)
}

func (fs *FS) recordMetadataLocked(n *node, mode uint32, modTime int64) error {
	if n.volume == "" {
		return nil
	}
	volume := fs.volumes[n.volume]
	effect := persistenceEffect{kind: "metadata", inode: n.inode, mode: mode, modTime: modTime}
	return volume.addOperations([]operationPlan{{kind: "metadata", effect: effect, stores: []string{metadataKey(n.inode)}, depends: []string{allocKey(n.inode), metadataKey(n.inode)}}})
}

func (fs *FS) recordNamespaceLocked(volumeID string, parents []uint64, effects []namespaceEffect, inode uint64, modTime int64) error {
	if volumeID == "" {
		return nil
	}
	volume := fs.volumes[volumeID]
	stores := make([]string, 0, len(effects))
	depends := make([]string, 0, len(parents)+len(effects)+1)
	for _, parent := range parents {
		depends = append(depends, allocKey(parent))
	}
	if inode != 0 {
		depends = append(depends, allocKey(inode))
	}
	for _, effect := range effects {
		key := namespaceKey(effect.path)
		stores = append(stores, key)
		depends = append(depends, key)
	}
	return volume.addOperations([]operationPlan{{
		kind: "namespace", effect: persistenceEffect{kind: "namespace", inode: inode, parents: append([]uint64(nil), parents...), modTime: modTime, namespace: append([]namespaceEffect(nil), effects...)},
		stores: stores, depends: depends,
	}})
}

func (volume *volumeState) addOperations(plans []operationPlan) error {
	operations := make([]*persistenceOperation, 0, len(plans))
	nextOperation := volume.nextOperation
	last := make(map[string]uint64, len(volume.last)+len(plans))
	for key, identity := range volume.last {
		last[key] = identity
	}
	for _, plan := range plans {
		dependencySet := make(map[uint64]struct{})
		for _, key := range plan.depends {
			if dependency := last[key]; dependency != 0 {
				dependencySet[dependency] = struct{}{}
			}
		}
		dependencies := make([]uint64, 0, len(dependencySet))
		for dependency := range dependencySet {
			dependencies = append(dependencies, dependency)
		}
		sort.Slice(dependencies, func(left, right int) bool { return dependencies[left] < dependencies[right] })
		operation := &persistenceOperation{id: nextOperation, kind: plan.kind, deps: dependencies, stores: append([]string(nil), plan.stores...), effect: plan.effect}
		operations = append(operations, operation)
		for _, key := range plan.stores {
			last[key] = nextOperation
		}
		nextOperation++
	}
	if volume.observer != nil {
		observed := make([]Operation, 0, len(operations))
		for _, operation := range operations {
			observed = append(observed, observedOperation(operation))
		}
		if err := volume.observer.BeforeVolumeOperations(volume.id, observed); err != nil {
			return err
		}
	}
	volume.nextOperation = nextOperation
	volume.last = last
	volume.pending = append(volume.pending, operations...)
	volume.history = append(volume.history, operations...)
	return nil
}

func observedOperation(operation *persistenceOperation) Operation {
	result := Operation{
		ID: operation.id, Kind: operation.kind, Dependencies: append([]uint64(nil), operation.deps...),
		Inode: operation.effect.inode, Offset: operation.effect.offset, Bytes: operation.effect.size,
		EffectSHA256: persistenceEffectIdentity(operation.effect),
	}
	if len(operation.effect.data) != 0 {
		result.Bytes = uint64(len(operation.effect.data))
		result.PayloadSHA256 = hashBytes(operation.effect.data)
	}
	if len(operation.effect.namespace) != 0 {
		result.Path = operation.effect.namespace[0].path
		if len(operation.effect.namespace) > 1 {
			result.Destination = operation.effect.namespace[len(operation.effect.namespace)-1].path
		}
	}
	return result
}

func snapshotEntries(state *diskState) []SnapshotEntry {
	paths := make([]string, 0, len(state.paths))
	for path := range state.paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	entries := make([]SnapshotEntry, 0, len(paths))
	for _, path := range paths {
		object := state.objects[state.paths[path]]
		entries = append(entries, snapshotEntry(path, object.mode, object.kind, object.modTime, object.data))
	}
	return entries
}

func snapshotEntry(path string, mode uint32, kind Kind, modTime int64, data []byte) SnapshotEntry {
	name := "file"
	if kind == KindDirectory {
		name = "directory"
	}
	entry := SnapshotEntry{Path: path, Mode: mode, Kind: name, ModTime: modTime, Size: uint64(len(data))}
	if kind == KindFile {
		entry.DataSHA256 = hashBytes(data)
	}
	return entry
}

func volumeSnapshotIdentity(snapshot VolumeSnapshot) string {
	hasher := gomadchoicewire.NewHasher()
	writeHashString(hasher, "gomadv3-volume-snapshot/v1")
	writeHashString(hasher, snapshot.ID)
	writeHashString(hasher, snapshot.Mount)
	writeHashUint64(hasher, snapshot.CapacityBytes)
	writeSnapshotEntries(hasher, snapshot.Persisted)
	writeSnapshotEntries(hasher, snapshot.Volatile)
	writeHashUint64(hasher, snapshot.PendingOperations)
	writeHashString(hasher, snapshot.PendingSHA256)
	writeHashUint64(hasher, snapshot.NextOperation)
	return hashDigest(hasher)
}

func writeSnapshotEntries(hasher *gomadchoicewire.Hasher, entries []SnapshotEntry) {
	writeHashUint64(hasher, uint64(len(entries)))
	for _, entry := range entries {
		writeHashString(hasher, entry.Path)
		writeHashUint64(hasher, uint64(entry.Mode))
		writeHashString(hasher, entry.Kind)
		writeHashUint64(hasher, uint64(entry.ModTime))
		writeHashUint64(hasher, entry.Size)
		writeHashString(hasher, entry.DataSHA256)
	}
}

func persistenceEffectIdentity(effect persistenceEffect) string {
	hasher := gomadchoicewire.NewHasher()
	writeHashString(hasher, "gomadv3-volume-effect/v1")
	writeEffectHash(hasher, effect)
	return hashDigest(hasher)
}

func hashBytes(value []byte) string {
	hasher := gomadchoicewire.NewHasher()
	hasher.Write(value)
	return hashDigest(hasher)
}

func (volume *volumeState) relative(path string) string {
	if path == volume.mount {
		return "/"
	}
	return strings.TrimPrefix(path, volume.mount)
}

func (volume *volumeState) pendingIdentity() string {
	hasher := gomadchoicewire.NewHasher()
	writeHashString(hasher, "gomadv3-volume-pending/v1")
	writeHashString(hasher, volume.id)
	for _, operation := range volume.pending {
		writeHashUint64(hasher, operation.id)
		writeHashString(hasher, operation.kind)
		for _, dependency := range operation.deps {
			writeHashUint64(hasher, dependency)
		}
		writeEffectHash(hasher, operation.effect)
	}
	return hashDigest(hasher)
}

func applyPersistenceEffect(state *diskState, effect persistenceEffect) error {
	switch effect.kind {
	case "alloc":
		if effect.object == nil || state.objects[effect.inode] != nil {
			return errors.New("invalid durable allocation effect")
		}
		state.objects[effect.inode] = cloneDiskObject(effect.object)
		state.owned[effect.inode] = struct{}{}
	case "resize":
		object := state.mutableObject(effect.inode)
		if object == nil || object.kind != KindFile || effect.size > MaximumFileBytes {
			return errors.New("invalid durable resize effect")
		}
		if effect.size <= uint64(len(object.data)) {
			object.data = object.data[:effect.size]
		} else {
			object.data = append(object.data, make([]byte, int(effect.size)-len(object.data))...)
		}
		object.modTime = effect.modTime
	case "write":
		object := state.mutableObject(effect.inode)
		end := effect.offset + uint64(len(effect.data))
		if object == nil || object.kind != KindFile || end < effect.offset || end > uint64(len(object.data)) {
			return errors.New("invalid durable write effect")
		}
		copy(object.data[effect.offset:end], effect.data)
		object.modTime = effect.modTime
	case "metadata":
		object := state.mutableObject(effect.inode)
		if object == nil {
			return errors.New("invalid durable metadata effect")
		}
		object.mode = effect.mode
		object.modTime = effect.modTime
	case "namespace":
		for _, part := range effect.namespace {
			if part.path == "/" {
				return errors.New("invalid durable root namespace effect")
			}
			if part.inode == 0 {
				delete(state.paths, part.path)
				continue
			}
			if state.objects[part.inode] == nil {
				return errors.New("durable namespace effect refers to an unavailable object")
			}
			state.paths[part.path] = part.inode
		}
		if effect.inode != 0 && state.objects[effect.inode] != nil {
			state.mutableObject(effect.inode).modTime = effect.modTime
		}
	default:
		return errors.New("unknown durable persistence effect")
	}
	return nil
}

func cloneDiskState(source *diskState) *diskState {
	result := &diskState{objects: make(map[uint64]*diskObject, len(source.objects)), paths: make(map[string]uint64, len(source.paths)), owned: make(map[uint64]struct{})}
	for inode, object := range source.objects {
		result.objects[inode] = object
	}
	for path, inode := range source.paths {
		result.paths[path] = inode
	}
	return result
}

func (state *diskState) mutableObject(inode uint64) *diskObject {
	object := state.objects[inode]
	if object == nil {
		return nil
	}
	if _, ok := state.owned[inode]; ok {
		return object
	}
	object = cloneDiskObject(object)
	state.objects[inode] = object
	state.owned[inode] = struct{}{}
	return object
}

func cloneDiskObject(source *diskObject) *diskObject {
	if source == nil {
		return nil
	}
	result := *source
	result.data = append([]byte(nil), source.data...)
	return &result
}

func diskObjectForNode(n *node) *diskObject {
	return &diskObject{inode: n.inode, mode: n.mode, kind: n.kind, modTime: n.modTime, data: append([]byte(nil), n.data...)}
}

func nodeForDiskObject(object *diskObject, volume string) *node {
	return &node{inode: object.inode, mode: object.mode, kind: object.kind, modTime: object.modTime, data: append([]byte(nil), object.data...), volume: volume, linked: true}
}

func diskEntries(state *diskState) []CrashEntry {
	paths := make([]string, 0, len(state.paths))
	for path := range state.paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	entries := make([]CrashEntry, 0, len(paths))
	for _, path := range paths {
		object := state.objects[state.paths[path]]
		entries = append(entries, CrashEntry{Path: path, Mode: object.mode, Kind: object.kind, ModTime: object.modTime, Data: append([]byte(nil), object.data...)})
	}
	return entries
}

func crashStateIdentity(state CrashState) string {
	hasher := gomadchoicewire.NewHasher()
	writeHashString(hasher, "gomadv3-volume-crash-state/v1")
	writeHashString(hasher, state.Volume)
	for _, entry := range state.Entries {
		writeHashString(hasher, entry.Path)
		writeHashUint64(hasher, uint64(entry.Mode))
		writeHashUint64(hasher, uint64(entry.Kind))
		writeHashUint64(hasher, uint64(entry.ModTime))
		writeHashBytes(hasher, entry.Data)
	}
	return hashDigest(hasher)
}

func crashStateBytes(state CrashState) uint64 {
	result := uint64(len(state.Volume) + len(state.PendingSHA256) + len(state.Identity))
	result += uint64(len(state.SelectedOperations)) * 8
	for _, entry := range state.Entries {
		result += uint64(len(entry.Path)+len(entry.Data)) + 32
	}
	return result
}

func allocKey(inode uint64) string    { return "alloc:" + strconv.FormatUint(inode, 10) }
func resizeKey(inode uint64) string   { return "resize:" + strconv.FormatUint(inode, 10) }
func writeKey(inode uint64) string    { return "write:" + strconv.FormatUint(inode, 10) }
func metadataKey(inode uint64) string { return "metadata:" + strconv.FormatUint(inode, 10) }
func namespaceKey(path string) string { return "namespace:" + path }

func writeHashUint64(hasher *gomadchoicewire.Hasher, value uint64) {
	var encoded [8]byte
	for index := range encoded {
		encoded[index] = byte(value >> (index * 8))
	}
	hasher.Write(encoded[:])
}

func writeHashString(hasher *gomadchoicewire.Hasher, value string) {
	writeHashBytes(hasher, []byte(value))
}

func writeHashBytes(hasher *gomadchoicewire.Hasher, value []byte) {
	writeHashUint64(hasher, uint64(len(value)))
	hasher.Write(value)
}

func writeEffectHash(hasher *gomadchoicewire.Hasher, effect persistenceEffect) {
	writeHashString(hasher, effect.kind)
	writeHashUint64(hasher, effect.inode)
	writeHashUint64(hasher, effect.parent)
	for _, parent := range effect.parents {
		writeHashUint64(hasher, parent)
	}
	writeHashUint64(hasher, effect.offset)
	writeHashUint64(hasher, effect.size)
	writeHashUint64(hasher, uint64(effect.mode))
	writeHashUint64(hasher, uint64(effect.modTime))
	writeHashBytes(hasher, effect.data)
	if effect.object != nil {
		writeHashUint64(hasher, effect.object.inode)
		writeHashUint64(hasher, uint64(effect.object.mode))
		writeHashUint64(hasher, uint64(effect.object.kind))
		writeHashUint64(hasher, uint64(effect.object.modTime))
		writeHashBytes(hasher, effect.object.data)
	}
	for _, part := range effect.namespace {
		writeHashString(hasher, part.path)
		writeHashUint64(hasher, part.inode)
	}
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+gomadchoicewire.DigestBytes*2 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, character := range value[len("sha256:"):] {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}

func hashDigest(hasher *gomadchoicewire.Hasher) string {
	sum := hasher.Sum()
	const digits = "0123456789abcdef"
	encoded := make([]byte, len("sha256:")+len(sum)*2)
	copy(encoded, "sha256:")
	for index, value := range sum {
		encoded[len("sha256:")+index*2] = digits[value>>4]
		encoded[len("sha256:")+index*2+1] = digits[value&0x0f]
	}
	return string(encoded)
}

func slicesContainsUint64(values []uint64, wanted uint64) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
