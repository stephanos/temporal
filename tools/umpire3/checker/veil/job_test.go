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
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func TestNormalizeJobReceiptRecordsReconstructedInvariantProof(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocolchecker.BackendJobInvariant)
	receipt.Axioms = []string{"Classical.choice", "Quot.sound", "propext"}

	result, err := normalizeJobReceipt(view, binding, protocolchecker.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocolcatalog.ResultClassInvariantProved, result.ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeReconstructedSolverProof, result.TrustBadge)
	require.Equal(t, protocolchecker.BackendTerminationGoalsClosed, result.Termination)
	require.True(t, result.Exact)
	require.Equal(t, receipt.Axioms, result.Axioms)
	require.Equal(t, binding.ArtifactDigest, result.BindingArtifactDigest)
	require.NoError(t, result.Validate())
}

func TestNormalizeJobReceiptRecordsSymbolicSolverTrust(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocolchecker.BackendJobSymbolicTrace)

	result, err := normalizeJobReceipt(view, binding, protocolchecker.BackendJobSymbolicTrace,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocolcatalog.ResultClassBoundedSafe, result.ResultClass)
	require.Equal(t, protocolchecker.BackendBounds{Depth: 6}, result.Bounds)
	require.Equal(t, protocolcatalog.TrustBadgeTrustedSolver, result.TrustBadge)
	require.NoError(t, result.Validate())
}

func TestNormalizeJobReceiptRejectsCompiledBindingMismatch(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocolchecker.BackendJobInvariant)
	receipt.Binding.ModuleName = "WrongModule"

	_, err := normalizeJobReceipt(view, binding, protocolchecker.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocolexperiment.DefaultDecodeLimit)
	require.ErrorContains(t, err, "compiled Veil binding")
}

func TestNormalizeJobReceiptRejectsAdmittedReconstructedInvariant(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocolchecker.BackendJobInvariant)
	receipt.Axioms = []string{"sorryAx"}

	_, err := normalizeJobReceipt(view, binding, protocolchecker.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocolexperiment.DefaultDecodeLimit)
	require.ErrorContains(t, err, "reconstructed Veil invariant contains sorryAx")
}

func TestNormalizeJobReceiptRecordsTrustedInvariantAxioms(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound-trusted.json")
	receipt := testJobReceipt(binding, protocolchecker.BackendJobInvariant)
	receipt.Axioms = []string{"Classical.choice", "Quot.sound", "propext", "sorryAx"}

	result, err := normalizeJobReceipt(view, binding, protocolchecker.BackendJobInvariant,
		strings.NewReader(encodeJobReceipt(t, receipt)),
		protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, protocolcatalog.TrustBadgeTrustedSolver, result.TrustBadge)
	require.Equal(t, receipt.Axioms, result.Axioms)
}

func TestRunJobBindsReceiptToCompiledEvidenceAndRequestedJob(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	receipt := testJobReceipt(binding, protocolchecker.BackendJobInvariant)
	receipt.Axioms = []string{"Classical.choice", "Quot.sound", "propext"}

	result, err := RunJob(context.Background(),
		explicitTestEnvironment([]string{
			"UMPIRE3_VEIL_JOB_HELPER=1",
			"UMPIRE3_VEIL_JOB_RECEIPT=" + encodeJobReceipt(t, receipt),
			"UMPIRE3_VEIL_JOB_SEMANTIC_HASH=" + view.SemanticHash,
		}, os.Args[0], "-test.run=^TestJobReceiptHelper$", "--"),
		view, binding, protocolchecker.BackendJobInvariant)
	require.NoError(t, err)
	require.Equal(t, protocolcatalog.ResultClassInvariantProved, result.ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeReconstructedSolverProof, result.TrustBadge)
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
	job protocolchecker.BackendJob,
) jobReceipt {
	receipt := jobReceipt{
		FormatVersion: veilJobReceiptFormatVersion, BackendRevision: protocolchecker.VeilBackendRevision,
		Binding: binding.Binding, Job: job, Axioms: []string{},
	}
	switch job {
	case protocolchecker.BackendJobSymbolicTrace:
		receipt.Status = protocolchecker.BackendTerminationBoundedSafe
		receipt.Depth = binding.Binding.View.Bounds.SymbolicDepth
		receipt.TrustBadge = protocolcatalog.TrustBadgeTrustedSolver
	case protocolchecker.BackendJobInvariant:
		receipt.Status = protocolchecker.BackendTerminationGoalsClosed
		if binding.Binding.TrustMode == ReconstructedSMT {
			receipt.TrustBadge = protocolcatalog.TrustBadgeReconstructedSolverProof
		} else {
			receipt.TrustBadge = protocolcatalog.TrustBadgeTrustedSolver
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
