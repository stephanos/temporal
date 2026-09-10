package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/tools/common/artifactio"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRunGenerationPublishesExactlySixCompleteClasses(t *testing.T) {
	configuration := generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}
	entries := productionManifest()
	rendered := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		rendered[entry.RendererArg] = encoded
	}
	var published bool
	dependencies := generationDependencies{
		Render: func(_ string, argument string) (rendererOutput, error) {
			return rendererOutput{Stdout: slices.Clone(rendered[argument])}, nil
		},
		Publish: func(set artifactio.Set, root string, artifacts map[string][]byte, validate func(string) error) error {
			published = true
			require.Equal(t, []string{"common/testing/testpilot/testdata/case-runtime-conformance"}, set.Roots)
			require.Len(t, set.Paths, 12)
			require.Len(t, artifacts, 12)
			return set.Publish(root, artifacts, validate)
		},
	}

	require.NoError(t, runGeneration(configuration, entries, dependencies))
	require.True(t, published)
	for _, entry := range entries {
		caseBytes, err := os.ReadFile(filepath.Join(configuration.OutputRoot, filepath.FromSlash(casePath(entry.Class))))
		require.NoError(t, err)
		require.Equal(t, rendered[entry.RendererArg], caseBytes)
		expectedBytes, err := os.ReadFile(filepath.Join(configuration.OutputRoot, filepath.FromSlash(expectedPath(entry.Class))))
		require.NoError(t, err)
		canonical, err := marshalExpected(entry.Expected)
		require.NoError(t, err)
		require.Equal(t, expectedBytes, canonical)
	}
}

func TestRunFunctionalGenerationPublishesOnlyCanonicalTestpilotCases(t *testing.T) {
	configuration := generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}
	entries := functionalManifest()
	rendered := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		rendered[entry.RendererArg] = encoded
	}
	stale := filepath.Join(configuration.OutputRoot, filepath.FromSlash(functionalFixtureRoot), "stale.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(stale), 0o700))
	require.NoError(t, os.WriteFile(stale, []byte("stale"), 0o600))

	require.NoError(t, runFunctionalGeneration(configuration, entries, generationDependencies{
		Render: func(_ string, argument string) (rendererOutput, error) {
			return rendererOutput{Stdout: slices.Clone(rendered[argument])}, nil
		},
		Publish: defaultGenerationDependencies().Publish,
	}))

	_, err := os.Stat(stale)
	require.ErrorIs(t, err, os.ErrNotExist)
	files, err := os.ReadDir(filepath.Join(configuration.OutputRoot, filepath.FromSlash(functionalFixtureRoot)))
	require.NoError(t, err)
	require.Len(t, files, len(entries))
	for _, entry := range entries {
		encoded, err := os.ReadFile(filepath.Join(configuration.OutputRoot, filepath.FromSlash(functionalCasePath(entry))))
		require.NoError(t, err)
		require.Equal(t, rendered[entry.RendererArg], encoded)
	}
}

func TestParseGenerationConfigSelectsExplicitFunctionalMode(t *testing.T) {
	configuration, err := parseGenerationConfig([]string{"--mode", "functional"})
	require.NoError(t, err)
	require.Equal(t, generationModeFunctional, configuration.Mode)

	_, err = parseGenerationConfig([]string{"--mode", "unknown"})
	require.ErrorContains(t, err, "unknown generation mode")
}

func TestRunFunctionalGenerationPreservesPublishedSetWhenPublicationFails(t *testing.T) {
	configuration := generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}
	entries := functionalManifest()
	rendered := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		rendered[entry.RendererArg] = encoded
		path := filepath.Join(configuration.OutputRoot, filepath.FromSlash(functionalCasePath(entry)))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))
	}

	err := runFunctionalGeneration(configuration, entries, generationDependencies{
		Render: func(_ string, argument string) (rendererOutput, error) {
			return rendererOutput{Stdout: slices.Clone(rendered[argument])}, nil
		},
		Publish: func(artifactio.Set, string, map[string][]byte, func(string) error) error {
			return errors.New("injected publication failure")
		},
	})
	require.ErrorContains(t, err, "injected publication failure")
	for _, entry := range entries {
		encoded, readErr := os.ReadFile(filepath.Join(configuration.OutputRoot, filepath.FromSlash(functionalCasePath(entry))))
		require.NoError(t, readErr)
		require.Equal(t, []byte("original"), encoded)
	}
}

