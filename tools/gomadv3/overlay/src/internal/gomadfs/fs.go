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
	mode     uint32
	kind     Kind
	data     []byte
	children []Child
	readonly bool
	linked   bool
	handles  uint64
	modTime  int64
}

type FS struct {
	mu          sync.Mutex
	nodes       map[string]*node
	loader      Loader
	clock       func() int64
	openHandles uint64
	usedBytes   uint64
	liveNodes   uint64
	cwd         string
}

type Handle struct {
	fs              *FS
	node            *node
	name            string
	offset          int64
	directoryOffset int
	readable        bool
	writable        bool
	append          bool
	closed          bool
}

var Default = New()

const (
	maximumPathBytes        = 4096
	maximumNodes            = 100_000
	maximumHandles          = 100_000
	maximumDirectoryEntries = 100_000
	maximumFileBytes        = 16 << 20
	maximumTotalBytes       = 64 << 20
)

func New() *FS {
	return &FS{nodes: map[string]*node{"/": {mode: 0o755, kind: KindDirectory, linked: true}}, liveNodes: 1, cwd: "/"}
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
	if name == "" || len(name) > maximumPathBytes || strings.IndexByte(name, 0) >= 0 {
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
	if strings.HasPrefix(name, "/") {
		return Normalize(name)
	}
	fs.mu.Lock()
	cwd := fs.cwd
	fs.mu.Unlock()
	return Normalize(cwd + "/" + name)
}

func (fs *FS) Resolve(name string) (string, string, error) {
	return fs.normalize(name)
}

func (fs *FS) Mkdir(name string, perm uint32) error {
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
	fs.nodes[path] = &node{mode: perm & 0o777, kind: KindDirectory, linked: true, modTime: fs.nowLocked()}
	fs.liveNodes++
	return nil
}

func (fs *FS) MkdirAll(name string, perm uint32) error {
	path, _, err := fs.normalize(name)
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	current := ""
	for _, component := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if component == "" {
			continue
		}
		current += "/" + component
		existing, lookupErr := fs.lookupLocked(current)
		if lookupErr == nil {
			if existing.kind != KindDirectory {
				return syscall.ENOTDIR
			}
			continue
		}
		if lookupErr != syscall.ENOENT {
			return lookupErr
		}
		parent, err := fs.lookupLocked(parentPath(current))
		if err != nil {
			return err
		}
		if parent.readonly {
			return syscall.EROFS
		}
		if fs.liveNodes == maximumNodes {
			return syscall.ENOSPC
		}
		if fs.directoryEntriesLocked(parentPath(current)) == maximumDirectoryEntries {
			return syscall.ENOSPC
		}
		fs.nodes[current] = &node{mode: perm & 0o777, kind: KindDirectory, linked: true, modTime: fs.nowLocked()}
		fs.liveNodes++
	}
	return nil
}

func (fs *FS) Stat(name string) (Entry, error) {
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
		n = &node{mode: perm & 0o777, kind: KindFile, linked: true, modTime: fs.nowLocked()}
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
	if flags.Truncate && flags.Write {
		fs.usedBytes -= uint64(len(n.data))
		n.data = nil
		n.modTime = fs.nowLocked()
	}
	fs.openHandles++
	n.handles++
	return &Handle{fs: fs, node: n, name: path, readable: flags.Read, writable: flags.Write, append: flags.Append}, nil
}

func (fs *FS) Rename(oldName, newName string) error {
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
	if n.kind == KindDirectory {
		for name, descendant := range fs.nodes {
			if strings.HasPrefix(name, oldPath+"/") && descendant.readonly {
				return syscall.EXDEV
			}
		}
	}
	if existing != nil {
		if existing.kind == KindDirectory {
			return syscall.EEXIST
		}
		existing.linked = false
		fs.releaseNodeLocked(existing)
	}
	if fs.nodes[newPath] == nil && parentPath(oldPath) != parentPath(newPath) && fs.directoryEntriesLocked(parentPath(newPath)) == maximumDirectoryEntries {
		return syscall.ENOSPC
	}
	delete(fs.nodes, newPath)
	fs.nodes[newPath] = n
	delete(fs.nodes, oldPath)
	n.modTime = fs.nowLocked()
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
	delete(fs.nodes, path)
	n.linked = false
	fs.releaseNodeLocked(n)
	return nil
}

