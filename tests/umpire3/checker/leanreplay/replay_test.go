package leanreplay

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/checker/finite"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
)

func TestReplayFiniteAndTemporalRequireDigestBoundLeanReceipts(t *testing.T) {
	finiteView, found, err := finite.DefaultFirstOrderView(
		protocolcatalog.TargetIDNexusCancellation, "stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	finiteInput := protocolchecker.TraceReplayInput{
		FormatVersion: protocolchecker.TraceReplayInputFormatVersion,
		Target:        finiteView.Target, Property: finiteView.Property, World: finiteView.World,
		Variant: finiteView.Variant, SemanticHash: finiteView.SemanticHash,
		Actions: []protocolcatalog.ActionKind{
			protocolcatalog.ActionKindDispatchTask,
			protocolcatalog.ActionKindAcquireOwnership,
			protocolcatalog.ActionKindWorkerReturnsSuccess,
			protocolcatalog.ActionKindPersistSuccess,
		},
	}
	finiteDigest, err := finiteInput.Digest()
	require.NoError(t, err)
	finiteReceipt := protocolchecker.TraceReplayReceipt{
		FormatVersion: protocolchecker.TraceReplayReceiptFormatVersion,
		TraceDigest:   finiteDigest, Target: finiteInput.Target, Property: finiteInput.Property,
		World: finiteInput.World, Variant: finiteInput.Variant, SemanticHash: finiteInput.SemanticHash,
		Actions: finiteInput.Actions, Status: protocolchecker.TraceReplayAccepted,
		TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate, Axioms: []string{},
	}
	finiteJSON, err := json.Marshal(finiteReceipt)
	require.NoError(t, err)
	finiteResult, err := ReplayFinite(context.Background(), helperCommand("finite", finiteJSON), finiteInput)
	require.NoError(t, err)
	require.Equal(t, finiteReceipt, finiteResult)

	temporalView, found, err := protocolchecker.DefaultTemporalView("delivery-fairness-removed")
	require.NoError(t, err)
	require.True(t, found)
	temporalInput := protocolchecker.TemporalLassoReplayInput{
		FormatVersion: protocolchecker.TemporalLassoReplayInputFormatVersion,
		Target:        temporalView.Target, Property: temporalView.Property, World: temporalView.World,
		Variant: temporalView.Variant, SemanticHash: temporalView.SemanticHash,
		Lasso: protocolchecker.TemporalLasso{
			States:  []string{"unavailable", "ready"},
			Actions: []protocolcatalog.ActionKind{protocolcatalog.ActionKindRecoverOwner, ""}, LoopStart: 1,
		},
	}
	temporalDigest, err := temporalInput.Digest()
	require.NoError(t, err)
	temporalReceipt := protocolchecker.TemporalLassoReplayReceipt{
		FormatVersion: protocolchecker.TemporalLassoReplayReceiptFormatVersion,
		LassoDigest:   temporalDigest, Target: temporalInput.Target, Property: temporalInput.Property,
		World: temporalInput.World, Variant: temporalInput.Variant,
		SemanticHash: temporalInput.SemanticHash, Lasso: temporalInput.Lasso,
		Status: protocolchecker.TraceReplayAccepted, TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate,
		Axioms: []string{},
	}
	temporalJSON, err := json.Marshal(temporalReceipt)
	require.NoError(t, err)
	temporal, err := ReplayTemporal(context.Background(), helperCommand("temporal", temporalJSON), temporalInput)
	require.NoError(t, err)
	require.Equal(t, temporalReceipt, temporal)
}

func helperCommand(mode string, receipt []byte) []string {
	return []string{"/usr/bin/env", "UMPIRE3_CANONICAL_HELPER=" + mode,
		"UMPIRE3_CANONICAL_RECEIPT=" + string(receipt), os.Args[0],
		"-test.run=^TestCanonicalReplayHelper$", "--"}
}

func TestCanonicalReplayHelper(t *testing.T) {
	mode := os.Getenv("UMPIRE3_CANONICAL_HELPER")
	if mode == "" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		//nolint:revive // The helper process reports malformed protocol input through its exit status.
		os.Exit(3)
	}
	arguments := os.Args[separator+1:]
	switch mode {
	case "finite":
		if len(arguments) != 10 || arguments[1] != string(protocolcatalog.TargetIDNexusCancellation) ||
			arguments[6] != string(protocolcatalog.ActionKindDispatchTask) {
			//nolint:revive // The helper process reports malformed protocol input through its exit status.
			os.Exit(4)
		}
	case "temporal":
		if len(arguments) != 12 || arguments[1] != string(protocolcatalog.TargetIDFoundationDeliverySafety) ||
			arguments[8] != "unavailable" || arguments[10] != string(protocolcatalog.ActionKindRecoverOwner) {
			//nolint:revive // The helper process reports malformed protocol input through its exit status.
			os.Exit(5)
		}
	default:
		//nolint:revive // The helper process reports an unsupported protocol mode through its exit status.
		os.Exit(6)
	}
	fmt.Print(os.Getenv("UMPIRE3_CANONICAL_RECEIPT"))
	//nolint:revive // The helper process must not append the Go test runner's PASS output to its response.
	os.Exit(0)
}
