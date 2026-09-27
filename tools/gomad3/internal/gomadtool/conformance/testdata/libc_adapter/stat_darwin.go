package main

import (
	"unsafe"

	"modernc.org/libc"
	"modernc.org/libc/sys/stat"
)

// darwin's libc package has no Tstat; its fstat fills the sys/stat layout.
func fstatSize(tls *libc.TLS, descriptor int32) (int32, int64) {
	var status stat.Stat
	result := libc.Xfstat64(tls, descriptor, uintptr(unsafe.Pointer(&status)))
	return result, int64(status.Fst_size)
}
