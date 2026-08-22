package main

import (
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: clock_spin <loop|select>")
	}
	time.AfterFunc(time.Hour, func() {
		os.Exit(99)
	})
	switch os.Args[1] {
	case "loop":
		for {
		}
	case "select":
		for {
			select {
			default:
			}
		}
	default:
		panic("unknown clock spin mode")
	}
}
