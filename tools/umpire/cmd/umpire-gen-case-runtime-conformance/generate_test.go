package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/common/artifactio"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRunGenerationPublishesExactlySixCompleteClasses(t *testing.T) {
	configuration := generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}
	entries := productionManifest()
	rendered := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&umpirespb.Case{CaseId: entry.CaseID})
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
			require.Equal(t, []string{fixtureRoot}, set.Roots)
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
