// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"encoding/binary"
	"internal/gomadtrace"
	"io"
	"strings"
	"sync"
	"syscall"
	"time"
	_ "unsafe"
)

//go:linkname gomadProfileEnabled runtime.gomadIOProfileEnabled
func gomadProfileEnabled() bool

func gomadIOEnabled() bool {
	return gomadProfileEnabled()
}

var gomadFilesystem = struct {
	sync.RWMutex
	directories map[string]FileMode
}{}

var gomadFilesystemOnce sync.Once

type gomadOpenHandle struct {
	sync.Mutex
	entry           gomadMountEntry
	offset          int64
	directoryOffset int
	closed          bool
}

const (
	gomadMountRequestDescriptor  = 9
	gomadMountResponseDescriptor = 10
	gomadMountRequestHeader      = 24
	gomadMountResponseHeader     = 40
	gomadMountPathBytes          = 4096
	gomadMountFileBytes          = 16 << 20
	gomadMountDirectoryEntries   = 100_000
)

const (
	gomadMountStatusOK uint16 = iota
	gomadMountStatusUnmounted
	gomadMountStatusNotExist
)

const (
	gomadMountKindFile uint8 = iota + 1
	gomadMountKindDirectory
)

var (
	gomadMountRequestMagic  = [8]byte{'G', 'O', 'M', 'A', 'D', 'R', 'O', 1}
	gomadMountResponseMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'R', 'S', 1}
)

type gomadMountChild struct {
	name string
	mode uint32
	kind uint8
}

type gomadMountEntry struct {
	mode     uint32
	kind     uint8
	data     []byte
	children []gomadMountChild
}

var gomadMountClient struct {
	sync.Mutex
	ordinal uint64
}

var gomadOpenHandles = struct {
	sync.RWMutex
	handles map[*file]*gomadOpenHandle
}{handles: make(map[*file]*gomadOpenHandle)}

func gomadInitializeFilesystem() {
	gomadFilesystemOnce.Do(func() {
		gomadFilesystem.directories = map[string]FileMode{"/": 0o755}
	})
}

func gomadMkdir(name string, perm FileMode) error {
	gomadInitializeFilesystem()
	path, _, err := gomadNormalizePath(name)
	if err != nil || path == "/" {
		return gomadPathError("os.mkdir", "mkdir", name, syscall.EINVAL)
	}
	parent := path[:strings.LastIndexByte(path, '/')]
	if parent == "" {
		parent = "/"
	}
	gomadFilesystem.Lock()
	defer gomadFilesystem.Unlock()
	if _, found := gomadFilesystem.directories[path]; found {
		return gomadPathError("os.mkdir", "mkdir", path, syscall.EEXIST)
	}
	if _, found := gomadFilesystem.directories[parent]; !found {
		return gomadPathError("os.mkdir", "mkdir", path, syscall.ENOENT)
	}
	gomadFilesystem.directories[path] = perm & ModePerm
	gomadRecordPath("os.mkdir", path, uint64(perm&ModePerm), nil)
	return nil
}

func gomadMkdirAll(name string, perm FileMode) error {
	gomadInitializeFilesystem()
	path, _, err := gomadNormalizePath(name)
	if err != nil {
		return gomadPathError("os.mkdirall", "mkdir", name, syscall.EINVAL)
	}
	gomadFilesystem.Lock()
	defer gomadFilesystem.Unlock()
	current := ""
	for _, component := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if component == "" {
			continue
		}
		current += "/" + component
		if _, found := gomadFilesystem.directories[current]; !found {
			gomadFilesystem.directories[current] = perm & ModePerm
		}
	}
	gomadRecordPath("os.mkdirall", path, uint64(perm&ModePerm), nil)
	return nil
}

func gomadStat(name string) (FileInfo, error) {
	gomadInitializeFilesystem()
	path, base, err := gomadNormalizePath(name)
	if err != nil {
		return nil, gomadPathError("os.stat", "stat", name, syscall.EINVAL)
	}
	if info, handled, err := gomadMountStat(path, base); handled {
		return info, err
	}
	gomadFilesystem.RLock()
	mode, found := gomadFilesystem.directories[path]
	gomadFilesystem.RUnlock()
	if !found {
		return nil, gomadPathError("os.stat", "stat", path, syscall.ENOENT)
	}
	gomadRecordPath("os.stat", path, uint64(mode|ModeDir), nil)
	return gomadFileInfo{name: base, mode: mode | ModeDir, directory: true}, nil
}

