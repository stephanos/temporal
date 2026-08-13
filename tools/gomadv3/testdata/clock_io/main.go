package main

import (
	"os"
	"syscall"
	"time"
)

func main() {
	var descriptors [2]int
	if err := syscall.Pipe(descriptors[:]); err != nil {
		panic(err)
	}
	time.AfterFunc(time.Hour, func() {
		os.Exit(99)
	})
	var buffer [1]byte
	if _, err := syscall.Read(descriptors[0], buffer[:]); err != nil {
		panic(err)
	}
	if err := syscall.Close(descriptors[1]); err != nil {
		panic(err)
	}
}
