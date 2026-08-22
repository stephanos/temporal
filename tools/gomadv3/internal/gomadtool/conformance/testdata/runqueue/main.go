package main

import (
	"fmt"
	"os"

	"gomadv3.test/internal/layout"
)

func main() {
	padding := layout.New(os.Args[1:])
	const (
		waves   = 8
		workers = 768
	)
	digest := uint64(14695981039346656037)
	for wave := range waves {
		start := make(chan struct{})
		completed := make(chan int, workers)
		for worker := range workers {
			go func() {
				<-start
				completed <- worker
			}()
		}
		close(start)
		seen := make([]bool, workers)
		for range workers {
			worker := <-completed
			if worker < 0 || worker >= workers || seen[worker] {
				panic("run queue lost or duplicated a goroutine")
			}
			seen[worker] = true
			digest ^= uint64(wave*workers + worker)
			digest *= 1099511628211
		}
	}
	fmt.Printf("runqueue:%016x\n", digest)
	padding.Finish()
}
