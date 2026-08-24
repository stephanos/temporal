package main

import (
	"context"
	"os"

	"go.temporal.io/server/tools/agentworkflow/internal/cli"
)

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