func gomadOpenFile(name string, flag int, perm FileMode) (*File, bool, error) {
	path, _, err := gomadNormalizePath(name)
	if err != nil {
		return nil, true, gomadPathError("os.mount.open", "open", name, syscall.EINVAL)
	}
	entry, status, err := gomadMountLookup(path)
	if err == syscall.EBADF {
		return nil, false, nil
	}
	if err != nil {
		panic("gomadv3: read-only mount broker failure")
	}
	if status == gomadMountStatusUnmounted {
		if flag&(O_WRONLY|O_RDWR|O_APPEND|O_CREATE|O_EXCL|O_TRUNC) != 0 {
			return nil, false, nil
		}
		return nil, true, gomadPathError("os.mount.open", "open", path, syscall.ENOENT)
	}
	if flag&(O_WRONLY|O_RDWR|O_APPEND|O_CREATE|O_EXCL|O_TRUNC) != 0 {
		return nil, true, gomadPathError("os.mount.open", "open", path, syscall.EROFS)
	}
	if status == gomadMountStatusNotExist {
		return nil, true, gomadPathError("os.mount.open", "open", path, syscall.ENOENT)
	}
	if status != gomadMountStatusOK {
		panic("gomadv3: invalid read-only mount status")
	}
	file := &File{file: &file{name: name}}
	handle := &gomadOpenHandle{entry: entry}
	gomadOpenHandles.Lock()
	gomadOpenHandles.handles[file.file] = handle
	gomadOpenHandles.Unlock()
	gomadtrace.Init()
	gomadtrace.Record("os.mount.open", []byte(path), entry.data, uint64(len(entry.data)), 0, 0, 0)
	return file, true, nil
}

func gomadMountStat(path, base string) (FileInfo, bool, error) {
	entry, status, err := gomadMountLookup(path)
	if err == syscall.EBADF || status == gomadMountStatusUnmounted {
		return nil, false, nil
	}
	if err != nil {
		panic("gomadv3: read-only mount broker failure")
	}
	if status == gomadMountStatusNotExist {
		return nil, true, gomadPathError("os.mount.stat", "stat", path, syscall.ENOENT)
	}
	info := gomadMountFileInfo(base, entry)
	gomadRecordPath("os.mount.stat", path, uint64(info.mode), nil)
	return info, true, nil
}

func gomadMountFileInfo(name string, entry gomadMountEntry) gomadFileInfo {
	mode := FileMode(entry.mode)
	directory := entry.kind == gomadMountKindDirectory
	if directory {
		mode |= ModeDir
	}
	return gomadFileInfo{name: name, size: int64(len(entry.data)), mode: mode, directory: directory}
}

func gomadHandle(file *File) *gomadOpenHandle {
	if file == nil || file.file == nil {
		return nil
	}
	gomadOpenHandles.RLock()
	handle := gomadOpenHandles.handles[file.file]
	gomadOpenHandles.RUnlock()
	return handle
}

func gomadFileRead(file *File, destination []byte) (int, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	handle.Lock()
	defer handle.Unlock()
	if handle.closed {
		return 0, ErrClosed, true
	}
	if handle.entry.kind != gomadMountKindFile {
		return 0, syscall.EISDIR, true
	}
	if handle.offset >= int64(len(handle.entry.data)) {
		return 0, io.EOF, true
	}
	read := copy(destination, handle.entry.data[handle.offset:])
	handle.offset += int64(read)
	return read, nil, true
}

func gomadFileReadAt(file *File, destination []byte, offset int64) (int, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	handle.Lock()
	defer handle.Unlock()
	if handle.closed {
		return 0, ErrClosed, true
	}
	if offset < 0 {
		return 0, syscall.EINVAL, true
	}
	if handle.entry.kind != gomadMountKindFile {
		return 0, syscall.EISDIR, true
	}
	if offset >= int64(len(handle.entry.data)) {
		return 0, io.EOF, true
	}
	read := copy(destination, handle.entry.data[offset:])
	if read != len(destination) {
		return read, io.EOF, true
	}
	return read, nil, true
}

