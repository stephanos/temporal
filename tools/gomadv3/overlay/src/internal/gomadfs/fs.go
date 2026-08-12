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
)

type Kind uint8

const (
	KindFile Kind = iota + 1
	KindDirectory
)

type MountStatus uint8

const (
	MountOK MountStatus = iota
	MountUnmounted
	MountNotExist
)

type Child struct {
	Name string
	Mode uint32
	Kind Kind
}

type Entry struct {
	Name     string
	Mode     uint32
	Kind     Kind
	Data     []byte
	Children []Child
}

type OpenFlags struct {
	Read, Write, Append, Create, Exclusive, Truncate bool
}

type Loader func(string) (Entry, MountStatus, error)

type node struct {
	mode     uint32
	kind     Kind
	data     []byte
	children []Child
	readonly bool
}

type FS struct {
	mu     sync.Mutex
	nodes  map[string]*node
	loader Loader
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

func New() *FS {
	return &FS{nodes: map[string]*node{"/": {mode: 0o755, kind: KindDirectory}}}
}

func (fs *FS) SetLoader(loader Loader) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.loader = loader
}

func Normalize(name string) (string, string, error) {
	if name == "" || strings.IndexByte(name, 0) >= 0 {
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

func (fs *FS) Mkdir(name string, perm uint32) error {
	path, _, err := Normalize(name)
	if err != nil || path == "/" {
		return syscall.EINVAL
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if _, found := fs.nodes[path]; found {
		return syscall.EEXIST
	}
	parent := parentPath(path)
	parentNode := fs.nodes[parent]
	if parentNode == nil {
		return syscall.ENOENT
	}
	if parentNode.kind != KindDirectory {
		return syscall.ENOTDIR
	}
	if parentNode.readonly {
		return syscall.EROFS
	}
	fs.nodes[path] = &node{mode: perm & 0o777, kind: KindDirectory}
	return nil
}

func (fs *FS) MkdirAll(name string, perm uint32) error {
	path, _, err := Normalize(name)
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
		if existing := fs.nodes[current]; existing != nil {
			if existing.kind != KindDirectory {
				return syscall.ENOTDIR
			}
			continue
		}
		parent := fs.nodes[parentPath(current)]
		if parent == nil {
			return syscall.ENOENT
		}
		if parent.readonly {
			return syscall.EROFS
		}
		fs.nodes[current] = &node{mode: perm & 0o777, kind: KindDirectory}
	}
	return nil
}

func (fs *FS) Stat(name string) (Entry, error) {
	path, base, err := Normalize(name)
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
	path, _, err := Normalize(name)
	if err != nil {
		return nil, err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n, lookupErr := fs.lookupLocked(path)
	if lookupErr != nil && lookupErr != syscall.ENOENT {
		return nil, lookupErr
	}
	if n == nil {
		if !flags.Create {
			return nil, syscall.ENOENT
		}
		parent := fs.nodes[parentPath(path)]
		if parent == nil {
			return nil, syscall.ENOENT
		}
		if parent.kind != KindDirectory {
			return nil, syscall.ENOTDIR
		}
		if parent.readonly {
			return nil, syscall.EROFS
		}
		n = &node{mode: perm & 0o777, kind: KindFile}
		fs.nodes[path] = n
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
		n.data = nil
	}
	return &Handle{fs: fs, node: n, name: path, readable: flags.Read, writable: flags.Write, append: flags.Append}, nil
}

func (fs *FS) Rename(oldName, newName string) error {
	oldPath, _, err := Normalize(oldName)
	if err != nil {
		return err
	}
	newPath, _, err := Normalize(newName)
	if err != nil || oldPath == "/" || newPath == "/" {
		return syscall.EINVAL
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n := fs.nodes[oldPath]
	if n == nil {
		return syscall.ENOENT
	}
	parent := fs.nodes[parentPath(newPath)]
	if parent == nil {
		return syscall.ENOENT
	}
	if n.readonly || parent.readonly {
		return syscall.EROFS
	}
	if existing := fs.nodes[newPath]; existing != nil && existing.kind == KindDirectory {
		return syscall.EEXIST
	}
	delete(fs.nodes, newPath)
	fs.nodes[newPath] = n
	delete(fs.nodes, oldPath)
	if n.kind == KindDirectory {
		for name, descendant := range fs.nodes {
			if strings.HasPrefix(name, oldPath+"/") {
				delete(fs.nodes, name)
				fs.nodes[newPath+strings.TrimPrefix(name, oldPath)] = descendant
			}
		}
	}
	return nil
}

func (fs *FS) Remove(name string) error {
	path, _, err := Normalize(name)
	if err != nil || path == "/" {
		return syscall.EINVAL
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n := fs.nodes[path]
	if n == nil {
		return syscall.ENOENT
	}
	if n.readonly {
		return syscall.EROFS
	}
	if n.kind == KindDirectory {
		for candidate := range fs.nodes {
			if strings.HasPrefix(candidate, path+"/") {
				return syscall.ENOTEMPTY
			}
		}
	}
	delete(fs.nodes, path)
	return nil
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
		n := &node{mode: entry.Mode & 0o777, kind: entry.Kind, data: append([]byte(nil), entry.Data...), children: append([]Child(nil), entry.Children...), readonly: true}
		fs.nodes[path] = n
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
	if end < handle.offset || end > int64(^uint(0)>>1) {
		return 0, syscall.EFBIG
	}
	if end > int64(len(handle.node.data)) {
		handle.node.data = append(handle.node.data, make([]byte, int(end)-len(handle.node.data))...)
	}
	copy(handle.node.data[handle.offset:end], source)
	handle.offset = end
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
	if end < offset || end > int64(^uint(0)>>1) {
		return 0, syscall.EFBIG
	}
	if end > int64(len(handle.node.data)) {
		handle.node.data = append(handle.node.data, make([]byte, int(end)-len(handle.node.data))...)
	}
	copy(handle.node.data[offset:end], source)
	return len(source), nil
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
	return nil
}

func parentPath(path string) string {
	parent := path[:strings.LastIndexByte(path, '/')]
	if parent == "" {
		return "/"
	}
	return parent
}

func entryForNode(name string, n *node) Entry {
	return Entry{Name: name, Mode: n.mode, Kind: n.kind, Data: append([]byte(nil), n.data...), Children: append([]Child(nil), n.children...)}
}
