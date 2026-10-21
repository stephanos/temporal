package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: overlay STOCK_ROOT SEMANTICS_ROOT OUTPUT_ROOT")
		os.Exit(2)
	}
	path, identity, err := toolchain.BuildOverlay(os.Args[1], os.Args[2], os.Args[3])
	if err == nil {
		err = json.NewEncoder(os.Stdout).Encode(struct {
			Path                 string `json:"path"`
			Identity             string `json:"identity"`
			ImplementationSHA256 string `json:"implementation_sha256"`
		}{path, identity, strings.TrimPrefix(toolchain.ImplementationSHA256(), "sha256:")})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
