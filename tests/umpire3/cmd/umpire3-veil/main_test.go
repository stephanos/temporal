package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunRejectsUnsupervisedRawConcretePromotion(t *testing.T) {
	temporary := t.TempDir()
	output := filepath.Join(temporary, "result.json")

	err := run([]string{
		"-operation", "normalize",
		"-input", "../../protocol/generated/nexus-cancellation.first-order.json",
		"-output", output,
	})
	require.ErrorContains(t, err, `unknown operation "normalize"`)
}

func TestRunRejectsRawJobReceiptPromotion(t *testing.T) {
	temporary := t.TempDir()
	err := run([]string{
		"-operation", "normalize-job",
		"-input", "../../protocol/generated/nexus-cancellation.first-order.json",
		"-output", filepath.Join(temporary, "result.json"),
		"-smt-trust", "reconstructed",
	})
	require.ErrorContains(t, err, `unknown operation "normalize-job"`)
}

func TestRunCheckJobRequiresReceiptCommand(t *testing.T) {
	err := run([]string{
		"-operation", "check-job",
		"-input", "../../protocol/generated/nexus-cancellation.first-order.json",
		"-output", filepath.Join(t.TempDir(), "result.json"),
		"-job", "invariant",
	})
	require.ErrorContains(t, err, "job-command is required")
}

func TestRunCheckConcreteRequiresBackendCommand(t *testing.T) {
	err := run([]string{
		"-operation", "check-concrete",
		"-input", "../../protocol/generated/nexus-cancellation.first-order.json",
		"-output", filepath.Join(t.TempDir(), "result.json"),
	})
	require.ErrorContains(t, err, "backend-command is required")
}
