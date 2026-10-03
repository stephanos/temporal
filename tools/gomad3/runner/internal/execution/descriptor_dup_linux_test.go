//go:build linux

package execution

import (
	"syscall"
	"testing"
)

func TestDup2SameDescriptorIsNoOp(t *testing.T) {
	var descriptors [2]int
	if err := syscall.Pipe(descriptors[:]); err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(descriptors[0])
	defer syscall.Close(descriptors[1])

	if err := dup2(descriptors[0], descriptors[0]); err != nil {
		t.Fatalf("dup2 descriptor onto itself: %v", err)
	}
}
