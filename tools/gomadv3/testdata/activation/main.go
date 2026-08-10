package main

import (
	"fmt"
	"runtime"
)

func init() {
	fmt.Printf("init GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
}

func main() {
	fmt.Printf("main GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
}
