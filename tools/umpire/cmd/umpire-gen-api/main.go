package main

import (
	"context"
	"fmt"
	"os"

	"go.temporal.io/server/tools/umpire/internal/generate/api"
)

func main() {
	if err := api.Run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
