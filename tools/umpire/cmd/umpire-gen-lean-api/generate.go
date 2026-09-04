package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/common/artifactio"
)

func Run(arguments []string) error {
	configuration, err := parseGenerationConfig(arguments)
	if err != nil {
		return err
	}
	return run(configuration)
}

func run(configuration generationConfig) error {
	inputs := make([]descriptorInput, 0, len(configuration.Descriptors))
	for _, input := range configuration.Descriptors {
		descriptor, err := descriptorFileInput(input.Path, input.Locator)
		if err != nil {
			return err
		}
		inputs = append(inputs, descriptor)
	}
	set, err := mergeDescriptorInputs(inputs)
	if err != nil {
		return err
	}
	projection, err := buildProjection(set)
	if err != nil {
		return err
	}
	artifacts, err := generateArtifacts(configuration, projection)
	if err != nil {
		return err
	}
	outputRoot, err := filepath.Abs(configuration.OutputRoot)
	if err != nil {
		return fmt.Errorf("resolve output root: %w", err)
	}
	return publishArtifacts(outputRoot, configuration.Layout, artifacts)
}

func generateArtifacts(configuration generationConfig, projection projection) (map[string][]byte, error) {
	plan, err := buildLeanPlan(projection, configuration)
	if err != nil {
		return nil, err
	}
	artifacts := renderArtifacts(plan)
	if err := validateArtifactMap(configuration.Layout, artifacts); err != nil {
		return nil, err
	}
	return artifacts, nil
}

func publishArtifacts(outputRoot string, layout outputLayout, artifacts map[string][]byte) error {
	return publishArtifactsWith(outputRoot, layout, artifacts, artifactio.Publish)
}

func publishArtifactsWith(
	outputRoot string,
	layout outputLayout,
	artifacts map[string][]byte,
	publish func(string, []byte) error,
) error {
	if err := validateArtifactMap(layout, artifacts); err != nil {
		return err
	}
	paths, err := resolvePublicationPaths(outputRoot, layout)
	if err != nil {
		return err
	}
	for _, owned := range []struct {
		relative string
		absolute string
	}{
		{relative: layout.APIPath, absolute: paths.api},
		{relative: layout.APIDirectory, absolute: paths.apiDirectory},
	} {
		if err := os.RemoveAll(owned.absolute); err != nil {
			return fmt.Errorf("remove owned output %q: %w", owned.relative, err)
		}
	}
	for _, artifact := range []struct {
		relative string
		absolute string
	}{
		{relative: layout.ProtoPath, absolute: paths.proto},
		{relative: layout.TypesPath, absolute: paths.types},
		{relative: layout.APIPath, absolute: paths.api},
	} {
		if err := publish(artifact.absolute, artifacts[artifact.relative]); err != nil {
			return fmt.Errorf("publish generated artifact %q: %w", artifact.relative, err)
		}
	}
	return nil
}

func validateArtifactMap(layout outputLayout, artifacts map[string][]byte) error {
	if layout != newOutputLayout(layout.RootModule) {
		return errors.New("generated output layout is inconsistent with the Lean root")
	}
	expected := []string{layout.APIPath, layout.ProtoPath, layout.TypesPath}
	if len(artifacts) != len(expected) {
		return errors.New("generated artifact map must contain exactly the three managed artifacts")
	}
	for _, path := range expected {
		if err := validateArtifactPath(path); err != nil {
			return err
		}
		if _, exists := artifacts[path]; !exists {
			return fmt.Errorf("generated artifact map must contain exactly the three managed artifacts: missing %q", path)
		}
	}
	return nil
}

func validateArtifactPath(path string) error {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if path == "" || clean != path || filepath.IsAbs(filepath.FromSlash(path)) || path == ".." || strings.HasPrefix(path, "../") {
		return fmt.Errorf("generated artifact path %q is unsafe", path)
	}
	return nil
}

type publicationPaths struct {
	api          string
	apiDirectory string
	proto        string
	types        string
}

func resolvePublicationPaths(outputRoot string, layout outputLayout) (publicationPaths, error) {
	if outputRoot == "" {
		return publicationPaths{}, errors.New("output root is required")
	}
	absoluteRoot, err := filepath.Abs(outputRoot)
	if err != nil {
		return publicationPaths{}, fmt.Errorf("resolve output root: %w", err)
	}
	if err := validateDirectoryIfPresent(absoluteRoot, "output root"); err != nil {
		return publicationPaths{}, err
	}
	resolvedRoot, err := resolveWithMissing(absoluteRoot)
	if err != nil {
		return publicationPaths{}, fmt.Errorf("resolve output root %q: %w", outputRoot, err)
	}
	moduleDirectory, err := joinInside(absoluteRoot, filepath.Dir(filepath.FromSlash(layout.APIDirectory)))
	if err != nil {
		return publicationPaths{}, err
	}
	if err := validateDirectoryChain(absoluteRoot, moduleDirectory); err != nil {
		return publicationPaths{}, err
	}
	resolvedModuleDirectory, err := resolveWithMissing(moduleDirectory)
	if err != nil {
		return publicationPaths{}, fmt.Errorf("resolve managed module directory %q: %w", moduleDirectory, err)
	}
	if !pathWithin(resolvedRoot, resolvedModuleDirectory) {
		return publicationPaths{}, fmt.Errorf(
			"managed module directory %q resolves outside output root %q",
			moduleDirectory, absoluteRoot,
		)
	}

	api, err := joinInside(absoluteRoot, filepath.FromSlash(layout.APIPath))
	if err != nil {
		return publicationPaths{}, err
	}
	apiDirectory, err := joinInside(absoluteRoot, filepath.FromSlash(layout.APIDirectory))
	if err != nil {
		return publicationPaths{}, err
	}
	proto, err := joinInside(absoluteRoot, filepath.FromSlash(layout.ProtoPath))
	if err != nil {
		return publicationPaths{}, err
	}
	types, err := joinInside(absoluteRoot, filepath.FromSlash(layout.TypesPath))
	if err != nil {
		return publicationPaths{}, err
	}
	return publicationPaths{api: api, apiDirectory: apiDirectory, proto: proto, types: types}, nil
}

func validateDirectoryIfPresent(path, description string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %s metadata %q: %w", description, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s %q is not a directory", description, path)
	}
	return nil
}

func validateDirectoryChain(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("managed module directory %q is outside output root %q", target, root)
	}
	current := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Stat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			return nil
		}
		if statErr != nil {
			return fmt.Errorf("read managed module directory metadata %q: %w", current, statErr)
		}
		if !info.IsDir() {
			return fmt.Errorf("managed module directory %q is not a directory", current)
		}
	}
	return nil
}

func resolveWithMissing(path string) (string, error) {
	current := filepath.Clean(path)
	var missing []string
	for {
		_, err := os.Lstat(current)
		if err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(current)
			if resolveErr != nil {
				return "", resolveErr
			}
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func joinInside(root, relative string) (string, error) {
	target := filepath.Join(root, relative)
	if !pathWithin(root, target) {
		return "", fmt.Errorf("generated target %q is outside output root %q", target, root)
	}
	return target, nil
}

func pathWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func sortedArtifactPaths(artifacts map[string][]byte) []string {
	paths := make([]string, 0, len(artifacts))
	for path := range artifacts {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}
