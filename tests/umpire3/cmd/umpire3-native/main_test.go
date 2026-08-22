package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/checker/finite"
	"go.temporal.io/server/tests/umpire3/internal/command"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestBindAndProduceUseTheSelectedFirstOrderView(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	view, found, err := finite.DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)
	encoded, err := view.CanonicalJSON()
	require.NoError(t, err)
	input := filepath.Join(directory, "view.json")
	require.NoError(t, os.WriteFile(input, encoded, 0o600))
	binding := filepath.Join(directory, "Generated", "Binding.lean")
	err = command.RunNative([]string{"-operation", "bind", "-input", input, "-output", binding})
	require.NoError(t, err)
	expectedBinding, err := finite.BindingSource(view)
	require.NoError(t, err)
	actualBinding, err := os.ReadFile(binding)
	require.NoError(t, err)
	require.Equal(t, expectedBinding, actualBinding)

	certificatePath := filepath.Join(directory, "result", "certificate.json")
	err = command.RunNative([]string{
		"-operation", "produce", "-input", input, "-output", certificatePath,
		"-workers", "4", "-replicas", "10",
	})
	require.NoError(t, err)
	certificateFile, err := os.Open(certificatePath)
	require.NoError(t, err)
	certificate, err := finite.DecodeCertificate(certificateFile, protocolexperiment.DefaultDecodeLimit, view)
	require.NoError(t, err)
	require.NoError(t, certificateFile.Close())
	require.Equal(t, 260, certificate.Statistics.ExpandedStates)
}

func TestValidateRetainedBenchmarkFailsClosedOnRecoveryDrift(t *testing.T) {
	t.Parallel()

	arguments := []string{
		"-operation", "validate-benchmark",
		"-input", "../../checker/finite/testdata/generated/nexus-cancellation.first-order.json",
		"-certificate", "../../checker/finite/testdata/generated/nexus-cancellation-scale.certificate.json",
		"-receipt", "../../checker/finite/testdata/generated/nexus-cancellation-scale.receipt.json",
		"-benchmark", "../../checker/finite/testdata/retained/nexus-cancellation-scale.benchmark.json",
	}
	require.NoError(t, command.RunNative(arguments))

	encoded, err := os.ReadFile("../../checker/finite/testdata/retained/nexus-cancellation-scale.benchmark.json")
	require.NoError(t, err)
	mutated := strings.Replace(string(encoded), `"partialPublicationRecovered":true`,
		`"partialPublicationRecovered":false`, 1)
	require.NotEqual(t, string(encoded), mutated)
	path := filepath.Join(t.TempDir(), "benchmark.json")
	require.NoError(t, os.WriteFile(path, []byte(mutated), 0o600))
	arguments[len(arguments)-1] = path
	require.Error(t, command.RunNative(arguments))
}
