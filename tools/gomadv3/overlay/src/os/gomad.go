// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"syscall"
	"time"
	_ "unsafe"

	"internal/gomadfs"
	"internal/gomadio/mount"
	"internal/gomadtrace"
)

//go:linkname gomadProfileEnabled runtime.gomadIOProfileEnabled
func gomadProfileEnabled() bool

//go:linkname gomadDeterministicEnabled runtime.gomadDeterministicEnabled
func gomadDeterministicEnabled() bool

func gomadIOEnabled() bool {
	return gomadDeterministicEnabled()
}

func gomadInterceptFileReaddir(file *File, count int) ([]FileInfo, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	if file == nil {
		return nil, ErrInvalid, true
	}
	_, _, infos, err, handled := gomadFileReaddir(file, count, readdirFileInfo)
	if !handled {
		return nil, nil, false
	}
	if infos == nil {
		infos = []FileInfo{}
	}
	return infos, err, true
}

func gomadInterceptFileReaddirnames(file *File, count int) ([]string, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	if file == nil {
		return nil, ErrInvalid, true
	}
	names, _, _, err, handled := gomadFileReaddir(file, count, readdirName)
	if !handled {
		return nil, nil, false
	}
	if names == nil {
		names = []string{}
	}
	return names, err, true
}

func gomadInterceptFileReadDir(file *File, count int) ([]DirEntry, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	if file == nil {
		return nil, ErrInvalid, true
	}
	_, entries, _, err, handled := gomadFileReaddir(file, count, readdirDirEntry)
	if !handled {
		return nil, nil, false
	}
	if entries == nil {
		entries = []DirEntry{}
	}
	return entries, err, true
}

func gomadInterceptReadDir(name string) ([]DirEntry, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	file, handled, err := gomadOpenFile(name, O_RDONLY, 0)
	if !handled {
		return nil, nil, false
	}
	if err != nil {
		return nil, err, true
	}
	defer file.Close()
	entries, err := file.ReadDir(-1)
	return entries, err, true
}

func gomadInterceptFileReadAt(file *File, destination []byte, offset int64) (int, error, bool) {
	if !gomadIOEnabled() {
		return 0, nil, false
	}
	if err := file.checkValid("read"); err != nil {
		return 0, err, true
	}
	if offset < 0 {
		return 0, &PathError{Op: "readat", Path: file.name, Err: errors.New("negative offset")}, true
	}
	read, err, handled := gomadFileReadAt(file, destination, offset)
	if !handled {
		return 0, nil, false
	}
	return read, file.wrapErr("read", err), true
}

func gomadInterceptFileWrite(file *File, source []byte) (int, error, bool) {
	if !gomadIOEnabled() {
		return 0, nil, false
	}
	if err := file.checkValid("write"); err != nil {
		return 0, err, true
	}
	written, err, handled := gomadFileWrite(file, source)
	if !handled {
		return 0, nil, false
	}
	return written, file.wrapErr("write", err), true
}

func gomadInterceptFileWriteAt(file *File, source []byte, offset int64) (int, error, bool) {
	if !gomadIOEnabled() {
		return 0, nil, false
	}
	if err := file.checkValid("write"); err != nil {
		return 0, err, true
	}
	written, err, handled := gomadFileWriteAt(file, source, offset)
	if !handled {
		return 0, nil, false
	}
	return written, file.wrapErr("write", err), true
}

func gomadInterceptFileSeek(file *File, offset int64, whence int) (int64, error, bool) {
	if !gomadIOEnabled() {
		return 0, nil, false
	}
	if err := file.checkValid("seek"); err != nil {
		return 0, err, true
	}
	next, err, handled := gomadFileSeek(file, offset, whence)
	if !handled {
		return 0, nil, false
	}
	return next, file.wrapErr("seek", err), true
}

func gomadInterceptMkdir(name string, perm FileMode) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadMkdir(name, perm), true
}

func gomadInterceptChdir(name string) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadChdir(name), true
}

func gomadInterceptRename(oldName, newName string) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadRename(oldName, newName), true
}

