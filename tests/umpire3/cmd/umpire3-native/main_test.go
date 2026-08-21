package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/model-checkers/native"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestBindAndProduceUseTheSelectedFirstOrderView(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	view, found, err := protocol.DefaultFirstOrderView(protocol.TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)
	encoded, err := view.CanonicalJSON()
	require.NoError(t, err)
	input := filepath.Join(directory, "view.json")
	require.NoError(t, os.WriteFile(input, encoded, 0o600))
	binding := filepath.Join(directory, "Generated", "Binding.lean")
	err = run([]string{"-operation", "bind", "-input", input, "-output", binding})
	require.NoError(t, err)
	expectedBinding, err := native.BindingSource(view)
	require.NoError(t, err)
	actualBinding, err := os.ReadFile(binding)
	require.NoError(t, err)
	require.Equal(t, expectedBinding, actualBinding)

	certificatePath := filepath.Join(directory, "result", "certificate.json")
	err = run([]string{
		"-operation", "produce", "-input", input, "-output", certificatePath,
		"-workers", "4", "-replicas", "10",
	})
	require.NoError(t, err)
	certificate, err := readCertificate(certificatePath, view)
	require.NoError(t, err)
	require.Equal(t, 260, certificate.Statistics.ExpandedStates)
}
