package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"gomadv3.test/internal/layout"
)

type workerResult struct {
	worker   int
	checksum uint64
}

func main() {
	padding := layout.New(os.Args[1:])
	const (
		workers = 8
		rounds  = 64
	)
	debug.SetGCPercent(10)
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := make(chan struct{})
	completed := make(chan workerResult, workers)
	for worker := range workers {
		go func() {
			<-start
			window := make([][]byte, 8)
			var checksum uint64
			for round := range rounds {
				block := make([]byte, 32<<10)
				value := byte(worker + round)
				for index := 0; index < len(block); index += 4096 {
					block[index] = value
					checksum += uint64(value)
				}
				window[round%len(window)] = block
				runtime.Gosched()
			}
			runtime.KeepAlive(window)
			completed <- workerResult{worker: worker, checksum: checksum}
		}()
	}
	close(start)
	digest := uint64(14695981039346656037)
	seen := make([]bool, workers)
	var checksum uint64
	for range workers {
		result := <-completed
		if result.worker < 0 || result.worker >= workers || seen[result.worker] {
			panic("automatic GC fixture lost or duplicated a worker")
		}
		seen[result.worker] = true
		checksum += result.checksum
		digest ^= uint64(result.worker)
		digest *= 1099511628211
	}
	var expected uint64
	for worker := range workers {
		for round := range rounds {
			expected += uint64(byte(worker+round)) * 8
		}
	}
	if checksum != expected {
		panic("automatic GC fixture computed an invalid result")
	}
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if after.NumGC-before.NumGC < 2 {
		panic("automatic GC did not complete two cycles")
	}
	fmt.Printf("automatic-gc:%016x\n", digest)
	padding.Finish()
}
