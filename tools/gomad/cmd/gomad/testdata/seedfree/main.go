// Command seedfree prints the same output for every seed, so two seeds of it
// complete with one outcome signature.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "seed-failure":
			if os.Getenv("GOMADSEED") == "2" {
				fmt.Fprintln(os.Stderr, "seed 2 failed")
				os.Exit(17)
			}
		case "stream":
			fmt.Println("gomad callback marker")
			chunk := strings.Repeat("bounded callback output\n", 1024)
			for range 64 {
				fmt.Print(chunk)
			}
			return
		default:
			panic("unknown seedfree fixture mode")
		}
	}
	fmt.Println("same output for every seed")
}
