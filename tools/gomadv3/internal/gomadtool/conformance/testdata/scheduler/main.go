package main

import (
	"fmt"
	"os"
	"runtime"

	"gomadv3.test/internal/layout"
)

func main() {
	padding := layout.New(os.Args[1:])
	const (
		workers = 8
		rounds  = 8
	)
	start := make(chan struct{})
	result := make(chan int, workers*rounds)
	for worker := range workers {
		go func() {
			<-start
			for range rounds {
				result <- worker
				runtime.Gosched()
			}
		}()
	}
	close(start)
	var order [workers * rounds]int
	for index := range order {
		order[index] = <-result
	}
	for _, worker := range order {
		fmt.Print(worker)
	}
	fmt.Println()
	padding.Finish()
}