func gomadInterceptReadlink(name string) (string, error, bool) {
	if !gomadIOEnabled() {
		return "", nil, false
	}
	return "", gomadUnsupportedPath("readlink", name), true
}

func gomadInterceptChmod(name string, mode FileMode) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadChmod(name, mode), true
}

func gomadInterceptFileChmod(file *File, mode FileMode) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	handled, err := gomadFileChmod(file, mode)
	if !handled {
		return nil, false
	}
	return file.wrapErr("chmod", err), true
}

func gomadInterceptChown(name string, _, _ int) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadUnsupportedPath("chown", name), true
}

func gomadInterceptLchown(name string, _, _ int) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadUnsupportedPath("lchown", name), true
}

func gomadInterceptFileChown(file *File, _, _ int) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	if err := file.checkValid("chown"); err != nil {
		return err, true
	}
	if gomadFileUnsupported(file) {
		return file.wrapErr("chown", syscall.ENOTSUP), true
	}
	return nil, false
}

func gomadInterceptFileTruncate(file *File, size int64) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	if err := file.checkValid("truncate"); err != nil {
		return err, true
	}
	handled, err := gomadFileTruncate(file, size)
	if !handled {
		return nil, false
	}
	return file.wrapErr("truncate", err), true
}

func gomadInterceptFileSync(file *File) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	if err := file.checkValid("sync"); err != nil {
		return err, true
	}
	handled, err := gomadFileSync(file)
	if !handled {
		return nil, false
	}
	return file.wrapErr("sync", err), true
}

func gomadInterceptChtimes(name string, _ time.Time, modified time.Time) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadChtimes(name, modified), true
}

func gomadInterceptFileChdir(file *File) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	if err := file.checkValid("chdir"); err != nil {
		return err, true
	}
	handled, err := gomadFileChdir(file)
	if !handled {
		return nil, false
	}
	return file.wrapErr("chdir", err), true
}

func gomadInterceptTruncate(name string, size int64) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadTruncate(name, size), true
}

func gomadInterceptRemove(name string) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadRemove(name), true
}

func gomadInterceptLink(oldName, newName string) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadUnsupportedLink("link", oldName, newName), true
}

func gomadInterceptSymlink(oldName, newName string) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadUnsupportedLink("symlink", oldName, newName), true
}

func gomadInterceptGetwd() (string, error, bool) {
	if !gomadIOEnabled() {
		return "", nil, false
	}
	return gomadGetwd(), nil, true
}

func gomadInterceptMkdirAll(name string, perm FileMode) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadMkdirAll(name, perm), true
}

func gomadInterceptRemoveAll(name string) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	return gomadRemoveAll(name), true
}

func gomadInterceptStat(name string) (FileInfo, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	info, err := gomadStat(name)
	return info, err, true
}

func gomadInterceptLstat(name string) (FileInfo, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	info, err := gomadStat(name)
	return info, err, true
}

func gomadInterceptFileStat(file *File) (FileInfo, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	if file == nil {
		return nil, ErrInvalid, true
	}
	info, handled, err := gomadFileStat(file)
	if !handled {
		return nil, nil, false
	}
	return info, file.wrapErr("stat", err), true
}

func gomadInterceptHostname() (string, error, bool) {
	if !gomadIOEnabled() {
		return "", nil, false
	}
	name, err := gomadHostname()
	return name, err, true
}

var gomadFilesystemOnce sync.Once

var gomadOpenHandles = struct {
	sync.RWMutex
	handles map[*file]*gomadfs.Handle
}{handles: make(map[*file]*gomadfs.Handle)}

func gomadInitializeFilesystem() {
	gomadFilesystemOnce.Do(func() {
		gomadfs.Default.SetLoader(gomadLoadMount)
		gomadfs.Default.SetClock(func() int64 { return time.Now().UnixNano() })
	})
}

