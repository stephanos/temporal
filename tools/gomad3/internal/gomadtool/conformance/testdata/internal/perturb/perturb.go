// Package perturb implements the -gomad-address-padding flag the conformance
// driver passes to scheduling fixtures. Padding shifts the heap layout so the
// driver can check that program output does not depend on addresses.
package perturb

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

const flagName = "-gomad-address-padding"

// MaxPaddingBytes bounds the padding a fixture accepts.
const MaxPaddingBytes = 4194304

var padding []byte
var requested bool

// Apply consumes a -gomad-address-padding flag from args and returns the rest.
// An invalid value terminates the process with status 2 before any output.
func Apply(args []string) []string {
	remaining := make([]string, 0, len(args))
	for _, argument := range args {
		value, found := strings.CutPrefix(argument, flagName+"=")
		if !found {
			remaining = append(remaining, argument)
			continue
		}
		bytes, err := strconv.ParseUint(value, 10, 64)
		if value == "" || err != nil || bytes > MaxPaddingBytes {
			fmt.Fprintf(os.Stderr, "%s must be a decimal byte count up to %d\n", flagName, MaxPaddingBytes)
			os.Exit(2)
		}
		requested = true
		padding = make([]byte, bytes)
		for index := range padding {
			padding[index] = byte(index)
		}
	}
	return remaining
}

// Marker prints the address of a fresh large allocation as the final output
// line when padding was requested. Large objects come straight from the page
// heap, so the address moves with the padding size.
func Marker() {
	if !requested {
		return
	}
	probe := make([]byte, 1<<16)
	if len(padding) != 0 {
		probe[0] = padding[len(padding)-1]
	}
	fmt.Printf("GOMAD3_ADDRESS 0x%x\n", uintptr(unsafe.Pointer(unsafe.SliceData(probe))))
}
