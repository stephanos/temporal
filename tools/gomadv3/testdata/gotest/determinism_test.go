package gotest

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

func TestDisabledCompatibility(t *testing.T) {
	if _, enabled := os.LookupEnv("GOMADSEED"); enabled {
		t.Skip("disabled-mode compatibility check")
	}
	gomaxprocs := runtime.GOMAXPROCS(0)
	if gomaxprocs != 2 {
		t.Fatalf("GOMAXPROCS = %d, want 2", gomaxprocs)
	}
	fmt.Printf("GOMADV3_COMPAT version=%s goos=%s goarch=%s gomaxprocs=%d\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, gomaxprocs)
}

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
