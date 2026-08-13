package main

import (
	"flag"
	"fmt"
	"os"

	"go.temporal.io/server/tools/gomadv3/internal/version"
)

func main() {
	check := flag.Bool("check", false, "check generated files without changing them")
	root := flag.String("root", ".", "Gomad v3 module root")
	flag.Parse()
	if err := version.Generate(*root, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
