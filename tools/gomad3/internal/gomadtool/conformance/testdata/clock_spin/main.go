// clock_spin never yields, so virtual time cannot advance and the wall watchdog
// must terminate it.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: clock_spin loop|select")
		os.Exit(2)
	}
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
		fmt.Fprintln(os.Stderr, "usage: clock_spin loop|select")
		os.Exit(2)
	}
}
