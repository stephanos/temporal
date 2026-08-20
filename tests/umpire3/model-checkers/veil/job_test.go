package veil

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const reconstructedInvariantReceipt = `{
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
}`

func TestNormalizeJobReceiptRecordsReconstructedInvariantProof(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := GenerateWithTrust(view, Interactive, ReconstructedSMT)
	require.NoError(t, err)

	result, err := NormalizeJobReceipt(view, generated, strings.NewReader(reconstructedInvariantReceipt),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassInvariantProved, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeReconstructedSolverProof, result.TrustBadge)
	require.Equal(t, protocol.BackendTerminationGoalsClosed, result.Termination)
	require.True(t, result.Exact)
	require.Empty(t, result.Axioms)
	require.NoError(t, result.Validate())
}

func TestNormalizeJobReceiptRecordsSymbolicBound(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := GenerateWithTrust(view, Interactive, ReconstructedSMT)
	require.NoError(t, err)
	receipt := strings.Replace(reconstructedInvariantReceipt,
		`"job": "invariant",
  "status": "goals-closed",`,
		`"job": "symbolic-trace",
  "status": "bounded-safe",
  "depth": 6,`, 1)

	result, err := NormalizeJobReceipt(view, generated, strings.NewReader(receipt),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassBoundedSafe, result.ResultClass)
	require.Equal(t, protocol.BackendBounds{Depth: 6}, result.Bounds)
	require.Equal(t, protocol.TrustBadgeReconstructedSolverProof, result.TrustBadge)
	require.NoError(t, result.Validate())
}

func TestNormalizeJobReceiptRejectsTrustMismatch(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := GenerateWithTrust(view, Interactive, ReconstructedSMT)
	require.NoError(t, err)
	receipt := strings.Replace(reconstructedInvariantReceipt,
		`"trustBadge": "reconstructed-solver-proof"`, `"trustBadge": "trusted-solver"`, 1)
	receipt = strings.Replace(receipt, `"smt-trust=false"`, `"smt-trust=true"`, 1)

	_, err = NormalizeJobReceipt(view, generated, strings.NewReader(receipt),
		protocol.DefaultDecodeLimit)
	require.ErrorContains(t, err, "does not match generated Veil trust mode")
}

func TestRunJobBindsReceiptToRequestedViewAndJob(t *testing.T) {
	t.Setenv("UMPIRE3_VEIL_JOB_HELPER", "1")
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := GenerateWithTrust(view, Interactive, ReconstructedSMT)
	require.NoError(t, err)

	result, err := RunJob(context.Background(),
		[]string{os.Args[0], "-test.run=^TestJobReceiptHelper$", "--"},
		view, generated, protocol.BackendJobInvariant)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassInvariantProved, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeReconstructedSolverProof, result.TrustBadge)
}

func TestJobReceiptHelper(t *testing.T) {
	if os.Getenv("UMPIRE3_VEIL_JOB_HELPER") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 || !slices.Equal(os.Args[separator+1:], []string{
		"sha256:91939fb7d186499518ed05a76483a9c378a8fe55ca07d8104ad7d1f9e9380e1a",
		"invariant",
	}) {
		os.Exit(3)
	}
	fmt.Print(reconstructedInvariantReceipt)
	os.Exit(0)
}
