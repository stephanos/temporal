package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	left, right := make(chan int, 1), make(chan int, 1)
	left <- 1
	right <- 2
	select {
	case value := <-left:
		fmt.Println("select", value)
	case value := <-right:
		fmt.Println("select", value)
	}
	done := make(chan int, 2)
	go func() { done <- 1 }()
	go func() { done <- 2 }()
	runtime.Gosched()
	fmt.Println("runq", <-done, <-done)
	time.Sleep(time.Millisecond)
	fmt.Println("timer")
}
