package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	_ "unsafe"
)

var runtimeCompleted atomic.Int32
var userProgress [2]atomic.Int32
var runtimeDone = make(chan struct{})

// The fixture gives its yielding worker a runtime symbol so the runtime's
// own system-goroutine classifier recognizes it without a production hook.
//
//go:linkname busyRuntime runtime.gomadConformanceBusyWorker
func busyRuntime() {
	for step := int32(1); step <= 64; step++ {
		runtimeCompleted.Store(step)
		for userProgress[0].Load() < step || userProgress[1].Load() < step {
			runtime.Gosched()
		}
		runtime.Gosched()
	}
	close(runtimeDone)
}

func main() {
	var group sync.WaitGroup
	var completed [2]int
	users := 2
	busy := false
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "zero":
			users = 0
		case "one":
			users = 1
		case "busy":
			busy = true
		}
	}
	if busy {
		go busyRuntime()
	}
	for user := range users {
		group.Go(func() {
			for range 64 {
				completed[user]++
				if busy {
					userProgress[user].Store(int32(completed[user]))
					for runtimeCompleted.Load() < int32(completed[user]) {
						runtime.Gosched()
					}
				}
				runtime.Gosched()
			}
		})
	}
	group.Wait()
	if busy {
		<-runtimeDone
		fmt.Println("runtime", runtimeCompleted.Load())
	}
	fmt.Println(completed)
}
