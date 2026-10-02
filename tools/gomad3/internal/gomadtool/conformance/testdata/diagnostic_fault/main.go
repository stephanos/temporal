package main

import (
	"fmt"
	"runtime"
	"sync"
)

func main() {
	var group sync.WaitGroup
	results := make(chan int, 32)
	for worker := range 4 {
		group.Add(1)
		go func() {
			defer group.Done()
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
		}()
	}
	group.Wait()
	close(results)
	total := 0
	for value := range results {
		total += value
	}
	fmt.Println(total)
}
