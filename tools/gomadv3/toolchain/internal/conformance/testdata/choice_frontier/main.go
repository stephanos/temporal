package main

import (
	"fmt"
	"runtime"
)

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	left := make(chan struct{}, 1)
	right := make(chan struct{}, 1)
	left <- struct{}{}
	right <- struct{}{}
	select {
	case <-left:
		fmt.Println("left")
	case <-right:
		fmt.Println("right")
	}
}
