package main

import (
	"fmt"
	"unsafe"

	"modernc.org/libc"
	"modernc.org/libc/fcntl"
)

func main() {
	tls := libc.NewTLS()
	defer tls.Close()
	directory := cString("workspace")
	defer libc.Xfree(tls, directory)
	if result := libc.Xmkdir(tls, directory, 0o750); result != 0 {
		panic(fmt.Sprintf("mkdir = %d", result))
	}
	path := cString("workspace/state")
	defer libc.Xfree(tls, path)
	mode := uint64(0o640)
	fd := libc.Xopen(tls, path, fcntl.O_CREAT|fcntl.O_EXCL|fcntl.O_RDWR, uintptr(unsafe.Pointer(&mode)))
	if fd < 0 {
		panic(fmt.Sprintf("open = %d", fd))
	}
	contents := []byte("state")
	if written := libc.Xwrite(tls, fd, uintptr(unsafe.Pointer(&contents[0])), libc.Tsize_t(len(contents))); int64(written) != int64(len(contents)) {
		panic(fmt.Sprintf("write = %d", written))
	}
	if offset := libc.Xlseek64(tls, fd, 0, fcntl.SEEK_SET); offset != 0 {
		panic(fmt.Sprintf("seek = %d", offset))
	}
	read := make([]byte, len(contents))
	if count := libc.Xread(tls, fd, uintptr(unsafe.Pointer(&read[0])), libc.Tsize_t(len(read))); int64(count) != int64(len(read)) || string(read) != string(contents) {
		panic(fmt.Sprintf("read = %d, %q", count, read))
	}
	if result := libc.Xfsync(tls, fd); result != 0 {
		panic(fmt.Sprintf("fsync = %d", result))
	}
	if result := libc.Xclose(tls, fd); result != 0 {
		panic(fmt.Sprintf("close = %d", result))
	}
	renamed := cString("workspace/renamed")
	defer libc.Xfree(tls, renamed)
	if result := libc.Xrename(tls, path, renamed); result != 0 {
		panic(fmt.Sprintf("rename = %d", result))
	}
	if result := libc.Xaccess(tls, renamed, 0); result != 0 {
		panic(fmt.Sprintf("access = %d", result))
	}
	if result := libc.Xunlink(tls, renamed); result != 0 {
		panic(fmt.Sprintf("unlink = %d", result))
	}
	var timeval [16]byte
	if result := libc.Xgettimeofday(tls, uintptr(unsafe.Pointer(&timeval[0])), 0); result != 0 {
		panic(fmt.Sprintf("gettimeofday = %d", result))
	}
	fmt.Println("ok")
}

func cString(value string) uintptr {
	result, err := libc.CString(value)
	if err != nil {
		panic(err)
	}
	return result
}
