package gotest

import (
	"fmt"
	"runtime"
	"testing"
)

func TestSeedReachesTestBinary(t *testing.T) {
	left := make(chan struct{}, 1)
	right := make(chan struct{}, 1)
	left <- struct{}{}
	right <- struct{}{}
	var result string
	for range 32 {
		select {
		case <-left:
			result += "L"
			left <- struct{}{}
		case <-right:
			result += "R"
			right <- struct{}{}
		}
	}
	fmt.Printf("GOMAXPROCS=%d choices=%s\n", runtime.GOMAXPROCS(0), result)
}
