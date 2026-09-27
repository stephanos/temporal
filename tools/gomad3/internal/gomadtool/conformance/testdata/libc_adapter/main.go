// libc_adapter exercises the modernc libc file boundary through the adapter:
// every operation must land in the deterministic filesystem, so the host
// working directory stays untouched.
package main

import (
	"fmt"
	"os"
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
		fail("mkdir = %d", result)
	}
	path := cString("workspace/state")
	defer libc.Xfree(tls, path)
	mode := uint64(0o640)
	descriptor := libc.Xopen(tls, path, fcntl.O_CREAT|fcntl.O_EXCL|fcntl.O_RDWR, uintptr(unsafe.Pointer(&mode)))
	if descriptor < 0 {
		fail("open = %d", descriptor)
	}
	contents := []byte("state")
	if written := libc.Xwrite(tls, descriptor, uintptr(unsafe.Pointer(&contents[0])), libc.Tsize_t(len(contents))); int64(written) != int64(len(contents)) {
		fail("write = %d", written)
	}
	if offset := libc.Xlseek64(tls, descriptor, 0, fcntl.SEEK_SET); offset != 0 {
		fail("seek = %d", offset)
	}
	read := make([]byte, len(contents))
	if count := libc.Xread(tls, descriptor, uintptr(unsafe.Pointer(&read[0])), libc.Tsize_t(len(read))); int64(count) != int64(len(read)) || string(read) != string(contents) {
		fail("read = %d, %q", count, read)
	}
	if result, size := fstatSize(tls, descriptor); result != 0 || size != int64(len(contents)) {
		fail("fstat = %d size=%d", result, size)
	}
	if result := libc.Xclose(tls, descriptor); result != 0 {
		fail("close = %d", result)
	}
	if result := libc.Xunlink(tls, path); result != 0 {
		fail("unlink = %d", result)
	}
	if result := libc.Xrmdir(tls, directory); result != 0 {
		fail("rmdir = %d", result)
	}
	fmt.Println("ok")
}

func cString(value string) uintptr {
	result, err := libc.CString(value)
	if err != nil {
		fail("cstring: %v", err)
	}
	return result
}

func fail(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(1)
}
