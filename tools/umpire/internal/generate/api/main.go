package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire/internal/artifactio"
)

type options struct {
	Mode                    string
	RepositoryRoot          string
	PublicModule            string
	PublicDescriptor        string
	APIDependencyDescriptor string
	InternalDescriptor      string
	CHASMDescriptor         string
	OutputRoot              string
}

func Run(ctx context.Context, arguments []string, stdout io.Writer) error {
	if len(arguments) == 0 {
		return errors.New("operation is required: generate, check, or inspect")
	}
	flags := flag.NewFlagSet("umpire-gen-api "+arguments[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configuration := options{Mode: arguments[0]}
	flags.StringVar(&configuration.RepositoryRoot, "repository-root", ".", "Temporal repository root")
	flags.StringVar(&configuration.PublicModule, "public-module", "go.temporal.io/api", "public Temporal API Go module")
	flags.StringVar(&configuration.PublicDescriptor, "public-descriptor", "", "prebuilt complete public descriptor set")
	flags.StringVar(&configuration.APIDependencyDescriptor, "api-dependencies", "proto/api.binpb", "public and third-party dependency descriptor set")
	flags.StringVar(&configuration.InternalDescriptor, "internal-descriptor", "proto/image.bin", "internal server descriptor set")
	flags.StringVar(&configuration.CHASMDescriptor, "chasm-descriptor", "proto/chasm.bin", "CHASM descriptor set")
	flags.StringVar(&configuration.OutputRoot, "output-root", "model", "Lean model source root")
	if err := flags.Parse(arguments[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	switch configuration.Mode {
	case "generate", "check", "inspect":
	default:
		return fmt.Errorf("unknown operation %q", configuration.Mode)
	}
	return run(ctx, configuration, stdout)
}

func run(ctx context.Context, configuration options, stdout io.Writer) error {
	repositoryRoot, err := filepath.Abs(configuration.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	var public descriptorInput
	var publicVersion string
	if configuration.PublicDescriptor == "" {
		public, publicVersion, err = exportPublicDescriptors(ctx, repositoryRoot, configuration.PublicModule)
	} else {
		public, err = descriptorFileInput("public", resolvePath(repositoryRoot, configuration.PublicDescriptor))
		if err == nil {
			public.Locator = filepath.ToSlash(configuration.PublicDescriptor)
			version, versionErr := commandOutput(ctx, repositoryRoot, "go", "list", "-m", "-f", "{{.Version}}", configuration.PublicModule)
			if versionErr != nil {
				return fmt.Errorf("resolve public API module version: %w", versionErr)
			}
			publicVersion = strings.TrimSpace(version)
		}
	}
	if err != nil {
		return err
	}
	inputs := []descriptorInput{public}
	for _, input := range []struct {
		name string
		path string
	}{
		{name: "api-dependencies", path: configuration.APIDependencyDescriptor},
		{name: "internal", path: configuration.InternalDescriptor},
		{name: "chasm", path: configuration.CHASMDescriptor},
	} {
		if input.path == "" {
			continue
		}
		descriptor, loadErr := descriptorFileInput(input.name, resolvePath(repositoryRoot, input.path))
		if loadErr != nil {
			return loadErr
		}
		descriptor.Locator = filepath.ToSlash(input.path)
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
	artifacts, manifest, err := generateArtifacts(configuration.PublicModule, publicVersion, inputs, projection)
	if err != nil {
		return err
	}
	if configuration.Mode == "inspect" {
		return writeInspect(stdout, manifest)
	}
	outputRoot := resolvePath(repositoryRoot, configuration.OutputRoot)
	if configuration.Mode == "check" {
		return checkArtifacts(outputRoot, artifacts)
	}
	return publishArtifacts(outputRoot, artifacts, manifest)
}

func resolvePath(repositoryRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repositoryRoot, path)
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

func checkArtifacts(outputRoot string, artifacts map[string][]byte) error {
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
	if len(drift) != 0 {
		return fmt.Errorf("generated Temporal API model is stale; run make umpire-gen-api: %s", strings.Join(drift, ", "))
	}
	return nil
}

func publishArtifacts(outputRoot string, artifacts map[string][]byte, manifest generationManifest) error {
	previous, err := loadPreviousManifest(filepath.Join(outputRoot, filepath.FromSlash(manifestPath)))
	if err != nil {
		return err
	}
	for _, path := range sortedArtifactPaths(artifacts) {
		if path == manifestPath {
			continue
		}
		if err := validateManagedPath(path); err != nil {
			return err
		}
		if err := artifactio.Publish(filepath.Join(outputRoot, filepath.FromSlash(path)), artifacts[path]); err != nil {
			return fmt.Errorf("publish generated artifact %q: %w", path, err)
		}
	}
	current := make(map[string]bool, len(manifest.GeneratedFiles))
	for _, file := range manifest.GeneratedFiles {
		current[file.Path] = true
	}
	for _, file := range previous.GeneratedFiles {
		if current[file.Path] {
			continue
		}
		if err := validateManagedPath(file.Path); err != nil {
			return fmt.Errorf("refuse to remove stale artifact: %w", err)
		}
		if err := artifactio.Remove(filepath.Join(outputRoot, filepath.FromSlash(file.Path))); err != nil {
			return fmt.Errorf("remove stale generated artifact %q: %w", file.Path, err)
		}
	}
	if err := artifactio.Publish(filepath.Join(outputRoot, filepath.FromSlash(manifestPath)), artifacts[manifestPath]); err != nil {
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

func validateManagedPath(path string) error {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if clean != path || filepath.IsAbs(path) || strings.HasPrefix(path, "../") {
		return fmt.Errorf("generated artifact path %q is unsafe", path)
	}
	if path != "Temporal/Generated.lean" && !strings.HasPrefix(path, "Temporal/Generated/") {
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
