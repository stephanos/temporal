package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const (
	leanOutput         = "Wire.lean"
	descriptorOutput   = "descriptor-manifest.json"
	dispositionsOutput = "field-dispositions.json"
	fixturesOutput     = "conformance-fixtures.json"
)

func main() {
	mode := flag.String("mode", "generate", "generate or check")
	selection := flag.String("selection", "tests/umpire3/model/Temporal/API/selection.json", "protobuf selection manifest")
	outputRoot := flag.String("output-root", "tests/umpire3/model/Temporal/API/Generated", "generated artifact directory")
	flag.Parse()
	if err := run(*mode, *selection, *outputRoot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(mode, selectionPath, outputRoot string) error {
	selection, err := loadSelection(selectionPath)
	if err != nil {
		return err
	}
	projection, err := buildProjection(selection)
	if err != nil {
		return err
	}
	artifacts, err := generateArtifacts(selection, projection)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(artifacts))
	for name := range artifacts {
		names = append(names, name)
	}
	slices.Sort(names)
	switch mode {
	case "generate":
		if err := os.MkdirAll(outputRoot, 0o755); err != nil {
			return fmt.Errorf("create generated API directory: %w", err)
		}
		for _, name := range names {
			if err := os.WriteFile(filepath.Join(outputRoot, name), artifacts[name], 0o644); err != nil {
				return fmt.Errorf("write generated API artifact %q: %w", name, err)
			}
		}
	case "check":
		for _, name := range names {
			current, err := os.ReadFile(filepath.Join(outputRoot, name))
			if err != nil {
				return fmt.Errorf("read generated API artifact %q: %w", name, err)
			}
			if !bytes.Equal(current, artifacts[name]) {
				return errors.New("generated Temporal API projection is stale; run make umpire3-gen-api")
			}
		}
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
	return nil
}
