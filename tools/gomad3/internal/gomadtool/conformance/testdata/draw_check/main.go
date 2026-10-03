// draw_check reaches the runtime's host-timed draw sites after its first
// choice points: idle windows send the scheduler through its steal pass, and
// collections and runtime.Callers walk stacks through the pcvalue cache. The
// host-timed fault of GOMAD3_DIAGNOSTIC_PERTURB_DRAW puts the next such draw
// on the seeded stream, which the diagnostic check must refuse.
package main

import (
	"fmt"
	"runtime"
	"sync"
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
