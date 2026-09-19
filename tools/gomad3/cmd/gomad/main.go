package main

import (
	"os"

	"go.temporal.io/server/tools/gomad3/cmd/gomad/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
