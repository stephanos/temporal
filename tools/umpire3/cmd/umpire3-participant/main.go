package main

import (
	"context"
	"fmt"
	"os"

	"go.temporal.io/server/tools/umpire3/internal/command"
)

func main() {
	if err := command.RunParticipant(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
