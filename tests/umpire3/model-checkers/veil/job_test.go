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
	generated, err := GenerateWithTrust(view, Interactive, ReconstructedSMT)
	require.NoError(t, err)
	receipt := testJobReceipt(view, generated, protocol.BackendJobInvariant)
	receipt.Axioms = []string{"Classical.choice", "Quot.sound", "propext"}

	result, err := normalizeJobReceipt(view, generated, protocol.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassInvariantProved, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeReconstructedSolverProof, result.TrustBadge)
	require.Equal(t, protocol.BackendTerminationGoalsClosed, result.Termination)
	require.True(t, result.Exact)
	require.Equal(t, receipt.Axioms, result.Axioms)
	require.NoError(t, result.Validate())
}

func TestNormalizeJobReceiptRecordsSymbolicSolverTrust(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := GenerateWithTrust(view, Interactive, ReconstructedSMT)
	require.NoError(t, err)
	receipt := testJobReceipt(view, generated, protocol.BackendJobSymbolicTrace)

	result, err := normalizeJobReceipt(view, generated, protocol.BackendJobSymbolicTrace,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassBoundedSafe, result.ResultClass)
	require.Equal(t, protocol.BackendBounds{Depth: 6}, result.Bounds)
	require.Equal(t, protocol.TrustBadgeTrustedSolver, result.TrustBadge)
	require.NoError(t, result.Validate())
}

func TestNormalizeJobReceiptRejectsGeneratedModelMismatch(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := GenerateWithTrust(view, Interactive, ReconstructedSMT)
	require.NoError(t, err)
	receipt := testJobReceipt(view, generated, protocol.BackendJobInvariant)
	receipt.GeneratedModelHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	_, err = normalizeJobReceipt(view, generated, protocol.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.ErrorContains(t, err, "generated Veil module")
}

func TestNormalizeJobReceiptRejectsAdmittedReconstructedInvariant(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := GenerateWithTrust(view, Interactive, ReconstructedSMT)
	require.NoError(t, err)
	receipt := testJobReceipt(view, generated, protocol.BackendJobInvariant)
	receipt.Axioms = []string{"sorryAx"}

	_, err = normalizeJobReceipt(view, generated, protocol.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.ErrorContains(t, err, "reconstructed Veil invariant contains sorryAx")
}

func TestNormalizeJobReceiptRecordsTrustedInvariantAxioms(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := GenerateWithTrust(view, Interactive, TrustedSMT)
	require.NoError(t, err)
	receipt := testJobReceipt(view, generated, protocol.BackendJobInvariant)
	receipt.Axioms = []string{"Classical.choice", "Quot.sound", "propext", "sorryAx"}

	result, err := normalizeJobReceipt(view, generated, protocol.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocol.TrustBadgeTrustedSolver, result.TrustBadge)
	require.Equal(t, receipt.Axioms, result.Axioms)
}

func TestRunJobBindsReceiptToCompiledEvidenceAndRequestedJob(t *testing.T) {
	t.Setenv("UMPIRE3_VEIL_JOB_HELPER", "1")
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := GenerateWithTrust(view, Interactive, ReconstructedSMT)
	require.NoError(t, err)
	receipt := testJobReceipt(view, generated, protocol.BackendJobInvariant)
	receipt.Axioms = []string{"Classical.choice", "Quot.sound", "propext"}
	t.Setenv("UMPIRE3_VEIL_JOB_RECEIPT", encodeJobReceipt(t, receipt))

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
	if separator < 0 || !slices.Equal(os.Args[separator+1:], []string{"invariant"}) {
		os.Exit(3)
	}
	fmt.Print(os.Getenv("UMPIRE3_VEIL_JOB_RECEIPT"))
	os.Exit(0)
}

func testJobReceipt(
	view protocol.FirstOrderView,
	generated GeneratedModule,
	job protocol.BackendJob,
) jobReceipt {
	receipt := jobReceipt{
		FormatVersion: veilJobReceiptFormatVersion, BackendRevision: protocol.VeilBackendRevision,
		ViewFormatVersion: view.FormatVersion, Target: view.Target, Property: view.Property,
		World: view.World, Variant: view.Variant, SemanticHash: view.SemanticHash,
		GeneratedModelHash: generated.ModelHash, Job: job, Axioms: []string{},
	}
	switch job {
	case protocol.BackendJobSymbolicTrace:
		receipt.Status = protocol.BackendTerminationBoundedSafe
		receipt.Depth = view.Bounds.SymbolicDepth
		receipt.TrustBadge = protocol.TrustBadgeTrustedSolver
	case protocol.BackendJobInvariant:
		receipt.Status = protocol.BackendTerminationGoalsClosed
		if generated.TrustMode == ReconstructedSMT {
			receipt.TrustBadge = protocol.TrustBadgeReconstructedSolverProof
		} else {
			receipt.TrustBadge = protocol.TrustBadgeTrustedSolver
		}
	}
	receipt.Options = []string{"grind+smt", "sequential", "smt-trust=false"}
	if generated.TrustMode == TrustedSMT {
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
