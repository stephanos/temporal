package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/internal/command"
)

func TestRunRejectsVeilSourceGeneration(t *testing.T) {
	output := filepath.Join(t.TempDir(), "generated.lean")
	err := command.RunVeil([]string{
		"-operation", "generate",
		"-input", "../../checker/finite/testdata/generated/nexus-cancellation.first-order.json",
		"-output", output,
	})
	require.ErrorContains(t, err, `unknown operation "generate"`)
	_, statErr := os.Stat(output)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestRunRejectsUnsupervisedRawConcretePromotion(t *testing.T) {
	temporary := t.TempDir()
	output := filepath.Join(temporary, "result.json")

	err := command.RunVeil([]string{
		"-operation", "normalize",
		"-input", "../../checker/finite/testdata/generated/nexus-cancellation.first-order.json",
		"-output", output,
	})
	require.ErrorContains(t, err, `unknown operation "normalize"`)
}

func TestRunRejectsRawJobReceiptPromotion(t *testing.T) {
	temporary := t.TempDir()
	err := command.RunVeil([]string{
		"-operation", "normalize-job",
		"-input", "../../checker/finite/testdata/generated/nexus-cancellation.first-order.json",
		"-output", filepath.Join(temporary, "result.json"),
	})
	require.ErrorContains(t, err, `unknown operation "normalize-job"`)
}

func TestRunCheckJobRequiresReceiptCommand(t *testing.T) {
	err := command.RunVeil([]string{
		"-operation", "check-job",
		"-input", "../../checker/finite/testdata/generated/nexus-cancellation.first-order.json",
		"-output", filepath.Join(t.TempDir(), "result.json"),
		"-job", "invariant",
	})
	require.ErrorContains(t, err, "job-command is required")
}

func TestRunCheckConcreteRequiresBackendCommand(t *testing.T) {
	err := command.RunVeil([]string{
		"-operation", "check-concrete",
		"-input", "../../checker/finite/testdata/generated/nexus-cancellation.first-order.json",
		"-output", filepath.Join(t.TempDir(), "result.json"),
	})
	require.ErrorContains(t, err, "backend-command is required")
}
