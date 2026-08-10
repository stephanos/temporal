package main

import (
	"fmt"
	"runtime"
)

func main() {
	start := make(chan struct{})
	result := make(chan int, 6)
	for worker := range 3 {
		go func() {
			<-start
			for range 2 {
				result <- worker
				runtime.Gosched()
			}
		}()
	}
	close(start)
	var order [6]int
	for index := range order {
		order[index] = <-result
	}
	for _, worker := range order {
		fmt.Print(worker)
	}
	fmt.Println()
}
