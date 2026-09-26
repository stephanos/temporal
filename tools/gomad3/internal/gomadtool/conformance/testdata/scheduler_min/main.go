// scheduler_min is the smallest scheduling fixture: without a seed and with one
// P and no preemption its order is fixed, so it detects accidental activation.
package main

import (
	"fmt"
	"sync"
)

func main() {
	const workers = 4
	var mutex sync.Mutex
	var order []int
	var group sync.WaitGroup
	for id := range workers {
		group.Go(func() {
			mutex.Lock()
			order = append(order, id)
			mutex.Unlock()
		})
	}
	group.Wait()
	fmt.Println(order)
}
