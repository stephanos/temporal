package main

import (
	"context"
	"fmt"
	"os"

	"go.temporal.io/server/tools/umpire/internal/generate/api"
)

func main() {
	if err := api.Run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
