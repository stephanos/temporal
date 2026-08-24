package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire/internal/artifactio"
)

func Run(_ context.Context, arguments []string, stdout io.Writer) error {
	configuration, err := parseGenerationConfig(arguments)
	if err != nil {
		return err
	}
	return run(configuration, stdout)
}

func run(configuration generationConfig, stdout io.Writer) error {
	inputs := make([]descriptorInput, 0, len(configuration.Descriptors))
	for _, input := range configuration.Descriptors {
		descriptor, loadErr := descriptorFileInput(input.Name, input.Path, input.Locator)
		if loadErr != nil {
			return loadErr
		}
		inputs = append(inputs, descriptor)
	}
	set, err := mergeDescriptorInputs(inputs)
	if err != nil {
		return err
	}
	projection, err := buildProjection(set, configuration.Classify)
	if err != nil {
		return err
	}
	artifacts, manifest, err := generateArtifacts(configuration, inputs, projection)
	if err != nil {
		return err
	}
	if configuration.Operation == "inspect" {
		return writeInspect(stdout, manifest)
	}
	outputRoot, err := filepath.Abs(configuration.OutputRoot)
	if err != nil {
		return fmt.Errorf("resolve output root: %w", err)
	}
	if configuration.Operation == "check" {
		return checkArtifacts(outputRoot, artifacts, configuration.Layout, configuration.Operation)
	}
	return publishArtifacts(outputRoot, artifacts, configuration.Layout)
}

func writeInspect(stdout io.Writer, manifest generationManifest) error {
	encoded, err := canonicalIndentedJSON(manifest)
	if err != nil {
		return err
	}
	if _, err := stdout.Write(encoded); err != nil {
		return fmt.Errorf("write descriptor inventory: %w", err)
	}
	return nil
}

func checkArtifacts(outputRoot string, artifacts map[string][]byte, layout outputLayout, operation string) error {
	var drift []string
	for _, path := range sortedArtifactPaths(artifacts) {
		current, err := os.ReadFile(filepath.Join(outputRoot, filepath.FromSlash(path)))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				drift = append(drift, path+" (missing)")
				continue
			}
			return fmt.Errorf("read generated artifact %q: %w", path, err)
		}
		if !bytes.Equal(current, artifacts[path]) {
			drift = append(drift, path+" (stale)")
		}
	}
	previous, err := loadPreviousManifest(filepath.Join(outputRoot, filepath.FromSlash(layout.ManifestPath)))
	if err != nil {
		return err
	}
	for _, file := range previous.GeneratedFiles {
		if _, expected := artifacts[file.Path]; !expected {
			drift = append(drift, file.Path+" (unexpected)")
		}
	}
	if len(drift) != 0 {
		slices.Sort(drift)
		return fmt.Errorf("generated protobuf Lean model is stale after %s: %s", operation, strings.Join(drift, ", "))
	}
	return nil
}

func publishArtifacts(
	outputRoot string,
	artifacts map[string][]byte,
	layout outputLayout,
) error {
	previous, err := loadPreviousManifest(filepath.Join(outputRoot, filepath.FromSlash(layout.ManifestPath)))
	if err != nil {
		return err
	}
	paths := sortedArtifactPaths(artifacts)
	current := make(map[string]bool, len(paths))
	for _, path := range paths {
		if err := validateManagedPath(layout, path); err != nil {
			return err
		}
		current[path] = true
	}
	stale := make([]string, 0, len(previous.GeneratedFiles))
	for _, file := range previous.GeneratedFiles {
		if current[file.Path] {
			continue
		}
		if err := validateManagedPath(layout, file.Path); err != nil {
			return fmt.Errorf("refuse to remove stale artifact: %w", err)
		}
		stale = append(stale, file.Path)
	}

	for _, path := range paths {
		if path == layout.ManifestPath {
			continue
		}
		if err := artifactio.Publish(filepath.Join(outputRoot, filepath.FromSlash(path)), artifacts[path]); err != nil {
			return fmt.Errorf("publish generated artifact %q: %w", path, err)
		}
	}
	for _, path := range stale {
		if err := artifactio.Remove(filepath.Join(outputRoot, filepath.FromSlash(path))); err != nil {
			return fmt.Errorf("remove stale generated artifact %q: %w", path, err)
		}
	}
	if err := artifactio.Publish(
		filepath.Join(outputRoot, filepath.FromSlash(layout.ManifestPath)),
		artifacts[layout.ManifestPath],
	); err != nil {
		return fmt.Errorf("publish generation manifest: %w", err)
	}
	return nil
}

func loadPreviousManifest(path string) (generationManifest, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return generationManifest{}, nil
		}
		return generationManifest{}, fmt.Errorf("read previous generation manifest: %w", err)
	}
	var manifest generationManifest
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		return generationManifest{}, fmt.Errorf("decode previous generation manifest: %w", err)
	}
	return manifest, nil
}

func validateManagedPath(layout outputLayout, path string) error {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if clean != path || filepath.IsAbs(path) || strings.HasPrefix(path, "../") {
		return fmt.Errorf("generated artifact path %q is unsafe", path)
	}
	if path != layout.CorePath && path != layout.UmbrellaPath && !strings.HasPrefix(path, layout.GeneratedPath+"/") {
		return fmt.Errorf("generated artifact path %q is outside the managed tree", path)
	}
	return nil
}

func sortedArtifactPaths(artifacts map[string][]byte) []string {
	paths := make([]string, 0, len(artifacts))
	for path := range artifacts {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}
