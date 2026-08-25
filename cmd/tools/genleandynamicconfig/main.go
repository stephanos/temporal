package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := mainError(); err != nil {
		if _, writeErr := fmt.Fprintf(os.Stderr, "genleandynamicconfig: %v\n", err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func mainError() error {
	if len(os.Args) == 2 && os.Args[1] == registryHelperArgument {
		if err := writeRegistryCatalog(os.Stdout); err != nil {
			return fmt.Errorf("helper: %w", err)
		}
		return nil
	}
	if len(os.Args) != 1 {
		return errors.New("unexpected arguments")
	}
	moduleRoot, err := findModuleRoot()
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
	outputRoot := filepath.Join(moduleRoot, "model")
	return publishCatalog(outputRoot, artifacts, func(candidateRoot string) error {
		return validateLeanCandidate(ctx, moduleRoot, candidateRoot)
	})
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
