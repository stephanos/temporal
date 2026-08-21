package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/model-checkers/tla"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestGenerateWritesReadableTLAAndConfig(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	view, found, err := protocol.DefaultTemporalView("sound")
	require.NoError(t, err)
	require.True(t, found)
	encoded, err := view.CanonicalJSON()
	require.NoError(t, err)
	input := filepath.Join(directory, "view.json")
	require.NoError(t, os.WriteFile(input, encoded, 0o600))
	tlaOutput := filepath.Join(directory, "generated", "model.tla")
	configOutput := filepath.Join(directory, "generated", "model.cfg")

	err = run([]string{
		"-operation", "generate", "-input", input,
		"-output", tlaOutput, "-config-output", configOutput,
	})
	require.NoError(t, err)
	generated, err := tla.Generate(view)
	require.NoError(t, err)
	actualTLA, err := os.ReadFile(tlaOutput)
	require.NoError(t, err)
	actualConfig, err := os.ReadFile(configOutput)
	require.NoError(t, err)
	require.Equal(t, generated.TLA, actualTLA)
	require.Equal(t, generated.Config, actualConfig)
}

func TestCheckRequiresKnownBackend(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	view, found, err := protocol.DefaultTemporalView("sound")
	require.NoError(t, err)
	require.True(t, found)
	encoded, err := view.CanonicalJSON()
	require.NoError(t, err)
	input := filepath.Join(directory, "view.json")
	require.NoError(t, os.WriteFile(input, encoded, 0o600))

	err = run([]string{
		"-operation", "check", "-input", input,
		"-output", filepath.Join(directory, "result.json"), "-backend", "unknown",
	})
	require.ErrorContains(t, err, "unknown temporal backend")
}
