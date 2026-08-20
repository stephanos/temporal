package main

import (
	"context"
	"os"

	"go.temporal.io/server/tests/umpire3/internal/command"
)

func main() {
	os.Exit(command.Main(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