func TestRunFunctionalGenerationRejectsIncompleteOrNondeterministicRenderingBeforePublication(t *testing.T) {
	entries := functionalManifest()
	encodedByArgument := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		encodedByArgument[entry.RendererArg] = encoded
	}
	for _, test := range []struct {
		name    string
		entries []functionalEntry
		render  func(string, string) (rendererOutput, error)
	}{
		{
			name: "incomplete manifest", entries: entries[:1],
			render: func(string, string) (rendererOutput, error) { return rendererOutput{}, nil },
		},
		{
			name: "renderer failure", entries: entries,
			render: func(string, string) (rendererOutput, error) {
				return rendererOutput{Stderr: []byte("failed")}, errors.New("exit status 1")
			},
		},
		{
			name: "non-deterministic bytes", entries: entries,
			render: func() func(string, string) (rendererOutput, error) {
				calls := make(map[string]int)
				return func(_ string, argument string) (rendererOutput, error) {
					calls[argument]++
					encoded := slices.Clone(encodedByArgument[argument])
					if calls[argument]%2 == 0 {
						encoded = append(encoded, '\n')
					}
					return rendererOutput{Stdout: encoded}, nil
				}
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			published := false
			err := runFunctionalGeneration(generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}, test.entries, generationDependencies{
				Render: test.render,
				Publish: func(artifactio.Set, string, map[string][]byte, func(string) error) error {
					published = true
					return nil
				},
			})
			require.Error(t, err)
			require.False(t, published)
		})
	}
}

func TestValidateFunctionalArtifactsRejectsStaleFile(t *testing.T) {
	entries := functionalManifest()
	artifacts := make(map[string][]byte, len(entries)+1)
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		artifacts[functionalCasePath(entry)] = encoded
	}
	artifacts[filepath.ToSlash(filepath.Join(functionalFixtureRoot, "stale.json"))] = []byte("stale")

	require.ErrorContains(t, validateFunctionalArtifacts(entries, artifacts), "has 7 files, want 6")
}

func TestRunGenerationRejectsIncompleteManifestAndRendererFailureBeforePublication(t *testing.T) {
	entries := productionManifest()
	for _, test := range []struct {
		name    string
		entries []manifestEntry
		render  func(string, string) (rendererOutput, error)
	}{
		{name: "incomplete manifest", entries: entries[:5], render: func(string, string) (rendererOutput, error) { return rendererOutput{}, nil }},
		{name: "renderer failure", entries: entries, render: func(string, string) (rendererOutput, error) {
			return rendererOutput{Stderr: []byte("failed")}, errors.New("exit status 1")
		}},
		{name: "renderer contradiction", entries: entries, render: func(string, string) (rendererOutput, error) {
			return rendererOutput{Stdout: []byte("partial")}, errors.New("exit status 1")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			published := false
			err := runGeneration(generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}, test.entries, generationDependencies{
				Render: test.render,
				Publish: func(artifactio.Set, string, map[string][]byte, func(string) error) error {
					published = true
					return nil
				},
			})
			require.Error(t, err)
			require.False(t, published)
		})
	}
}

func TestRunGenerationRejectsNondeterministicRenderingBeforePublication(t *testing.T) {
	entries := productionManifest()
	encodedByArgument := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		encodedByArgument[entry.RendererArg] = encoded
	}
	calls := make(map[string]int)
	published := false
	err := runGeneration(
		generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()},
		entries,
		generationDependencies{
			Render: func(_ string, argument string) (rendererOutput, error) {
				calls[argument]++
				encoded := slices.Clone(encodedByArgument[argument])
				if calls[argument]%2 == 0 {
					encoded = append(encoded, '\n')
				}
				return rendererOutput{Stdout: encoded}, nil
			},
			Publish: func(artifactio.Set, string, map[string][]byte, func(string) error) error {
				published = true
				return nil
			},
		},
	)
	require.ErrorContains(t, err, "non-deterministic bytes")
	require.False(t, published)
}
