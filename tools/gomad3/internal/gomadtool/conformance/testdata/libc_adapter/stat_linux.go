package main

import (
	"unsafe"

	"modernc.org/libc"
)

func fstatSize(tls *libc.TLS, descriptor int32) (int32, int64) {
	var status libc.Tstat
	result := libc.Xfstat64(tls, descriptor, uintptr(unsafe.Pointer(&status)))
	return result, int64(status.Fst_size)
}
