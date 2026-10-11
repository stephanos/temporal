package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	a, b := make(chan int, 1), make(chan int, 1)
	a <- 1
	b <- 2
	var selected int
	select {
	case selected = <-a:
	case selected = <-b:
	}
	time.Sleep(time.Millisecond)
	fmt.Printf("selected=%d\n", selected)
	if len(os.Args) > 1 && os.Args[1] == "canary" {
		os.Exit(17)
	}
}
