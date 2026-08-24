package main

import (
	"fmt"
	"os"

	"go.temporal.io/server/tools/umpire3/internal/command"
)

func main() {
	if err := command.RunVeil(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
