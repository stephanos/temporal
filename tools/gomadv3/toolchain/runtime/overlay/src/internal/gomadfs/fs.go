// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"io"
	"sort"
	"strings"
	"sync"
	"syscall"

	"internal/gomadmodelwire"
	"internal/gomadwire"
)

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
	handles      map[*Handle]struct{}
	mappings     map[*Mapping]struct{}
	unavailable  error
}

type Handle struct {
	fs              *FS
	processHandle   uint64
	node            *node
	name            string
	offset          int64
	directoryOffset int
	readable        bool
	writable        bool
	append          bool
	closed          bool
	revoked         bool
	generation      uint64
}

type Mapping struct {
	fs            *FS
	processHandle uint64
	node          *node
	data          []byte
	closed        bool
	revoked       bool
	generation    uint64
}

type Statistics struct {
	OpenHandles uint64
	UsedBytes   uint64
}

var Default = New()

const (
	MaximumPathBytes        = 4096
	maximumNodes            = 100_000
	maximumHandles          = 100_000
	maximumDirectoryEntries = 100_000
	MaximumFileBytes        = 16 << 20
	maximumTotalBytes       = 64 << 20
)

func New() *FS {
	return &FS{nodes: map[string]*node{"/": {inode: 1, mode: 0o755, kind: KindDirectory, linked: true}}, liveNodes: 1, cwd: "/", nextInode: 2, generation: 1, handles: make(map[*Handle]struct{}), mappings: make(map[*Mapping]struct{})}
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
	return Statistics{OpenHandles: fs.openHandles, UsedBytes: fs.usedBytes}
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
	handle := &Handle{fs: fs, node: n, name: path, readable: flags.Read, writable: flags.Write, append: flags.Append, generation: fs.generation}
	fs.handles[handle] = struct{}{}
	return handle, nil
}

