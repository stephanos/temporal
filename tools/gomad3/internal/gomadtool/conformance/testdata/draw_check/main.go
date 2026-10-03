// draw_check reaches the runtime's host-timed draw sites after its first
// choice points: idle windows send the scheduler through its steal pass, and
// collections and runtime.Callers walk stacks through the pcvalue cache. The
// host-timed fault of GOMAD3_DIAGNOSTIC_PERTURB_DRAW puts every later such
// draw on the seeded stream; the diagnostic check must refuse the first one
// inside a bracketed host-timed path, the steal pass of the idle window before
// the fixture prints.
package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

//go:noinline
func depth(level int, pcs []uintptr) int {
	if level == 0 {
		return runtime.Callers(0, pcs)
	}
	return depth(level-1, pcs) + 1
}

func main() {
	var group sync.WaitGroup
	results := make(chan int, 32)
	for worker := range 4 {
		group.Go(func() {
			for range 8 {
				left, right := make(chan int, 1), make(chan int, 1)
				left <- worker
				right <- worker + 4
				select {
				case value := <-left:
					results <- value
				case value := <-right:
					results <- value
				}
				runtime.Gosched()
			}
		})
	}
	group.Wait()
	// Main is the only goroutine left and it sleeps: the idle window takes
	// the scheduler through its steal pass before virtual time advances.
	time.Sleep(time.Millisecond)
	close(results)
	total := 0
	for value := range results {
		total += value
	}
	pcs := make([]uintptr, 64)
	frames := 0
	for range 4 {
		runtime.GC()
		frames += depth(16, pcs)
	}
	fmt.Println(total, frames)
}
