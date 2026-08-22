package main

import (
	"fmt"
	_ "unsafe"
)

//go:linkname runtimeRand runtime.rand
func runtimeRand() uint64

//go:linkname runtimeCheaprand runtime.cheaprand
func runtimeCheaprand() uint32

func main() {
	var random [8]uint64
	var cheap [8]uint32
	for index := range random {
		random[index] = runtimeRand()
		cheap[index] = runtimeCheaprand()
	}
	for index := range random {
		fmt.Printf("%016x %08x\n", random[index], cheap[index])
	}
}
