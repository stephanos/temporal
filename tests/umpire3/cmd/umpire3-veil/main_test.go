package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestRunNormalizesConcreteExhaustionWithoutCompleteness(t *testing.T) {
	temporary := t.TempDir()
	raw := filepath.Join(temporary, "raw.json")
	output := filepath.Join(temporary, "result.json")
	require.NoError(t, os.WriteFile(raw, []byte(`{
  "explored_states": 26,
  "result": "no_violation_found",
  "termination_reason": {"kind": "explored_all_reachable_states"}
}`), 0o600))

	require.NoError(t, run([]string{
		"-operation", "normalize",
		"-input", "../../protocol/generated/nexus-cancellation.first-order.json",
		"-raw-result", raw,
		"-output", output,
	}))
	encoded, err := os.ReadFile(output)
	require.NoError(t, err)
	result, err := protocol.DecodeBackendResult(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassExternalNoCounterexample, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, result.TrustBadge)
	require.False(t, result.Exact)
}

func TestRunRequiresCanonicalReplayForConcreteViolation(t *testing.T) {
	temporary := t.TempDir()
	raw := filepath.Join(temporary, "raw.json")
	require.NoError(t, os.WriteFile(raw, []byte(`{
  "result": "found_violation",
  "state_fingerprint": "1",
  "trace": {
    "states": [
      {"fields": "<unrepresentable>", "index": 0, "transition": "after_init"},
      {"fields": "<unrepresentable>", "index": 1, "transition": "DispatchTask"}
    ],
    "theory": "<unrepresentable>"
  },
  "violation": {
    "kind": "safety_failure",
    "violates": ["NexusCancellationWonExcludesSuccess"]
  }
}`), 0o600))

	err := run([]string{
		"-operation", "normalize",
		"-input", "../../protocol/generated/nexus-cancellation-mutated.first-order.json",
		"-raw-result", raw,
		"-output", filepath.Join(temporary, "result.json"),
	})
	require.ErrorContains(t, err, "replay-command is required")
}

func TestRunRejectsRawJobReceiptPromotion(t *testing.T) {
	temporary := t.TempDir()
	err := run([]string{
		"-operation", "normalize-job",
		"-input", "../../protocol/generated/nexus-cancellation.first-order.json",
		"-raw-result", filepath.Join(temporary, "receipt.json"),
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
