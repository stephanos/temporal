package main

import (
	"fmt"
	"os"

	"go.temporal.io/server/tests/umpire3/internal/command"
)

func main() {
	if err := command.QualifyCompatibility(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
