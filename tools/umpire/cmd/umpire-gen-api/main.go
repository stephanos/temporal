package main

import (
	"fmt"
	"os"

	"go.temporal.io/server/tools/umpire/internal/generate/api"
)

func main() {
	if err := api.Run(os.Args[1:]); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