func (fs *FS) RemoveAll(name string) error {
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
	for candidate, descendant := range fs.nodes {
		if strings.HasPrefix(candidate, path+"/") && descendant.readonly {
			return syscall.EROFS
		}
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
	n.mode = mode & 0o777
	n.modTime = fs.nowLocked()
	return nil
}

func (fs *FS) Chtimes(name string, modTime int64) error {
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
	n.modTime = modTime
	return nil
}

func (fs *FS) Chdir(name string) error {
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
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.cwd
}

func (fs *FS) lookupLocked(path string) (*node, error) {
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
		if fs.liveNodes == maximumNodes || len(entry.Children) > maximumNodes || uint64(len(entry.Data)) > maximumFileBytes || uint64(len(entry.Data)) > maximumTotalBytes-fs.usedBytes {
			return nil, syscall.ENOSPC
		}
		n := &node{mode: entry.Mode & 0o777, kind: entry.Kind, data: append([]byte(nil), entry.Data...), children: append([]Child(nil), entry.Children...), readonly: true, linked: true, modTime: fs.nowLocked()}
		fs.nodes[path] = n
		fs.liveNodes++
		fs.usedBytes += uint64(len(n.data))
		return n, nil
	default:
		return nil, syscall.EPROTO
	}
}

func (handle *Handle) Read(destination []byte) (int, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed {
		return 0, syscall.EBADF
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
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed || !handle.readable {
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
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed || !handle.writable {
		return 0, syscall.EBADF
	}
	if handle.node.readonly {
		return 0, syscall.EROFS
	}
	if handle.append {
		handle.offset = int64(len(handle.node.data))
	}
	end := handle.offset + int64(len(source))
	if end < handle.offset || end > maximumFileBytes {
		return 0, syscall.EFBIG
	}
	if end > int64(len(handle.node.data)) {
		growth := uint64(end - int64(len(handle.node.data)))
		if growth > maximumTotalBytes-handle.fs.usedBytes {
			return 0, syscall.ENOSPC
		}
		handle.node.data = append(handle.node.data, make([]byte, int(end)-len(handle.node.data))...)
		handle.fs.usedBytes += growth
	}
	copy(handle.node.data[handle.offset:end], source)
	handle.offset = end
	handle.node.modTime = handle.fs.nowLocked()
	return len(source), nil
}

func (handle *Handle) WriteAt(source []byte, offset int64) (int, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed || !handle.writable {
		return 0, syscall.EBADF
	}
	if handle.append || offset < 0 {
		return 0, syscall.EINVAL
	}
	if handle.node.readonly {
		return 0, syscall.EROFS
	}
	end := offset + int64(len(source))
	if end < offset || end > maximumFileBytes {
		return 0, syscall.EFBIG
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
	handle.node.modTime = handle.fs.nowLocked()
	return len(source), nil
}

func (handle *Handle) Truncate(size int64) error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed || !handle.writable {
		return syscall.EBADF
	}
	if handle.node.readonly {
		return syscall.EROFS
	}
	if size < 0 {
		return syscall.EINVAL
	}
	if size > maximumFileBytes {
		return syscall.EFBIG
	}
	if size <= int64(len(handle.node.data)) {
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
	handle.node.modTime = handle.fs.nowLocked()
	return nil
}

func (handle *Handle) Chmod(mode uint32) error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed {
		return syscall.EBADF
	}
	if handle.node.readonly {
		return syscall.EROFS
	}
	handle.node.mode = mode & 0o777
	handle.node.modTime = handle.fs.nowLocked()
	return nil
}

func (handle *Handle) Chtimes(modTime int64) error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed {
		return syscall.EBADF
	}
	if handle.node.readonly {
		return syscall.EROFS
	}
	handle.node.modTime = modTime
	return nil
}

func (handle *Handle) Chdir() error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed {
		return syscall.EBADF
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
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed {
		return 0, syscall.EBADF
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
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed {
		return Entry{}, syscall.EBADF
	}
	_, base, _ := Normalize(handle.name)
	return entryForNode(base, handle.node), nil
}

func (handle *Handle) Path() string {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	return handle.name
}

func (handle *Handle) ReadDir(count int) ([]Entry, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed {
		return nil, syscall.EBADF
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
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if handle.closed {
		return syscall.EBADF
	}
	handle.closed = true
	handle.fs.openHandles--
	handle.node.handles--
	handle.fs.releaseNodeLocked(handle.node)
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
