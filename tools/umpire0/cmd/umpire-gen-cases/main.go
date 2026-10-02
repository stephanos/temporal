package main

import (
	"flag"
	"fmt"
	"os"

	_ "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/model/scalav2/goir/testpilot"
)

func main() {
	update := flag.Bool("update", false, "rewrite generated Cases and manifest")
	flag.Parse()
	files, err := testpilot.GenerateCases("model/scalav2/ir")
	if err == nil {
		err = testpilot.SyncCases("model/scalav2/cases", files, *update)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
