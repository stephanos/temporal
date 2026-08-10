package main

import (
	"fmt"
	"sync"
)

func main() {
	const workers = 10
	var mutex sync.Mutex
	var waitGroup sync.WaitGroup
	start := make(chan struct{})
	ready := make(chan struct{}, workers)
	order := make(chan int, workers)
	mutex.Lock()
	for worker := range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			ready <- struct{}{}
			mutex.Lock()
			order <- worker
			mutex.Unlock()
		}()
	}
	close(start)
	for range workers {
		<-ready
	}
	mutex.Unlock()
	waitGroup.Wait()
	close(order)
	for worker := range order {
		fmt.Print(worker)
	}
	fmt.Println()
}
