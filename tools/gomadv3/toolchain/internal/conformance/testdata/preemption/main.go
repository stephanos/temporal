package main

import (
	"fmt"
)

func main() {
	started := make(chan struct{})
	go func() {
		close(started)
		for {
		}
	}()
	<-started
	fmt.Println("async preemption enabled")
}
