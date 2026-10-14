package libc

import (
	"testing"
	"unsafe"

	modernclibc "modernc.org/libc"
	"modernc.org/libc/fcntl"
)

func TestLibcFileBoundaryPreservesContents(t *testing.T) {
	tls := modernclibc.NewTLS()
	defer tls.Close()
	directory := cString(t, "workspace")
	defer modernclibc.Xfree(tls, directory)
	if result := modernclibc.Xmkdir(tls, directory, 0o750); result != 0 {
		t.Fatalf("mkdir = %d", result)
	}
	path := cString(t, "workspace/state")
	defer modernclibc.Xfree(tls, path)
	mode := uint64(0o640)
	descriptor := modernclibc.Xopen(tls, path, fcntl.O_CREAT|fcntl.O_EXCL|fcntl.O_RDWR, uintptr(unsafe.Pointer(&mode)))
	if descriptor < 0 {
		t.Fatalf("open = %d", descriptor)
	}
	contents := []byte("state")
	if written := modernclibc.Xwrite(tls, descriptor, uintptr(unsafe.Pointer(&contents[0])), modernclibc.Tsize_t(len(contents))); int64(written) != int64(len(contents)) {
		t.Fatalf("write = %d", written)
	}
	if offset := modernclibc.Xlseek64(tls, descriptor, 0, fcntl.SEEK_SET); offset != 0 {
		t.Fatalf("seek = %d", offset)
	}
	read := make([]byte, len(contents))
	if count := modernclibc.Xread(tls, descriptor, uintptr(unsafe.Pointer(&read[0])), modernclibc.Tsize_t(len(read))); int64(count) != int64(len(read)) || string(read) != string(contents) {
		t.Fatalf("read = %d, %q", count, read)
	}
	if result := modernclibc.Xclose(tls, descriptor); result != 0 {
		t.Fatalf("close = %d", result)
	}
}

func cString(t *testing.T, value string) uintptr {
	t.Helper()
	result, err := modernclibc.CString(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
