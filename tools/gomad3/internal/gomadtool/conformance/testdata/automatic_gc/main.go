// automatic_gc allocates enough garbage to trigger collections while goroutines
// interleave, and prints their completion order.
package main

import (
	"fmt"
	"runtime"
	"sync"
)

const workers = 6
const steps = 64
const stepBytes = 64 << 10

var sink [][]byte

func main() {
	var mutex sync.Mutex
	var order []int
	var group sync.WaitGroup
	for id := range workers {
		group.Go(func() {
			retained := make([][]byte, 0, steps)
			for step := range steps {
				block := make([]byte, stepBytes)
				block[step%stepBytes] = byte(id)
				if step%8 == 0 {
					retained = append(retained, block)
				}
				runtime.Gosched()
			}
			mutex.Lock()
			sink = append(sink, retained...)
			order = append(order, id)
			mutex.Unlock()
		})
	}
	group.Wait()
	fmt.Println(order, len(sink))
}
