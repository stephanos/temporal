package veil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestNormalizeJobReceiptRecordsReconstructedInvariantProof(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocol.BackendJobInvariant)
	receipt.Axioms = []string{"Classical.choice", "Quot.sound", "propext"}

	result, err := normalizeJobReceipt(view, binding, protocol.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassInvariantProved, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeReconstructedSolverProof, result.TrustBadge)
	require.Equal(t, protocol.BackendTerminationGoalsClosed, result.Termination)
	require.True(t, result.Exact)
	require.Equal(t, receipt.Axioms, result.Axioms)
	require.Equal(t, binding.ArtifactDigest, result.GeneratedArtifactDigest)
	require.NoError(t, result.Validate())
}

func TestNormalizeJobReceiptRecordsSymbolicSolverTrust(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocol.BackendJobSymbolicTrace)

	result, err := normalizeJobReceipt(view, binding, protocol.BackendJobSymbolicTrace,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassBoundedSafe, result.ResultClass)
	require.Equal(t, protocol.BackendBounds{Depth: 6}, result.Bounds)
	require.Equal(t, protocol.TrustBadgeTrustedSolver, result.TrustBadge)
	require.NoError(t, result.Validate())
}

func TestNormalizeJobReceiptRejectsCompiledBindingMismatch(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocol.BackendJobInvariant)
	receipt.Binding.ModuleName = "WrongModule"

	_, err := normalizeJobReceipt(view, binding, protocol.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.ErrorContains(t, err, "compiled Veil binding")
}

func TestNormalizeJobReceiptRejectsAdmittedReconstructedInvariant(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocol.BackendJobInvariant)
	receipt.Axioms = []string{"sorryAx"}

	_, err := normalizeJobReceipt(view, binding, protocol.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.ErrorContains(t, err, "reconstructed Veil invariant contains sorryAx")
}

func TestNormalizeJobReceiptRecordsTrustedInvariantAxioms(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound-trusted.json")
	receipt := testJobReceipt(binding, protocol.BackendJobInvariant)
	receipt.Axioms = []string{"Classical.choice", "Quot.sound", "propext", "sorryAx"}

	result, err := normalizeJobReceipt(view, binding, protocol.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.TrustBadgeTrustedSolver, result.TrustBadge)
	require.Equal(t, receipt.Axioms, result.Axioms)
}

func TestRunJobBindsReceiptToCompiledEvidenceAndRequestedJob(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocol.BackendJobInvariant)
	receipt.Axioms = []string{"Classical.choice", "Quot.sound", "propext"}

	result, err := RunJob(context.Background(),
		explicitTestEnvironment([]string{
			"UMPIRE3_VEIL_JOB_HELPER=1",
			"UMPIRE3_VEIL_JOB_RECEIPT=" + encodeJobReceipt(t, receipt),
			"UMPIRE3_VEIL_JOB_SEMANTIC_HASH=" + view.SemanticHash,
		}, os.Args[0], "-test.run=^TestJobReceiptHelper$", "--"),
		view, binding, protocol.BackendJobInvariant)
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
		os.Getenv("UMPIRE3_VEIL_JOB_SEMANTIC_HASH"),
		"invariant",
	}) {
		//nolint:revive // The subprocess helper reports malformed invocation through its exit status.
		os.Exit(3)
	}
	fmt.Print(os.Getenv("UMPIRE3_VEIL_JOB_RECEIPT"))
	//nolint:revive // The subprocess helper must not emit the Go test runner's PASS output.
	os.Exit(0)
}

func testJobReceipt(
	binding BindingArtifact,
	job protocol.BackendJob,
) jobReceipt {
	receipt := jobReceipt{
		FormatVersion: veilJobReceiptFormatVersion, BackendRevision: protocol.VeilBackendRevision,
		Binding: binding.Binding, Job: job, Axioms: []string{},
	}
	switch job {
	case protocol.BackendJobSymbolicTrace:
		receipt.Status = protocol.BackendTerminationBoundedSafe
		receipt.Depth = binding.Binding.View.Bounds.SymbolicDepth
		receipt.TrustBadge = protocol.TrustBadgeTrustedSolver
	case protocol.BackendJobInvariant:
		receipt.Status = protocol.BackendTerminationGoalsClosed
		if binding.Binding.TrustMode == ReconstructedSMT {
			receipt.TrustBadge = protocol.TrustBadgeReconstructedSolverProof
		} else {
			receipt.TrustBadge = protocol.TrustBadgeTrustedSolver
		}
	default:
	}
	receipt.Options = []string{"grind+smt", "sequential", "smt-trust=false"}
	if binding.Binding.TrustMode == TrustedSMT {
		receipt.Options[2] = "smt-trust=true"
	}
	return receipt
}

func encodeJobReceipt(t *testing.T, receipt jobReceipt) string {
	t.Helper()
	encoded, err := json.Marshal(receipt)
	require.NoError(t, err)
	return string(encoded)
}
