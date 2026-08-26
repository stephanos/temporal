package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/common/artifactio"
)

func TestRunGenerationInspectsOnceAndPublishesTheCompleteSetDeterministically(t *testing.T) {
	configuration, entry, encoded, dependencies := newGenerationFixture(t)
	var inspections atomic.Int32
	dependencies.Inspect = func(modelRoot, identity string) (inspectorOutput, error) {
		inspections.Add(1)
		require.Equal(t, filepath.Join(configuration.RepositoryRoot, "model"), modelRoot)
		require.Equal(t, entry.Identity, identity)
		return inspectorOutput{Stdout: slices.Clone(encoded)}, nil
	}
	var publications atomic.Int32
	productionPublish := dependencies.Publish
	dependencies.Publish = func(
		set artifactio.Set,
		root string,
		artifacts map[string][]byte,
		validate func(string) error,
	) error {
		publications.Add(1)
		expected := managedArtifactPaths([]manifestEntry{entry})
		require.Equal(t, expected, set.Roots)
		require.Equal(t, expected, set.Paths)
		require.Len(t, artifacts, 2)
		return productionPublish(set, root, artifacts, validate)
	}

	require.NoError(t, runGeneration(configuration, []manifestEntry{entry}, dependencies))
	first := readGeneratedSet(t, configuration.OutputRoot, entry)
	require.EqualValues(t, 1, inspections.Load())
	require.EqualValues(t, 1, publications.Load())

	require.NoError(t, runGeneration(configuration, []manifestEntry{entry}, dependencies))
	require.Equal(t, first, readGeneratedSet(t, configuration.OutputRoot, entry))
	require.EqualValues(t, 2, inspections.Load())
	require.EqualValues(t, 2, publications.Load())
}

func TestRunGenerationRejectsEveryStaleDisplayedFixtureFieldBeforePublication(t *testing.T) {
	tests := map[string]func(*testing.T, string, *experimentEnvelope){
		"format": func(_ *testing.T, _ string, fixture *experimentEnvelope) {
			fixture.FormatVersion = "umpire-experiment/unsupported"
		},
		"identity": func(_ *testing.T, _ string, fixture *experimentEnvelope) {
			fixture.Plan.QueryIdentity = "query.changed"
		},
		"canonical sources": func(t *testing.T, repositoryRoot string, fixture *experimentEnvelope) {
			writeLeanSource(t, filepath.Join(repositoryRoot, "model"), "Changed.lean")
			fixture.Provenance.Sources[0].Path = "Changed.lean"
		},
		"property identities": func(_ *testing.T, _ string, fixture *experimentEnvelope) {
			fixture.Properties[0].Identity = "property.changed"
		},
		"observation-requirement identities": func(_ *testing.T, _ string, fixture *experimentEnvelope) {
			fixture.ObservationRequirements[0] = "observation.changed"
		},
		"semantic fingerprint": func(_ *testing.T, _ string, fixture *experimentEnvelope) {
			fixture.SemanticIdentity = "semantic-identity-changed"
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			configuration, entry, inspected, dependencies := newGenerationFixture(t)
			fixture := decodeGenerationFixture(t, inspected)
			mutate(t, configuration.RepositoryRoot, &fixture)
			writeGenerationFixture(t, configuration.RepositoryRoot, entry.FixturePath, fixture)
			dependencies.Inspect = staticInspector(inspected)
			published := false
			dependencies.Publish = func(
				artifactio.Set,
				string,
				map[string][]byte,
				func(string) error,
			) error {
				published = true
				return nil
			}

			err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
			require.Error(t, err)
			require.Contains(t, err.Error(), entry.Identity)
			require.False(t, published)
		})
	}
}

