// preemption spins until another goroutine stores a flag. Only asynchronous
// preemption lets that goroutine run on a single P, so the seeded runtime,
// which disables preemption, never finishes.
package main

import (
	"fmt"
	"sync/atomic"
)

func main() {
	var stop atomic.Bool
	go func() {
		stop.Store(true)
	}()
	for !stop.Load() {
	}
	fmt.Println("async preemption enabled")
}
