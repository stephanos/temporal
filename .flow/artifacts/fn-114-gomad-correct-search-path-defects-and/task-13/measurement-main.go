package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	_ "unsafe"
)

var runtimeCompleted int
var runtimeDone = make(chan struct{})

// The fixture gives its yielding worker a runtime symbol so the runtime's
// own system-goroutine classifier recognizes it without a production hook.
//
//go:linkname busyRuntime runtime.gomadConformanceBusyWorker
func busyRuntime() {
	for range 64 {
		runtimeCompleted++
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
				runtime.Gosched()
			}
		})
	}
	group.Wait()
	if busy {
		<-runtimeDone
		fmt.Println("runtime", runtimeCompleted)
	}
	fmt.Println(completed)
}