func TestRunGenerationRejectsInspectorFailuresAndOutputContradictions(t *testing.T) {
	tests := map[string]struct {
		output inspectorOutput
		err    error
		want   string
	}{
		"process failure": {
			err:  errors.New("exit status 1"),
			want: "inspector failed",
		},
		"structured diagnostic": {
			output: inspectorOutput{Stderr: []byte(`{"kind":"unknown-scenario","subject":"fixed"}`)},
			err:    errors.New("exit status 1"),
			want:   "stderr present",
		},
		"failed with stdout": {
			output: inspectorOutput{Stdout: []byte("large-semantic-artifact"), Stderr: []byte("diagnostic")},
			err:    errors.New("exit status 1"),
			want:   "failed while also producing stdout",
		},
		"empty success": {
			output: inspectorOutput{},
			want:   "empty artifact",
		},
		"empty success with stderr": {
			output: inspectorOutput{Stderr: []byte("diagnostic")},
			want:   "succeeded without an artifact",
		},
		"artifact with stderr": {
			output: inspectorOutput{Stdout: []byte(`{"artifact":true}`), Stderr: []byte("diagnostic")},
			want:   "contradictory stderr",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			configuration, entry, _, dependencies := newGenerationFixture(t)
			var inspections atomic.Int32
			dependencies.Inspect = func(string, string) (inspectorOutput, error) {
				inspections.Add(1)
				return test.output, test.err
			}
			published := false
			dependencies.Publish = func(
				artifactio.Set,
				string,
				map[string][]byte,
				func(string) error,
			) error {
				published = true
				return nil
			}

			err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
			require.ErrorContains(t, err, test.want)
			require.Contains(t, err.Error(), entry.Identity)
			require.NotContains(t, err.Error(), "large-semantic-artifact")
			require.EqualValues(t, 1, inspections.Load())
			require.False(t, published)
		})
	}
}

func TestRequireInspectorArtifactDoesNotExposeStderr(t *testing.T) {
	semanticArtifact := []byte(`{"format":"umpire-experiment/v1","semanticIdentity":"do-not-expose-semantic-identity"}`)
	tests := map[string]struct {
		output     inspectorOutput
		inspectErr error
	}{
		"failed with stdout": {
			output:     inspectorOutput{Stdout: []byte(`{"artifact":true}`), Stderr: semanticArtifact},
			inspectErr: errors.New("exit status 1"),
		},
		"failed without stdout": {
			output:     inspectorOutput{Stderr: semanticArtifact},
			inspectErr: errors.New("exit status 1"),
		},
		"succeeded without stdout": {
			output: inspectorOutput{Stderr: semanticArtifact},
		},
		"succeeded with stdout": {
			output: inspectorOutput{Stdout: []byte(`{"artifact":true}`), Stderr: semanticArtifact},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := requireInspectorArtifact("stable-identity", test.output, test.inspectErr)

			require.Error(t, err)
			require.Contains(t, err.Error(), "stable-identity")
			require.NotContains(t, err.Error(), string(semanticArtifact))
			require.NotContains(t, err.Error(), "do-not-expose-semantic-identity")
		})
	}
}

func TestRunGenerationRejectsInvalidInspectorAndFixtureArtifactsBeforePublication(t *testing.T) {
	inspectorCases := map[string][]byte{
		"malformed JSON": []byte("{"),
		"unsupported format": syntheticExperiment(t, syntheticOptions{
			format: "umpire-experiment/unsupported",
		}),
		"query identity mismatch": syntheticExperiment(t, syntheticOptions{
			identity: "query.changed",
		}),
		"missing provenance": syntheticExperiment(t, syntheticOptions{
			sources: []string{},
		}),
		"nonexistent provenance": syntheticExperiment(t, syntheticOptions{
			sources: []string{"Missing.lean"},
		}),
	}
	for name, inspected := range inspectorCases {
		t.Run("inspector "+name, func(t *testing.T) {
			configuration, entry, _, dependencies := newGenerationFixture(t)
			dependencies.Inspect = staticInspector(inspected)
			published := false
			dependencies.Publish = recordingPublisher(&published)

			err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
			require.Error(t, err)
			require.Contains(t, err.Error(), entry.Identity)
			require.False(t, published)
		})
	}

	fixtureCases := map[string][]byte{
		"missing":   nil,
		"malformed": []byte("{"),
	}
	for name, fixture := range fixtureCases {
		t.Run("fixture "+name, func(t *testing.T) {
			configuration, entry, inspected, dependencies := newGenerationFixture(t)
			dependencies.Inspect = staticInspector(inspected)
			if fixture == nil {
				require.NoError(t, os.Remove(filepath.Join(configuration.RepositoryRoot, filepath.FromSlash(entry.FixturePath))))
			} else {
				require.NoError(t, os.WriteFile(
					filepath.Join(configuration.RepositoryRoot, filepath.FromSlash(entry.FixturePath)),
					fixture,
					0o600,
				))
			}
			published := false
			dependencies.Publish = recordingPublisher(&published)

			err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
			require.Error(t, err)
			require.Contains(t, err.Error(), entry.Identity)
			require.False(t, published)
		})
	}
}

