package main

import (
	"flag"
	"fmt"
	"os"

	_ "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/tools/umpire/lower"
)

func main() {
	update := flag.Bool("update", false, "rewrite generated Cases and manifest")
	flag.Parse()
	files, err := lower.GenerateCases("model/ir")
	if err == nil {
		err = lower.SyncCases("model/cases", files, *update)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
