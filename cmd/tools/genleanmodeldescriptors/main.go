package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if err := Run(context.Background(), os.Args[1:]); err != nil {
		if _, writeErr := fmt.Fprintf(os.Stderr, "genleanmodeldescriptors: %v\n", err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
