package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.temporal.io/server/tools/gomad/internal/model/volume/internal/projection"
)

func main() {
	check := flag.Bool("check", false, "verify generated source without writing")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(check bool) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		module, err := os.ReadFile(filepath.Join(root, "go.mod"))
		if err == nil && bytes.HasPrefix(module, []byte("module go.temporal.io/server/tools/gomad\n")) {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			return fmt.Errorf("gomad3 module root not found")
		}
		root = parent
	}
	outputs, err := projection.Generate(root)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(outputs))
	for path := range outputs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		destination := filepath.Join(root, "internal/model/volume", path)
		if check {
			actual, err := os.ReadFile(destination)
			if err != nil {
				return err
			}
			if !bytes.Equal(actual, outputs[path]) {
				return fmt.Errorf("generated volume source is stale: %s", path)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(destination, outputs[path], 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
