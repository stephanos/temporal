// clock_io parks a goroutine in a raw blocking read that the boundary does not
// model. The runtime can never reach quiescence, so virtual time stands still,
// the sleep never returns, and only the wall watchdog ends the run.
package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

func main() {
	var descriptors [2]int
	if err := syscall.Pipe(descriptors[:]); err != nil {
		fmt.Fprintln(os.Stderr, "pipe:", err)
		os.Exit(1)
	}
	go func() {
		buffer := make([]byte, 1)
		_, _ = syscall.Read(descriptors[0], buffer)
	}()
	time.Sleep(time.Second)
	fmt.Println("clock io ok")
}
