// sync contends on a mutex, a condition variable, and a Once and prints who
// won each acquisition.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"gomad3.test/internal/perturb"
)

const workers = 8
const acquisitions = 16

func main() {
	perturb.Apply(os.Args[1:])
	var mutex sync.Mutex
	var order []int
	counter := 0
	var group sync.WaitGroup
	for id := range workers {
		group.Go(func() {
			for range acquisitions {
				mutex.Lock()
				counter++
				order = append(order, id)
				mutex.Unlock()
				runtime.Gosched()
			}
		})
	}
	group.Wait()

	var once sync.Once
	onceWinner := -1
	var onceGroup sync.WaitGroup
	for id := range workers {
		onceGroup.Go(func() {
			once.Do(func() { onceWinner = id })
		})
	}
	onceGroup.Wait()

	var gate sync.Mutex
	ready := sync.NewCond(&gate)
	released := 0
	var waiters []int
	var condGroup sync.WaitGroup
	for id := range workers {
		condGroup.Go(func() {
			gate.Lock()
			for released == 0 {
				ready.Wait()
			}
			waiters = append(waiters, id)
			gate.Unlock()
		})
	}
	runtime.Gosched()
	gate.Lock()
	released = 1
	ready.Broadcast()
	gate.Unlock()
	condGroup.Wait()

	fields := make([]string, 0, len(order))
	for _, id := range order {
		fields = append(fields, fmt.Sprint(id))
	}
	fmt.Println(strings.Join(fields, ""))
	fmt.Printf("once=%d waiters=%v\n", onceWinner, waiters)
	if counter == workers*acquisitions && len(waiters) == workers && onceWinner >= 0 {
		fmt.Println("sync-oracle:ok")
	} else {
		fmt.Printf("sync-oracle:counter=%d waiters=%d once=%d\n", counter, len(waiters), onceWinner)
	}
	perturb.Marker()
}
