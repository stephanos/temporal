package main

import (
	"flag"
	"fmt"
	"os"

	"go.temporal.io/server/tests/umpire3/migration"
)

func main() {
	testsRoot := flag.String("tests-root", "tests", "root tests directory")
	output := flag.String("output", "tests/umpire3/migration/ledger.json", "checked ledger output")
	flag.Parse()

	ledger, err := migration.Build(*testsRoot)
	if err == nil {
		err = migration.Write(*output, ledger)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
