package main

import (
	"fmt"
	"time"
)

var marker uint64

//go:noinline
func auditStart() {
	marker++
}

func main() {
	auditStart()
	var observed int64
	for range 1_001 {
		observed ^= time.Now().UnixNano()
	}
	fmt.Println(observed, marker)
}
