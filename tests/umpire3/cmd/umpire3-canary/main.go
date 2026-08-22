package main

import (
	"context"
	"fmt"
	"os"

	"go.temporal.io/server/tests/umpire3/internal/command"
)

func main() {
	if err := command.RunCanary(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
