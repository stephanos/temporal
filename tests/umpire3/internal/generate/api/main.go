package api

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"go.temporal.io/server/tests/umpire3/internal/artifactio"
)

const (
	leanOutput         = "Wire.lean"
	descriptorOutput   = "descriptor-manifest.json"
	dispositionsOutput = "field-dispositions.json"
	fixturesOutput     = "conformance-fixtures.json"
)

func Run(arguments []string) error {
	flags := flag.NewFlagSet("umpire3-api", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "generate", "generate or check")
	selection := flags.String("selection", "tests/umpire3/model/Temporal/API/testdata/fixtures/selection.json", "protobuf selection manifest")
	outputRoot := flags.String("output-root", "tests/umpire3/model/Temporal/API/Generated", "generated artifact directory")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	return run(*mode, *selection, *outputRoot)
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
		for _, name := range names {
			if err := artifactio.Publish(filepath.Join(outputRoot, name), artifacts[name]); err != nil {
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
