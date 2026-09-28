package main

import (
	"fmt"
	"syscall"
	"time"
)

var marker uint64

//go:noinline
func auditStart() {
	marker++
}

func main() {
	// The audit attaches DTrace while the process is stopped here: after an
	// activated runtime has re-executed itself unslid, which would discard
	// probes placed on the first image, and before the audited region.
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGSTOP); err != nil {
		panic(err)
	}
	auditStart()
	var observed int64
	for range 1_001 {
		observed ^= time.Now().UnixNano()
	}
	fmt.Println(observed, marker)
}
