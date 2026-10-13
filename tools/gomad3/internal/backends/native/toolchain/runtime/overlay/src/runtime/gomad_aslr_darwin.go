// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin && arm64

package runtime

import (
	"internal/abi"
	"internal/goarch"
	"unsafe"
)

// POSIX_SPAWN_SETEXEC is <sys/spawn.h>'s exec-in-place flag: posix_spawn then
// replaces the calling image the way execve does while still honoring the
// attributes. _POSIX_SPAWN_DISABLE_ASLR is the private flag debuggers (lldb,
// Delve) set to load an image unslid; it is not in the public headers.
const (
	gomadPosixSpawnSetExec     = 0x0040
	gomadPosixSpawnDisableASLR = 0x0100
)

//go:cgo_import_dynamic libc_posix_spawnattr_init posix_spawnattr_init "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_posix_spawnattr_setflags posix_spawnattr_setflags "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_posix_spawn posix_spawn "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc__dyld_get_image_vmaddr_slide _dyld_get_image_vmaddr_slide "/usr/lib/libSystem.B.dylib"

//go:nosplit
//go:cgo_unsafe_args
func gomadPosixSpawnattrInit(attr *uintptr) int32 {
	ret := libcCall(unsafe.Pointer(abi.FuncPCABI0(gomadPosixSpawnattrInit_trampoline)), unsafe.Pointer(&attr))
	KeepAlive(attr)
	return ret
}
func gomadPosixSpawnattrInit_trampoline()

//go:nosplit
//go:cgo_unsafe_args
func gomadPosixSpawnattrSetflags(attr *uintptr, flags int16) int32 {
	ret := libcCall(unsafe.Pointer(abi.FuncPCABI0(gomadPosixSpawnattrSetflags_trampoline)), unsafe.Pointer(&attr))
	KeepAlive(attr)
	return ret
}
func gomadPosixSpawnattrSetflags_trampoline()

//go:nosplit
//go:cgo_unsafe_args
func gomadPosixSpawn(pid *int32, path *byte, fileActions unsafe.Pointer, attr *uintptr, argv, envp **byte) int32 {
	ret := libcCall(unsafe.Pointer(abi.FuncPCABI0(gomadPosixSpawn_trampoline)), unsafe.Pointer(&pid))
	KeepAlive(pid)
	KeepAlive(path)
	KeepAlive(attr)
	KeepAlive(argv)
	KeepAlive(envp)
	return ret
}
func gomadPosixSpawn_trampoline()

// gomadImageSlide returns the load address slide of image index (0 is the
// executable), which is zero exactly when ASLR left the image where it was
// linked.
//
//go:nosplit
//go:cgo_unsafe_args
func gomadImageSlide(index uint32) uintptr {
	var args struct {
		index uint32
		slide uintptr
	}
	args.index = index
	libcCall(unsafe.Pointer(abi.FuncPCABI0(gomadImageSlide_trampoline)), unsafe.Pointer(&args))
	return args.slide
}
func gomadImageSlide_trampoline()

const gomadASLRMarker = "GOMAD3_ASLR_DISABLED=1"

// gomadDisableASLR re-executes an activated target unslid. The darwin/arm64
// linker produces only position-independent executables and the kernel slides
// every image, so the addresses of type descriptors, globals, and functions
// differ between two runs of one binary. Programs key caches by those
// addresses (reflect2's type cache, reflect's own lookup caches, every
// map[reflect.Type]), so the hash-trie nodes they allocate during package
// initialization, and with them the heap layout and every later collection,
// followed the slide. posix_spawn with POSIX_SPAWN_SETEXEC replaces this image
// in place, keeping the pid, inherited descriptors, and process group the
// Runner supervises; the marker variable makes a kernel that ignores the flag
// fail closed instead of re-executing forever.
func gomadDisableASLR() {
	if gomadImageSlide(0) == 0 {
		return
	}
	if _, present := gomadEnv(gomadASLRMarker[:len(gomadASLRMarker)-1]); present {
		print("runtime: Gomad could not disable ASLR\n")
		exit(2)
	}
	count := int32(0)
	for argv_index(argv, argc+1+count) != nil {
		count++
	}
	// The heap does not exist yet, so the vector (entries, marker, nil) and
	// the marker's bytes are mapped directly, zeroed; the exec discards the
	// mapping with the rest of this image.
	vectorBytes := uintptr(count+2) * goarch.PtrSize
	mapping := sysAlloc(vectorBytes+uintptr(len(gomadASLRMarker))+1, &memstats.other_sys, "gomad aslr")
	if mapping == nil {
		print("runtime: Gomad could not map the ASLR re-execution environment\n")
		exit(2)
	}
	envp := unsafe.Slice((**byte)(mapping), int(count)+2)
	marker := unsafe.Slice((*byte)(unsafe.Add(mapping, vectorBytes)), len(gomadASLRMarker)+1)
	copy(marker, gomadASLRMarker)
	for i := int32(0); i < count; i++ {
		envp[i] = argv_index(argv, argc+1+i)
	}
	envp[count] = &marker[0]
	var attr uintptr
	if gomadPosixSpawnattrInit(&attr) != 0 || gomadPosixSpawnattrSetflags(&attr, gomadPosixSpawnSetExec|gomadPosixSpawnDisableASLR) != 0 {
		print("runtime: Gomad could not prepare the ASLR re-execution\n")
		exit(2)
	}
	var pid int32
	gomadPosixSpawn(&pid, unsafe.StringData(executablePath), nil, &attr, argv, &envp[0])
	print("runtime: Gomad could not re-execute without ASLR\n")
	exit(2)
}