func gomadFileSeek(file *File, offset int64, whence int) (int64, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	handle.Lock()
	defer handle.Unlock()
	if handle.closed {
		return 0, ErrClosed, true
	}
	var next int64
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = handle.offset + offset
	case io.SeekEnd:
		next = int64(len(handle.entry.data)) + offset
	default:
		return 0, syscall.EINVAL, true
	}
	if next < 0 {
		return 0, syscall.EINVAL, true
	}
	handle.offset = next
	return next, nil, true
}

func gomadFileWrite(file *File) (bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return false, nil
	}
	handle.Lock()
	defer handle.Unlock()
	if handle.closed {
		return true, ErrClosed
	}
	return true, syscall.EROFS
}

func gomadFileClose(file *File) (bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return false, nil
	}
	handle.Lock()
	defer handle.Unlock()
	if handle.closed {
		return true, ErrClosed
	}
	handle.closed = true
	return true, nil
}

func gomadFileStat(file *File) (FileInfo, bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return nil, false, nil
	}
	handle.Lock()
	defer handle.Unlock()
	if handle.closed {
		return nil, true, ErrClosed
	}
	_, base, _ := gomadNormalizePath(file.name)
	return gomadMountFileInfo(base, handle.entry), true, nil
}

func gomadFileReaddir(file *File, count int, mode readdirMode) ([]string, []DirEntry, []FileInfo, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return nil, nil, nil, nil, false
	}
	handle.Lock()
	defer handle.Unlock()
	if handle.closed {
		return nil, nil, nil, ErrClosed, true
	}
	if handle.entry.kind != gomadMountKindDirectory {
		return nil, nil, nil, syscall.ENOTDIR, true
	}
	remaining := len(handle.entry.children) - handle.directoryOffset
	if remaining == 0 {
		if count > 0 {
			return nil, nil, nil, io.EOF, true
		}
		return nil, nil, nil, nil, true
	}
	take := remaining
	if count > 0 && count < take {
		take = count
	}
	children := handle.entry.children[handle.directoryOffset : handle.directoryOffset+take]
	handle.directoryOffset += take
	names := make([]string, 0, take)
	dirents := make([]DirEntry, 0, take)
	infos := make([]FileInfo, 0, take)
	for _, child := range children {
		entry := gomadMountEntry{mode: child.mode, kind: child.kind}
		info := gomadMountFileInfo(child.name, entry)
		names = append(names, child.name)
		dirents = append(dirents, gomadDirEntry{info: info})
		infos = append(infos, info)
	}
	switch mode {
	case readdirName:
		return names, nil, nil, nil, true
	case readdirDirEntry:
		return nil, dirents, nil, nil, true
	default:
		return nil, nil, infos, nil, true
	}
}

func gomadHostname() (string, error) {
	const hostname = "gomad-host"
	gomadRecordPath("os.hostname", hostname, uint64(len(hostname)), nil)
	return hostname, nil
}

func gomadPathError(operation, pathOperation, path string, err error) error {
	gomadRecordPath(operation, path, 0, err)
	return &PathError{Op: pathOperation, Path: path, Err: err}
}

func gomadRecordPath(operation, path string, count uint64, err error) {
	gomadtrace.Init()
	result := uint32(0)
	switch err {
	case nil:
	case syscall.EINVAL:
		result = 1
	case syscall.EEXIST:
		result = 2
	case syscall.ENOENT:
		result = 3
	default:
		result = 4
	}
	gomadtrace.Record(operation, []byte(path), nil, count, result, 0, 0)
}

