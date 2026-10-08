package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/toolchain"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	root := "/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad3"
	data, err := os.ReadFile(filepath.Join(root, "toolchain/version/version.json"))
	if err != nil { return err }
	var descriptor struct { Archive struct { Name, URL, SHA256 string } }
	if err := json.Unmarshal(data, &descriptor); err != nil { return err }
	archive, err := toolchain.EnsureSource(context.Background(), toolchain.SourceSpec{CacheDir: filepath.Join(root, ".toolchain/downloads"), Name: descriptor.Archive.Name, URL: descriptor.Archive.URL, SHA256: descriptor.Archive.SHA256})
	if err != nil { return err }
	fmt.Println("verified archive", archive, descriptor.Archive.SHA256)
	return nil
}