func gomadLoadMount(path string) (gomadfs.Entry, gomadfs.MountStatus, error) {
	entry, status, err := mount.Default.Lookup(path)
	if err == syscall.EBADF {
		return gomadfs.Entry{}, gomadfs.MountUnmounted, nil
	}
	if err != nil {
		return gomadfs.Entry{}, 0, err
	}
	converted := gomadfs.Entry{Mode: entry.Mode, Kind: gomadfs.Kind(entry.Kind), Data: entry.Data, Children: make([]gomadfs.Child, 0, len(entry.Children))}
	for _, child := range entry.Children {
		converted.Children = append(converted.Children, gomadfs.Child{Name: child.Name, Mode: child.Mode, Kind: gomadfs.Kind(child.Kind)})
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
	return gomadFileInfo{name: base, size: int64(len(entry.Data)), mode: mode, modTime: entry.ModTime, directory: entry.Kind == gomadfs.KindDirectory}, nil
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
	if gomadProfileEnabled() {
		gomadtrace.Init()
		gomadtrace.Record("os.open", []byte(path), nil, uint64(flag), 0, 0, 0)
	}
	return file, true, nil
}

func gomadInterceptOpenFile(name string, flag int, perm FileMode) (*File, error, bool) {
	if !gomadIOEnabled() {
		return nil, nil, false
	}
	file, handled, err := gomadOpenFile(name, flag, perm)
	return file, err, handled
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
	gomadRecordFile("os.read", handle.Path(), nil, destination[:read], uint64(read), err)
	return read, err, true
}

func gomadInterceptFileRead(file *File, destination []byte) (int, error, bool) {
	if !gomadIOEnabled() {
		return 0, nil, false
	}
	if err := file.checkValid("read"); err != nil {
		return 0, err, true
	}
	read, err, handled := gomadFileRead(file, destination)
	if !handled {
		return 0, nil, false
	}
	return read, file.wrapErr("read", err), true
}

func gomadFileReadAt(file *File, destination []byte, offset int64) (int, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	read, err := handle.ReadAt(destination, offset)
	gomadRecordFile("os.readat", handle.Path(), gomadInt64Argument(offset), destination[:read], uint64(read), err)
	return read, err, true
}

func gomadFileSeek(file *File, offset int64, whence int) (int64, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	next, err := handle.Seek(offset, whence)
	gomadRecordFile("os.seek", handle.Path(), gomadTwoInt64Arguments(offset, int64(whence)), nil, uint64(next), err)
	return next, err, true
}

func gomadFileWrite(file *File, source []byte) (int, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	written, err := handle.Write(source)
	gomadRecordFile("os.write", handle.Path(), nil, source[:written], uint64(written), err)
	return written, err, true
}

func gomadFileWriteAt(file *File, source []byte, offset int64) (int, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return 0, nil, false
	}
	written, err := handle.WriteAt(source, offset)
	gomadRecordFile("os.writeat", handle.Path(), gomadInt64Argument(offset), source[:written], uint64(written), err)
	return written, err, true
}

func gomadFileTruncate(file *File, size int64) (bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return false, nil
	}
	err := handle.Truncate(size)
	gomadRecordFile("os.truncate", handle.Path(), gomadInt64Argument(size), nil, 0, err)
	return true, err
}

func gomadTruncate(name string, size int64) error {
	file, _, err := gomadOpenFile(name, O_WRONLY, 0)
	if err != nil {
		return err
	}
	truncateErr := file.Truncate(size)
	closeErr := file.Close()
	if truncateErr != nil {
		return truncateErr
	}
	return closeErr
}

func gomadChmod(name string, mode FileMode) error {
	gomadInitializeFilesystem()
	if err := gomadfs.Default.Chmod(name, uint32(mode)); err != nil {
		gomadRecordFile("os.chmod", name, gomadInt64Argument(int64(mode)), nil, 0, err)
		return &PathError{Op: "chmod", Path: name, Err: err}
	}
	gomadRecordFile("os.chmod", name, gomadInt64Argument(int64(mode)), nil, 0, nil)
	return nil
}

func gomadChtimes(name string, mtime time.Time) error {
	gomadInitializeFilesystem()
	if mtime.IsZero() {
		gomadRecordFile("os.chtimes", name, gomadInt64Argument(0), nil, 0, nil)
		return nil
	}
	if err := gomadfs.Default.Chtimes(name, mtime.UnixNano()); err != nil {
		gomadRecordFile("os.chtimes", name, gomadInt64Argument(mtime.UnixNano()), nil, 0, err)
		return &PathError{Op: "chtimes", Path: name, Err: err}
	}
	gomadRecordFile("os.chtimes", name, gomadInt64Argument(mtime.UnixNano()), nil, 0, nil)
	return nil
}

