//go:build wasip1 && wasm

package main

import "unsafe"

//go:wasmimport wasi_snapshot_preview1 clock_time_get
//go:noescape
func clockTimeGet(id uint32, precision uint64, timestamp unsafe.Pointer) uint32

func main() {
	var timestamp uint64
	clockTimeGet(2, 1, unsafe.Pointer(&timestamp))
}
