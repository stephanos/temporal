// static_addresses prints the addresses of a global, a type descriptor, and a
// function, which caches keyed by those addresses (reflect2, reflect's lookup
// caches, map[reflect.Type]) hash on. They must not move between two runs of
// one seeded binary; on darwin/arm64 every executable is position independent
// and the kernel slides it, so the runtime re-executes an activated target
// with ASLR disabled.
package main

import (
	"fmt"
	"reflect"
	"unsafe"
)

type descriptor struct{ field int }

var global int

func main() {
	var typ interface{} = reflect.TypeOf(descriptor{})
	fmt.Printf("global=%p type=%#x func=%#x\n", &global, (*[2]uintptr)(unsafe.Pointer(&typ))[1], reflect.ValueOf(main).Pointer())
}