func gomadFileChmod(file *File, mode FileMode) (bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return false, nil
	}
	err := handle.Chmod(uint32(mode))
	gomadRecordFile("os.chmod", handle.Path(), gomadInt64Argument(int64(mode)), nil, 0, err)
	return true, err
}

func gomadFileChdir(file *File) (bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return false, nil
	}
	err := handle.Chdir()
	gomadRecordFile("os.chdir", handle.Path(), nil, nil, 0, err)
	return true, err
}

func gomadFileUnsupported(file *File) bool {
	return gomadHandle(file) != nil
}

func gomadFileSync(file *File) (bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return false, nil
	}
	gomadRecordFile("os.sync", handle.Path(), nil, nil, 0, nil)
	return true, nil
}

func gomadFileClose(file *File) (bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return false, nil
	}
	path := handle.Path()
	err := handle.Close()
	gomadRecordFile("os.close", path, nil, nil, 0, err)
	return true, err
}

func gomadInterceptFileClose(file *File) (error, bool) {
	if !gomadIOEnabled() {
		return nil, false
	}
	if file == nil {
		return ErrInvalid, true
	}
	handled, err := gomadFileClose(file)
	if !handled {
		return nil, false
	}
	return file.wrapErr("close", err), true
}

func gomadFileStat(file *File) (FileInfo, bool, error) {
	handle := gomadHandle(file)
	if handle == nil {
		return nil, false, nil
	}
	entry, err := handle.Stat()
	if err != nil {
		gomadRecordFile("os.fstat", handle.Path(), nil, nil, 0, err)
		return nil, true, err
	}
	mode := FileMode(entry.Mode)
	if entry.Kind == gomadfs.KindDirectory {
		mode |= ModeDir
	}
	gomadRecordFile("os.fstat", handle.Path(), nil, nil, uint64(len(entry.Data)), nil)
	return gomadFileInfo{name: entry.Name, size: int64(len(entry.Data)), mode: mode, modTime: entry.ModTime, directory: entry.Kind == gomadfs.KindDirectory}, true, nil
}

func gomadFileReaddir(file *File, count int, mode readdirMode) ([]string, []DirEntry, []FileInfo, error, bool) {
	handle := gomadHandle(file)
	if handle == nil {
		return nil, nil, nil, nil, false
	}
	entries, err := handle.ReadDir(count)
	if err != nil {
		gomadRecordFile("os.readdir", handle.Path(), gomadInt64Argument(int64(count)), nil, 0, err)
		return nil, nil, nil, err, true
	}
	gomadRecordFile("os.readdir", handle.Path(), gomadInt64Argument(int64(count)), nil, uint64(len(entries)), nil)
	switch mode {
	case readdirName:
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name)
		}
		return names, nil, nil, nil, true
	case readdirDirEntry:
		dirents := make([]DirEntry, 0, len(entries))
		for _, entry := range entries {
			fileMode := FileMode(entry.Mode)
			if entry.Kind == gomadfs.KindDirectory {
				fileMode |= ModeDir
			}
			dirents = append(dirents, gomadDirEntry{info: gomadFileInfo{name: entry.Name, size: int64(len(entry.Data)), mode: fileMode, modTime: entry.ModTime, directory: entry.Kind == gomadfs.KindDirectory}})
		}
		return nil, dirents, nil, nil, true
	default:
		infos := make([]FileInfo, 0, len(entries))
		for _, entry := range entries {
			fileMode := FileMode(entry.Mode)
			if entry.Kind == gomadfs.KindDirectory {
				fileMode |= ModeDir
			}
			infos = append(infos, gomadFileInfo{name: entry.Name, size: int64(len(entry.Data)), mode: fileMode, modTime: entry.ModTime, directory: entry.Kind == gomadfs.KindDirectory})
		}
		return nil, nil, infos, nil, true
	}
}

