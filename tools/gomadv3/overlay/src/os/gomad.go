// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"encoding/binary"
	"internal/gomadfs"
	"internal/gomadtrace"
	"io"
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

var gomadFilesystemOnce sync.Once

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
	handles map[*file]*gomadfs.Handle
}{handles: make(map[*file]*gomadfs.Handle)}

func gomadInitializeFilesystem() {
	gomadFilesystemOnce.Do(func() {
		gomadfs.Default.SetLoader(gomadLoadMount)
	})
}

func gomadLoadMount(path string) (gomadfs.Entry, gomadfs.MountStatus, error) {
	entry, status, err := gomadMountLookup(path)
	if err == syscall.EBADF {
		return gomadfs.Entry{}, gomadfs.MountUnmounted, nil
	}
	if err != nil {
		return gomadfs.Entry{}, 0, err
	}
	converted := gomadfs.Entry{Mode: entry.mode, Kind: gomadfs.Kind(entry.kind), Data: entry.data, Children: make([]gomadfs.Child, 0, len(entry.children))}
	for _, child := range entry.children {
		converted.Children = append(converted.Children, gomadfs.Child{Name: child.name, Mode: child.mode, Kind: gomadfs.Kind(child.kind)})
	}
	return converted, gomadfs.MountStatus(status), nil
}

func gomadMkdir(name string, perm FileMode) error {
	gomadInitializeFilesystem()
	path, _, normalizeErr := gomadNormalizePath(name)
	if normalizeErr != nil {
		return gomadPathError("os.mkdir", "mkdir", name, normalizeErr)
	}
	if err := gomadfs.Default.Mkdir(path, uint32(perm)); err != nil {
		return gomadPathError("os.mkdir", "mkdir", path, err)
	}
	gomadRecordPath("os.mkdir", path, uint64(perm&ModePerm), nil)
	return nil
}

func gomadMkdirAll(name string, perm FileMode) error {
	gomadInitializeFilesystem()
	path, _, err := gomadNormalizePath(name)
	if err != nil {
		return gomadPathError("os.mkdirall", "mkdir", name, syscall.EINVAL)
	}
	if err := gomadfs.Default.MkdirAll(path, uint32(perm)); err != nil {
		return gomadPathError("os.mkdirall", "mkdir", path, err)
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
	entry, err := gomadfs.Default.Stat(path)
	if err != nil {
		return nil, gomadPathError("os.stat", "stat", path, err)
	}
	mode := FileMode(entry.Mode)
	if entry.Kind == gomadfs.KindDirectory {
		mode |= ModeDir
	}
	gomadRecordPath("os.stat", path, uint64(mode|ModeDir), nil)
	return gomadFileInfo{name: base, size: int64(len(entry.Data)), mode: mode, directory: entry.Kind == gomadfs.KindDirectory}, nil
}

func gomadOpenFile(name string, flag int, perm FileMode) (*File, bool, error) {
	gomadInitializeFilesystem()
	path, _, err := gomadNormalizePath(name)
	if err != nil {
		return nil, true, gomadPathError("os.mount.open", "open", name, syscall.EINVAL)
	}
	access := flag & (O_WRONLY | O_RDWR)
	handle, err := gomadfs.Default.Open(path, gomadfs.OpenFlags{
		Read: access != O_WRONLY, Write: access != 0, Append: flag&O_APPEND != 0,
		Create: flag&O_CREATE != 0, Exclusive: flag&O_EXCL != 0, Truncate: flag&O_TRUNC != 0,
	}, uint32(perm))
	if err != nil {
		return nil, true, gomadPathError("os.open", "open", path, err)
	}
	file := &File{file: &file{name: name}}
	gomadOpenHandles.Lock()
	gomadOpenHandles.handles[file.file] = handle
	gomadOpenHandles.Unlock()
	gomadtrace.Init()
	gomadtrace.Record("os.open", []byte(path), nil, uint64(flag), 0, 0, 0)
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

func gomadHandle(file *File) *gomadfs.Handle {
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
	read, err := handle.Read(destination)
	return read, err, true
}

func gomadFileReadAt(file *File, destination []byte, offset int64) (int, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	read, err := handle.ReadAt(destination, offset)
	return read, err, true
}

func gomadFileSeek(file *File, offset int64, whence int) (int64, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	next, err := handle.Seek(offset, whence)
	return next, err, true
}

func gomadFileWrite(file *File, source []byte) (int, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	written, err := handle.Write(source)
	return written, err, true
}

func gomadFileWriteAt(file *File, source []byte, offset int64) (int, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	written, err := handle.WriteAt(source, offset)
	return written, err, true
}

func gomadFileClose(file *File) (bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return false, nil
	}
	return true, handle.Close()
}

func gomadFileStat(file *File) (FileInfo, bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return nil, false, nil
	}
	entry, err := handle.Stat()
	if err != nil {
		return nil, true, err
	}
	mode := FileMode(entry.Mode)
	if entry.Kind == gomadfs.KindDirectory {
		mode |= ModeDir
	}
	return gomadFileInfo{name: entry.Name, size: int64(len(entry.Data)), mode: mode, directory: entry.Kind == gomadfs.KindDirectory}, true, nil
}

func gomadFileReaddir(file *File, count int, mode readdirMode) ([]string, []DirEntry, []FileInfo, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return nil, nil, nil, nil, false
	}
	entries, err := handle.ReadDir(count)
	if err != nil {
		return nil, nil, nil, err, true
	}
	names := make([]string, 0, len(entries))
	dirents := make([]DirEntry, 0, len(entries))
	infos := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		fileMode := FileMode(entry.Mode)
		if entry.Kind == gomadfs.KindDirectory {
			fileMode |= ModeDir
		}
		info := gomadFileInfo{name: entry.Name, size: int64(len(entry.Data)), mode: fileMode, directory: entry.Kind == gomadfs.KindDirectory}
		names = append(names, entry.Name)
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

func gomadRename(oldName, newName string) error {
	gomadInitializeFilesystem()
	if err := gomadfs.Default.Rename(oldName, newName); err != nil {
		return &LinkError{Op: "rename", Old: oldName, New: newName, Err: err}
	}
	return nil
}

func gomadRemove(name string) error {
	gomadInitializeFilesystem()
	if err := gomadfs.Default.Remove(name); err != nil {
		return &PathError{Op: "remove", Path: name, Err: err}
	}
	return nil
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
	return gomadfs.Normalize(name)
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
