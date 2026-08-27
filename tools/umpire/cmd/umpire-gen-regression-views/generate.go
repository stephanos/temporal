package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/common/artifactio"
)

const inspectorExecutable = "temporal-model-inspect"

type generationConfig struct {
	RepositoryRoot string
	OutputRoot     string
}

type inspectorOutput struct {
	Stdout []byte
	Stderr []byte
}

type generationDependencies struct {
	Inspect  func(modelRoot, identity string) (inspectorOutput, error)
	ReadFile func(string) ([]byte, error)
	Render   func([]generatedViewRecord) (map[string][]byte, error)
	Publish  func(artifactio.Set, string, map[string][]byte, func(string) error) error
}

func Run(arguments []string) error {
	configuration, err := parseGenerationConfig(arguments)
	if err != nil {
		return err
	}
	return runGeneration(configuration, productionManifest(), defaultGenerationDependencies())
}

func parseGenerationConfig(arguments []string) (generationConfig, error) {
	configuration := generationConfig{
		RepositoryRoot: ".",
		OutputRoot:     ".",
	}
	flags := flag.NewFlagSet("umpire-gen-regression-views", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(
		&configuration.RepositoryRoot,
		"repository-root",
		configuration.RepositoryRoot,
		"repository root containing the model and canonical fixture",
	)
	flags.StringVar(
		&configuration.OutputRoot,
		"output-root",
		configuration.OutputRoot,
		"repository-shaped root receiving generated views",
	)
	if err := flags.Parse(arguments); err != nil {
		return generationConfig{}, fmt.Errorf("parse regression generated view generation arguments: %w", err)
	}
	if flags.NArg() != 0 {
		return generationConfig{}, errors.New("regression generated view generation accepts no positional arguments")
	}
	if strings.TrimSpace(configuration.RepositoryRoot) == "" {
		return generationConfig{}, errors.New("repository root is required")
	}
	if strings.TrimSpace(configuration.OutputRoot) == "" {
		return generationConfig{}, errors.New("output root is required")
	}
	return configuration, nil
}

func defaultGenerationDependencies() generationDependencies {
	return generationDependencies{
		Inspect:  inspectExperiment,
		ReadFile: os.ReadFile,
		Render:   renderGeneratedViews,
		Publish: func(
			set artifactio.Set,
			root string,
			artifacts map[string][]byte,
			validate func(string) error,
		) error {
			return set.Publish(root, artifacts, validate)
		},
	}
}

func runGeneration(
	configuration generationConfig,
	entries []manifestEntry,
	dependencies generationDependencies,
) error {
	if err := validateGenerationDependencies(dependencies); err != nil {
		return err
	}
	if err := validateManifest(entries); err != nil {
		return fmt.Errorf("validate regression generated view manifest: %w", err)
	}
	repositoryRoot, err := resolveRepositoryRoot(configuration.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve regression generated view repository root: %w", err)
	}
	modelRoot := filepath.Join(repositoryRoot, "model")
	if _, _, err := resolveModelRoot(modelRoot); err != nil {
		return fmt.Errorf("resolve regression generated view model root: %w", err)
	}

	records := make([]generatedViewRecord, 0, len(entries))
	for _, entry := range entries {
		inspected, inspectErr := dependencies.Inspect(modelRoot, entry.Identity)
		encoded, err := requireInspectorArtifact(entry.Identity, inspected, inspectErr)
		if err != nil {
			return err
		}
		live, err := extractGeneratedView(entry, encoded, modelRoot)
		if err != nil {
			return fmt.Errorf("inspect regression generated view %q: %w", entry.Identity, err)
		}

		fixturePath, err := resolveFixturePath(repositoryRoot, entry.FixturePath)
		if err != nil {
			return fmt.Errorf("read regression generated view fixture for %q: %w", entry.Identity, err)
		}
		fixtureBytes, err := dependencies.ReadFile(fixturePath)
		if err != nil {
			return fmt.Errorf(
				"read regression generated view fixture %q for %q: %w",
				entry.FixturePath,
				entry.Identity,
				err,
			)
		}
		fixture, err := extractGeneratedView(entry, fixtureBytes, modelRoot)
		if err != nil {
			return fmt.Errorf("validate regression generated view fixture for %q: %w", entry.Identity, err)
		}
		if err := compareGeneratedViewRecords(live, fixture); err != nil {
			return fmt.Errorf("cross-check regression generated view fixture for %q: %w", entry.Identity, err)
		}
		records = append(records, live)
	}

	artifacts, err := dependencies.Render(records)
	if err != nil {
		return fmt.Errorf("render regression generated views: %w", err)
	}
	if err := validateGeneratedArtifacts(entries, records, artifacts); err != nil {
		return fmt.Errorf("validate rendered regression generated views: %w", err)
	}
	outputRoot, err := filepath.Abs(configuration.OutputRoot)
	if err != nil {
		return fmt.Errorf("resolve regression generated view output root: %w", err)
	}
	paths := managedArtifactPaths(entries)
	set := artifactio.Set{Roots: slices.Clone(paths), Paths: slices.Clone(paths)}
	validateCandidate := func(candidateRoot string) error {
		candidate := make(map[string][]byte, len(paths))
		for _, relative := range paths {
			encoded, err := os.ReadFile(filepath.Join(candidateRoot, filepath.FromSlash(relative)))
			if err != nil {
				return fmt.Errorf("read staged artifact %q: %w", relative, err)
			}
			candidate[relative] = encoded
		}
		return validateGeneratedArtifacts(entries, records, candidate)
	}
	if err := dependencies.Publish(set, outputRoot, artifacts, validateCandidate); err != nil {
		return fmt.Errorf("publish regression generated views: %w", err)
	}
	return nil
}

func validateGenerationDependencies(dependencies generationDependencies) error {
	switch {
	case dependencies.Inspect == nil:
		return errors.New("regression generated view inspector is required")
	case dependencies.ReadFile == nil:
		return errors.New("regression generated view fixture reader is required")
	case dependencies.Render == nil:
		return errors.New("regression generated view renderer is required")
	case dependencies.Publish == nil:
		return errors.New("regression generated view publisher is required")
	default:
		return nil
	}
}

func inspectExperiment(modelRoot, identity string) (inspectorOutput, error) {
	command := exec.Command("lake", "exe", inspectorExecutable, identity)
	command.Dir = modelRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return inspectorOutput{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
}

func requireInspectorArtifact(identity string, output inspectorOutput, inspectErr error) ([]byte, error) {
	stdoutPresent := len(bytes.TrimSpace(output.Stdout)) != 0
	stderr := bytes.TrimSpace(output.Stderr)
	if inspectErr != nil {
		if stdoutPresent {
			return nil, fmt.Errorf(
				"inspect regression generated view %q: inspector failed while also producing stdout; diagnostic: %s: %w",
				identity,
				diagnosticSummary(stderr),
				inspectErr,
			)
		}
		if len(stderr) != 0 {
			return nil, fmt.Errorf(
				"inspect regression generated view %q: inspector diagnostic: %s: %w",
				identity,
				diagnosticSummary(stderr),
				inspectErr,
			)
		}
		return nil, fmt.Errorf("inspect regression generated view %q: inspector failed: %w", identity, inspectErr)
	}
	if !stdoutPresent {
		if len(stderr) != 0 {
			return nil, fmt.Errorf(
				"inspect regression generated view %q: inspector succeeded without an artifact and wrote diagnostic: %s",
				identity,
				diagnosticSummary(stderr),
			)
		}
		return nil, fmt.Errorf("inspect regression generated view %q: inspector produced an empty artifact", identity)
	}
	if len(stderr) != 0 {
		return nil, fmt.Errorf(
			"inspect regression generated view %q: inspector succeeded with contradictory stderr: %s",
			identity,
			diagnosticSummary(stderr),
		)
	}
	return output.Stdout, nil
}

func diagnosticSummary(diagnostic []byte) string {
	if len(diagnostic) == 0 {
		return "(none)"
	}
	return fmt.Sprintf("stderr present (%d bytes)", len(diagnostic))
}

func resolveRepositoryRoot(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", root)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func resolveFixturePath(repositoryRoot, relative string) (string, error) {
	if err := validateRepositoryPath(relative); err != nil {
		return "", err
	}
	target := filepath.Join(repositoryRoot, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", err
	}
	contained, err := pathIsWithin(repositoryRoot, resolved)
	if err != nil {
		return "", err
	}
	if !contained {
		return "", fmt.Errorf("fixture %q escapes the repository root", relative)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("fixture %q is not a regular file", relative)
	}
	return resolved, nil
}

func pathIsWithin(root, target string) (bool, error) {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false, err
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)), nil
}