func gomadRename(oldName, newName string) error {
	gomadInitializeFilesystem()
	if err := gomadfs.Default.Rename(oldName, newName); err != nil {
		gomadRecordFile("os.rename", oldName, []byte(newName), nil, 0, err)
		return &LinkError{Op: "rename", Old: oldName, New: newName, Err: err}
	}
	gomadRecordFile("os.rename", oldName, []byte(newName), nil, 0, nil)
	return nil
}

func gomadRemove(name string) error {
	gomadInitializeFilesystem()
	if err := gomadfs.Default.Remove(name); err != nil {
		gomadRecordFile("os.remove", name, nil, nil, 0, err)
		return &PathError{Op: "remove", Path: name, Err: err}
	}
	gomadRecordFile("os.remove", name, nil, nil, 0, nil)
	return nil
}

func gomadRemoveAll(name string) error {
	gomadInitializeFilesystem()
	if err := gomadfs.Default.RemoveAll(name); err != nil {
		gomadRecordFile("os.removeall", name, nil, nil, 0, err)
		return &PathError{Op: "removeall", Path: name, Err: err}
	}
	gomadRecordFile("os.removeall", name, nil, nil, 0, nil)
	return nil
}

func gomadChdir(name string) error {
	gomadInitializeFilesystem()
	if err := gomadfs.Default.Chdir(name); err != nil {
		gomadRecordFile("os.chdir", name, nil, nil, 0, err)
		return &PathError{Op: "chdir", Path: name, Err: err}
	}
	gomadRecordFile("os.chdir", name, nil, nil, 0, nil)
	return nil
}

func gomadGetwd() string {
	gomadInitializeFilesystem()
	workingDirectory := gomadfs.Default.Getwd()
	gomadRecordFile("os.getwd", workingDirectory, nil, nil, uint64(len(workingDirectory)), nil)
	return workingDirectory
}

func gomadUnsupportedPath(operation, name string) error {
	return &PathError{Op: operation, Path: name, Err: syscall.ENOTSUP}
}

func gomadUnsupportedLink(operation, oldName, newName string) error {
	return &LinkError{Op: operation, Old: oldName, New: newName, Err: syscall.ENOTSUP}
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
	if !gomadProfileEnabled() {
		return
	}
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

func gomadRecordFile(operation, path string, arguments, content []byte, count uint64, err error) {
	if !gomadProfileEnabled() {
		return
	}
	result := uint32(0)
	if err != nil && err != io.EOF {
		result = 1
	}
	gomadtrace.Init()
	gomadtrace.Record(operation, append(append([]byte(path), 0), arguments...), content, count, result, 0, 0)
}

func gomadInt64Argument(value int64) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], uint64(value))
	return encoded[:]
}

func gomadTwoInt64Arguments(first, second int64) []byte {
	encoded := make([]byte, 16)
	binary.BigEndian.PutUint64(encoded[:8], uint64(first))
	binary.BigEndian.PutUint64(encoded[8:], uint64(second))
	return encoded
}

func gomadNormalizePath(name string) (string, string, error) {
	gomadInitializeFilesystem()
	return gomadfs.Default.Resolve(name)
}

type gomadFileInfo struct {
	name      string
	size      int64
	mode      FileMode
	modTime   int64
	directory bool
}

func (info gomadFileInfo) Name() string       { return info.name }
func (info gomadFileInfo) Size() int64        { return info.size }
func (info gomadFileInfo) Mode() FileMode     { return info.mode }
func (info gomadFileInfo) ModTime() time.Time { return time.Unix(0, info.modTime) }
func (info gomadFileInfo) IsDir() bool        { return info.directory }
func (info gomadFileInfo) Sys() any           { return nil }

type gomadDirEntry struct {
	info gomadFileInfo
}

func (entry gomadDirEntry) Name() string            { return entry.info.Name() }
func (entry gomadDirEntry) IsDir() bool             { return entry.info.IsDir() }
func (entry gomadDirEntry) Type() FileMode          { return entry.info.Mode().Type() }
func (entry gomadDirEntry) Info() (FileInfo, error) { return entry.info, nil }
