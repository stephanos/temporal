package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublishCatalogRejectsInvalidCandidateBeforeMutation(t *testing.T) {
	t.Parallel()
	outputRoot := t.TempDir()
	writePublishedCatalog(t, outputRoot, "old")
	artifacts, err := renderCatalog(renderFixtureCatalog())
	require.NoError(t, err)
	validatorCalled := false

	err = publishCatalog(outputRoot, artifacts, func(string) error {
		validatorCalled = true
		return errors.New("invalid Lean")
	})
	require.ErrorContains(t, err, "invalid Lean")
	require.True(t, validatorCalled)
	requirePublishedCatalog(t, outputRoot, "old")
}

func TestPublishCatalogOwnsOnlyTheFacadeAndChildDirectory(t *testing.T) {
	t.Parallel()
	outputRoot := t.TempDir()
	writePublishedCatalog(t, outputRoot, "old")
	authored := filepath.Join(outputRoot, "Temporal", "Authored.lean")
	require.NoError(t, os.WriteFile(authored, []byte("authored"), 0o600))
	artifacts := generatedFixtureArtifacts("new")

	require.NoError(t, publishCatalog(outputRoot, artifacts, nil))
	requirePublishedCatalog(t, outputRoot, "new")
	authoredBytes, err := os.ReadFile(authored)
	require.NoError(t, err)
	require.Equal(t, []byte("authored"), authoredBytes)
}

func TestPublishCatalogRejectsUnexpectedArtifactSet(t *testing.T) {
	t.Parallel()
	outputRoot := t.TempDir()
	writePublishedCatalog(t, outputRoot, "old")
	artifacts := generatedFixtureArtifacts("new")
	artifacts["Temporal/DynamicConfig/Unexpected.lean"] = []byte("unexpected")

	err := publishCatalog(outputRoot, artifacts, nil)
	require.ErrorContains(t, err, "exactly the managed paths")
	requirePublishedCatalog(t, outputRoot, "old")
}

func generatedFixtureArtifacts(value string) map[string][]byte {
	return map[string][]byte{
		dynamicConfigFacadePath:   []byte("facade-" + value),
		dynamicConfigTypesPath:    []byte("types-" + value),
		dynamicConfigSettingsPath: []byte("settings-" + value),
	}
}

func writePublishedCatalog(t *testing.T, outputRoot string, value string) {
	t.Helper()
	for path, encoded := range generatedFixtureArtifacts(value) {
		absolute := filepath.Join(outputRoot, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(absolute), 0o700))
		require.NoError(t, os.WriteFile(absolute, encoded, 0o600))
	}
	stale := filepath.Join(outputRoot, "Temporal", "DynamicConfig", "Stale.lean")
	require.NoError(t, os.WriteFile(stale, []byte("stale"), 0o600))
}

func requirePublishedCatalog(t *testing.T, outputRoot string, value string) {
	t.Helper()
	for path, want := range generatedFixtureArtifacts(value) {
		encoded, err := os.ReadFile(filepath.Join(outputRoot, filepath.FromSlash(path)))
		require.NoError(t, err)
		require.Equal(t, want, encoded)
	}
	_, err := os.Stat(filepath.Join(outputRoot, "Temporal", "DynamicConfig", "Stale.lean"))
	if value == "old" {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, os.ErrNotExist)
	}
}
