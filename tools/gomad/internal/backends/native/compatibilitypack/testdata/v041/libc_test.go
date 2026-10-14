package compatibility

import (
	"testing"
	"unsafe"

	"modernc.org/libc"
	"modernc.org/libc/fcntl"
)

func TestLibcCompatibilityClosure(t *testing.T) {
	tls := libc.NewTLS()
	defer tls.Close()
	path, err := libc.CString("state")
	if err != nil {
		t.Fatal(err)
	}
	defer libc.Xfree(tls, path)
	mode := uint64(0o640)
	descriptor := libc.Xopen(tls, path, fcntl.O_CREAT|fcntl.O_TRUNC|fcntl.O_RDWR, uintptr(unsafe.Pointer(&mode)))
	if descriptor < 0 {
		t.Fatalf("open = %d", descriptor)
	}
	contents := []byte("state")
	if written := libc.Xwrite(tls, descriptor, uintptr(unsafe.Pointer(&contents[0])), libc.Tsize_t(len(contents))); int64(written) != int64(len(contents)) {
		t.Fatalf("write = %d", written)
	}
	if offset := libc.Xlseek64(tls, descriptor, 0, fcntl.SEEK_SET); offset != 0 {
		t.Fatalf("seek = %d", offset)
	}
	read := make([]byte, len(contents))
	if count := libc.Xread(tls, descriptor, uintptr(unsafe.Pointer(&read[0])), libc.Tsize_t(len(read))); int64(count) != int64(len(read)) || string(read) != string(contents) {
		t.Fatalf("read = %d, %q", count, read)
	}
	if result := libc.Xclose(tls, descriptor); result != 0 {
		t.Fatalf("close = %d", result)
	}
}