func compareGeneratedViewRecords(inspected, fixture generatedViewRecord) error {
	switch {
	case inspected.Format != fixture.Format:
		return errors.New("fixture format differs from inspector output")
	case inspected.Identity != fixture.Identity:
		return errors.New("fixture query identity differs from inspector output")
	case !slices.Equal(inspected.Sources, fixture.Sources):
		return errors.New("fixture canonical sources differ from inspector output")
	case !slices.Equal(inspected.Properties, fixture.Properties):
		return errors.New("fixture property identities differ from inspector output")
	case !slices.Equal(inspected.ObservationRequirements, fixture.ObservationRequirements):
		return errors.New("fixture observation-requirement identities differ from inspector output")
	case inspected.ArtifactChecksum != fixture.ArtifactChecksum:
		return errors.New("fixture artifact checksum differs from inspector output")
	default:
		return nil
	}
}

func managedArtifactPaths(entries []manifestEntry) []string {
	paths := make([]string, 0, len(entries)*2)
	for _, entry := range entries {
		paths = append(paths, entry.GoOutputPath, entry.MarkdownOutputPath)
	}
	slices.Sort(paths)
	return paths
}

func validateGeneratedArtifacts(
	entries []manifestEntry,
	records []generatedViewRecord,
	artifacts map[string][]byte,
) error {
	paths := managedArtifactPaths(entries)
	if len(artifacts) != len(paths) {
		return errors.New("generated artifact map must contain exactly the managed generated-view paths")
	}
	for _, relative := range paths {
		if _, exists := artifacts[relative]; !exists {
			return fmt.Errorf(
				"generated artifact map must contain exactly the managed generated-view paths: missing %q",
				relative,
			)
		}
	}
	for _, record := range records {
		goSource := artifacts[record.GoOutputPath]
		formatted, err := format.Source(goSource)
		if err != nil {
			return fmt.Errorf("format generated Go generated view %q: %w", record.Identity, err)
		}
		if !bytes.Equal(goSource, formatted) {
			return fmt.Errorf("generated Go generated view %q is not gofmt-normalized", record.Identity)
		}
		if err := validateRenderedPair(
			record,
			goSource,
			artifacts[record.MarkdownOutputPath],
		); err != nil {
			return err
		}
	}
	return nil
}
