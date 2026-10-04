// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"sort"
	"strings"
	"sync"
	"syscall"

	"internal/gomadwire"
)

var ErrClosed error = closedHandleError{}

type closedHandleError struct{}

func (closedHandleError) Error() string { return syscall.EBADF.Error() }
func (closedHandleError) Unwrap() error { return syscall.EBADF }

type Kind = gomadwire.MountKind

const (
	KindFile      = gomadwire.MountKindFile
	KindDirectory = gomadwire.MountKindDirectory
)

type MountStatus = gomadwire.MountStatus

const (
	MountOK        = gomadwire.MountStatusOK
	MountUnmounted = gomadwire.MountStatusUnmounted
	MountNotExist  = gomadwire.MountStatusNotExist
)

type Child = gomadwire.MountChild
type LoadEntry = gomadwire.MountEntry

type Entry struct {
	Name     string
	Mode     uint32
	Kind     Kind
	ModTime  int64
	Data     []byte
	Children []Child
}

type OpenFlags struct {
	Read, Write, Append, Create, Exclusive, Truncate bool
}

type Loader func(string) (LoadEntry, MountStatus, error)

type node struct {
	inode     uint64
	mode      uint32
	kind      Kind
	data      []byte
	children  []Child
	readonly  bool
	linked    bool
	handles   uint64
	modTime   int64
	volume    string
	mountRoot bool
}

type FS struct {
	mu           sync.Mutex
	process      bool
	nodes        map[string]*node
	loader       Loader
	clock        func() int64
	openHandles  uint64
	usedBytes    uint64
	liveNodes    uint64
	cwd          string
	nextInode    uint64
	generation   uint64
	volumes      map[string]*volumeState
	volumeLimits VolumeLimits
	handles      map[*localHandle]struct{}
	mappings     map[*localMapping]struct{}
	mappedBytes  uint64
	unavailable  error
}

type Statistics struct {
	OpenHandles uint64
	UsedBytes   uint64
	MappedBytes uint64
}

var Default = New()

const (
	MaximumPathBytes        = 4096
	maximumNodes            = 100_000
	maximumHandles          = 100_000
	maximumDirectoryEntries = 100_000
	// A WAL-mode SQLite database of a large functional suite grows past
	// 16 MiB; a write refused with EFBIG surfaces as SQLITE_IOERR_WRITE and
	// fails every later transaction, so the file and total bounds leave room
	// for the largest ./tests suites while still failing closed on a runaway.
	MaximumFileBytes   = 256 << 20
	maximumTotalBytes  = 1 << 30
	maximumMappedBytes = 64 << 20
)

// TempDirectory is the directory os.TempDir resolves to when TMPDIR is unset.
// A Go program assumes it exists, so every fresh in-memory filesystem carries
// it instead of making the first scratch-file open fail closed.
const TempDirectory = "/tmp"

// initialNodeCount is the number of nodes a filesystem holds before any
// operation: the root and the temp directory.
const initialNodeCount = 2

func New() *FS {
	fs := &FS{cwd: "/", nextInode: 2, generation: 1, handles: make(map[*localHandle]struct{}), mappings: make(map[*localMapping]struct{})}
	fs.resetNodesLocked()
	return fs
}

func (fs *FS) resetNodesLocked() {
	fs.nodes = map[string]*node{
		"/":           {inode: 1, mode: 0o755, kind: KindDirectory, linked: true},
		TempDirectory: {inode: fs.allocateInodeLocked(), mode: 0o777, kind: KindDirectory, linked: true, modTime: fs.nowLocked()},
	}
	fs.liveNodes = initialNodeCount
}

func (fs *FS) initialLocked() bool {
	return len(fs.nodes) == initialNodeCount && fs.nodes["/"] != nil && fs.nodes[TempDirectory] != nil && fs.openHandles == 0 && fs.usedBytes == 0
}

func NewSimulation() *FS {
	Default.mu.Lock()
	defer Default.mu.Unlock()
	fs := New()
	fs.clock = Default.clock
	fs.nodes["/"].modTime = Default.nodes["/"].modTime
	return fs
}