func gomadNormalizePath(name string) (string, string, error) {
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

func gomadMountLookup(path string) (gomadMountEntry, uint16, error) {
	gomadMountClient.Lock()
	defer gomadMountClient.Unlock()
	if len(path) > gomadMountPathBytes {
		return gomadMountEntry{}, 0, syscall.ENAMETOOLONG
	}
	ordinal := gomadMountClient.ordinal
	var request [gomadMountRequestHeader]byte
	copy(request[:8], gomadMountRequestMagic[:])
	binary.BigEndian.PutUint16(request[8:10], 1)
	binary.BigEndian.PutUint16(request[10:12], 1)
	binary.BigEndian.PutUint64(request[12:20], ordinal)
	binary.BigEndian.PutUint32(request[20:24], uint32(len(path)))
	if err := gomadWriteMountBytes(gomadMountRequestDescriptor, request[:]); err != nil {
		return gomadMountEntry{}, 0, err
	}
	if err := gomadWriteMountBytes(gomadMountRequestDescriptor, []byte(path)); err != nil {
		return gomadMountEntry{}, 0, err
	}
	var response [gomadMountResponseHeader]byte
	if err := gomadReadMountBytes(gomadMountResponseDescriptor, response[:]); err != nil {
		return gomadMountEntry{}, 0, err
	}
	if string(response[:8]) != string(gomadMountResponseMagic[:]) || binary.BigEndian.Uint16(response[8:10]) != 1 || binary.BigEndian.Uint64(response[12:20]) != ordinal {
		return gomadMountEntry{}, 0, syscall.EPROTO
	}
	gomadMountClient.ordinal++
	status := binary.BigEndian.Uint16(response[10:12])
	entry := gomadMountEntry{kind: response[20], mode: binary.BigEndian.Uint32(response[24:28])}
	dataBytes := binary.BigEndian.Uint64(response[28:36])
	children := binary.BigEndian.Uint32(response[36:40])
	if dataBytes > gomadMountFileBytes || children > gomadMountDirectoryEntries {
		return gomadMountEntry{}, 0, syscall.EOVERFLOW
	}
	entry.data = make([]byte, int(dataBytes))
	if err := gomadReadMountBytes(gomadMountResponseDescriptor, entry.data); err != nil {
		return gomadMountEntry{}, 0, err
	}
	entry.children = make([]gomadMountChild, 0, children)
	for range children {
		var header [8]byte
		if err := gomadReadMountBytes(gomadMountResponseDescriptor, header[:]); err != nil {
			return gomadMountEntry{}, 0, err
		}
		nameBytes := binary.BigEndian.Uint16(header[:2])
		if nameBytes > gomadMountPathBytes {
			return gomadMountEntry{}, 0, syscall.EOVERFLOW
		}
		name := make([]byte, nameBytes)
		if err := gomadReadMountBytes(gomadMountResponseDescriptor, name); err != nil {
			return gomadMountEntry{}, 0, err
		}
		entry.children = append(entry.children, gomadMountChild{name: string(name), kind: header[2], mode: binary.BigEndian.Uint32(header[4:8])})
	}
	return entry, status, nil
}

func gomadWriteMountBytes(descriptor int, data []byte) error {
	for len(data) != 0 {
		written, err := syscall.Write(descriptor, data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}

func gomadReadMountBytes(descriptor int, data []byte) error {
	for len(data) != 0 {
		read, err := syscall.Read(descriptor, data)
		if err != nil {
			return err
		}
		if read == 0 {
			return io.ErrUnexpectedEOF
		}
		data = data[read:]
	}
	return nil
}

type gomadFileInfo struct {
	name      string
	size      int64
	mode      FileMode
	directory bool
}

func (info gomadFileInfo) Name() string       { return info.name }
func (info gomadFileInfo) Size() int64        { return info.size }
func (info gomadFileInfo) Mode() FileMode     { return info.mode }
func (info gomadFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (info gomadFileInfo) IsDir() bool        { return info.directory }
func (info gomadFileInfo) Sys() any           { return nil }

type gomadDirEntry struct {
	info gomadFileInfo
}

func (entry gomadDirEntry) Name() string            { return entry.info.Name() }
func (entry gomadDirEntry) IsDir() bool             { return entry.info.IsDir() }
func (entry gomadDirEntry) Type() FileMode          { return entry.info.Mode().Type() }
func (entry gomadDirEntry) Info() (FileInfo, error) { return entry.info, nil }