func (fs *FS) Rename(oldName, newName string) error {
	if fs.process {
		return processPathOperation(gomadmodelwire.VolumeRename, oldName, newName, 0, 0)
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
	if n.kind == KindDirectory && (newPath == oldPath || strings.HasPrefix(newPath, oldPath+"/")) {
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
		return processPathOperation(gomadmodelwire.VolumeRemove, name, "", 0, 0)
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
		return processPathOperation(gomadmodelwire.VolumeRemoveAll, name, "", 0, 0)
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
		return processPathOperation(gomadmodelwire.VolumeChmod, name, "", 0, uint64(mode))
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
		return processPathOperation(gomadmodelwire.VolumeChtimes, name, "", modTime, 0)
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
		return processPathOperation(gomadmodelwire.VolumeChdir, name, "", 0, 0)
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

func (handle *Handle) Read(destination []byte) (int, error) {
	if handle.fs.process {
		return processHandleRead(handle, destination, 0, false)
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	if !handle.readable {
		return 0, syscall.EBADF
	}
	if handle.node.kind != KindFile {
		return 0, syscall.EISDIR
	}
	if handle.offset >= int64(len(handle.node.data)) {
		return 0, io.EOF
	}
	n := copy(destination, handle.node.data[handle.offset:])
	handle.offset += int64(n)
	return n, nil
}

func (handle *Handle) ReadAt(destination []byte, offset int64) (int, error) {
	if handle.fs.process {
		return processHandleRead(handle, destination, offset, true)
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	if !handle.readable {
		return 0, syscall.EBADF
	}
	if offset < 0 {
		return 0, syscall.EINVAL
	}
	if handle.node.kind != KindFile {
		return 0, syscall.EISDIR
	}
	if offset >= int64(len(handle.node.data)) {
		return 0, io.EOF
	}
	n := copy(destination, handle.node.data[offset:])
	if n != len(destination) {
		return n, io.EOF
	}
	return n, nil
}

func (handle *Handle) Write(source []byte) (int, error) {
	if handle.fs.process {
		return processHandleWrite(handle, source, 0, false)
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	if !handle.writable {
		return 0, syscall.EBADF
	}
	if handle.node.readonly {
		return 0, syscall.EROFS
	}
	if handle.append {
		handle.offset = int64(len(handle.node.data))
	}
	end := handle.offset + int64(len(source))
	if end < handle.offset || end > MaximumFileBytes {
		return 0, syscall.EFBIG
	}
	operationCount := uint64(1)
	if end > int64(len(handle.node.data)) {
		operationCount++
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, operationCount); err != nil {
		return 0, err
	}
	if err := handle.fs.preflightVolumeGrowthLocked(handle.node, end); err != nil {
		return 0, err
	}
	start := handle.offset
	modTime := handle.fs.nowLocked()
	if err := handle.fs.recordWriteLocked(handle.node, uint64(start), source, uint64(end), modTime); err != nil {
		return 0, err
	}
	if end > int64(len(handle.node.data)) {
		growth := uint64(end - int64(len(handle.node.data)))
		if growth > maximumTotalBytes-handle.fs.usedBytes {
			return 0, syscall.ENOSPC
		}
		handle.node.data = append(handle.node.data, make([]byte, int(end)-len(handle.node.data))...)
		handle.fs.usedBytes += growth
	}
	copy(handle.node.data[start:end], source)
	handle.fs.updateMappingsLocked(handle.node, uint64(start), source)
	handle.offset = end
	handle.node.modTime = modTime
	return len(source), nil
}

func (handle *Handle) WriteAt(source []byte, offset int64) (int, error) {
	if handle.fs.process {
		return processHandleWrite(handle, source, offset, true)
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	if !handle.writable {
		return 0, syscall.EBADF
	}
	if handle.append || offset < 0 {
		return 0, syscall.EINVAL
	}
	if handle.node.readonly {
		return 0, syscall.EROFS
	}
	end := offset + int64(len(source))
	if end < offset || end > MaximumFileBytes {
		return 0, syscall.EFBIG
	}
	operationCount := uint64(1)
	if end > int64(len(handle.node.data)) {
		operationCount++
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, operationCount); err != nil {
		return 0, err
	}
	if err := handle.fs.preflightVolumeGrowthLocked(handle.node, end); err != nil {
		return 0, err
	}
	modTime := handle.fs.nowLocked()
	if err := handle.fs.recordWriteLocked(handle.node, uint64(offset), source, uint64(end), modTime); err != nil {
		return 0, err
	}
	if end > int64(len(handle.node.data)) {
		growth := uint64(end - int64(len(handle.node.data)))
		if growth > maximumTotalBytes-handle.fs.usedBytes {
			return 0, syscall.ENOSPC
		}
		handle.node.data = append(handle.node.data, make([]byte, int(end)-len(handle.node.data))...)
		handle.fs.usedBytes += growth
	}
	copy(handle.node.data[offset:end], source)
	handle.fs.updateMappingsLocked(handle.node, uint64(offset), source)
	handle.node.modTime = modTime
	return len(source), nil
}

func (handle *Handle) Truncate(size int64) error {
	if handle.fs.process {
		_, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleTruncate, size, 0, 0)
		return err
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	if !handle.writable {
		return syscall.EBADF
	}
	if handle.node.readonly {
		return syscall.EROFS
	}
	if size < 0 {
		return syscall.EINVAL
	}
	if size > MaximumFileBytes {
		return syscall.EFBIG
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, 1); err != nil {
		return err
	}
	if err := handle.fs.preflightVolumeGrowthLocked(handle.node, size); err != nil {
		return err
	}
	modTime := handle.fs.nowLocked()
	if err := handle.fs.recordResizeLocked(handle.node, uint64(size), modTime); err != nil {
		return err
	}
	if size <= int64(len(handle.node.data)) {
		handle.fs.truncateMappingsLocked(handle.node, uint64(size), uint64(len(handle.node.data)))
		handle.fs.usedBytes -= uint64(int64(len(handle.node.data)) - size)
		handle.node.data = handle.node.data[:size]
	} else {
		growth := uint64(size - int64(len(handle.node.data)))
		if growth > maximumTotalBytes-handle.fs.usedBytes {
			return syscall.ENOSPC
		}
		handle.node.data = append(handle.node.data, make([]byte, int(size)-len(handle.node.data))...)
		handle.fs.usedBytes += growth
	}
	handle.node.modTime = modTime
	return nil
}

func (handle *Handle) Chmod(mode uint32) error {
	if handle.fs.process {
		_, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleChmod, 0, 0, uint64(mode))
		return err
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	if handle.node.readonly {
		return syscall.EROFS
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, 1); err != nil {
		return err
	}
	nextMode := mode & 0o777
	nextModTime := handle.fs.nowLocked()
	if err := handle.fs.recordMetadataLocked(handle.node, nextMode, nextModTime); err != nil {
		return err
	}
	handle.node.mode = nextMode
	handle.node.modTime = nextModTime
	return nil
}

func (handle *Handle) Chtimes(modTime int64) error {
	if handle.fs.process {
		_, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleChtimes, modTime, 0, 0)
		return err
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	if handle.node.readonly {
		return syscall.EROFS
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, 1); err != nil {
		return err
	}
	if err := handle.fs.recordMetadataLocked(handle.node, handle.node.mode, modTime); err != nil {
		return err
	}
	handle.node.modTime = modTime
	return nil
}

func (handle *Handle) Chdir() error {
	if handle.fs.process {
		_, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleChdir, 0, 0, 0)
		return err
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	if handle.node.kind != KindDirectory {
		return syscall.ENOTDIR
	}
	if !handle.node.linked {
		return syscall.ENOENT
	}
	for name, candidate := range handle.fs.nodes {
		if candidate == handle.node {
			handle.fs.cwd = name
			return nil
		}
	}
	return syscall.ENOENT
}

func (handle *Handle) Seek(offset int64, whence int) (int64, error) {
	if handle.fs.process {
		response, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleSeek, offset, int64(whence), 0)
		return response.Int1, err
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	var next int64
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = handle.offset + offset
	case io.SeekEnd:
		next = int64(len(handle.node.data)) + offset
	default:
		return 0, syscall.EINVAL
	}
	if next < 0 {
		return 0, syscall.EINVAL
	}
	handle.offset = next
	return next, nil
}

func (handle *Handle) Stat() (Entry, error) {
	if handle.fs.process {
		return processHandleStat(handle)
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return Entry{}, err
	}
	_, base, _ := Normalize(handle.name)
	return entryForNode(base, handle.node), nil
}

func (handle *Handle) Path() string {
	if handle.fs.process {
		return handle.name
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	return handle.name
}

func (handle *Handle) ReadDir(count int) ([]Entry, error) {
	if handle.fs.process {
		return processHandleReadDir(handle, count)
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return nil, err
	}
	if handle.node.kind != KindDirectory {
		return nil, syscall.ENOTDIR
	}
	entries := make(map[string]Entry)
	for _, child := range handle.node.children {
		entries[child.Name] = Entry{Name: child.Name, Mode: child.Mode, Kind: child.Kind}
	}
	for name, childNode := range handle.fs.nodes {
		if parentPath(name) == handle.name && name != handle.name {
			_, base, _ := Normalize(name)
			entries[base] = entryForNode(base, childNode)
		}
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	if handle.directoryOffset == len(names) {
		if count > 0 {
			return nil, io.EOF
		}
		return []Entry{}, nil
	}
	take := len(names) - handle.directoryOffset
	if count > 0 && count < take {
		take = count
	}
	result := make([]Entry, 0, take)
	for _, name := range names[handle.directoryOffset : handle.directoryOffset+take] {
		result = append(result, entries[name])
	}
	handle.directoryOffset += take
	return result, nil
}

func (handle *Handle) Close() error {
	if handle.fs.process {
		_, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleClose, 0, 0, 0)
		if err == nil {
			handle.closed = true
		}
		return err
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	handle.closed = true
	delete(handle.fs.handles, handle)
	handle.fs.openHandles--
	handle.node.handles--
	handle.fs.releaseNodeLocked(handle.node)
	return nil
}

func (handle *Handle) Sync() error {
	if handle.fs.process {
		_, err := processHandleOperation(handle, gomadmodelwire.VolumeHandleSync, 0, 0, 0)
		return err
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	return handle.fs.syncNodeLocked(handle.node)
}

func (handle *Handle) Map(length uint64) (*Mapping, error) {
	if handle.fs.process {
		return processHandleMap(handle, length)
	}
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return nil, err
	}
	if !handle.readable {
		return nil, syscall.EBADF
	}
	if handle.node.kind != KindFile {
		return nil, syscall.ENODEV
	}
	if length == 0 || length > MaximumFileBytes {
		return nil, syscall.EINVAL
	}
	mapping := &Mapping{fs: handle.fs, node: handle.node, data: make([]byte, length), generation: handle.fs.generation}
	copy(mapping.data, handle.node.data)
	handle.fs.mappings[mapping] = struct{}{}
	return mapping, nil
}

func (mapping *Mapping) Bytes() ([]byte, error) {
	if mapping.fs.process {
		return processMappingBytes(mapping)
	}
	mapping.fs.mu.Lock()
	defer mapping.fs.mu.Unlock()
	if err := mapping.errorLocked(); err != nil {
		return nil, err
	}
	return mapping.data, nil
}

func (mapping *Mapping) Close() error {
	if mapping.fs.process {
		return processMappingClose(mapping)
	}
	mapping.fs.mu.Lock()
	defer mapping.fs.mu.Unlock()
	if err := mapping.errorLocked(); err != nil {
		return err
	}
	mapping.closed = true
	delete(mapping.fs.mappings, mapping)
	mapping.data = nil
	return nil
}

func (mapping *Mapping) errorLocked() error {
	if mapping.fs.unavailable != nil {
		return mapping.fs.unavailable
	}
	if mapping.revoked || mapping.generation != mapping.fs.generation {
		return syscall.ESTALE
	}
	if mapping.closed {
		return syscall.EINVAL
	}
	return nil
}

func (fs *FS) truncateMappingsLocked(n *node, size, previous uint64) {
	for mapping := range fs.mappings {
		if mapping.node != n || size >= uint64(len(mapping.data)) {
			continue
		}
		end := min(previous, uint64(len(mapping.data)))
		clear(mapping.data[size:end])
	}
}

func (fs *FS) updateMappingsLocked(n *node, offset uint64, source []byte) {
	for mapping := range fs.mappings {
		if mapping.node != n || offset >= uint64(len(mapping.data)) {
			continue
		}
		copy(mapping.data[offset:], source)
	}
}

func (handle *Handle) errorLocked() error {
	if handle.fs.unavailable != nil {
		return handle.fs.unavailable
	}
	if handle.revoked || handle.generation != handle.fs.generation {
		return syscall.ESTALE
	}
	if handle.closed {
		return syscall.EBADF
	}
	return nil
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
