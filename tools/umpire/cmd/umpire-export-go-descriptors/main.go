package main

import (
	"context"
	"fmt"
	"os"

	"go.temporal.io/server/tools/umpire/internal/export/godescriptors"
)

func main() {
	if err := godescriptors.Run(context.Background(), os.Args[1:]); err != nil {
		if _, writeErr := fmt.Fprintf(os.Stderr, "umpire-export-go-descriptors: %v\n", err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