func (fs *FS) allocateInodeLocked() uint64 {
	inode := fs.nextInode
	fs.nextInode++
	return inode
}

func (fs *FS) Statistics() Statistics {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return Statistics{OpenHandles: fs.openHandles, UsedBytes: fs.usedBytes, MappedBytes: fs.mappedBytes}
}

func (fs *FS) SetLoader(loader Loader) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.loader = loader
}

func (fs *FS) SetClock(clock func() int64) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.clock = clock
	fs.nodes["/"].modTime = fs.nowLocked()
}

func Normalize(name string) (string, string, error) {
	if name == "" || len(name) > MaximumPathBytes || strings.IndexByte(name, 0) >= 0 {
		return "", "", syscall.EINVAL
	}
	components := make([]string, 0, strings.Count(name, "/")+1)
	for _, component := range strings.Split(name, "/") {
		switch component {
		case "", ".":
		case "..":
			if len(components) == 0 {
				return "", "", syscall.EINVAL
			}
			components = components[:len(components)-1]
		default:
			components = append(components, component)
		}
	}
	if len(components) == 0 {
		return "/", "/", nil
	}
	return "/" + strings.Join(components, "/"), components[len(components)-1], nil
}

func (fs *FS) normalize(name string) (string, string, error) {
	if fs.unavailable != nil {
		return "", "", fs.unavailable
	}
	if strings.HasPrefix(name, "/") {
		return Normalize(name)
	}
	fs.mu.Lock()
	cwd := fs.cwd
	fs.mu.Unlock()
	return Normalize(cwd + "/" + name)
}

func (fs *FS) Resolve(name string) (string, string, error) {
	if fs.process {
		return processResolve(name)
	}
	return fs.normalize(name)
}

func (fs *FS) Mkdir(name string, perm uint32) error {
	if fs.process {
		return processMkdir(name, perm, false)
	}
	path, _, err := fs.normalize(name)
	if err != nil || path == "/" {
		return syscall.EINVAL
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if _, lookupErr := fs.lookupLocked(path); lookupErr == nil {
		return syscall.EEXIST
	} else if lookupErr != syscall.ENOENT {
		return lookupErr
	}
	parent := parentPath(path)
	parentNode, err := fs.lookupLocked(parent)
	if err != nil {
		return err
	}
	if parentNode.kind != KindDirectory {
		return syscall.ENOTDIR
	}
	if parentNode.readonly {
		return syscall.EROFS
	}
	if fs.liveNodes == maximumNodes {
		return syscall.ENOSPC
	}
	if fs.directoryEntriesLocked(parent) == maximumDirectoryEntries {
		return syscall.ENOSPC
	}
	if err := fs.preflightVolumeOperationsLocked(parentNode.volume, 2); err != nil {
		return err
	}
	n := &node{inode: fs.allocateInodeLocked(), mode: perm & 0o777, kind: KindDirectory, linked: true, modTime: fs.nowLocked(), volume: parentNode.volume}
	if err := fs.recordAllocationAndLinkLocked(path, n, parentNode); err != nil {
		return err
	}
	fs.nodes[path] = n
	fs.liveNodes++
	return nil
}

func (fs *FS) MkdirAll(name string, perm uint32) error {
	if fs.process {
		return processMkdir(name, perm, true)
	}
	path, _, err := fs.normalize(name)
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	planned := make(map[string]*node)
	links := make([]allocationLink, 0)
	newEntries := make(map[string]int)
	current := ""
	for _, component := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if component == "" {
			continue
		}
		current += "/" + component
		existing := planned[current]
		lookupErr := error(nil)
		if existing == nil {
			existing, lookupErr = fs.lookupLocked(current)
		}
		if lookupErr == nil {
			if existing.kind != KindDirectory {
				return syscall.ENOTDIR
			}
			continue
		}
		if lookupErr != syscall.ENOENT {
			return lookupErr
		}
		parentName := parentPath(current)
		parent := planned[parentName]
		if parent == nil {
			parent, err = fs.lookupLocked(parentName)
			if err != nil {
				return err
			}
		}
		if parent.readonly {
			return syscall.EROFS
		}
		if fs.liveNodes+uint64(len(links))+1 > maximumNodes {
			return syscall.ENOSPC
		}
		if fs.directoryEntriesLocked(parentName)+newEntries[parentName] == maximumDirectoryEntries {
			return syscall.ENOSPC
		}
		n := &node{inode: fs.allocateInodeLocked(), mode: perm & 0o777, kind: KindDirectory, linked: true, modTime: fs.nowLocked(), volume: parent.volume}
		planned[current] = n
		links = append(links, allocationLink{path: current, node: n, parent: parent})
		newEntries[parentName]++
	}
	operations := make(map[string]uint64)
	for _, link := range links {
		operations[link.node.volume] += 2
	}
	for volume, count := range operations {
		if err := fs.preflightVolumeOperationsLocked(volume, count); err != nil {
			return err
		}
	}
	if err := fs.recordAllocationLinksLocked(links); err != nil {
		return err
	}
	for _, link := range links {
		fs.nodes[link.path] = link.node
		fs.liveNodes++
	}
	return nil
}

