package main

import (
	"context"
	"fmt"
	"os"

	"go.temporal.io/server/tools/umpire3/internal/command"
)

func main() {
	if err := command.RunDeveloper(context.Background(), os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
