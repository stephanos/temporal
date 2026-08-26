package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if err := mainError(); err != nil {
		if _, writeErr := fmt.Fprintf(os.Stderr, "umpire-gen-dynamic-config: %v\n", err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func mainError() error {
	args := os.Args[1:]
	if len(args) == 1 && args[0] == registryHelperArgument {
		if err := writeRegistryCatalog(os.Stdout); err != nil {
			return fmt.Errorf("helper: %w", err)
		}
		return nil
	}
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return err
	}
	outputRoot, err := parseOutputRoot(args, moduleRoot)
	if err != nil {
		return err
	}
	ctx := context.Background()
	catalog, err := run(ctx, moduleRoot)
	if err != nil {
		return err
	}
	artifacts, err := renderCatalog(catalog)
	if err != nil {
		return fmt.Errorf("render catalog: %w", err)
	}
	return publishCatalog(outputRoot, artifacts, func(candidateRoot string) error {
		return validateLeanCandidate(ctx, moduleRoot, candidateRoot)
	})
}

func parseOutputRoot(args []string, moduleRoot string) (string, error) {
	flags := flag.NewFlagSet("umpire-gen-dynamic-config", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	outputRoot := flags.String("output-root", "", "root directory for generated Lean modules")
	if err := flags.Parse(args); err != nil {
		return "", fmt.Errorf("parse arguments: %w", err)
	}
	if flags.NArg() != 0 {
		return "", errors.New("unexpected arguments")
	}
	if *outputRoot == "" {
		return "", errors.New("--output-root is required")
	}
	if filepath.IsAbs(*outputRoot) {
		return filepath.Clean(*outputRoot), nil
	}
	return filepath.Join(moduleRoot, filepath.Clean(*outputRoot)), nil
}

func findModuleRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect module root %q: %w", directory, err)
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", errors.New("no go.mod found above working directory")
		}
		directory = parent
	}
}