func (fs *FS) Stat(name string) (Entry, error) {
	if fs.process {
		return processStat(name)
	}
	path, base, err := fs.normalize(name)
	if err != nil {
		return Entry{}, err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n, err := fs.lookupLocked(path)
	if err != nil {
		return Entry{}, err
	}
	return entryForNode(base, n), nil
}

func (fs *FS) Open(name string, flags OpenFlags, perm uint32) (*Handle, error) {
	if fs.process {
		return processOpen(name, flags, perm)
	}
	path, _, err := fs.normalize(name)
	if err != nil {
		return nil, err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.openHandles == maximumHandles {
		return nil, syscall.EMFILE
	}
	n, lookupErr := fs.lookupLocked(path)
	if lookupErr != nil && lookupErr != syscall.ENOENT {
		return nil, lookupErr
	}
	if n == nil {
		if !flags.Create {
			return nil, syscall.ENOENT
		}
		parent, err := fs.lookupLocked(parentPath(path))
		if err != nil {
			return nil, err
		}
		if parent.kind != KindDirectory {
			return nil, syscall.ENOTDIR
		}
		if parent.readonly {
			return nil, syscall.EROFS
		}
		if fs.liveNodes == maximumNodes {
			return nil, syscall.ENOSPC
		}
		if fs.directoryEntriesLocked(parentPath(path)) == maximumDirectoryEntries {
			return nil, syscall.ENOSPC
		}
		if err := fs.preflightVolumeOperationsLocked(parent.volume, 2); err != nil {
			return nil, err
		}
		n = &node{inode: fs.allocateInodeLocked(), mode: perm & 0o777, kind: KindFile, linked: true, modTime: fs.nowLocked(), volume: parent.volume}
		if err := fs.recordAllocationAndLinkLocked(path, n, parent); err != nil {
			return nil, err
		}
		fs.nodes[path] = n
		fs.liveNodes++
	} else if flags.Create && flags.Exclusive {
		return nil, syscall.EEXIST
	}
	if n.readonly && (flags.Write || flags.Truncate || flags.Create) {
		return nil, syscall.EROFS
	}
	if n.kind == KindDirectory && flags.Write {
		return nil, syscall.EISDIR
	}
	if flags.Truncate && flags.Write && len(n.data) != 0 {
		if err := fs.preflightVolumeOperationsLocked(n.volume, 1); err != nil {
			return nil, err
		}
		modTime := fs.nowLocked()
		if err := fs.recordResizeLocked(n, 0, modTime); err != nil {
			return nil, err
		}
		fs.usedBytes -= uint64(len(n.data))
		fs.truncateMappingsLocked(n, 0, uint64(len(n.data)))
		n.data = nil
		n.modTime = modTime
	}
	fs.openHandles++
	n.handles++
	handle := &localHandle{fs: fs, node: n, name: path, readable: flags.Read, writable: flags.Write, append: flags.Append, generation: fs.generation}
	fs.handles[handle] = struct{}{}
	return &Handle{implementation: handle}, nil
}

func (fs *FS) Rename(oldName, newName string) error {
	if fs.process {
		return processPathOperation(processVolumeCommand{Operation: processVolumeRenameOp, Path: oldName, Destination: newName})
	}
	oldPath, _, err := fs.normalize(oldName)
	if err != nil {
		return err
	}
	newPath, _, err := fs.normalize(newName)
	if err != nil || oldPath == "/" || newPath == "/" {
		return syscall.EINVAL
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n, err := fs.lookupLocked(oldPath)
	if err != nil {
		return err
	}
	parent, err := fs.lookupLocked(parentPath(newPath))
	if err != nil {
		return err
	}
	if n.readonly || parent.readonly {
		return syscall.EXDEV
	}
	if n.mountRoot || n.volume != parent.volume {
		return syscall.EXDEV
	}
	if parent.kind != KindDirectory {
		return syscall.ENOTDIR
	}
	if n.kind == KindDirectory && strings.HasPrefix(newPath, oldPath+"/") {
		return syscall.EINVAL
	}
	existing, lookupErr := fs.lookupLocked(newPath)
	if lookupErr != nil && lookupErr != syscall.ENOENT {
		return lookupErr
	}
	if existing != nil && existing.readonly {
		return syscall.EXDEV
	}
	if existing != nil && (existing.mountRoot || existing.volume != n.volume) {
		return syscall.EXDEV
	}
	if n.kind == KindDirectory {
		for name, descendant := range fs.nodes {
			if strings.HasPrefix(name, oldPath+"/") && descendant.readonly {
				return syscall.EXDEV
			}
		}
	}
	if oldPath == newPath {
		return nil
	}
	if existing != nil && existing.kind == KindDirectory {
		return syscall.EEXIST
	}
	if fs.nodes[newPath] == nil && parentPath(oldPath) != parentPath(newPath) && fs.directoryEntriesLocked(parentPath(newPath)) == maximumDirectoryEntries {
		return syscall.ENOSPC
	}
	if err := fs.preflightVolumeOperationsLocked(n.volume, 1); err != nil {
		return err
	}
	oldParent := fs.nodes[parentPath(oldPath)]
	effects := []namespaceEffect(nil)
	if n.volume != "" {
		effects = append(effects, namespaceEffect{path: fs.volumes[n.volume].relative(oldPath)})
		if n.kind == KindDirectory {
			paths := make([]string, 0)
			for name := range fs.nodes {
				if strings.HasPrefix(name, oldPath+"/") {
					paths = append(paths, name)
				}
			}
			sort.Strings(paths)
			for _, name := range paths {
				effects = append(effects, namespaceEffect{path: fs.volumes[n.volume].relative(name)})
			}
		}
		effects = append(effects, namespaceEffect{path: fs.volumes[n.volume].relative(newPath), inode: n.inode})
		if n.kind == KindDirectory {
			paths := make([]string, 0)
			for name := range fs.nodes {
				if strings.HasPrefix(name, oldPath+"/") {
					paths = append(paths, name)
				}
			}
			sort.Strings(paths)
			for _, name := range paths {
				effects = append(effects, namespaceEffect{path: fs.volumes[n.volume].relative(newPath + strings.TrimPrefix(name, oldPath)), inode: fs.nodes[name].inode})
			}
		}
	}
	modTime := fs.nowLocked()
	parents := []uint64{oldParent.inode}
	if parent.inode != oldParent.inode {
		parents = append(parents, parent.inode)
	}
	if err := fs.recordNamespaceLocked(n.volume, parents, effects, n.inode, modTime); err != nil {
		return err
	}
	if existing != nil {
		existing.linked = false
		fs.releaseNodeLocked(existing)
	}
	delete(fs.nodes, newPath)
	fs.nodes[newPath] = n
	delete(fs.nodes, oldPath)
	n.modTime = modTime
	if n.kind == KindDirectory {
		for name, descendant := range fs.nodes {
			if strings.HasPrefix(name, oldPath+"/") {
				delete(fs.nodes, name)
				fs.nodes[newPath+strings.TrimPrefix(name, oldPath)] = descendant
			}
		}
		if fs.cwd == oldPath || strings.HasPrefix(fs.cwd, oldPath+"/") {
			fs.cwd = newPath + strings.TrimPrefix(fs.cwd, oldPath)
		}
	}
	return nil
}

func (fs *FS) Remove(name string) error {
	if fs.process {
		return processPathOperation(processVolumeCommand{Operation: processVolumeRemoveOp, Path: name})
	}
	path, _, err := fs.normalize(name)
	if err != nil || path == "/" {
		return syscall.EINVAL
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n, err := fs.lookupLocked(path)
	if err != nil {
		return err
	}
	if n.readonly {
		return syscall.EROFS
	}
	if n.mountRoot {
		return syscall.EBUSY
	}
	if fs.cwd == path || strings.HasPrefix(fs.cwd, path+"/") {
		return syscall.EBUSY
	}
	if n.kind == KindDirectory {
		for candidate := range fs.nodes {
			if strings.HasPrefix(candidate, path+"/") {
				return syscall.ENOTEMPTY
			}
		}
	}
	if err := fs.preflightVolumeOperationsLocked(n.volume, 1); err != nil {
		return err
	}
	parent := fs.nodes[parentPath(path)]
	var effectPath string
	if n.volume != "" {
		effectPath = fs.volumes[n.volume].relative(path)
	}
	if err := fs.recordNamespaceLocked(n.volume, []uint64{parent.inode}, []namespaceEffect{{path: effectPath}}, n.inode, n.modTime); err != nil {
		return err
	}
	delete(fs.nodes, path)
	n.linked = false
	fs.releaseNodeLocked(n)
	return nil
}

func (fs *FS) RemoveAll(name string) error {
	if fs.process {
		return processPathOperation(processVolumeCommand{Operation: processVolumeRemoveAllOp, Path: name})
	}
	path, _, err := fs.normalize(name)
	if err != nil || path == "/" {
		return syscall.EINVAL
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n, err := fs.lookupLocked(path)
	if err == syscall.ENOENT {
		return nil
	}
	if err != nil {
		return err
	}
	if fs.cwd == path || strings.HasPrefix(fs.cwd, path+"/") {
		return syscall.EBUSY
	}
	if n.readonly {
		return syscall.EROFS
	}
	if n.mountRoot {
		return syscall.EBUSY
	}
	for candidate, descendant := range fs.nodes {
		if strings.HasPrefix(candidate, path+"/") && descendant.readonly {
			return syscall.EROFS
		}
	}
	if err := fs.preflightVolumeOperationsLocked(n.volume, 1); err != nil {
		return err
	}
	parent := fs.nodes[parentPath(path)]
	effects := make([]namespaceEffect, 0)
	if n.volume != "" {
		paths := make([]string, 0)
		for candidate := range fs.nodes {
			if candidate == path || strings.HasPrefix(candidate, path+"/") {
				paths = append(paths, candidate)
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(paths)))
		for _, candidate := range paths {
			effects = append(effects, namespaceEffect{path: fs.volumes[n.volume].relative(candidate)})
		}
	}
	if err := fs.recordNamespaceLocked(n.volume, []uint64{parent.inode}, effects, n.inode, n.modTime); err != nil {
		return err
	}
	for candidate, descendant := range fs.nodes {
		if candidate == path || strings.HasPrefix(candidate, path+"/") {
			delete(fs.nodes, candidate)
			descendant.linked = false
			fs.releaseNodeLocked(descendant)
		}
	}
	return nil
}

func (fs *FS) Chmod(name string, mode uint32) error {
	if fs.process {
		return processPathOperation(processVolumeCommand{Operation: processVolumeChmodOp, Path: name, Mode: mode})
	}
	path, _, err := fs.normalize(name)
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n, err := fs.lookupLocked(path)
	if err != nil {
		return err
	}
	if n.readonly {
		return syscall.EROFS
	}
	if err := fs.preflightVolumeOperationsLocked(n.volume, 1); err != nil {
		return err
	}
	nextMode := mode & 0o777
	nextModTime := fs.nowLocked()
	if err := fs.recordMetadataLocked(n, nextMode, nextModTime); err != nil {
		return err
	}
	n.mode = nextMode
	n.modTime = nextModTime
	return nil
}

func (fs *FS) Chtimes(name string, modTime int64) error {
	if fs.process {
		return processPathOperation(processVolumeCommand{Operation: processVolumeChtimesOp, Path: name, ModTime: modTime})
	}
	path, _, err := fs.normalize(name)
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n, err := fs.lookupLocked(path)
	if err != nil {
		return err
	}
	if n.readonly {
		return syscall.EROFS
	}
	if err := fs.preflightVolumeOperationsLocked(n.volume, 1); err != nil {
		return err
	}
	if err := fs.recordMetadataLocked(n, n.mode, modTime); err != nil {
		return err
	}
	n.modTime = modTime
	return nil
}

func (fs *FS) Chdir(name string) error {
	if fs.process {
		return processPathOperation(processVolumeCommand{Operation: processVolumeChdirOp, Path: name})
	}
	path, _, err := fs.normalize(name)
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n, err := fs.lookupLocked(path)
	if err != nil {
		return err
	}
	if n.kind != KindDirectory {
		return syscall.ENOTDIR
	}
	fs.cwd = path
	return nil
}

func (fs *FS) Getwd() string {
	if fs.process {
		return processGetwd()
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.cwd
}

func (fs *FS) lookupLocked(path string) (*node, error) {
	if fs.unavailable != nil {
		return nil, fs.unavailable
	}
	if n := fs.nodes[path]; n != nil {
		return n, nil
	}
	if fs.loader == nil {
		return nil, syscall.ENOENT
	}
	entry, status, err := fs.loader(path)
	if err != nil {
		return nil, err
	}
	switch status {
	case MountUnmounted, MountNotExist:
		return nil, syscall.ENOENT
	case MountOK:
		if fs.liveNodes == maximumNodes || len(entry.Children) > maximumNodes || uint64(len(entry.Data)) > MaximumFileBytes || uint64(len(entry.Data)) > maximumTotalBytes-fs.usedBytes {
			return nil, syscall.ENOSPC
		}
		n := &node{inode: fs.allocateInodeLocked(), mode: entry.Mode & 0o777, kind: entry.Kind, data: append([]byte(nil), entry.Data...), children: append([]Child(nil), entry.Children...), readonly: true, linked: true, modTime: fs.nowLocked()}
		fs.nodes[path] = n
		fs.liveNodes++
		fs.usedBytes += uint64(len(n.data))
		return n, nil
	default:
		return nil, syscall.EPROTO
	}
}

func (fs *FS) releaseNodeLocked(n *node) {
	if n.linked || n.handles != 0 {
		return
	}
	fs.usedBytes -= uint64(len(n.data))
	fs.liveNodes--
}

func (fs *FS) directoryEntriesLocked(path string) int {
	entries := 0
	for name := range fs.nodes {
		if name != path && parentPath(name) == path {
			entries++
		}
	}
	return entries
}

func (fs *FS) nowLocked() int64 {
	if fs.clock == nil {
		return 0
	}
	return fs.clock()
}

func parentPath(path string) string {
	parent := path[:strings.LastIndexByte(path, '/')]
	if parent == "" {
		return "/"
	}
	return parent
}

func entryForNode(name string, n *node) Entry {
	return Entry{Name: name, Mode: n.mode, Kind: n.kind, ModTime: n.modTime, Data: append([]byte(nil), n.data...), Children: append([]Child(nil), n.children...)}
}
