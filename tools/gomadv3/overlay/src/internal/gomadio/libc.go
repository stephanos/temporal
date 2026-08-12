// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"errors"
	"io"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const firstLibcDescriptor int32 = 1000

var libcDescriptors = struct {
	sync.Mutex
	next  int32
	files map[int32]libcDescriptor
}{next: firstLibcDescriptor, files: make(map[int32]libcDescriptor)}

type libcDescriptor struct {
	file    *os.File
	entropy bool
}

//go:linkname LibcOpen
func LibcOpen(name string, flags int, mode uint32) (int32, syscall.Errno) {
	descriptorState := libcDescriptor{}
	if name == "/dev/urandom" && flags&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC) == 0 {
		descriptorState.entropy = true
	} else {
		file, err := os.OpenFile(name, flags, os.FileMode(mode))
		if err != nil {
			return -1, libcErrno(err)
		}
		descriptorState.file = file
	}
	libcDescriptors.Lock()
	descriptor := libcDescriptors.next
	libcDescriptors.next++
	libcDescriptors.files[descriptor] = descriptorState
	libcDescriptors.Unlock()
	return descriptor, 0
}

//go:linkname LibcClose
func LibcClose(descriptor int32) syscall.Errno {
	libcDescriptors.Lock()
	state, found := libcDescriptors.files[descriptor]
	delete(libcDescriptors.files, descriptor)
	libcDescriptors.Unlock()
	if !found {
		return syscall.EBADF
	}
	if state.file == nil {
		return 0
	}
	return libcErrno(state.file.Close())
}

//go:linkname LibcRead
func LibcRead(descriptor int32, address uintptr, size uint64, offset int64, positional bool) (int64, syscall.Errno) {
	state, found := libcFile(descriptor)
	if !found {
		return -1, syscall.EBADF
	}
	buffer, errno := libcBuffer(address, size)
	if errno != 0 {
		return -1, errno
	}
	if state.entropy {
		count, err := RandomReader().Read(buffer)
		return int64(count), libcErrno(err)
	}
	var count int
	var err error
	if positional {
		count, err = state.file.ReadAt(buffer, offset)
	} else {
		count, err = state.file.Read(buffer)
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}
	if err != nil {
		return -1, libcErrno(err)
	}
	return int64(count), 0
}

//go:linkname LibcWrite
func LibcWrite(descriptor int32, address uintptr, size uint64, offset int64, positional bool) (int64, syscall.Errno) {
	state, found := libcFile(descriptor)
	if !found || state.file == nil {
		return -1, syscall.EBADF
	}
	buffer, errno := libcBuffer(address, size)
	if errno != 0 {
		return -1, errno
	}
	var count int
	var err error
	if positional {
		count, err = state.file.WriteAt(buffer, offset)
	} else {
		count, err = state.file.Write(buffer)
	}
	if err != nil {
		return -1, libcErrno(err)
	}
	return int64(count), 0
}

//go:linkname LibcSeek
func LibcSeek(descriptor int32, offset int64, whence int) (int64, syscall.Errno) {
	state, found := libcFile(descriptor)
	if !found || state.file == nil {
		return -1, syscall.EBADF
	}
	position, err := state.file.Seek(offset, whence)
	if err != nil {
		return -1, libcErrno(err)
	}
	return position, 0
}

//go:linkname LibcTruncate
func LibcTruncate(descriptor int32, size int64) syscall.Errno {
	state, found := libcFile(descriptor)
	if !found || state.file == nil {
		return syscall.EBADF
	}
	return libcErrno(state.file.Truncate(size))
}

//go:linkname LibcSync
func LibcSync(descriptor int32) syscall.Errno {
	state, found := libcFile(descriptor)
	if !found || state.file == nil {
		return syscall.EBADF
	}
	return libcErrno(state.file.Sync())
}

//go:linkname LibcRemove
func LibcRemove(name string) syscall.Errno {
	return libcErrno(os.Remove(name))
}

//go:linkname LibcRename
func LibcRename(oldName, newName string) syscall.Errno {
	return libcErrno(os.Rename(oldName, newName))
}

//go:linkname LibcMkdir
func LibcMkdir(name string, mode uint32) syscall.Errno {
	return libcErrno(os.Mkdir(name, os.FileMode(mode)))
}

//go:linkname LibcAccess
func LibcAccess(name string) syscall.Errno {
	_, err := os.Stat(name)
	return libcErrno(err)
}

//go:linkname LibcStat
func LibcStat(name string, descriptor int32) (uint32, int64, syscall.Errno) {
	var info os.FileInfo
	var err error
	if descriptor >= firstLibcDescriptor {
		state, found := libcFile(descriptor)
		if !found {
			return 0, 0, syscall.EBADF
		}
		if state.entropy {
			return 0o444, 0, 0
		}
		info, err = state.file.Stat()
	} else {
		info, err = os.Stat(name)
	}
	if err != nil {
		return 0, 0, libcErrno(err)
	}
	mode := uint32(info.Mode().Perm()) | 0o100000
	if info.IsDir() {
		mode = uint32(info.Mode().Perm()) | 0o040000
	}
	return mode, info.Size(), 0
}

//go:linkname LibcIsDescriptor
func LibcIsDescriptor(descriptor int32) bool {
	_, found := libcFile(descriptor)
	return found
}

//go:linkname LibcNow
func LibcNow() (int64, int64) {
	now := time.Now()
	return now.Unix(), int64(now.Nanosecond() / 1000)
}

func libcBuffer(address uintptr, size uint64) ([]byte, syscall.Errno) {
	if size > uint64(^uint(0)>>1) {
		return nil, syscall.EOVERFLOW
	}
	if address == 0 && size != 0 {
		return nil, syscall.EFAULT
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(address)), int(size)), 0
}

func libcFile(descriptor int32) (libcDescriptor, bool) {
	libcDescriptors.Lock()
	defer libcDescriptors.Unlock()
	state, found := libcDescriptors.files[descriptor]
	return state, found
}

func libcErrno(err error) syscall.Errno {
	if err == nil {
		return 0
	}
	var pathError *os.PathError
	if errors.As(err, &pathError) {
		err = pathError.Err
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno
	}
	return syscall.EIO
}
