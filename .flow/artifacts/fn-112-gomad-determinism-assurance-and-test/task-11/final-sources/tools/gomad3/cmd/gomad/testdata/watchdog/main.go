package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	seed := os.Getenv("GOMADSEED")
	payload := []byte("seed=" + seed)
	if err := os.WriteFile("/campaign-fixture", payload, 0o600); err != nil {
		panic(err)
	}
	observed, err := os.ReadFile("/campaign-fixture")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", observed)
	if len(os.Args) > 1 && os.Args[1] == "watchdog" {
		for {
			runtime.Gosched()
		}
	}
	if seed == "2" {
		fmt.Fprintln(os.Stderr, "known failing seed")
		os.Exit(1)
	}
}
