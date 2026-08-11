package main

import (
	"fmt"
	"os"
	"time"
)

var initTime = time.Now()

func main() {
	if len(os.Args) != 2 {
		panic("usage: clock <initial|sleep>")
	}

	start := time.Now()
	switch os.Args[1] {
	case "initial":
		if got, want := initTime.UnixNano(), int64(946684800000000000); got != want {
			panic(fmt.Sprintf("init time = %d, want %d", got, want))
		}
		if !start.Equal(initTime) {
			panic(fmt.Sprintf("main time = %v, want %v", start, initTime))
		}
	case "sleep":
		time.Sleep(24 * time.Hour)
		if got, want := time.Since(start), 24*time.Hour; got != want {
			panic(fmt.Sprintf("elapsed time = %v, want %v", got, want))
		}
	default:
		panic(fmt.Sprintf("unknown clock case %q", os.Args[1]))
	}

	fmt.Printf("clock %s ok\n", os.Args[1])
}
