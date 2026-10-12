package main

import (
	"fmt"
	"runtime/debug"
)

const goroutineCount = 1024

func main() {
	// Keep the creator runnable until the one-P local queue overflows; GC
	// assists must not schedule the children before the batch shuffle runs.
	debug.SetGCPercent(-1)
	completed := make(chan int, goroutineCount)
	for id := range goroutineCount {
		go func() {
			completed <- id
		}()
	}
	order := make([]int, 0, goroutineCount)
	seen := make([]bool, goroutineCount)
	for range goroutineCount {
		id := <-completed
		if seen[id] {
			panic("goroutine completed twice")
		}
		seen[id] = true
		order = append(order, id)
	}
	fmt.Println(order)
}