func TestRunGenerationValidatesRenderedCompleteSetBeforePublication(t *testing.T) {
	tests := map[string]func([]projectionRecord) (map[string][]byte, error){
		"renderer error": func([]projectionRecord) (map[string][]byte, error) {
			return nil, errors.New("injected render failure")
		},
		"incomplete map": func(records []projectionRecord) (map[string][]byte, error) {
			artifacts, err := renderProjections(records)
			if err == nil {
				delete(artifacts, records[0].MarkdownOutputPath)
			}
			return artifacts, err
		},
		"unformatted Go": func(records []projectionRecord) (map[string][]byte, error) {
			artifacts, err := renderProjections(records)
			if err == nil {
				artifacts[records[0].GoOutputPath] = []byte("package regression\nfunc broken( {\n")
			}
			return artifacts, err
		},
		"inconsistent Markdown": func(records []projectionRecord) (map[string][]byte, error) {
			artifacts, err := renderProjections(records)
			if err == nil {
				artifacts[records[0].MarkdownOutputPath] = []byte("stale\n")
			}
			return artifacts, err
		},
	}

	for name, render := range tests {
		t.Run(name, func(t *testing.T) {
			configuration, entry, inspected, dependencies := newGenerationFixture(t)
			dependencies.Inspect = staticInspector(inspected)
			dependencies.Render = render
			published := false
			dependencies.Publish = recordingPublisher(&published)

			err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
			require.Error(t, err)
			require.False(t, published)
		})
	}
}

func TestRunGenerationPreservesPriorCompleteSetOnInjectedPublicationFailure(t *testing.T) {
	configuration, entry, inspected, dependencies := newGenerationFixture(t)
	dependencies.Inspect = staticInspector(inspected)
	old := map[string][]byte{
		entry.GoOutputPath:       []byte("old Go projection\n"),
		entry.MarkdownOutputPath: []byte("old Markdown projection\n"),
	}
	writeGeneratedSet(t, configuration.OutputRoot, old)
	dependencies.Publish = func(
		set artifactio.Set,
		root string,
		artifacts map[string][]byte,
		_ func(string) error,
	) error {
		return set.Publish(root, artifacts, func(string) error {
			return errors.New("injected candidate failure")
		})
	}

	err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
	require.ErrorContains(t, err, "injected candidate failure")
	require.Equal(t, old, readGeneratedSet(t, configuration.OutputRoot, entry))
}

func TestRunGenerationRejectsUnsafeRootsAndPaths(t *testing.T) {
	t.Run("missing repository root", func(t *testing.T) {
		configuration, entry, inspected, dependencies := newGenerationFixture(t)
		configuration.RepositoryRoot = filepath.Join(t.TempDir(), "missing")
		dependencies.Inspect = staticInspector(inspected)
		err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
		require.ErrorContains(t, err, "repository root")
	})

	t.Run("missing output root", func(t *testing.T) {
		configuration, entry, inspected, dependencies := newGenerationFixture(t)
		configuration.OutputRoot = filepath.Join(t.TempDir(), "missing")
		dependencies.Inspect = staticInspector(inspected)
		err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
		require.ErrorContains(t, err, "artifact set root")
	})

	t.Run("unwritable output root", func(t *testing.T) {
		configuration, entry, inspected, dependencies := newGenerationFixture(t)
		dependencies.Inspect = staticInspector(inspected)
		dependencies.Publish = func(
			artifactio.Set,
			string,
			map[string][]byte,
			func(string) error,
		) error {
			return fs.ErrPermission
		}
		err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
		require.ErrorIs(t, err, fs.ErrPermission)
		require.ErrorContains(t, err, "publish regression projections")
	})

	t.Run("fixture traversal", func(t *testing.T) {
		configuration, entry, inspected, dependencies := newGenerationFixture(t)
		entry.FixturePath = "../fixture.json"
		dependencies.Inspect = staticInspector(inspected)
		err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
		require.ErrorContains(t, err, "unsafe")
	})

	t.Run("fixture symlink escape", func(t *testing.T) {
		configuration, entry, inspected, dependencies := newGenerationFixture(t)
		external := filepath.Join(t.TempDir(), "fixture.json")
		require.NoError(t, os.WriteFile(external, inspected, 0o600))
		fixture := filepath.Join(configuration.RepositoryRoot, filepath.FromSlash(entry.FixturePath))
		require.NoError(t, os.Remove(fixture))
		require.NoError(t, os.Symlink(external, fixture))
		dependencies.Inspect = staticInspector(inspected)
		err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
		require.ErrorContains(t, err, "escapes the repository root")
	})

	t.Run("managed output symlink", func(t *testing.T) {
		configuration, entry, inspected, dependencies := newGenerationFixture(t)
		external := filepath.Join(t.TempDir(), "external.go")
		require.NoError(t, os.WriteFile(external, []byte("external\n"), 0o600))
		managed := filepath.Join(configuration.OutputRoot, filepath.FromSlash(entry.GoOutputPath))
		require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o700))
		require.NoError(t, os.Symlink(external, managed))
		dependencies.Inspect = staticInspector(inspected)
		err := runGeneration(configuration, []manifestEntry{entry}, dependencies)
		require.ErrorContains(t, err, "symlink")
		externalBytes, readErr := os.ReadFile(external)
		require.NoError(t, readErr)
		require.Equal(t, []byte("external\n"), externalBytes)
	})
}

