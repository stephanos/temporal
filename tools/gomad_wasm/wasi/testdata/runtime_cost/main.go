package main

import (
	"fmt"
	"time"
)

func main() {
	left, right := make(chan int, 1), make(chan int, 1)
	left <- 1
	right <- 1
	total := 0
	for range 100000 {
		select {
		case value := <-left:
			total += value
			left <- value
		case value := <-right:
			total += value
			right <- value
		}
	}
	time.Sleep(time.Nanosecond)
	fmt.Println("runtime-cost", total)
}
