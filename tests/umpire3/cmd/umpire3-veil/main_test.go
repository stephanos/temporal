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

func TestRunNormalizesReconstructedInvariantReceipt(t *testing.T) {
	temporary := t.TempDir()
	receipt := filepath.Join(temporary, "receipt.json")
	output := filepath.Join(temporary, "result.json")
	require.NoError(t, os.WriteFile(receipt, []byte(`{
  "formatVersion": "umpire3/veil-job-receipt/v1",
  "backendRevision": "300c305e945750ab3fb62de4a79c23161b24da39",
  "viewFormatVersion": "umpire3/first-order-view/v1",
  "target": "nexus-cancellation",
  "property": "nexus.cancellation.won-excludes-success",
  "world": "smoke",
  "variant": "sound",
  "semanticHash": "sha256:91939fb7d186499518ed05a76483a9c378a8fe55ca07d8104ad7d1f9e9380e1a",
  "job": "invariant",
  "status": "goals-closed",
  "trustBadge": "reconstructed-solver-proof",
  "options": ["grind+smt", "sequential", "smt-trust=false"],
  "axioms": []
}`), 0o600))

	require.NoError(t, run([]string{
		"-operation", "normalize-job",
		"-input", "../../protocol/generated/nexus-cancellation.first-order.json",
		"-raw-result", receipt,
		"-output", output,
		"-smt-trust", "reconstructed",
	}))
	encoded, err := os.ReadFile(output)
	require.NoError(t, err)
	result, err := protocol.DecodeBackendResult(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassInvariantProved, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeReconstructedSolverProof, result.TrustBadge)
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