func TestRunGenerationRejectsConcurrentPublisher(t *testing.T) {
	configuration, entry, inspected, dependencies := newGenerationFixture(t)
	dependencies.Inspect = staticInspector(inspected)
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDependencies := dependencies
	firstDependencies.Publish = func(
		set artifactio.Set,
		root string,
		artifacts map[string][]byte,
		validate func(string) error,
	) error {
		return set.Publish(root, artifacts, func(candidateRoot string) error {
			close(entered)
			<-release
			return validate(candidateRoot)
		})
	}
	firstResult := make(chan error, 1)
	go func() {
		firstResult <- runGeneration(configuration, []manifestEntry{entry}, firstDependencies)
	}()
	<-entered

	secondErr := runGeneration(configuration, []manifestEntry{entry}, dependencies)
	require.ErrorContains(t, secondErr, "concurrent writer")
	close(release)
	require.NoError(t, <-firstResult)
}

func TestGenerationArgumentsExposeOnlyRepositoryAndOutputRoots(t *testing.T) {
	configuration, err := parseGenerationConfig([]string{
		"--repository-root", "source",
		"--output-root", "output",
	})
	require.NoError(t, err)
	require.Equal(t, generationConfig{RepositoryRoot: "source", OutputRoot: "output"}, configuration)

	for _, arguments := range [][]string{
		{"--scenario", "query"},
		{"--runtime", "cluster"},
		{"--exploration", "candidate"},
		{"query.identity"},
		{"--repository-root", ""},
		{"--output-root", ""},
	} {
		_, err := parseGenerationConfig(arguments)
		require.Error(t, err, arguments)
	}
}

func newGenerationFixture(
	t *testing.T,
) (generationConfig, manifestEntry, []byte, generationDependencies) {
	t.Helper()
	repositoryRoot := t.TempDir()
	outputRoot := t.TempDir()
	modelRoot := filepath.Join(repositoryRoot, "model")
	writeLeanSource(t, modelRoot, "One.lean")
	entry := syntheticEntry(callerClosureIdentity)
	encoded := syntheticExperiment(t, syntheticOptions{})
	fixture := decodeGenerationFixture(t, encoded)
	writeGenerationFixture(t, repositoryRoot, entry.FixturePath, fixture)
	return generationConfig{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     outputRoot,
	}, entry, encoded, defaultGenerationDependencies()
}

func staticInspector(encoded []byte) func(string, string) (inspectorOutput, error) {
	return func(string, string) (inspectorOutput, error) {
		return inspectorOutput{Stdout: slices.Clone(encoded)}, nil
	}
}

func recordingPublisher(published *bool) func(
	artifactio.Set,
	string,
	map[string][]byte,
	func(string) error,
) error {
	return func(artifactio.Set, string, map[string][]byte, func(string) error) error {
		*published = true
		return nil
	}
}

func decodeGenerationFixture(t *testing.T, encoded []byte) experimentEnvelope {
	t.Helper()
	fixture, err := decodeExperiment(encoded)
	require.NoError(t, err)
	return fixture
}

func writeGenerationFixture(
	t *testing.T,
	repositoryRoot string,
	relative string,
	fixture experimentEnvelope,
) {
	t.Helper()
	encoded, err := json.Marshal(fixture)
	require.NoError(t, err)
	target := filepath.Join(repositoryRoot, filepath.FromSlash(relative))
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o700))
	require.NoError(t, os.WriteFile(target, encoded, 0o600))
}

func readGeneratedSet(t *testing.T, root string, entry manifestEntry) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte, 2)
	for _, relative := range []string{entry.GoOutputPath, entry.MarkdownOutputPath} {
		encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		require.NoError(t, err)
		result[relative] = encoded
	}
	return result
}

func writeGeneratedSet(t *testing.T, root string, artifacts map[string][]byte) {
	t.Helper()
	for relative, encoded := range artifacts {
		target := filepath.Join(root, filepath.FromSlash(relative))
		require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o700))
		require.NoError(t, os.WriteFile(target, encoded, 0o600))
	}
}
