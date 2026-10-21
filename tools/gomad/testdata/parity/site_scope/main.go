package main

import (
	"runtime"

	"go.temporal.io/server/tools/gomad/testdata/parity/site_scope/noise"
)

func main() {
	for range 3 {
		println("dependency", noise.Choose())
	}
	left, right := make(chan int, 1), make(chan int, 1)
	left <- 1
	right <- 2
	select {
	case value := <-left:
		println("application", value)
	case value := <-right:
		println("application", value)
	}
	done := make(chan int, 2)
	noise.Start(done, 1)
	go func() { done <- 2 }()
	runtime.Gosched()
	println("mixed runnable", <-done, <-done)
	noise.Start(done, 3)
	noise.Start(done, 4)
	runtime.Gosched()
	println("dependency runnable", <-done, <-done)
}
